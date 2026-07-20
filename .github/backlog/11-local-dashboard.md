---
title: "feature: local content + telemetry dashboard (reducer-boundary, shells out to cmd/run)"
labels: [feature]
---

## Summary

A small local web app to manage the game's text as it grows and to see what agents actually
said, in two views: a **telemetry view** (a map of what agents said at each interaction) and,
once backlog 10 settles the interaction schema, an **author view** (edit interactions against
the schema instead of hand-editing JSON). This spec **supersedes** the live-filed #24 by
adding the hard boundary clause below. It is telemetry-first; the author view waits on
backlog 10. The three build phases are tracked as their own issues: backlog 19 (telemetry
view), 20 (showcase runner), 21 (author view).

## References

- GDD §11 (results/telemetry — the "what models did" map), §13 (content authoring; the
  rejection/utterance log is the dictionary's backlog).
- ADR-000 D6 (replay format is the telemetry source — fold the canonical log to attribute
  utterances), D8 (a shell hosts the core; wall-clock stays outside the reducer).
- ADR-001 (engine-exposure contract, backlog 13) — the dashboard is a CLI consumer, never a
  library embedding of `/engine`.
- Supersedes **#24**. Build issues: backlog 19/20/21.

## Pillar

**P4 — built for machines, legible to pilots.** Tooling that makes machine-facing content and
agent behavior legible and manageable. Secondary **P5** — authoring stays rigorous: the tool
only ever writes schema-valid content the engine can load.

## Spec

### Boundary clause (verbatim, non-negotiable)

The dashboard may (a) read/write hash-addressed content data and (b) render the event stream.
It may **not** compute, judge, score, or interpret anything the reducer computes. Episodes are
run by shelling out to `cmd/run`; events are displayed, **never re-derived**. This is the
"no shadow engine" line (ADR-001).

### Deploy

Mac mini, localhost, reached via Tailscale from phone / Steam Machine. Ollama on the same host
for local showcases. No public surface.

### Build order (inside the track)

1. **Telemetry view** (backlog 19) — no schema dependency; can start immediately once the
   trace format (backlog 17, `cmd/trace`) exists.
2. **Showcase runner** (backlog 20) — shells out to `cmd/run` + local Ollama; streams events
   into the telemetry view.
3. **Author view** (backlog 21) — waits on backlog 10's interaction schema and backlog 18's
   published content schema; saves through the backlog 16 (`cmd/validate`) validator.

### Constraints

- **Not on the determinism path.** The app only reads replays/traces and reads/writes content
  JSON; the engine stays dependency-free (ADR-000). Prefer a stdlib Go backend + static
  frontend, or a pure client reading JSON — no heavy framework.
- **Content policy (GDD §11).** Public training-grounds content only; the tool must not bake
  held-out / sealed-season content into the public repo.

## Definition of done

- The two views exist behind the boundary clause; a CI grep audit (see backlog 19–21) proves
  no resolution/judgement logic lives in dashboard code.
- Dashboard-rendered events for an episode are byte-identical to a direct `cmd/run` of the same
  seed + log.
- Documented run instructions; no new engine dependency.

## Determinism impact

**none.** Outside `/engine`. It writes content JSON (validated by `cmd/validate` /
`ParseContent`) and reads replays/traces; nothing enters `CanonicalBytes` or the state hash.

## Anti-scope check

Checked against GDD §12. Tooling, not a game mechanic — not a doom clock, combat, inventory,
or the game UX itself, and it respects the public/sealed content split. The engine stays
untouched.
