---
title: "feature: content-authoring + telemetry webapp (edit interactions, map what agents said)"
labels: [feature]
---

> **Retro-filed 2026-07-20 for issue #24** (live-filed 2026-07-18) — closes audit
> Divergence 02. This file mirrors the existing issue so the backlog is complete; repo-sync
> skips it (the title already matches #24) and creates no duplicate.
>
> **Superseded by backlog spec 11 (A3 — local content + telemetry dashboard).** Spec 11 adds the
> hard boundary clause (the tool renders the event stream and reads/writes content, but never
> computes/judges/scores what the reducer computes; episodes run by shelling out to `cmd/run`),
> and breaks the build into backlog 19 (telemetry view), 20 (showcase runner), 21 (author view).
> Build from spec 11; close #24 as superseded once spec 11 materializes.

## Summary

A small local web app to manage the game's text as it grows, with two views: an **author view**
(edit interactions / dialogue / narration against the interaction schema from #22, instead of
hand-editing JSON) and a **telemetry view** (a map of what agents actually said at each
interaction — ingest replays, fold them through the engine, and bucket every agent utterance
under the interaction node it hit). It reads/writes the **same JSON schema on disk** the game
loads, so an agent can edit content programmatically while a human reviews it visually.

## References

- GDD §11 (results/telemetry & publicity — the "what models did" map), §13 (content authoring;
  the rejection/utterance log is the dictionary's backlog).
- ADR-000 D6 (replay format is the telemetry source), D3 (the tool is outside the engine; never on
  the rules/parser lethal path).

## Pillar

**P4 — built for machines, legible to pilots.** Tooling that makes machine-facing content and
agent behavior legible and manageable. Secondary **P5** — authoring stays rigorous (only ever
writes schema-valid content the engine can load).

## Spec

- **Telemetry view (ships first — no schema dependency).** Input: a folder of replays + death/win
  reports; fold each through the engine to reconstruct, per round, the interaction node the agent
  was at and what it said. Output: per-interaction aggregation with counts, read-only.
- **Author view (after #22 settles the schema).** CRUD over the interaction/content JSON; validates
  on save so it only ever writes content `engine.ParseContent` accepts; round-trips deterministically
  (stable key order, LF, byte-clean).
- **Constraints.** Not on the determinism path — the webapp only reads replays and reads/writes
  content JSON; the engine stays dependency-free. Public training-grounds content only (GDD §11).

## Definition of done

- Telemetry view renders the per-interaction utterance map from a real replay set.
- Author view edits the #22 schema and writes back content the engine loads unchanged (round-trip
  test).
- Documented run instructions; no new engine dependency.

## Determinism impact

**none.** Outside `/engine`. It writes content JSON (validated by `ParseContent`) and reads
replays; nothing enters `CanonicalBytes` or the state hash.

## Anti-scope check

Checked against GDD §12. Tooling, not a game mechanic — not a doom clock, combat, inventory, or the
game UX itself, and it respects the public/sealed split. The engine stays untouched.

---

Depends on: #22 (author view needs the interaction schema; telemetry view does not and can start
now). Related: #21, #23.
