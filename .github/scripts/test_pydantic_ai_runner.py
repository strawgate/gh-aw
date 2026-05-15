"""Tests for the Pydantic AI gh-aw harness (.github/scripts/pydantic-ai-runner).

Offline tests cover the gh-aw compatibility surface (argv tolerance, prompt
recovery, model resolution, MCP-config translation, stream-json schema) and
need no network or credentials.

The live test is skipped unless an OpenAI-compatible endpoint is provided via
env:  GH_AW_HARNESS_LIVE_API_KEY, GH_AW_HARNESS_LIVE_BASE_URL,
GH_AW_HARNESS_LIVE_MODEL.  No credentials are stored in this repo.

Run:  uv run --with pytest pytest .github/scripts/test_pydantic_ai_runner.py
"""

from __future__ import annotations

import importlib.machinery
import importlib.util
import io
import json
import os
import sys
from contextlib import redirect_stdout
from pathlib import Path

import pytest

_SCRIPT = Path(__file__).parent / "pydantic-ai-runner"


def _load_harness():
    # The harness has no .py extension, so use an explicit source loader.
    loader = importlib.machinery.SourceFileLoader("par_harness", str(_SCRIPT))
    spec = importlib.util.spec_from_loader("par_harness", loader)
    mod = importlib.util.module_from_spec(spec)
    loader.exec_module(mod)
    return mod


har = _load_harness()


# The exact argv gh-aw's claude_harness.cjs passes, with the prompt appended
# as the final positional argument.
GHAW_ARGV = [
    "--print",
    "--no-chrome",
    "--allowed-tools",
    "Bash,Read,mcp__github__get_me,mcp__safeoutputs",
    "--debug-file",
    "/tmp/gh-aw/agent-stdio.log",
    "--verbose",
    "--permission-mode",
    "bypassPermissions",
    "--output-format",
    "stream-json",
    "--mcp-config",
    "/tmp/mcp-servers.json",
    "--prompt-file",
    "/tmp/gh-aw/aw-prompts/prompt.txt",
]


def test_parses_full_claude_argv_without_error():
    args, extra = har.parse_args([*GHAW_ARGV, "do the thing"])
    assert args.output_format == "stream-json"
    assert args.mcp_config == "/tmp/mcp-servers.json"
    assert args.prompt_file == "/tmp/gh-aw/aw-prompts/prompt.txt"
    assert args.print is True


def test_unknown_future_claude_flags_are_tolerated():
    # gh-aw / Claude may add flags later; the harness must not crash.
    args, extra = har.parse_args([*GHAW_ARGV, "--some-future-flag", "x", "prompt"])
    assert "--some-future-flag" in extra


def test_prompt_recovered_from_trailing_positional():
    args, extra = har.parse_args([*GHAW_ARGV, "Investigate the failing CI run."])
    assert har.resolve_prompt(args, extra) == "Investigate the failing CI run."


def test_prompt_falls_back_to_prompt_file(tmp_path):
    pf = tmp_path / "prompt.txt"
    pf.write_text("from file", encoding="utf-8")
    args, extra = har.parse_args(["--prompt-file", str(pf)])
    assert har.resolve_prompt(args, extra) == "from file"


def test_prompt_falls_back_to_env(tmp_path, monkeypatch):
    pf = tmp_path / "p.txt"
    pf.write_text("from env path", encoding="utf-8")
    monkeypatch.setenv("GH_AW_PROMPT", str(pf))
    args, extra = har.parse_args(["--print"])
    assert har.resolve_prompt(args, extra) == "from env path"


def test_model_defaults_to_anthropic(monkeypatch):
    for v in ("GH_AW_HARNESS_MODEL", "GH_AW_MODEL_AGENT_CLAUDE", "ANTHROPIC_MODEL", "OPENAI_BASE_URL"):
        monkeypatch.delenv(v, raising=False)
    args, _ = har.parse_args(["--print"])
    model, label = har.build_model(args)
    assert model == "anthropic:claude-sonnet-4-5"
    assert label == "anthropic:claude-sonnet-4-5"


def test_model_openai_prefix_builds_openai_model(monkeypatch):
    monkeypatch.setenv("OPENAI_API_KEY", "test-key")
    monkeypatch.delenv("OPENAI_BASE_URL", raising=False)
    args, _ = har.parse_args(["--model", "openai:gpt-4o-mini"])
    model, label = har.build_model(args)
    assert label == "openai-compatible:gpt-4o-mini"
    assert model.__class__.__name__ in ("OpenAIChatModel", "OpenAIModel")


