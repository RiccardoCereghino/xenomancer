---
title: "content: schema v2 — pond branch and per-seed facts"
labels: [content, feature, determinism]
milestone: sprint-1 — woods to gate
---

## Summary

Extend the Zone 1 content pack from the current two-room stub
(`clearing → forest_path`) to the full vertical-slice map and introduce the
first per-seed hidden fact: the agent's eye color, observed in the still pond.
This is the "plant" half of the slice's plant → noise → recall → lethal-check
loop (GDD §7).

## References

- GDD §3 (the seed is the world; per-seed values, not obscurity, are the
  difficulty), §5.3 (Observation & Recall — the pond/eye-color archetype),
  §7 (Zone 1 content plan).
- ADR-000 D5.3 (all randomness derives from the run seed via sub-seeding),
  D5.5 (content is inert, hash-addressed data), D3 (state = seed + log).

## Spec

### Map

Extend `content/zone1/map.json` to the full topology:

```
clearing → forest_path → gate
                  └────→ still_pond   (branch off forest_path)
```

- `clearing` — start location, exits: `forest_path`. No hazard (GDD §7).
- `forest_path` — exits: `clearing`, `still_pond`, `gate`. (Hazard fuse is
  added in backlog 02; this issue only wires the topology and narration.)
- `still_pond` — exits: `forest_path`. The branch is optional and secretly
  survival-critical (GDD §7).
- `gate` — exits: `forest_path`. (The guard check itself is backlog 03.)

Movement remains a `perform` on `legs` targeting the destination location id
(GDD §5.2). Illegal moves stay a structured rejection (`illegal_move`), never
an error (ADR-000 D1).

### Per-seed fact: eye color

- Add an eye-color fact drawn once per run from a **closed palette**:
  `blue`, `green`, `brown`, `grey`.
- Selection is deterministic from the seed via the vendored rng only:
  `rng.Subseed(seed, "facts.eye_color")` seeds a `SplitMix64`; one `Next()`
  mod `len(palette)` indexes the palette. No `math/rand`, no floats, no map
  iteration (ADR-000 D5). The palette is an ordered slice so the index is
  stable forever.
- `inspect` on the pond reflection at `still_pond` emits a new
  `observed{fact:"eye_color", value:<palette word>}` event. Add the
  `observed` event kind to the engine's event set and narrate it from the
  content pack (a reflection line that names the color).
- Self-inspection in the `clearing` reveals species and hair, **not** eyes
  (GDD §7) — the eye color is discoverable only at the pond.

### Canonical world

Document **seed 0** as the canonical world (GDD §3, §5.3): record the
eye-color value seed 0 produces in the content pack's README or a comment so
knowledge runs and goldens have a fixed, memorizable reference.

## Definition of done

- `map.json` describes `clearing`, `forest_path`, `still_pond`, `gate` with the
  exits above; the stdio shell can walk clearing → forest_path → pond → back →
  gate.
- `inspect` at the pond emits exactly one `observed{eye_color}` event with a
  value from the closed palette; the value is a pure function of the seed.
- The eye color for seed 0 is documented as the canonical value.
- Narration exists for every new location and for the pond observation.
- Determinism guard still passes; no banned imports added under `/engine`.

## Supersedes / goldens

This changes world state, so it **supersedes the existing golden replays** —
regenerate them (`agent/scripted/testdata/golden_replay.json` and any
committed state hash) as part of this change and note the regeneration in the
PR's "Golden hash → superseded" box.

## Out of scope

- The wolf fuse and telegraph ladder on `forest_path` (backlog 02).
- The gate guard interrogation, palette matching, and death reports
  (backlog 03).
- The freeform parser and dictionary (backlog 04) — this issue only adds
  canonical content and events.
