# Lab input status

The user supplied the original PDF resume path and dedicated Notion page on 13 November 2026. The PDF exists locally. Its path and the page identifier are stored only in the ignored `profiles/ishita.yaml` file.

The local `prepare` command has now written `profiles/ishita-summary.json` with 15 skill-evidence entries and `reviewed: false`. Nothing was uploaded. The user must review the local summary before a personal research run. An isolated ready configuration and setup metadata are under ignored `private/ready-001/`; its absolute paths were parsed successfully by the probe. That home still needs the user's Codex sign-in. It points to `bin/validated`, which was kept separate while the benchmark binaries ran.

On 14 November, all four ready binaries were rebuilt from `eeeaaa78a3b31c305607e3e630b6d8d303ed26e8`, with executable hashes retained privately in `private/ready-001/builds.json`. The probe version check and local scout doctor completed successfully. Doctor confirmed a valid profile and configured Notion page, and correctly reported missing GitHub/Notion tokens and cloud configuration. This check made no provider calls and does not approve the resume summary or establish live access.

After all full-size rows completed, all four executables were freshly built from clean source `f55aaac3f4fe5a8cf47433674d706fe2779927c0`. Runtime Go sources and module files are unchanged from the previously tested ready build. Both `bin/` executable links and `bin/validated` now resolve to these fresh D: builds; hashes and build information are in ignored `private/ready-001/current-binaries.json`. Hash checks through both link paths, the version check and offline doctor passed. GitHub/Notion tokens and cloud configuration remain absent, and the resume summary remains unapproved. The exact older scan-matrix executable was preserved at `D:/stackguard-data/builds/scan-matrix-preserved/nhi-probe.exe`, with its hash checked against the collected scan evidence. Benchmark processes and their keep-awake helpers have exited.

The selected AWS profile is `default`. On 13 November 2026, `aws login --profile default` completed and `aws sts get-caller-identity --profile default` succeeded. The CLI identity matched the account signed into the AWS console in Edge. Account identifiers are omitted from this tracked note. The current identity is root; a restricted lab role is still required before agent workloads. Model access and the Notion integration token are not yet verified.

The user described submission timing as today or at most tomorrow (13–14 November 2026, Asia/Calcutta); there is no exact cut-off time. Do not invent one.

Large repository/replay data will use `D:\stackguard-data`, selected by the user. C: does not have sufficient free space for those workloads.

The evaluator's acceptance of AgentCore remains unresolved. User-provided inputs do not by themselves establish cloud integration or assignment compliance.
