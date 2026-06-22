# Agent Skills

Reusable, SOP-style **Agent Skills** for working on AHRM. Each subfolder is a skill: a
`SKILL.md` (with `name` + `description` frontmatter) plus any supporting docs/scripts.

## Discovery (no setup required)

These follow the open [Agent Skills](https://agentskills.io) standard and live in
`.claude/skills/`, which is auto-discovered by both **Cursor** (desktop and Cloud Agents)
and **Claude Code** — no installation step. The agent reads each skill's `description`
and reaches for it when the task fits; user-invoked ones can also be triggered by typing
`/<skill-name>`.

## Provenance

Vendored (curated subset) from [`mattpocock/skills`](https://github.com/mattpocock/skills)
@ commit `6eeb81b5fcfeeb5bd531dd47ab2f9f2bbea27461`, MIT License (see `LICENSE`).
Only skills relevant to this Go backend were included; TypeScript/UI/personal and the
issue-tracker workflow skills were intentionally left out. To add more later, use the
upstream installer: `npx skills@latest add mattpocock/skills`.

## Included skills

### User-invoked (type `/<name>`)
| Skill | Use it to |
|-------|-----------|
| `grill-me` | Get relentlessly interviewed about a plan/design until every open question is resolved (stateless; no repo writes). |
| `grill-with-docs` | Same grilling, but for a codebase — sharpens terminology and records decisions in `CONTEXT.md` / ADRs as you go. |
| `improve-codebase-architecture` | Scan the codebase for "deepening" opportunities and work through a chosen one. |
| `handoff` | Compact the current conversation into a handoff file so a fresh session/agent can continue. |

### Model-invoked (auto or `/<name>`)
| Skill | Use it to |
|-------|-----------|
| `tdd` | Build features / fix bugs test-first with a red-green-refactor loop and integration-style tests. |
| `diagnosing-bugs` | Disciplined loop for hard bugs / perf regressions: reproduce → minimise → hypothesise → instrument → fix → regression-test. |
| `codebase-design` | Shared discipline/vocabulary for designing deep modules (a lot of behaviour behind a small interface at a clean seam). |
| `domain-modeling` | Actively build/sharpen the project's domain model and keep `CONTEXT.md` + ADRs current. |
| `grilling` | The reusable interview loop behind `grill-me` and `grill-with-docs`. |

## Notes for this repo

- `domain-modeling` and `grill-with-docs` maintain a root `CONTEXT.md` and ADRs under
  `docs/adr/`. AHRM has a meaningful Persian options-trading domain (`اهرم`/`ضهرم`/`طهرم`,
  arbitrage R formula, breadth thrust), which these skills help capture precisely.
- `tdd` and `diagnosing-bugs` pair well with the Go test suite (`make test`) and the
  integration tests (`make test-integration`).
