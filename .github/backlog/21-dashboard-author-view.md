---
title: "feature: dashboard author view (forms from the content schema, save through cmd/validate)"
labels: [feature, content]
---

## Summary

The third dashboard view (backlog 11), and the one that **waits on backlog 10's interaction
schema**: an author view that edits interactions / dialogue / narration as forms rendered from the
published content schema (backlog 18, plus backlog 10's interaction schema), saving through the
`cmd/validate` (backlog 16) validator so it only ever writes content the engine loads. It reads
and writes the **same JSON on disk** the game loads, so an agent can edit content programmatically
while a human reviews it visually — one shared source of truth.

## References

- GDD §13 (content authoring; edit against the interaction schema instead of hand-editing JSON),
  §11 (public/sealed content policy; no un-reviewed player-facing content ships).
- ADR-000 D5.5 (content is inert, hash-addressed data), ADR-001 (no shadow engine — the editor
  writes schema-valid content and validates via `cmd/validate`, never by re-deriving rules).
- Depends on: backlog 10 (interaction schema), backlog 18 (base content schema), backlog 16
  (`cmd/validate` save path). Part of: backlog 11 (dashboard). Governed by: backlog 14's
  content-review merge rule (explicit human sign-off on player-facing prose).

## Pillar

**P5 — goofy surface, rigorous core** and **P4 — legible to pilots.** Authoring stays rigorous:
the tool only ever writes schema-valid content the engine can load.

## Spec

- **Forms from the schema.** Render editing forms from the published content schema (backlog 18 +
  backlog 10's interaction schema): NPC states, transitions, and lines. No hand-authored second
  schema.
- **Save through `cmd/validate`.** Validate on save so it only ever writes content
  `engine.ParseContent` accepts; surface the validator verdict inline (including, once Track E
  lands, the lethal-telegraph rule). Round-trips deterministically — stable key order, LF,
  byte-clean — so it does not churn diffs.
- **Shared source of truth.** Same on-disk JSON the game loads, so agent edits + human visual
  review operate on one file set. Player-facing prose still needs explicit human sign-off
  (backlog 14).
- **Boundary clause (from backlog 11).** The author view reads/writes content data; it may **not**
  compute, judge, or score anything the reducer computes.

## Definition of done

- The author view edits the interaction/content schema and writes back content the engine loads
  unchanged (round-trip test: load → edit → save → `engine.ParseContent` accepts, byte-clean).
- Saves are gated by `cmd/validate`; the verdict shows inline.
- The boundary-audit CI grep passes; content-review sign-off (backlog 14) governs prose merges.

## Determinism impact

**none.** Outside `/engine`. It reads/writes content JSON validated by `cmd/validate`; nothing
enters `CanonicalBytes` or the state hash.

## Anti-scope check

Checked against GDD §12. A content author view — not a mechanic, doom clock, combat, inventory,
game UX, an LLM in the rules path, or repo-secrecy. It respects the public/sealed content split.
