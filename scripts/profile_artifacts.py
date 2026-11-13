"""Validate finalized profile framing and indexed filenames, not CPU semantics."""
import gzip
import json
from pathlib import Path


def profile_files(base):
    base = Path(base)
    index = Path(str(base)+'.segments.jsonl')
    files = []
    if index.exists():
        with index.open(encoding='utf-8') as source:
            for number, line in enumerate(source):
                row = json.loads(line)
                expected = base.name if number == 0 else f'{base.name}.part-{number:06d}'
                if row.get('index') != number or row.get('file') != expected:
                    raise RuntimeError('invalid CPU profile segment index')
                files.append(base.parent / expected)
        if not files:
            raise RuntimeError('no finalized CPU profiles')
        actual = set(base.parent.glob(base.name+'.part-*')) | {base}
        if actual != set(files):
            raise RuntimeError('unindexed CPU profile segment')
    else:
        if any(base.parent.glob(base.name+'.part-*')):
            raise RuntimeError('CPU profile parts exist without their index')
        files = [base]
    for path in files:
        try:
            with gzip.open(path, 'rb') as stream:
                total = 0
                while chunk := stream.read(64*1024):
                    total += len(chunk)
            if not total:
                raise RuntimeError('empty CPU profile')
        except (OSError, EOFError) as exc:
            raise RuntimeError('missing or incomplete CPU profile') from exc
    return files
