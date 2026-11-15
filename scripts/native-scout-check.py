"""Capture a real Codex -> Harness -> provider run on Windows.

Requires an existing Codex login, READY Harness and approved cloud budget.
All outputs must be in ignored private/: they can contain personal research.
Only the local provider server receives the two dedicated provider tokens.
"""
import argparse
from datetime import datetime, timezone
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess

from lab_env import tokens

spec = importlib.util.spec_from_file_location('transport', Path(__file__).with_name('transport-check.py'))
transport = importlib.util.module_from_spec(spec)
spec.loader.exec_module(transport)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--codex', type=Path, required=True)
    p.add_argument('--codex-home', type=Path, required=True)
    p.add_argument('--profile', type=Path, required=True)
    p.add_argument('--summary', type=Path, required=True)
    p.add_argument('--env-file', type=Path, default=Path('.env'))
    p.add_argument('--bin-dir', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    p.add_argument('--verify-only', action='store_true', help='Check an existing capture without invoking any service')
    a = p.parse_args()
    root = Path(__file__).resolve().parents[1]
    dest = a.out.resolve()
    if not dest.is_relative_to(root/'private'):
        p.error('--out must be within the ignored repository private directory')
    if a.verify_only:
        verify(dest)
        return
    if not a.codex_home.is_dir():
        p.error('--codex-home must already contain your authenticated CLI setup')
    secret_values = tokens(a.env_file)
    summary = json.loads(a.summary.read_text())
    if summary.get('reviewed') is not True:
        p.error('Review the skill summary before running the cloud agent')
    dest.mkdir(parents=True, exist_ok=False)
    bins, profile, skills = a.bin_dir.resolve(), a.profile.resolve(), a.summary.resolve()
    base_env = {k:v for k,v in os.environ.items() if not any(s in k.upper() for s in ('TOKEN','SECRET','API_KEY','AWS_','NOTION','GH_','GITHUB_','VAMO'))}
    # CODEX_HOME contains only the operator's existing login; never copy auth files.
    base_env['CODEX_HOME'] = str(a.codex_home.resolve())
    server_env = dict(base_env, **secret_values)
    records = {'started_at':datetime.now(timezone.utc).isoformat(), 'source_commit':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),
               'binary_sha256':{n:hashlib.sha256((bins/n).read_bytes()).hexdigest() for n in ('oss-scout.exe','nhi-probe.exe','sg-collect.exe')}}
    handles=[]
    processes=[]

    def start(name,args,env):
        log=(dest/(name+'.log')).open('wb'); handles.append(log)
        startup=subprocess.STARTUPINFO(); startup.dwFlags|=subprocess.STARTF_USESHOWWINDOW; startup.wShowWindow=0
        child=subprocess.Popen(args,stdin=subprocess.DEVNULL,stdout=log,stderr=log,env=env,
                               startupinfo=startup,creationflags=subprocess.CREATE_NEW_CONSOLE)
        processes.append((name,child)); return child

    failure=None
    try:
        collector=start('collector',[str(bins/'sg-collect.exe'),'--listen','127.0.0.1:14318','--out',str(dest/'native')],base_env)
        transport.wait_port(14318,collector)
        server=start('tools',[str(bins/'oss-scout.exe'),'serve','--profile',str(profile),'--summary',str(skills),'--listen','127.0.0.1:18766','--out',str(dest/'tools')],server_env)
        transport.wait_port(18766,server)
        proxy=start('http',[str(bins/'nhi-probe.exe'),'proxy','--mode','http','--listen','127.0.0.1:18765','--upstream','http://127.0.0.1:18766/mcp','--agent','agentcore-researcher','--client','local-bridge','--out',str(dest/'http')],base_env)
        transport.wait_port(18765,proxy)
        config={
            'mcp_servers.stackguard.command':str(bins/'nhi-probe.exe'),
            'mcp_servers.stackguard.args':['proxy','--mode','stdio','--agent','codex','--client','codex-cli','--out',str(dest/'stdio'),'--',str(bins/'oss-scout.exe'),'bridge','--profile',str(profile),'--summary',str(skills),'--endpoint','http://127.0.0.1:18765/mcp','--out',str(dest/'bridge')],
            'mcp_servers.stackguard.enabled_tools':['scout_research'],
            'mcp_servers.stackguard.tools.scout_research.approval_mode':'approve',
            'mcp_servers.stackguard.tool_timeout_sec':360,
            'mcp_servers.stackguard.required':True,
            'otel.log_user_prompt':False,'otel.log_agent_responses':False,
            'otel.exporter.otlp-http.endpoint':'http://127.0.0.1:14318/v1/logs',
            'otel.exporter.otlp-http.protocol':'json','analytics.enabled':False,
        }
        args=[str(a.codex.resolve()),'exec','--ignore-user-config','--sandbox','read-only','--skip-git-repo-check','-C',str(dest),'--json']
        for key,value in config.items(): args+=['-c',key+'='+json.dumps(value)]
        args+=['Call only the stackguard scout_research MCP tool exactly once with no arguments. Wait for its result, then summarize the returned checked issues and links. This delegates to the configured cloud researcher. Do not use shell, browse, edit files, retry the tool, or call any other tool. If it fails, report the failure.']
        with (dest/'codex-output.jsonl').open('wb') as out,(dest/'codex-stderr.txt').open('wb') as err:
            result=subprocess.run(args,stdin=subprocess.DEVNULL,stdout=out,stderr=err,env=base_env,timeout=420)
        records['codex_exit']=result.returncode
        if result.returncode: raise RuntimeError('Codex failed; private diagnostics retained')
    except Exception as e:
        failure=e
    finally:
        for name,child in reversed(processes):
            try: records[name+'_shutdown']=transport.stop(child)
            except Exception as e:
                records[name+'_shutdown_error']=str(e)
                if failure is None: failure=e
        for handle in handles: handle.close()
        records['ended_at']=datetime.now(timezone.utc).isoformat()
        (dest/'capture-status.json').write_text(json.dumps(records,indent=2)+'\n')
    if failure: raise failure
    verify(dest)


