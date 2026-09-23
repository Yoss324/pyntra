# Multi-Agent Eino Overview

Pyntra supports several orchestration modes built around CloudWeGo Eino.

## Modes
- `deep`: coordinator plus focused sub-agents for delegated work
- `plan_execute`: planner, executor, and replanning loop
- `supervisor`: orchestrator that routes work and exits cleanly

## Key Files
- `agents/orchestrator.md`
- `agents/orchestrator-plan-execute.md`
- `agents/orchestrator-supervisor.md`
- `internal/multiagent/`

## Implementation Notes
1. Keep prompts short and structured so downstream agents can merge evidence cleanly.
2. Make tool output reproducible and easy to summarize.
3. Favor small, auditable steps over large speculative plans.
4. Record assumptions, stop conditions, and next-agent handoff points in English.
