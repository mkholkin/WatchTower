"""Read Docker Engine statistics once per second; keep raw counters for audit."""
import http.client
import json
import os
from pathlib import Path
import signal
import socket
import time


def io_bytes(stats, operation):
    if 'cgroup_io_stat' in stats:
        total = 0
        for line in stats['cgroup_io_stat'].splitlines():
            fields = dict(field.split('=') for field in line.split()[1:])
            total += int(fields.get(operation[0]+'bytes', 0))
        return total
    entries = stats.get('blkio_stats', {}).get('io_service_bytes_recursive')
    if entries is None:
        return None
    return sum(x['value'] for x in entries if x['op'].lower() == operation)


def sample(current, previous, seconds):
    cpu = current.get('cpu_stats', {})
    before = previous.get('cpu_stats', {})
    usage = cpu.get('cpu_usage', {}).get('total_usage')
    old_usage = before.get('cpu_usage', {}).get('total_usage')
    system = cpu.get('system_cpu_usage', 0) - before.get('system_cpu_usage', 0)
    cpu_percent = None
    if usage is not None and old_usage is not None and system > 0:
        cpu_percent = (usage-old_usage)/system*cpu.get('online_cpus', 1)*100
    memory = current.get('memory_stats', {})
    working = memory.get('usage')
    if working is not None:
        working = max(0, working-memory.get('stats', {}).get('inactive_file', 0))
    result = {'cpu_percent': cpu_percent, 'memory_bytes': working}
    for op in ('read', 'write'):
        now, old = io_bytes(current, op), io_bytes(previous, op)
        result[op+'_bytes_s'] = max(0, now-old)/seconds if now is not None and old is not None and seconds > 0 else None
    return result


class Engine(http.client.HTTPConnection):
    def __init__(self):
        super().__init__('localhost', timeout=3)

    def connect(self):
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.settimeout(self.timeout)
        self.sock.connect('/var/run/docker.sock')


def get(path):
    conn = Engine()
    try:
        conn.request('GET', path)
        response = conn.getresponse()
        data = json.loads(response.read())
        if response.status != 200:
            raise RuntimeError(f'Docker GET {path}: {response.status}')
        return data
    finally:
        conn.close()


def main():
    stopping = False

    def stop(*_):
        nonlocal stopping
        stopping = True

    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    containers = json.loads(os.environ['LAB4_CONTAINERS'])
    containers['collector'] = socket.gethostname()
    cgroups = {}
    for name, container in containers.items():
        identity = get(f'/containers/{container}/json')['Id']
        candidates = [Path(f'/host-cgroup/docker/{identity}/io.stat'),
                      Path(f'/host-cgroup/system.slice/docker-{identity}.scope/io.stat')]
        cgroups[name] = candidates
    previous = {}
    with Path('/artifacts/resources.jsonl').open('w', buffering=1) as output:
        while not stopping:
            tick = time.monotonic()
            for name, container in containers.items():
                stamp = time.time()
                monotonic = time.monotonic()
                try:
                    current = get(f'/containers/{container}/stats?stream=false&one-shot=true')
                    # An empty, readable cgroup v2 io.stat means no charged block I/O.
                    # Docker represents it as null, which alone cannot distinguish missing counters.
                    io_file = next((p for p in cgroups[name] if p.is_file()), None)
                    if io_file is not None:
                        current['cgroup_io_stat'] = io_file.read_text()
                    old_time, old = previous.get(name, (monotonic, {}))
                    row = {'time': stamp, 'clock_offset_s': stamp-monotonic, 'component': name, 'raw': current,
                           **sample(current, old, monotonic-old_time)}
                    previous[name] = (monotonic, current)
                except (OSError, ValueError, RuntimeError, http.client.HTTPException) as error:
                    row = {'time': stamp, 'component': name, 'error': str(error)}
                output.write(json.dumps(row)+'\n')
            time.sleep(max(0, 1-(time.monotonic()-tick)))


if __name__ == '__main__':
    main()
