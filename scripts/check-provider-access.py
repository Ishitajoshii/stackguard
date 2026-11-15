"""Verify only lab-owned tokens with read-only requests; publish no provider content."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, urlopen

from lab_env import tokens


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--env-file', default='.env')
    p.add_argument('--notion-page', required=True)
    p.add_argument('--github-user', required=True)
    p.add_argument('--out', required=True)
    args = p.parse_args()
    dest = Path(args.out)
    if dest.exists():
        p.error('Choose a fresh evidence path')
    page = args.notion_page.replace('-', '')
    if len(page) != 32 or any(c not in '0123456789abcdefABCDEF' for c in page):
        p.error('Invalid Notion page ID')
    values = tokens(args.env_file)
    results = []
    checks = [
        ('github_identity', 'github', 'https://api.github.com/user'),
        ('github_public_repositories', 'github', f'https://api.github.com/users/{args.github_user}/repos?per_page=1'),
        ('github_issue_search', 'github', 'https://api.github.com/search/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22&per_page=1'),
        ('notion_identity', 'notion', 'https://api.notion.com/v1/users/me'),
        ('notion_dedicated_page', 'notion', f'https://api.notion.com/v1/pages/{page}'),
        ('notion_dedicated_children', 'notion', f'https://api.notion.com/v1/blocks/{page}/children?page_size=100')]
    for name, provider, url in checks:
        token = values['STACKGUARD_'+provider.upper()+'_TOKEN']
        headers = {'Authorization': 'Bearer '+token, 'User-Agent': 'stackguard-owned-verification'}
        if provider == 'notion':
            headers['Notion-Version'] = '2025-09-03'
        try:
            with urlopen(Request(url, headers=headers), timeout=30) as response:
                value = json.load(response)
                row = {'check': name, 'status': response.status, 'passed': response.status == 200}
                if name == 'github_identity':
                    row['expected_user_matches'] = value.get('login', '').lower() == args.github_user.lower()
                    row['passed'] &= row['expected_user_matches']
                    row['reported_oauth_scopes'] = response.headers.get('X-OAuth-Scopes', '')
                if name == 'notion_dedicated_children':
                    row['block_count_first_page'] = len(value.get('results', []))
                    row['has_more'] = bool(value.get('has_more'))
        except HTTPError as error:
            row = {'check': name, 'status': error.code, 'passed': False}
        results.append(row)
    report = {'checked_at': datetime.now(timezone.utc).isoformat(),
              'passed': all(x['passed'] for x in results), 'checks': results,
              'credentials': [{'provider': name, 'fingerprint': hashlib.sha256(values[key].encode()).hexdigest(),
                               'binding': key} for name, key in [('github', 'STACKGUARD_GITHUB_TOKEN'), ('notion', 'STACKGUARD_NOTION_TOKEN')]],
              'limits': 'Successful reads do not establish that tokens lack write access or access to other pages. No write operations were tested.'}
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    print(json.dumps({'passed': report['passed'], 'checks': results}))
    return 0 if report['passed'] else 1


if __name__ == '__main__':
    raise SystemExit(main())
