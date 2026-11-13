# Private lab data

Keep credentials, original resumes, extracted resume text and private Notion responses in `private/` or `lab/private/`. Keep generated personal profiles in `profiles/`. These paths are ignored by Git, alongside local `.env` files and `.work/` intermediates.

Use a dedicated lab configuration. Do not copy existing Codex authentication stores or unrelated account credentials into the lab. Example configuration must contain references or placeholders only.

Original redacted submission captures belong in `captures/`; reviewed outputs belong in `results/`. Those directories are deliberately not ignored. Inspect their contents for full credentials and private source material before staging.

An ignore rule does not sanitize a file, remove a previously committed secret, or exclude files from an ordinary filesystem ZIP. Review the explicit submission file list separately.
