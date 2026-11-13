# Lab input status

The user supplied the original PDF resume path and dedicated Notion page on 13 November 2026. The PDF exists locally. Its path and the page identifier are stored only in the ignored `profiles/ishita.yaml` file.

The local `prepare` command has now written `profiles/ishita-summary.json` with 15 skill-evidence entries and `reviewed: false`. Nothing was uploaded. The user must review the local summary before a personal research run. An isolated ready configuration and setup metadata are under ignored `private/ready-001/`; its absolute paths were parsed successfully by the probe. That home still needs the user's Codex sign-in. It points to `bin/validated`, allowing benchmark binaries already running elsewhere to finish without replacement.

The selected AWS profile is `default`. On 13 November 2026, `aws login --profile default` completed and `aws sts get-caller-identity --profile default` succeeded. The CLI identity matched the account signed into the AWS console in Edge. Account identifiers are omitted from this tracked note. The current identity is root; a restricted lab role is still required before agent workloads. Model access and the Notion integration token are not yet verified.

The user described submission timing as today or at most tomorrow (13–14 November 2026, Asia/Calcutta); there is no exact cut-off time. Do not invent one.

Large repository/replay data will use `D:\stackguard-data`, selected by the user. C: does not have sufficient free space for those workloads.

The evaluator's acceptance of AgentCore remains unresolved. User-provided inputs do not by themselves establish cloud integration or assignment compliance.
