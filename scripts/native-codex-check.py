"""Windows: one real Codex file-tool call, with native telemetry and proxy evidence.

Use an ignored private --out directory: CLI diagnostics remain private. This uses
the operator's existing Codex sign-in, never copies credentials, and invokes no AWS.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess

spec = importlib.util.spec_from_file_location("transport", Path(__file__).with_name("transport-check.py"))
transport = importlib.util.module_from_spec(spec)
spec.loader.exec_module(transport)


def attributes(record):
    return {a["key"]: next(iter(a.get("value", {}).values()), None)
            for a in record.get("attributes", [])}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--codex", type=Path, required=True, help="installed native codex.exe")
    p.add_argument("--repo", type=Path, required=True, help="real checkout containing README.md")
    p.add_argument("--out", type=Path, required=True, help="fresh ignored private directory")
    p.add_argument("--bin-dir", type=Path, default=Path("bin/validated"))
    p.add_argument("--transport", choices=["stdio", "http"], default="stdio")
    a = p.parse_args()
    dest, bins, repo = a.out.resolve(), a.bin_dir.resolve(), a.repo.resolve()
    dest.mkdir(parents=True, exist_ok=False)
    capture = dest/"capture"
    collector = transport.start([str(bins/"sg-collect.exe"), "--listen", "127.0.0.1:14318", "--out", str(capture/"native")])
    server = proxy = None
    summary = {"transport": a.transport, "scope": "Original Codex call to a real repository-file tool; no provider credentials or cloud delegation."}
    try:
        transport.wait_port(14318, collector)
        config = {
            "mcp_servers.stackguard_file.tools.repository_file.approval_mode": "approve",
            "mcp_servers.stackguard_file.enabled_tools": ["repository_file"],
            "mcp_servers.stackguard_file.required": True,
            "otel.log_user_prompt": False,
            "otel.log_agent_responses": False,
            "otel.exporter.otlp-http.endpoint": "http://127.0.0.1:14318/v1/logs",
            "otel.exporter.otlp-http.protocol": "json",
            "analytics.enabled": False,
        }
        if a.transport == "stdio":
            config["mcp_servers.stackguard_file.command"] = str(bins/"nhi-probe.exe")
            config["mcp_servers.stackguard_file.args"] = ["proxy", "--mode", "stdio", "--agent", "codex", "--client", "codex-cli", "--out", str(capture/"proxy"), "--", str(bins/"oss-scout.exe"), "serve", "--stdio", "--files", str(repo)]
        else:
            server = transport.start([str(bins/"oss-scout.exe"), "serve", "--files", str(repo), "--listen", "127.0.0.1:18766"])
            transport.wait_port(18766, server)
            proxy = transport.start([str(bins/"nhi-probe.exe"), "proxy", "--mode", "http", "--listen", "127.0.0.1:18765", "--upstream", "http://127.0.0.1:18766/mcp", "--agent", "codex", "--client", "codex-cli", "--out", str(capture/"proxy")])
            transport.wait_port(18765, proxy)
            config["mcp_servers.stackguard_file.url"] = "http://127.0.0.1:18765/mcp"
        args = [str(a.codex.resolve()), "exec", "--ignore-user-config", "--ephemeral", "--sandbox", "read-only", "--skip-git-repo-check", "-C", str(dest), "--json"]
        for key, value in config.items():
            args += ["-c", key+"="+json.dumps(value)]
        args += ["Use only the stackguard_file MCP repository_file tool to inspect README.md. Make exactly one tool call with path README.md, then state the returned byte count and SHA-256. Do not use shell, edit files, browse, or call any other tool. This is a read-only transport validation."]
        env = {k: v for k, v in os.environ.items() if not any(s in k.upper() for s in ("TOKEN", "SECRET", "API_KEY", "AWS_", "NOTION", "GH_", "GITHUB_", "VAMO"))}
        with (dest/"codex-output.jsonl").open("wb") as out, (dest/"codex-stderr.txt").open("wb") as err:
            result = subprocess.run(args, stdin=subprocess.DEVNULL, stdout=out, stderr=err, env=env, timeout=180)
        summary["codex_exit"] = result.returncode
        if result.returncode:
            raise RuntimeError("Codex failed; inspect private diagnostics")
    finally:
        # Each service gets cleanup even if another cleanup fails.
        try:
            if proxy:
                summary["proxy_shutdown"] = transport.stop(proxy)
        finally:
            try:
                if server:
                    transport.stop(server)
            finally:
                summary["collector_shutdown"] = transport.stop(collector)
    cli = [json.loads(line) for line in (dest/"codex-output.jsonl").read_text(encoding="utf-8").splitlines()]
    calls = [v["item"] for v in cli if v.get("type") == "item.completed" and v.get("item", {}).get("type") == "mcp_tool_call"]
    if len(calls) != 1 or calls[0].get("status") != "completed" or calls[0].get("server") != "stackguard_file":
        raise RuntimeError("Expected exactly one successful native MCP call")
    value = json.loads(calls[0]["result"]["content"][0]["text"])
    with (repo/"README.md").open("rb") as source:
        expected = source.read(256*1024)
    if value.get("sha256") != hashlib.sha256(expected).hexdigest() or value.get("bytes_read") != len(expected):
        raise RuntimeError("Native tool returned the wrong repository file hash")
    observations = [json.loads(line) for line in (capture/"proxy/proxy.jsonl").read_text().splitlines()]
    if len(observations) != 2 or [v.get("kind") for v in observations] != ["tool_call", "tool_completion"] or observations[1].get("status") != "succeeded":
        raise RuntimeError("Proxy did not observe exactly one successful call")
    native = [json.loads(line) for line in (capture/"native/native-otlp.jsonl").read_text().splitlines()]
    tool_records = [attributes(r) for d in native for resource in d.get("resourceLogs", []) for scope in resource.get("scopeLogs", []) for r in scope.get("logRecords", []) if attributes(r).get("event.name") == "codex.tool_result" and attributes(r).get("mcp_server") == "stackguard_file"]
    if len(tool_records) != 1 or str(tool_records[0].get("success")).lower() != "true":
        raise RuntimeError("Native telemetry did not confirm the MCP tool result")
    summary.update(tool_result=value, native_tool_result=tool_records[0], proxy_records=len(observations),
                   proxy_manifest_present=(capture/"proxy/manifest.json").exists(),
                   correlation="Native conversation/call IDs and proxy session/request IDs are separate; no cross-source parent link is inferred from this one-call run.")
    (dest/"summary.json").write_text(json.dumps(summary, indent=2)+"\n", encoding="utf-8")
    print(json.dumps(summary), flush=True)


if __name__ == "__main__":
    main()
