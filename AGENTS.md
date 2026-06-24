# AGENTS.md

Operating guide for AI agents and automated contributors working in this repo.
The canonical rules live in [CLAUDE.md](CLAUDE.md); this file adds agent-specific workflow.

## Before you start
1. Read [CLAUDE.md](CLAUDE.md) and [agent_plan.md](agent_plan.md) (epics, stories, Sprint board).
2. Pick the story you're implementing; note its `CAL-###` id, acceptance criteria, and deps.
3. Branch: `feature/CAL-###-short-slug`.

## While you work
- Stay inside the hexagonal boundaries (domain imports no adapters/platform).
- Protobuf-first: change `proto/`, run `make proto`, commit regenerated `internal/gen`.
- Keep the build green: `make build && make lint && make test` before committing.
- Maintain **≥80% coverage**; add tests in the same change.
- Honor the UX standards and the **no-fabrication** guardrail.

## Commits & PRs
- Small, logical commits titled `CAL-### imperative summary`.
- End every commit message with:
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- PRs are created from these commits; CI + SonarQube + 1 review gate the merge.

## When you finish a story
- Update its status in `agent_plan.md` (story line + Sprint board + epic roll-up).
- Update docs if the change affects API/proto, workflow, or these guide files.

## AI roles (house convention)
- **Claude** — planning & documentation. **Kimi** — research & analysis. **Codex** — code generation.
- All model access in the app routes through the `LLMClient` port (default: Claude/Anthropic).

## Guardrails to never cross
- No secrets in code or VCS.
- No fabricated candidate skills/experience in the candidate-agent path.
- No domain → infrastructure imports.
- No unrelated changes bundled into a story's PR.
