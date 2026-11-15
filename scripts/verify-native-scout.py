"""Offline correctness oracle for the reviewed 15 November native research capture."""
from collections import Counter
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import argparse


def rows(path):
    return [json.loads(line) for line in path.read_text(encoding='utf-8').splitlines()]


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out',type=Path,required=True)
    args=parser.parse_args()
    capture=Path('captures/native-scout-20261115')
    result=Path('results/native-scout-20261115')
    providers=rows(capture/'providers.jsonl')
    bridge=rows(capture/'bridge-events.jsonl')
    audit=rows(capture/'bridge-audit.jsonl')
    http=rows(capture/'http.jsonl')
    stdio=rows(capture/'stdio.jsonl')
    edges=rows(result/'edges.jsonl')
    root=next(e for e in bridge if e['kind']=='delegation')
    completed=json.loads((capture/'codex-result-link.json').read_text())['payload']
    item=completed['item'];session=json.loads(item['result']['content'][0]['text'])['session']
    assert session==root['session'] and item['status']=='completed'
    local=next(e for e in stdio if e['kind']=='tool_call')
    assert local['session']==root['parent_session'] and local['request_id']==root['parent_request_id']
    link=rows(capture/'correlation.jsonl')[0]
    assert link['session']==local['session'] and link['span']==local['span']
    assert link['parent_session']==completed['thread_id'] and link['parent_span']==item['id']
    native=[]
    for envelope in rows(capture/'codex-native.jsonl'):
        for resource in envelope.get('resourceLogs',[]):
            for scope in resource.get('scopeLogs',[]):
                for record in scope.get('logRecords',[]):
                    a={v['key']:next(iter(v['value'].values()),None) for v in record.get('attributes',[])}
                    if a.get('tool_name')=='scout_research':native.append(a)
    assert len(native)==1 and native[0]['call_id']==item['id'] and native[0]['conversation.id']==completed['thread_id']
    assert str(native[0]['success']).lower()=='true'

    # Hand-reviewed expected operations; native output and proxy requests must
    # independently agree with this list, not just with an implementation fixture.
    tool_counts={'github_profile':1,'github_search':2,'github_shortlist':1}
    requests=[e for e in bridge if e['kind']=='tool_call' and e['agent']=='agentcore-researcher']
    assert Counter(e['target'] for e in requests)==tool_counts
    mappings=[e for e in audit if e['source']=='bridge-correlation']
    assert len(mappings)==4
    for mapping in mappings:
        call=next(e for e in requests if e['span']==mapping['cloud_tool_use_id'])
        downstream=next(e for e in http if e['kind']=='tool_call' and e['request_id']==mapping['downstream_request_id'])
        assert downstream['parent_span']==call['span'] and downstream['parent_session']==session
        assert downstream['target']==call['target']==mapping['tool']
    calls=[e for e in providers if e['kind']=='tool_call']
    assert Counter(e['agent'] for e in calls)=={'github-tool':101,'notion-tool':1}
    assert len(providers)==204 and all(e['status']=='succeeded' for e in providers if e['kind']=='tool_completion')
    assert all(e['parent_request_id'] in {m['downstream_request_id'] for m in mappings} for e in calls)
    ids={e['nhi'] for e in calls}
    inv=rows(capture/'inventory.jsonl')
    assert ids=={e['id'] for e in inv if e['provider'] in ('github','notion')}
    sdk=[e for e in bridge if e.get('target')=='AgentCore.InvokeHarness']
    assert len(sdk)==3
    temporary={c['id'] for e in sdk for c in e['credential_uses'] if not c['id'].startswith('aws_role:')}
    assert len(temporary)==1 and len(ids|temporary)==3
    assert len(edges)==6 and not any(e.get('unresolved') for e in edges)
    assert all(e['path'][:4]==['codex','scout_research','codex','scout_research'] for e in edges)
    assert sum(e['call_count'] for e in edges if e['agent']=='github-tool')==101
    assert sum(e['call_count'] for e in edges if e['agent']=='notion-tool')==1
    assert all(e['attribution_kinds']==['direct'] for e in edges if e['agent'].endswith('-tool'))
    assert all(e['call_count']==3 and e['attribution_kinds']==['implicit'] for e in edges if e['agent']=='local-bridge')

    spans=[json.loads(e['message']) for e in json.loads((capture/'cloudwatch.json').read_text())['events']]
    assert len(spans)==44 and all(e['attributes']['session.id']==session for e in spans)
    contexts={e['session']:e['span'] for e in bridge if e['source']=='bridge-trace-context'}
    assert set(contexts)=={e['traceId'] for e in spans}
    native_tools={e['attributes']['gen_ai.tool.call.id'] for e in spans if e['attributes'].get('gen_ai.tool.name')}
    assert native_tools=={e['span'] for e in requests}
    present={(e['traceId'],e['spanId']) for e in spans}
    missing={(e['traceId'],e['parentSpanId']) for e in spans if e.get('parentSpanId') and (e['traceId'],e['parentSpanId']) not in present}
    assert len(missing)==3
    assert sum(e.get('status',{}).get('code')=='ERROR' for e in spans)==2
    manifest=json.loads((result/'manifest.json').read_text())
    assert manifest['complete'] and manifest['errors']==0 and manifest['records']==277
    assert (result/'quarantine.jsonl').stat().st_size==0 and (result/'findings.jsonl').stat().st_size==0
    report={'checked_at':datetime.now(timezone.utc).isoformat(),'passed':True,'normalized_events':277,
            'provider_calls':{'github':101,'notion':1},'sdk_invocations':3,'cloud_selected_tools':tool_counts,
            'secret_credential_identities':3,'additional_typed_role_principals':1,'native_cloud_spans':44,
            'explicit_native_cloud_tool_id_matches':4,'explicit_native_trace_id_matches':3,
            'credential_paths_resolved':True,'native_ingress_parents_missing':3,'retained_native_optional_memory_errors':2,
            'source_hashes':{f.name:hashlib.sha256(f.read_bytes()).hexdigest() for f in capture.iterdir() if f.is_file()},
            'limits':'Direct/implicit describe credential presence; delegated paths describe operations, not credential transfer. Native/proxy duplicate observations remain visible. Three AWS ingress spans are absent; no missing native parents were fabricated. Two optional memory-read errors are retained from this run; later remediation is separate evidence.'}
    with args.out.open('x',encoding='utf-8') as f:json.dump(report,f,indent=2);f.write('\n')
    print('Native oracle passed: three credentials, explicit source links, 102 provider reads, and disclosed native gaps.')


if __name__=='__main__':main()
