---
emoji: "🧪"
description: "Demo: fetch an agent prompt from Logfire managed variables at runtime (no recompile/commit needed to change the prompt)"
on:
  workflow_dispatch:
permissions:
  contents: read
network: defaults
engine: claude
tools:
  cli-proxy: true
safe-outputs:
  noop:
timeout-minutes: 10

jobs:
  fetch_dynamic_prompt:
    runs-on: ubuntu-latest
    timeout-minutes: 5
    permissions:
      contents: none
    outputs:
      dynamic_prompt: ${{ steps.logfire.outputs.dynamic_prompt }}
    steps:
      - name: Fetch dynamic prompt from Logfire managed variables
        id: logfire
        env:
          LOGFIRE_READ_KEY: ${{ secrets.LOGFIRE_READ_EXTERNAL_VARIABLES_KEY }}
          LOGFIRE_API_HOST: logfire-api.pydantic.dev
          LOGFIRE_VARIABLE_KEY: gh_aw_demo_prompt
          TARGETING_KEY: gh-aw-${{ github.repository }}
        run: |
          set -euo pipefail

          if [ -z "${LOGFIRE_READ_KEY:-}" ]; then
            echo "::error::LOGFIRE_READ_EXTERNAL_VARIABLES_KEY secret is not set"
            exit 1
          fi

          RESPONSE="$(curl --fail --silent --show-error \
            --max-time 20 \
            -X POST \
            -H "Authorization: Bearer ${LOGFIRE_READ_KEY}" \
            -H "Content-Type: application/json" \
            -d "{\"context\":{\"targetingKey\":\"${TARGETING_KEY}\"}}" \
            "https://${LOGFIRE_API_HOST}/v1/ofrep/v1/evaluate/flags/${LOGFIRE_VARIABLE_KEY}")"

          PROMPT="$(printf '%s' "$RESPONSE" | jq -er '.value')"

          if [ -z "$PROMPT" ] || [ "$PROMPT" = "null" ]; then
            REASON="$(printf '%s' "$RESPONSE" | jq -r '.reason // "UNKNOWN"')"
            echo "::error::Logfire returned no value for ${LOGFIRE_VARIABLE_KEY} (reason: ${REASON})"
            exit 1
          fi

          {
            echo "dynamic_prompt<<__GH_AW_DYNAMIC_PROMPT_EOF__"
            printf '%s\n' "$PROMPT"
            echo "__GH_AW_DYNAMIC_PROMPT_EOF__"
          } >> "$GITHUB_OUTPUT"

          echo "Loaded dynamic prompt (${#PROMPT} chars) from Logfire variable '${LOGFIRE_VARIABLE_KEY}'."
---

# Dynamic Prompt Demo

The instructions below are loaded at run time from a Logfire managed variable,
so the prompt can be edited (and A/B-tested or rolled back) from the Logfire UI
without recompiling or committing this workflow.

## Static context

- Repository: ${{ github.repository }}

## Dynamic instructions

${{ needs.fetch_dynamic_prompt.outputs.dynamic_prompt }}
