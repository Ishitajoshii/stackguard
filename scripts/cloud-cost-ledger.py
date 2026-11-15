"""Summarize original smoke-run metadata without treating failed runs as free."""
import argparse
from datetime import datetime, timezone
from decimal import Decimal
import hashlib
import json
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--budget', type=Path, default=Path('results/checks/cloud-budget.json'))
    parser.add_argument('--run', type=Path, action='append', required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    budget = json.loads(args.budget.read_text())
    rates = budget['usd_per_1000_tokens']
    rows = []
    known_model = Decimal(0)
    for run in args.run:
        stream = run/'stream.jsonl'
        data = stream.read_bytes()
        events = [json.loads(line) for line in data.splitlines()]
        # The raw metadata event and the round summary describe the SAME usage.
        # Count only the provider metadata event, never both representations.
        usage = [e['event']['Value']['Usage'] for e in events
                 if e.get('event_type') == '*types.InvokeHarnessStreamOutputMemberMetadata']
        inputs = sum(e['InputTokens'] for e in usage)
        outputs = sum(e['OutputTokens'] for e in usage)
        if any(e.get('CacheReadInputTokens') or e.get('CacheWriteInputTokens') for e in usage):
            raise ValueError('Check cache pricing separately before accounting for this run')
        subtotal = (Decimal(inputs)*Decimal(rates['Input tokens'])
                    + Decimal(outputs)*Decimal(rates['Output tokens']))/Decimal(1000)
        known_model += subtotal
        summary = run/'summary.json'
        rows.append({'run': run.name, 'original_stream_sha256': hashlib.sha256(data).hexdigest(),
                     'passed': summary.exists() and json.loads(summary.read_text()).get('passed') is True,
                     'reported_metadata_events': len(usage),
                     'reported_input_tokens': inputs if usage else None,
                     'reported_output_tokens': outputs if usage else None,
                     'reported_model_estimate_usd': str(subtotal) if usage else None})
    scenario = Decimal(budget['conservative_session_scenario']['total_usd'])
    reserve = Decimal(budget['reserve_logs_storage_tracing_usd'])
    report = {'checked_at': datetime.now(timezone.utc).isoformat(), 'attempts': rows,
              'attempt_count': len(rows), 'known_reported_model_estimate_usd': str(known_model),
              'all_attempts_scenario_plus_reserve_usd': str(scenario*len(rows)+reserve),
              'initial_attempt_limit': budget['initial_session_limit'],
              'remaining_attempts_before_replanning': max(0,budget['initial_session_limit']-len(rows)),
              'final_bill_reconciled': False,
              'limits': 'Estimates using the checked regional rates, not the final bill. Null usage is unknown, not free. The conservative runtime/model scenario is applied to every attempt including failures; actual runtime shape, logs, memory and other service charges still require reconciliation.'}
    with args.out.open('x', encoding='utf-8') as output:
        json.dump(report, output, indent=2)
        output.write('\n')
    print(f'{len(rows)} attempts recorded; {sum(r["passed"] for r in rows)} passed; billing reconciliation remains open.')


if __name__ == '__main__':
    main()
