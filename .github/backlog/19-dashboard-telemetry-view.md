---
title: "feature: dashboard telemetry view (consumes cmd/trace output)"
labels: [feature]
---

## Summary

The first dashboard view (backlog 11), and the one with no schema dependency — it can start the
moment `cmd/trace` (backlog 17) merges. It ingests traces and renders a map of **what agents
actually said at each interaction**: every utterance/action bucketed under the interaction node
(location + NPC state) it hit, with counts, so under/over-covered lines are visible. Read-only,
and strictly behind the backlog 11 boundary clause.

## References

- GDD §11 (results/telemetry — the "what models did" map), §13 (the rejection/utterance log is the
  dictionary's backlog).
- ADR-000 D6 (replay format is the telemetry source), D2 (events are the only seam), ADR-001
  (no shadow engine — the view renders `cmd/trace` output, it does not re-derive).
- Consumes: backlog 17 (`cmd/trace`). Part of: backlog 11 (dashboard).

## Pillar

**P4 — legible to pilots.** The map of what models did, made legible so lines can be refined
against real behavior.

## Spec

- **Input:** a folder of replays (and death/win reports), fed through `cmd/trace` (backlog 17) to
  get per-round canonical actions + re-derived events + terminal outcome.
- **Output:** per-interaction aggregation — "at the guard's `await_claim` state, agents said: …",
  with counts — so under/over-covered lines are visible. **Read-only.**
- **Boundary clause (from backlog 11).** The view may render the event stream; it may **not**
  compute, judge, score, or interpret anything the reducer computes. It displays `cmd/trace`
  output; it never re-derives events itself.
- **Boundary audit.** A CI grep (same spirit as the engine's determinism lint) asserts no
  resolution/judgement logic lives in the telemetry-view code.

## Definition of done

- The telemetry view renders the per-interaction utterance map from a real replay set (e.g. the
  showcase artifacts), with counts, read-only.
- The rendered events are byte-identical to `cmd/trace` output (no re-derivation).
- The boundary-audit CI grep passes.

## Determinism impact

**none.** Outside `/engine`. It reads traces and renders; nothing enters `CanonicalBytes` or the
state hash.

## Anti-scope check

Checked against GDD §12. A read-only telemetry view — not a mechanic, doom clock, combat,
inventory, game UX, an LLM in the rules path, or repo-secrecy.