def test_openai_base_url_triggers_openai_compatible(monkeypatch):
    # An OpenAI-compatible endpoint (CanopyWave/vLLM/Together) with a bare model id.
    monkeypatch.setenv("OPENAI_BASE_URL", "https://inference.canopywave.io/v1")
    monkeypatch.setenv("OPENAI_API_KEY", "test-key")
    monkeypatch.setenv("GH_AW_HARNESS_MODEL", "moonshotai/kimi-k2.6")
    args, _ = har.parse_args(["--print"])
    model, label = har.build_model(args)
    assert label == "openai-compatible:moonshotai/kimi-k2.6"
    assert model.__class__.__name__ in ("OpenAIChatModel", "OpenAIModel")


def test_mcp_missing_config_degrades_gracefully():
    assert har.build_mcp_servers("/no/such/file.json") == []


def test_mcp_translates_stdio_and_http(tmp_path):
    cfg = tmp_path / "mcp.json"
    cfg.write_text(
        json.dumps(
            {
                "mcpServers": {
                    "github": {"command": "docker", "args": ["run"], "env": {"X": "1"}},
                    "safeoutputs": {
                        "type": "http",
                        "url": "http://host.docker.internal:1234",
                        "headers": {"Authorization": "k"},
                    },
                }
            }
        ),
        encoding="utf-8",
    )
    servers = har.build_mcp_servers(str(cfg))
    # Both servers should translate into pydantic-ai MCP server objects.
    assert len(servers) == 2
    names = sorted(s.__class__.__name__ for s in servers)
    assert names == ["MCPServerStdio", "MCPServerStreamableHTTP"]


def test_emit_result_matches_claude_stream_json_schema():
    buf = io.StringIO()
    with redirect_stdout(buf):
        har.emit_result("answer", usage=None, session_id="run-1")
    obj = json.loads(buf.getvalue().strip())
    assert obj["type"] == "result"
    assert obj["subtype"] == "success"
    assert obj["is_error"] is False
    assert obj["result"] == "answer"
    for k in ("input_tokens", "output_tokens", "cache_creation_input_tokens", "cache_read_input_tokens"):
        assert k in obj["usage"]


def test_emit_result_error_subtype():
    buf = io.StringIO()
    with redirect_stdout(buf):
        har.emit_result("boom", usage=None, session_id="run-1", is_error=True)
    obj = json.loads(buf.getvalue().strip())
    assert obj["subtype"] == "error"
    assert obj["is_error"] is True


def test_emit_result_reads_usage_attributes():
    class U:
        input_tokens = 22
        output_tokens = 292

    buf = io.StringIO()
    with redirect_stdout(buf):
        har.emit_result("x", usage=U(), session_id="s")
    usage = json.loads(buf.getvalue().strip())["usage"]
    assert usage["input_tokens"] == 22
    assert usage["output_tokens"] == 292


@pytest.mark.skipif(
    not os.environ.get("GH_AW_HARNESS_LIVE_API_KEY"),
    reason="set GH_AW_HARNESS_LIVE_API_KEY/_BASE_URL/_MODEL to run the live test",
)
def test_live_openai_compatible_endpoint(monkeypatch):
    """End-to-end against a real OpenAI-compatible endpoint, using the exact
    argv gh-aw passes. Credentials come from env only — never committed."""
    monkeypatch.setenv("OPENAI_API_KEY", os.environ["GH_AW_HARNESS_LIVE_API_KEY"])
    monkeypatch.setenv(
        "OPENAI_BASE_URL",
        os.environ.get("GH_AW_HARNESS_LIVE_BASE_URL", "https://inference.canopywave.io/v1"),
    )
    model = os.environ.get("GH_AW_HARNESS_LIVE_MODEL", "moonshotai/kimi-k2.6")
    # Exercise the model path without the MCP gateway (not available outside a
    # gh-aw run): drop the --mcp-config <path> pair from the gh-aw argv.
    argv = [a for a in GHAW_ARGV]
    i = argv.index("--mcp-config")
    del argv[i : i + 2]
    argv += ["--model", f"openai:{model}", "Reply with exactly: HARNESS_OK"]
    monkeypatch.setattr(sys, "argv", ["pydantic-ai-runner", *argv])
    buf = io.StringIO()
    with redirect_stdout(buf):
        rc = har.main()
    assert rc == 0
    lines = [json.loads(x) for x in buf.getvalue().splitlines() if x.strip()]
    result = next(x for x in lines if x["type"] == "result")
    assert result["is_error"] is False
    assert "HARNESS_OK" in result["result"]
    assert result["usage"]["output_tokens"] > 0
