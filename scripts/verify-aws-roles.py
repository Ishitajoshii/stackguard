"""Read-only IAM/STS checks; no inference, deployment, or credential output."""
import argparse
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--account', required=True)
    parser.add_argument('--region', required=True)
    parser.add_argument('--harness-name', default='StackGuardResearcher')
    parser.add_argument('--model', default='amazon.nova-micro-v1:0')
    parser.add_argument('--inference-profile', default='apac.amazon.nova-micro-v1:0')
    parser.add_argument('--provisioning-profile', default='default')
    parser.add_argument('--caller-profile', default='stackguard')
    parser.add_argument('--scope', choices=['caller', 'execution', 'all'], default='all')
    parser.add_argument('--out', required=True)
    args = parser.parse_args()
    output = Path(args.out)
    if output.exists():
        parser.error('--out must be fresh; retain previous evidence')
    env = dict(os.environ, AWS_PAGER='')

    def aws(*command, profile=None):
        result = subprocess.run(
            ['aws', *command, '--profile', profile or args.provisioning_profile,
             '--region', args.region, '--output', 'json'],
            env=env, capture_output=True, text=True, timeout=90)
        if result.returncode:
            raise RuntimeError(f'AWS {command[0]} {command[1]} failed; exit {result.returncode}')
        return json.loads(result.stdout)

    base = f'arn:aws:bedrock-agentcore:{args.region}:{args.account}'
    profile_arn = f'arn:aws:bedrock:{args.region}:{args.account}:inference-profile/{args.inference_profile}'
    model_arn = f'arn:aws:bedrock:{args.region}::foundation-model/{args.model}'
    cases = []

    def case(name, role, action, resource, allow, context=None):
        cases.append((name, role, action, resource, allow, context or {}))

    if args.scope in ('caller', 'all'):
        caller = 'StackGuardCaller'
        identity = aws('sts', 'get-caller-identity', profile=args.caller_profile)
        if not identity['Arn'].startswith(f'arn:aws:sts::{args.account}:assumed-role/{caller}/'):
            raise RuntimeError('Caller profile did not assume the expected role')
        case('named_harness_allowed', caller, 'bedrock-agentcore:InvokeHarness', f'{base}:harness/{args.harness_name}-validation', True)
        case('other_harness_denied', caller, 'bedrock-agentcore:InvokeHarness', f'{base}:harness/OtherProject-validation', False)
        runtime = f'{base}:runtime/test-validation'
        case('tagged_runtime_allowed', caller, 'bedrock-agentcore:InvokeAgentRuntime', runtime, True, {'aws:ResourceTag/Project': 'StackGuard'})
        case('untagged_runtime_denied', caller, 'bedrock-agentcore:InvokeAgentRuntime', runtime, False)
        case('other_project_runtime_denied', caller, 'bedrock-agentcore:InvokeAgentRuntime', runtime, False, {'aws:ResourceTag/Project': 'OtherProject'})
        case('runtime_commands_denied', caller, 'bedrock-agentcore:InvokeAgentRuntimeCommand', runtime, False, {'aws:ResourceTag/Project': 'StackGuard'})
        case('caller_s3_denied', caller, 's3:ListAllMyBuckets', '*', False)
        case('caller_iam_admin_denied', caller, 'iam:CreateRole', f'arn:aws:iam::{args.account}:role/Test', False)
        case('caller_direct_model_denied', caller, 'bedrock:InvokeModel', model_arn, False)
    if args.scope in ('execution', 'all'):
        role = 'StackGuardHarnessExecution'
        context = {'bedrock:InferenceProfileArn': profile_arn}
        for action in ('bedrock:InvokeModel', 'bedrock:InvokeModelWithResponseStream'):
            case(action+'_profile_allowed', role, action, profile_arn, True)
            case(action+'_model_via_profile_allowed', role, action, model_arn, True, context)
        case('direct_model_without_profile_denied', role, 'bedrock:InvokeModel', model_arn, False)
        other_model = f'arn:aws:bedrock:{args.region}::foundation-model/other.model-v1:0'
        case('other_model_denied', role, 'bedrock:InvokeModel', other_model, False, context)
        case('execution_role_chaining_denied', role, 'sts:AssumeRole', f'arn:aws:iam::{args.account}:role/StackGuardCaller', False)
        case('execution_s3_denied', role, 's3:ListAllMyBuckets', '*', False)
        case('execution_iam_admin_denied', role, 'iam:CreateRole', f'arn:aws:iam::{args.account}:role/Test', False)
        case('public_image_token_allowed', role, 'ecr-public:GetAuthorizationToken', '*', True)
        case('ecr_bearer_token_allowed', role, 'sts:GetServiceBearerToken', '*', True, {'sts:AWSServiceName': 'ecr-public.amazonaws.com'})
        case('other_bearer_service_denied', role, 'sts:GetServiceBearerToken', '*', False, {'sts:AWSServiceName': 'codeartifact.amazonaws.com'})

    def check(item):
        name, role, action, resource, allow, context = item
        command = ['iam', 'simulate-principal-policy', '--policy-source-arn',
                   f'arn:aws:iam::{args.account}:role/{role}', '--action-names', action,
                   '--resource-arns', resource]
        if context:
            command += ['--context-entries', json.dumps([
                {'ContextKeyName': key, 'ContextKeyValues': [value], 'ContextKeyType': 'string'}
                for key, value in context.items()])]
        rows = aws(*command)['EvaluationResults']
        if len(rows) != 1:
            raise RuntimeError(f'{name}: unexpected simulator result count')
        row = rows[0]
        specifics = row.get('ResourceSpecificResults', [])
        if specifics:
            matching = [x for x in specifics if x['EvalResourceName'] == resource]
            if len(matching) != 1:
                raise RuntimeError(f'{name}: missing exact resource result')
            decision = matching[0]['EvalResourceDecision']
        else:
            decision = row['EvalDecision']
        passed = (decision == 'allowed') if allow else decision in ('implicitDeny', 'explicitDeny')
        return {'name': name, 'expected': 'allowed' if allow else 'denied',
                'decision': decision, 'passed': passed}

    with ThreadPoolExecutor(max_workers=4) as pool:
        checks = list(pool.map(check, cases))
    report = {'checked_at': datetime.now(timezone.utc).isoformat(), 'scope': args.scope,
              'region': args.region, 'passed': all(x['passed'] for x in checks),
              'caller_sts_verified': args.scope in ('caller', 'all'), 'checks': checks,
              'limits': 'IAM simulation and STS only; no live Harness/model invocation or service-role assumption.'}
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    print(f'{sum(x["passed"] for x in checks)}/{len(checks)} checks passed; report: {output}')
    return 0 if report['passed'] else 1


if __name__ == '__main__':
    raise SystemExit(main())
