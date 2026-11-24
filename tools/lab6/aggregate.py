"""Integrate working set over a shared window and sum measured component costs."""
import math

COMPONENTS = ('api', 'collector', 'otel-collector', 'tempo')


def finite_nonnegative(value):
    return isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value) and value >= 0


def memory_mean(rows, component, start, end):
    if not finite_nonnegative(start) or not finite_nonnegative(end) or end <= start:
        raise ValueError('Invalid memory measurement window')
    selected = [row for row in rows if row.get('component') == component]
    if any(not finite_nonnegative(row.get('time')) for row in selected):
        raise ValueError(f'{component}: invalid memory sample time')
    selected.sort(key=lambda row: row['time'])
    before = [index for index, row in enumerate(selected) if row['time'] <= start]
    after = [index for index, row in enumerate(selected) if row['time'] >= end]
    if not before or not after:
        raise ValueError(f'{component}: memory samples do not bracket measurement window')
    relevant = selected[before[-1]:after[0]+1]
    for row in relevant:
        if row.get('error') or not finite_nonnegative(row['time']) or not finite_nonnegative(row.get('memory_bytes')):
            raise ValueError(f'{component}: invalid memory sample')
    integral = 0
    for left, right in zip(relevant, relevant[1:]):
        duration = right['time']-left['time']
        if not 0 < duration <= 5:
            raise ValueError(f'{component}: duplicate memory time or sampling gap')
        begin, finish = max(start, left['time']), min(end, right['time'])
        if finish <= begin:
            continue
        slope = (right['memory_bytes']-left['memory_bytes'])/duration
        first = left['memory_bytes']+slope*(begin-left['time'])
        last = left['memory_bytes']+slope*(finish-left['time'])
        integral += (first+last)/2*(finish-begin)
    mean = integral/(end-start)
    if not finite_nonnegative(mean):
        raise ValueError(f'{component}: invalid integrated memory mean')
    return mean


def totals(profile, rows):
    cpu, memory = {}, {}
    for component in COMPONENTS:
        value = profile['resources'][component]['cpu_seconds']
        if not finite_nonnegative(value):
            raise ValueError(f'{component}: invalid summary CPU seconds')
        cpu[component] = value
        memory[component] = memory_mean(rows, component, profile['measurement_start'], profile['measurement_end'])
    return {'total_cpu_seconds': sum(cpu.values()),
            'diagnostics_cpu_seconds': sum(cpu[name] for name in COMPONENTS if name != 'api'),
            'total_memory_mean_bytes': sum(memory.values()),
            'diagnostics_memory_mean_bytes': sum(memory[name] for name in COMPONENTS if name != 'api')}
