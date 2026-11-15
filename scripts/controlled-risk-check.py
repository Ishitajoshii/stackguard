"""Reproduce two inventory risks using owned tokens and one real GitHub read.

No synthetic events or fake credentials. Keep --out private until publication review.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
from urllib.request import Request, urlopen

from lab_env import tokens


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--env-file', default='.env')
    p.add_argument('--profile', required=True)
    p.add_argument('--summary', required=True)
    p.add_argument('--out', required=True)
    p.add_argument('--bin-dir', default='bin/validated')
    a = p.parse_args()
    root = Path(a.out).resolve()
    root.mkdir(parents=True, exist_ok=False)
    bins = Path(a.bin_dir).resolve()
    values = tokens(a.env_file)
    base_env = {k: v for k, v in os.environ.items() if k not in values}
    env = dict(base_env, **values)
    spec = importlib.util.spec_from_file_location('transport', Path(__file__).with_name('transport-check.py'))
    transport = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(transport)

    def run(args):
        with (root/'diagnostics.txt').open('ab') as diagnostics:
            r = subprocess.run([str(bins/'nhi-probe.exe'), *args], env=base_env,
                               stdout=diagnostics, stderr=diagnostics, timeout=60)
        if r.returncode:
            raise RuntimeError('Probe operation failed; see private diagnostics')

    inputs = root/'inputs'
    inputs.mkdir()
    files = []
    try:
        for name, selected in [('full', values), ('missing-github', {'STACKGUARD_NOTION_TOKEN': values['STACKGUARD_NOTION_TOKEN']})]:
            path = inputs/(name+'.env')
            path.write_text(''.join(k+'='+v+'\n' for k, v in selected.items()), encoding='utf-8')
            files.append(path)
            run(['scan', '--root', str(path), '--agent', 'github-tool', '--out', str(root/('scan-'+name))])
    finally:
        for path in files:
            path.unlink(missing_ok=True)
    startup = subprocess.STARTUPINFO()
    startup.dwFlags |= subprocess.STARTF_USESHOWWINDOW
    startup.wShowWindow = 0
    server = subprocess.Popen([str(bins/'oss-scout.exe'), 'serve', '--profile', str(Path(a.profile).resolve()),
                               '--summary', str(Path(a.summary).resolve()), '--listen', '127.0.0.1:18776',
                               '--out', str(root/'tools')], env=env, stdin=subprocess.DEVNULL,
                              stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                              startupinfo=startup, creationflags=subprocess.CREATE_NEW_CONSOLE)
    try:
        transport.wait_port(18776, server)
        request = {'jsonrpc': '2.0', 'id': 'controlled-real-read-1', 'method': 'tools/call',
                   'params': {'name': 'github_repo', 'arguments': {'repository': 'golang/go'}}}
        headers = {'Content-Type': 'application/json', 'Accept': 'application/json, text/event-stream',
                   'Mcp-Protocol-Version': '2025-06-18', 'X-Stackguard-Session': 'controlled-original-github-read'}
        with urlopen(Request('http://127.0.0.1:18776/mcp', data=json.dumps(request).encode(), headers=headers), timeout=45) as response:
            result = json.load(response)
        if result.get('error') or result.get('result', {}).get('isError'):
            raise RuntimeError('Real provider read failed')
        (root/'response.json').write_text(json.dumps(result, indent=2)+'\n', encoding='utf-8')
    finally:
        transport.stop(server)
    events = root/'tools'/'tool-events.jsonl'
    rows = [json.loads(x) for x in events.read_text(encoding='utf-8').splitlines()]
    assert len(rows) == 2 and rows[0]['attribution_kind'] == 'direct' and rows[1]['status'] == 'succeeded'
    ids = {name: hashlib.sha256(value.encode()).hexdigest() for name, value in values.items()}
    assert rows[0]['nhi'] == ids['STACKGUARD_GITHUB_TOKEN']
    findings = {}
    for name in ('full', 'missing-github'):
        run(['trace', '--input', 'normalized='+str(events), '--inventory', str(root/('scan-'+name)/'inventory.jsonl'),
             '--out', str(root/('trace-'+name)), '--lineage'])
        findings[name] = [json.loads(x) for x in (root/('trace-'+name)/'findings.jsonl').read_text().splitlines()]
    assert any(x['rule'] == 'unused_in_window' and x['nhi'] == ids['STACKGUARD_NOTION_TOKEN'] for x in findings['full'])
    assert any(x['rule'] == 'unwired_observation' and x['nhi'] == ids['STACKGUARD_GITHUB_TOKEN'] for x in findings['missing-github'])
    summary = {'passed': True, 'real_provider_requests': 1, 'provider_response_success': True,
               'source_sha256': hashlib.sha256(events.read_bytes()).hexdigest(),
               'scenarios': ['Owned Notion token present in the inventory but not used during the GitHub-only window',
                             'Owned GitHub token actually used but deliberately omitted from a separate binding snapshot'],
               'limits': 'Two inventory scenarios over the same original read; no agent delegation claim. Service stopped with Ctrl-C, so its manifest is incomplete; exact request/response pair is independently checked.'}
    (root/'summary.json').write_text(json.dumps(summary, indent=2)+'\n')
    print(json.dumps(summary))


if __name__ == '__main__':
    main()

