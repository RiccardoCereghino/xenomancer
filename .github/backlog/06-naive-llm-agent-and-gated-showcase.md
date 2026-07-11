---
title: "agent: naive LLM agent and gated showcase"
labels: [agent, ci, feature]
milestone: sprint-1 — woods to gate
---

## Summary

Add a naive LLM agent that plays the slice through the parser (the quarantine
proving itself on a real, sloppy input source) and a gated CI showcase job that
runs it best-of-3. This is the Phase-0 "a naive LLM agent dies instructively"
half of the definition of done (GDD §12), and the raw material for the first
publicity post (GDD §11).

## References

- GDD §10 (CI: hermetic scripted tests **plus** a gated live-agent showcase —
  spends tokens, allowed to flake, best-of-3), §11 (publicity: multi-model
  results post; AI stays on the player side), §12 (Phase 0 DoD), §5.2 (freeform
  goes through the quarantined parser).
- ADR-000 D3 (parser outside the replay path), D8 (agent is a subprocess under
  the runner).

## Dependencies

Depends on **backlog 04** (parser + freeform shell input) and **backlog 05**
(the `cmd/run` bidirectional runner). Build on both; do not duplicate them.

## Spec

### The agent

- A **stdlib-only** (`net/http`) client for the Anthropic Messages API — no
  third-party SDK. It is a player agent, not part of the engine or rules path.
- The agent receives observation packets (JSONL from the runner), sends them to
  the model, and replies with the model's **freeform** text. That text goes
  **through the parser** (backlog 04) — this is the quarantine proving itself
  on genuinely sloppy input; misparses become free rejections, never deaths
  (GDD P3).
- Runs as an `--agent` subprocess under `cmd/run` (backlog 05).

### The showcase workflow

- `.github/workflows/showcase.yml`, triggered on **`workflow_dispatch`** only
  (never on push — it spends tokens).
- Uses the repository secret **`ANTHROPIC_API_KEY`**.
- Plays **best-of-3** episodes; the job is **allowed to flake** (a loss or a
  model hiccup is not a red build).
- Uploads the **replay file and the death report** as workflow artifacts — the
  post-mortem is the deliverable and the publicity material (GDD §11).

## Definition of done

- The LLM agent plays a full episode via `cmd/run`, with every model reply
  routed through the parser; canonical actions only reach the engine.
- `showcase.yml` runs on `workflow_dispatch`, best-of-3, allowed to flake, and
  uploads replay + death report artifacts.
- No token spend on normal push/PR CI — the hermetic scripted CI is untouched.
- Stdlib-only HTTP client (no new third-party dependency).

## Setup note

The repo owner must add the **`ANTHROPIC_API_KEY`** secret manually before the
showcase job can run; document this in the workflow and/or CONTRIBUTING.

## Out of scope

- Any model on the rules/parser lethal path — the LLM is strictly the player
  (GDD P1, ADR-000 D5.LLM-quarantine).
- Multi-model matrix / the full publicity post (GDD §11 — that rides on this
  but is its own milestone).
- Local-model bracket and attested runners (GDD §11, later tightening).
