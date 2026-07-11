---
title: "engine: hazard framework and the wolf"
labels: [engine, feature]
milestone: sprint-1 — woods to gate
---

## Summary

Add the first hazard: a per-zone fuse counter with a telegraph ladder,
realized as the Wolf on `forest_path`. This is the slice's one
attention-under-noise test and the reference implementation of the fairness
law "no fuse without a telegraph ladder" (GDD P3, §5.6).

## References

- GDD §5.6 (Hazards: Local Doom Clocks; the Wolf reference table),
  §5.1 (sustained claims / grapple), §5.7 (death report struct),
  P3 (fair doom — telegraphed, attributable, reported).
- ADR-000 D1 (rejection is an event, death is an event), D2 (`telegraph{stage}`,
  `died{report}` are engine events), D5 (integer fuses; no floats/time/map
  iteration).

## Spec

### Fuse

- A per-zone integer fuse counter, incremented once per round while the agent
  is **in-zone** on `forest_path`. Fuse length = **12** rounds (GDD §5.6; the
  exact number is an [OPEN] tuning value, so read it from content, not a
  literal in code).
- **Leaving the zone pauses and resets the fuse** ("the wolf will not pursue
  you past the treeline", GDD §5.6). Re-entering starts it from zero.
- The fuse advances only on rounds actually spent in the zone; rejected
  submissions cost no tick (ADR-000 D1) and so do not advance it.

### Telegraph ladder

Emit `telegraph{stage}` events, woven into normal narration where a skimming
agent will miss them, at these fuse points (content-driven strings):

| Fuse | Stage | Signal |
|---|---|---|
| 6/12 | 1 | A howl, distant, almost decorative. |
| 9/12 | 2 | Rustling. The howl again — no longer decorative. |
| 11/12 | 3 | A low growl, sourced disturbingly locally. |

### The maul

- At **12/12**: the Wolf grapples. Create a **sustained claim on both hands
  and legs** (`hand_left`, `hand_right`, `legs`) — a `Hold` per GDD §5.1.
- The agent has **2 rounds** of `perform` with target `struggle` to break free.
  Two successful struggles release the holds and reset the fuse.
- If the grapple is not broken within 2 rounds → `died{report}` with cause
  `hazard.wolf`.
- While grappled, submissions claiming the held resources for anything other
  than `struggle` are rejected (resource conflict), never silently dropped.

### Death report

Emit the full death-report struct (GDD §5.7) as the terminal packet. For a
wolf death the report's `telegraphs_ignored` (or equivalent) field **lists the
telegraph stages emitted before death** — this is what makes the doom
auditable as fair. Cause = `hazard.wolf`; include the last acknowledged
telegraph stage in `detail`.

## Definition of done

- A scripted agent that lingers on `forest_path` receives stages 1–3 at fuse
  6/9/11 and is grappled at 12.
- Struggling twice within the 2-round window breaks free and resets the fuse;
  failing to kills with `hazard.wolf`.
- Leaving the zone before 12 pauses and resets the fuse (re-entry re-telegraphs
  from stage 1).
- The death report lists the telegraph stages that fired.
- Determinism guard passes; fuse math is integer-only; no map iteration in the
  hazard path.
- Tests cover: each telegraph firing, the grapple, the break, and the death
  class.

## Out of scope

- Any hazard other than the Wolf (suspicion, curfew, practice-clearing) — GDD
  §5.6 lists these for later.
- Whether any hazard should be un-pausable (GDD §5.6 [OPEN]).
- The gate guard and social death classes (backlog 03).
