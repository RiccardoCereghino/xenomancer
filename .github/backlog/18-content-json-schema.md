---
title: "content: publish machine-readable content JSON Schema (base topology/facts/hazards/narration)"
labels: [content, docs]
---

## Summary

Publish a machine-readable JSON Schema for the content pack, checked into the repo, so the editor
(backlog 21) can render forms and validate client-side without inventing its own truth, and
`cmd/validate` (backlog 16) validates against it. Ship the **base** schema now — topology (map),
facts, hazards, narration — because the interaction-schema portion will churn with backlog 10;
that portion lands **with** backlog 10, not here.

## References

- GDD §3, §5.3, §5.6, §7 (the content a Zone 1 pack carries: map/topology, per-seed facts,
  hazards + telegraph ladders, narration), §11 (public/sealed content policy), §13 (content is
  authored as data).
- ADR-000 D5.5 (content is inert, hash-addressed data), ADR-001 (the schema is the shared truth
  the CLIs and the editor consume — no shadow engine).
- Consumed by: backlog 16 (`cmd/validate`), backlog 21 (author view). Interaction schema: backlog
  10 (lands there).

## Pillar

**P1 — the referee is singular** and **P4 — legible to pilots.** One published schema is the
single source of truth for what a valid content pack is.

## Spec

- A JSON Schema (checked in, e.g. under `content/schema/` or `docs/`) describing the **base**
  content pack: map/topology (locations, adjacency, inspectables), per-seed facts (the pond /
  eye-color archetype), hazards (fuse length, telegraph stages, grapple parameters — content
  data, not code literals), and narration (keyed lines, telegraph rungs, epitaph pools).
- **Parity with the engine's loader.** The schema describes exactly the shape `engine.ParseContent`
  accepts for these sections; `cmd/validate` uses it. It is not a second, drifting definition.
- **Interaction schema explicitly deferred.** The per-NPC interaction state-machine schema
  (`greet → ask → await_claim → judge`, `asked_fact`, transitions keyed on canonical actions)
  is **not** in this base schema — it churns with backlog 10's breaking bump and is published
  there. This spec ships base topology/facts/hazards/narration only.

## Definition of done

- A JSON Schema for the base content pack is checked into the repo and referenced by
  `cmd/validate` (backlog 16).
- Validating the current Zone 1 pack against the schema passes, and it matches
  `engine.ParseContent` acceptance for the covered sections (parity).
- The schema documents that the interaction schema is deferred to backlog 10.

## Determinism impact

**none.** A schema file + its consumers live outside `/engine`. Nothing enters `CanonicalBytes`
or the state hash.

## Anti-scope check

Checked against GDD §12. A published schema — not a mechanic, doom clock, combat, inventory,
player UX, an LLM in the rules path, or repo-secrecy. It respects the public/sealed content split.
