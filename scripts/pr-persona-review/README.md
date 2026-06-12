# PR persona review (GitHub Actions)

Read-only automated review using `.github/prompts/` personas. Posts a single advisory
PR comment with PRS score and findings. **Never commits or modifies the branch.**

## Setup

1. Add repository secret `PERSONA_REVIEW_API_KEY` (Anthropic or OpenAI API key).
2. Optional repository variables:
   - `PERSONA_REVIEW_PROVIDER` — `anthropic` (default) or `openai`
   - `PERSONA_REVIEW_MODEL` — override model id

## Triggers

- `pull_request` to `main` (same-repo branches only)
- `workflow_dispatch` for manual runs

## Local equivalent

Use the Cursor skill `.cursor/skills/pr-review/SKILL.md` for local branch review
with optional fix loops before push.
