"""Local-only PDF preflight/extraction. stdout contains counts, never resume text."""

from __future__ import annotations

import argparse
import json
import logging
from pathlib import Path
import sys

# Parser diagnostics may contain document content; return our own bounded errors.
logging.getLogger("pypdf").setLevel(logging.CRITICAL)


class ResumeError(Exception):
    pass


def extract_pages(source: Path) -> list[dict]:
    try:
        from pypdf import PdfReader
    except ImportError:
        raise ResumeError("Install the lab dependency: python -m pip install -r lab/requirements-pdf.txt") from None
    if not source.is_file():
        raise ResumeError("PDF file not found. Supply an existing local file path.")
    if source.suffix.lower() != ".pdf":
        raise ResumeError("Choose a PDF file; text input is handled by the scout profile flow.")
    try:
        if source.stat().st_size > 20 * 1024 * 1024:
            raise ResumeError("PDF exceeds the 20 MiB resume input limit. Use a smaller export or pasted text.")
        with source.open("rb") as stream:
            reader = PdfReader(stream)
            if reader.is_encrypted:
                raise ResumeError("Encrypted PDF. Supply an unlocked local export, pasted text, or skip the resume.")
            if not 1 <= len(reader.pages) <= 20:
                raise ResumeError("Resume must contain 1 to 20 pages. Use a shorter export or pasted text.")
            pages = [{"page": number, "text": page.extract_text() or ""}
                     for number, page in enumerate(reader.pages, start=1)]
    except ResumeError:
        raise
    except Exception:
        # Do not echo third-party exception text or PDF contents into diagnostics.
        raise ResumeError("PDF could not be read. Try a fresh PDF export, pasted text, or skip the resume.") from None
    if not any(page["text"].strip() for page in pages):
        raise ResumeError("No extractable text. This may be a scanned PDF; supply a text-based export, paste text, or skip.")
    return pages


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true", help="extract in memory and report counts only")
    mode.add_argument("--output", type=Path, help="new private JSON destination; never printed to stdout")
    args = parser.parse_args()
    try:
        pages = extract_pages(args.source)
        if args.output:
            if args.output.resolve() == args.source.resolve():
                raise ResumeError("Output must not replace the source PDF.")
            args.output.parent.mkdir(parents=True, exist_ok=True)
            created = False
            try:
                # Exclusive creation protects an existing private export.
                with args.output.open("x", encoding="utf-8") as stream:
                    created = True
                    json.dump({"schema_version": 1, "pages": pages}, stream, ensure_ascii=False)
                    stream.write("\n")
            except Exception:
                if created:
                    args.output.unlink(missing_ok=True)
                raise
        print(json.dumps({"status": "ok", "pages": len(pages),
                          "text_characters": sum(len(p["text"]) for p in pages),
                          "empty_pages": [p["page"] for p in pages if not p["text"].strip()]}))
        return 0
    except ResumeError as exc:
        print(str(exc), file=sys.stderr)
    except FileExistsError:
        print("Output already exists. Choose a new private output path.", file=sys.stderr)
    except OSError:
        print("Cannot access input or private output. Check local file permissions and available space.", file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main())