def verify(dest):
    records=json.loads((dest/'capture-status.json').read_text(encoding='utf-8'))
    cli=[json.loads(x) for x in (dest/'codex-output.jsonl').read_text(encoding='utf-8').splitlines()]
    calls=[v['item'] for v in cli if v.get('type')=='item.completed' and v.get('item',{}).get('type')=='mcp_tool_call']
    if len(calls)!=1 or calls[0].get('status')!='completed' or calls[0].get('server')!='stackguard' or calls[0].get('result',{}).get('isError'):
        raise RuntimeError('Expected one successful native scout call; inspect preserved capture')
    value=json.loads(calls[0]['result']['content'][0]['text'])
    session=value['session']
    report=dest/'bridge'/session
    research=json.loads((report/'recommendations.json').read_text(encoding='utf-8'))
    events=[json.loads(x) for x in (dest/'tools/tool-events.jsonl').read_text(encoding='utf-8').splitlines()]
    credentials={c['id'] for e in events for c in e.get('credential_uses',[])}
    credentials.update(e['nhi'] for e in events if e.get('nhi'))
    if len(credentials)!=2:
        raise RuntimeError('Expected both real provider credentials in the capture')
    manifest=json.loads((report/'manifest.json').read_text())
    if not manifest.get('complete'):
        raise RuntimeError('Research report is partial; retain it without counting a full pass')
    records.update(cloud_session=session,canonical_report=str(report.relative_to(dest)),
                   provider_events=len(events),provider_credential_ids=sorted(credentials),
                   native_cli_item_id=calls[0].get('id'),
                   native_cloud_export_verified=False,
                   correlation_limit='CLI item IDs are not assumed equal to native telemetry call IDs. Inspect explicit source fields before adding cross-source links.')
    (dest/'summary.json').write_text(json.dumps(records,indent=2)+'\n')
    print('Real scout call completed; original private captures and checked report retained. Native cloud export and correlation still require verification.',flush=True)


if __name__=='__main__':
    main()
