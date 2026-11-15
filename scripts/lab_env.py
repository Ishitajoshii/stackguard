"""Load only the two dedicated lab tokens from a local dotenv file, without eval."""
from pathlib import Path

TOKEN_NAMES = frozenset(('STACKGUARD_GITHUB_TOKEN', 'STACKGUARD_NOTION_TOKEN'))


def tokens(path):
    result = {}
    for line in Path(path).read_text(encoding='utf-8-sig').splitlines():
        line = line.strip()
        if not line or line.startswith('#'):
            continue
        key, separator, value = line.partition('=')
        key = key.strip()
        if key not in TOKEN_NAMES:
            continue
        value = value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
            value = value[1:-1]
        if not separator or not value or key in result or '\x00' in value:
            raise ValueError('Missing, duplicate, or invalid dedicated token entry')
        result[key] = value
    if set(result) != TOKEN_NAMES:
        raise ValueError('Both dedicated lab tokens are required')
    return result
