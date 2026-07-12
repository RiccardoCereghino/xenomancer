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

## Expectations: this is a data-gathering run, not the publicity artifact

The **first frontier-model run is a data-gathering run, not the publicity
post.** Its job is to produce raw material and expose tuning gaps — not to be
the thing we ship to the AI/evals community. Setting this expectation up front
keeps the milestone honest (GDD §11 minimum publicity unit is a *good* death
report, not the first one).

Concretely, expect **several tuning iterations between "first model meets the
guard" and "a death report worth posting"** — the knobs likely to need turning:

- **narration verbosity** (GDD §3) — too thin and there is no attention-under-
  noise test; too thick and the model drowns in friction rather than the game;
- **fuse numbers** (the wolf's 12, telegraph points 6/9/11 — all [OPEN] tuning
  values, GDD §5.6, §13) — tuned so a naive agent dies *instructively*, not
  instantly or never;
- **dictionary coverage** (GDD §13) — thin coverage makes the agent fight the
  *parser*, not the *game*; the rejection log is the backlog that closes this.

**The publicity post is gated on the *cause* of the death, not on getting one.**
A postable death is one whose cause is a **game failure — `recall.*` or
`social.*`** (the agent was understood and wrong: it forgot its eye color or
claimed the wrong one, the game's thesis, GDD §5.3/§5.4/P3). A death caused by
**parser friction** (drowning in "I don't understand", or a timeout cascade —
see the friction telemetry, backlog 07) is a **tuning signal, not a publicity
artifact**: it means the dictionary/verbosity need work, and the post waits.

## Definition of done

- The LLM agent plays a full episode via `cmd/run`, with every model reply
  routed through the parser; canonical actions only reach the engine.
- `showcase.yml` runs on `workflow_dispatch`, best-of-3, allowed to flake, and
  uploads replay + death report artifacts.
- No token spend on normal push/PR CI — the hermetic scripted CI is untouched.
- Stdlib-only HTTP client (no new third-party dependency).
- **The showcase captures the full event stream *and* the friction telemetry**
  (the death report's friction block + replay-header `meta`, backlog 07) as
  artifacts — this is the raw data that drives the tuning iterations above and
  tells a game-failure death apart from a parser-friction death.
- **The DoD explicitly records the expectation** that this first run is for data
  gathering, that tuning iterations (verbosity, fuse numbers, dictionary
  coverage) are expected before a postable death, and that the publicity post is
  gated on a `recall.*`/`social.*` cause, not parser friction.

## Setup note

The repo owner must add the **`ANTHROPIC_API_KEY`** secret manually before the
showcase job can run; document this in the workflow and/or CONTRIBUTING.

## Out of scope

- Any model on the rules/parser lethal path — the LLM is strictly the player
  (GDD P1, ADR-000 D5.LLM-quarantine).
- Multi-model matrix / the full publicity post (GDD §11 — that rides on this
  but is its own milestone).
- Local-model bracket and attested runners (GDD §11, later tightening).
