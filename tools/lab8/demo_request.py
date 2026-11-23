"""Read registration for the sample monitor retained by the running Lab 8 demo."""
import json
from pathlib import Path
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[2]


def main():
    state = json.loads((ROOT/'build/lab8/demo.json').read_text())
    session = json.loads((Path(state['directory'])/'demo-session.json').read_text())
    request = Request('http://localhost:3088/monitors/'+session['monitor_id']+'/registration',
                      headers={'Authorization': 'Bearer '+session['token']})
    with urlopen(request, timeout=15) as response:
        print(json.dumps(json.load(response), ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
