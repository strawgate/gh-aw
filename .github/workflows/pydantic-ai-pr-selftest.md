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
network:
  allowed:
    - defaults
    # CanopyWave host comes from the ${{ vars.OPENAI_BASE_URL }} expression,
    # which gh-aw cannot resolve into the compile-time firewall allowlist, so
    # it must be listed explicitly (not a secret; it already appears in logs).
    - inference.canopywave.io
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
  # The checked-out workspace is mounted no-exec in the AWF sandbox (spawning
  # a repo script gives EACCES). gh-aw's exec-able convention is /tmp/gh-aw/bin
  # — a pre-step stages a launcher there that runs `uv run --script` against
  # the workspace harness (uv READS the file, so no-exec/exec-bit is moot).
  command: /tmp/gh-aw/bin/pydantic-ai-runner-launch
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
  # Stage (not install) a launcher at gh-aw's exec-able /tmp/gh-aw/bin path.
  # uv itself is installed by runtimes.uv; this only writes a wrapper file.
  # It runs `uv run --script` on the workspace harness at agent time, after
  # gh-aw's checkout, so the no-exec workspace mount is not a problem.
  - name: Stage Pydantic AI harness launcher
    run: |
      set -euo pipefail
      mkdir -p /tmp/gh-aw/bin
      cat > /tmp/gh-aw/bin/pydantic-ai-runner-launch <<'WRAP'
      #!/usr/bin/env bash
      set -euo pipefail
      # setup-uv points UV_CACHE_DIR at ${RUNNER_TEMP}/setup-uv-cache, which is
      # not writable by the chrooted sandbox user (UID 1001). Only /tmp/gh-aw
      # is owned by that user, so redirect every uv-writable dir there.
      export UV_CACHE_DIR=/tmp/gh-aw/uv/cache
      export UV_PYTHON_INSTALL_DIR=/tmp/gh-aw/uv/python
      export UV_TOOL_DIR=/tmp/gh-aw/uv/tool
      export XDG_DATA_HOME=/tmp/gh-aw/uv/data
      export XDG_CACHE_HOME=/tmp/gh-aw/uv/xdg-cache
      mkdir -p "$UV_CACHE_DIR" "$UV_PYTHON_INSTALL_DIR" "$UV_TOOL_DIR" "$XDG_DATA_HOME" "$XDG_CACHE_HOME"
      runner="${GITHUB_WORKSPACE}/.github/scripts/pydantic-ai-runner"
      echo "[harness-launch] cwd=$(pwd) GITHUB_WORKSPACE=${GITHUB_WORKSPACE:-unset} UV_CACHE_DIR=${UV_CACHE_DIR}" >&2
      echo "[harness-launch] runner=${runner} exists=$([ -f "${runner}" ] && echo yes || echo no)" >&2
      uv_bin=""
      if command -v uv >/dev/null 2>&1; then
        uv_bin="$(command -v uv)"
      else
        for c in "${HOME}/.local/bin/uv" "${RUNNER_TOOL_CACHE:-/opt/hostedtoolcache}"/uv/*/*/uv /opt/hostedtoolcache/uv/*/*/uv /home/runner/work/_tool/uv/*/*/uv /usr/local/bin/uv; do
          [ -x "$c" ] && uv_bin="$c" && break
        done
      fi
      if [ -z "${uv_bin}" ]; then
        echo "[harness-launch] FATAL: uv not found; PATH=${PATH}" >&2
        exit 127
      fi
      echo "[harness-launch] using uv=${uv_bin}" >&2
      exec "${uv_bin}" run --script "${runner}" "$@"
      WRAP
      chmod +x /tmp/gh-aw/bin/pydantic-ai-runner-launch
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
