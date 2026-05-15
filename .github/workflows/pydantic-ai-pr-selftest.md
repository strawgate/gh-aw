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
# KNOWN LIMITATION (verified against the compiled lock): gh-aw strips every
# ${{ secrets.* }} reference out of engine.env and adds --exclude-env for it,
# even under strict:false. A bring-your-own `command` harness has no registered
# provider, so the AWF api-proxy has no secret->provider binding to inject
# (that binding is created by built-in engine registration in gh-aw Go code,
# and `provider:` cannot coexist with `command:`). Result: the model
# credential reaches neither the agent container nor the proxy. This workflow
# therefore cannot authenticate to CanopyWave until a built-in `pydantic-ai`
# engine is registered in gh-aw. It is kept as a documented reproduction.
strict: false
engine:
  id: claude
  command: /tmp/gh-aw/bin/pydantic-ai-runner
  env:
    OPENAI_BASE_URL: ${{ vars.OPENAI_BASE_URL }}
    GH_AW_HARNESS_API_KEY: ${{ secrets.OPENAI_API_KEY }}
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
  - name: Install uv and the Pydantic AI harness runner
    run: |
      set -euo pipefail
      python3 -m pip install --quiet --upgrade uv
      mkdir -p /tmp/gh-aw/bin
      install -m 0755 "${GITHUB_WORKSPACE}/.github/scripts/pydantic-ai-runner" /tmp/gh-aw/bin/pydantic-ai-runner
      uv --version
  - name: Run harness unit tests (offline)
    run: |
      set -euo pipefail
      uv run --with pytest --with "pydantic-ai-slim[openai,mcp]>=1.95.1" \
        pytest "${GITHUB_WORKSPACE}/.github/scripts/test_pydantic_ai_runner.py" -q
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
