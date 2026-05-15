---
emoji: "🔁"
description: "PR self-test: run the Pydantic AI harness engine on this PR so we can iterate via the job log"
on:
  pull_request:
    types: [opened, synchronize]
    paths:
      - .github/scripts/pydantic-ai-runner
      - .github/scripts/test_pydantic_ai_runner.py
      - .github/workflows/pydantic-ai-pr-selftest.md
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
network: defaults
# We register as the built-in `claude` engine and only override `command`, so
# gh-aw runs its full Claude proxy + credential-injection machinery for us.
# Per the gh-aw auth docs, a custom Anthropic-compatible endpoint is supported
# by setting ANTHROPIC_BASE_URL in engine.env; ANTHROPIC_API_KEY is the
# engine's own provider secret (injected by the AWF api-proxy, excluded from
# the agent container). We map the CanopyWave OpenAI-compatible secret/vars
# onto those names. The harness treats ANTHROPIC_BASE_URL as the
# OpenAI-compatible base URL (CanopyWave speaks the OpenAI protocol).
runtimes:
  uv: {}
engine:
  id: claude
  # gh-aw checks out the repo and (via runtimes.uv) installs uv after
  # checkout, both before the agent step. Use the committed launcher: it
  # locates uv robustly inside the firewall sandbox (where PATH is rebuilt)
  # and emits diagnostics so any startup failure is visible in the log.
  command: .github/scripts/pydantic-ai-runner-launch
  env:
    ANTHROPIC_BASE_URL: ${{ vars.OPENAI_BASE_URL }}
    ANTHROPIC_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    GH_AW_HARNESS_MODEL: ${{ vars.GH_AW_HARNESS_MODEL }}
tools:
  github:
    mode: gh-proxy
    toolsets: [default]
safe-outputs:
  add-comment:
    max: 1
  noop:
timeout-minutes: 15
imports:
  - shared/otel-logfire.md
pre-steps:
  # Setting engine.command makes gh-aw skip ALL engine installation steps
  # (claude_engine.go GetInstallationSteps returns []), which also drops the
  # bundled AWF firewall binary install — so the agent step's `sudo -E awf`
  # fails with "awf: command not found". Re-run gh-aw's own installer (the
  # exact call gh-aw makes for non-custom-command jobs). The helper is staged
  # by the preceding "Setup Scripts" step and needs no repo checkout.
  - name: Install AWF firewall binary (skipped by custom engine.command)
    run: bash "${RUNNER_TEMP}/gh-aw/actions/install_awf_binary.sh" v0.25.46
---

# Pydantic AI Harness PR Self-Test

You are running under the **Pydantic AI harness engine** (not the Claude Code
CLI), backed by an OpenAI-compatible CanopyWave endpoint. This workflow exists
so we can iterate on the harness against a real PR and read the job log.

## Task

Do exactly the following, concisely:

1. Print a clearly recognizable banner line: `PYDANTIC_AI_HARNESS_SELFTEST_OK`.
2. State the repository (`${{ github.repository }}`) and the model you are running on.
3. Use a GitHub tool to fetch the title of this pull request and quote it back.
4. Post one short PR comment confirming the harness ran end-to-end, including
   the banner and the model name.

Keep total output under ~200 words. Do not modify any files.
