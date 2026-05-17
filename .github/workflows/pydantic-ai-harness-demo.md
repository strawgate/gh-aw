---
emoji: "🧬"
description: "Prototype: run a Pydantic AI harness as the gh-aw engine instead of the Claude Code CLI (drop-in via engine.command)"
on:
  workflow_dispatch:
permissions:
  contents: read
network: defaults
engine:
  id: claude
  command: /tmp/gh-aw/bin/pydantic-ai-runner
tools:
  github:
    mode: gh-proxy
    toolsets: [default]
safe-outputs:
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
      # Warm the dependency cache so the first agent invocation is not slowed
      # by uv resolving the PEP 723 inline dependencies.
      uv --version
---

# Pydantic AI Harness Prototype

This workflow does not use the Claude Code CLI. Instead, gh-aw's `engine.command`
override points the agent invocation at a Pydantic AI based harness
(`.github/scripts/pydantic-ai-runner`) that accepts the Claude Code CLI argument
surface, reads the gh-aw prompt, bridges gh-aw's MCP gateway (GitHub tools plus
the `safeoutputs` write-sink), and emits Claude-compatible stream-json.

## Task

Report the repository name and confirm which engine harness is running.

- Repository: ${{ github.repository }}
