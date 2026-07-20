---
title: "feature: cmd/trace — replay → structured trace dump (telemetry consumer)"
labels: [feature]
---

## Summary

Add `cmd/trace`: a CLI that takes a replay file and emits a structured JSON trace — per-round
canonical actions, re-derived events, and the terminal outcome. This is the July post-mortem
summarizer made first-class, and the exact format the dashboard's telemetry view (backlog 19)
consumes. Zero engine changes: a pure event-stream **consumer** (ADR-001). **New tool** — today
only `cmd/run` exists.

## References

- GDD §5.7 (death & the post-mortem — the report is a first-class feature the centaur loop runs
  on), §11 (results/telemetry).
- ADR-000 D2 (events are the only seam; the trace re-derives events by folding the canonical log),
  D6 (replay file format — the input), ADR-001 (engine-exposure contract — `cmd/trace` is a CLI
  consumer, never a library embedding).
- Consumed by: backlog 19 (telemetry view). Related: #11 (rejection telemetry in death reports —
  Track E; when it lands, the trace surfaces the friction block too).

## Pillar

**P4 — legible to pilots** and **P1 — the referee is public.** A replay folds to a legible trace
anyone can read, not an opaque log.

## Spec

- **Input:** a replay file (ADR-000 D6). **Output:** JSON with, per round, the canonical action(s),
  the events re-derived by folding the log through the engine, and the terminal outcome
  (won / died-with-cause / — ). Deterministic ordering (ordered rounds/events, no map iteration in
  the emitted structure).
- **Re-derive, never re-invent.** Events come from folding the canonical log through the same
  engine the game uses — the trace must match a direct `cmd/run` of the same seed + log
  byte-for-byte where they overlap. No second interpretation of the rules.
- **Consumer-only.** Lives in `cmd/trace`, outside `/engine`; the engine does not change.

## Definition of done

- `cmd/trace <replay>` emits the per-round canonical-actions + re-derived-events + terminal-outcome
  JSON for a real replay (e.g. a showcase artifact).
- The re-derived events match a direct `cmd/run` of the same seed + log (parity test).
- `/engine` does not import `cmd/trace`; golden unchanged.

## Determinism impact

**none.** Outside `/engine`. It reads a replay and re-derives events by folding; nothing enters
`CanonicalBytes` or the state hash.

## Anti-scope check

Checked against GDD §12. A replay-summarizer CLI — not a mechanic, doom clock, combat, inventory,
player UX, an LLM in the rules path, or repo-secrecy.
