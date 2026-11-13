# Local PDF extraction

Use Python with pypdf in the scout/lab layer. The current machine has Python 3.14 and pypdf 6.21.3. Pin the tested package with:

```powershell
python -m pip install -r lab/requirements-pdf.txt
```

The helper will preserve one-based page numbers and extract text locally. No PDF or extracted text is uploaded. Review the resulting skills summary before sending it to a cloud agent.

Text extraction cannot reliably read image-only scans and is not a layout renderer. Encrypted, corrupt, empty and scanned inputs must return an actionable fallback; OCR is outside the initial scope. See the [pypdf extraction documentation](https://pypdf.readthedocs.io/en/stable/user/extract-text.html).

This is a lab dependency, not an additional Go library. Its memory usage is separate from the probe's benchmark measurements. Extracted private text must never be included in the public example data or submission captures.

## Run and verify

```powershell
python lab/extract_resume.py 'C:\path with spaces\resume.pdf' --check
python lab/extract_resume.py 'C:\path with spaces\resume.pdf' --output private/resume-pages.json
python -m unittest discover -s lab -p 'test_extract_resume.py' -v
```

The check mode prints page/character counts only, retains no extracted file and never prints resume text. Explicit output preserves one-based page numbers in private JSON and refuses to overwrite existing files. The source PDF is never modified. Errors exit with code 2 and suppress raw parser diagnostics.

Verification on the user-supplied PDF at a Windows path containing spaces: one page, 4,175 extracted characters, no empty pages. Page numbering was verified as `[1]`. No private text or PDF is stored in this repository. Extraction completeness and semantic skill interpretation still require the later user review flow.
