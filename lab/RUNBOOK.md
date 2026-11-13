# Lab runbook

## Cloud budget

The assignment's total cloud-spend ceiling is approximately **US$5**. Include model inference, AgentCore runtime, CloudWatch logging/querying, storage and any optional compute in that total.

Before the first paid run, select an available model and region, check their current rates, estimate the run, and set explicit invocation, token, iteration and timeout limits. Track cumulative measured or estimated spend with timestamps. A budget alert is not a guaranteed hard cap; stop optional runs before the remaining balance becomes uncertain.

Do not launch EC2 or persistent storage as an implicit fallback. Use the local machine for the initial lab; large-data storage location is still being decided. Terminate unneeded sessions/resources after tests and record cleanup.

No cloud resources have been deployed or paid invocations performed by this project yet. The existing default AWS login must be refreshed before validating account/model access.
