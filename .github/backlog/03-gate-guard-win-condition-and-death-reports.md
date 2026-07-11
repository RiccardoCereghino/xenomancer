---
title: "engine: gate guard interrogation, win condition, and death reports"
labels: [engine, content, feature]
milestone: sprint-1 — woods to gate
---

## Summary

Implement the gate guard: the lethal social check that ends the slice. The
guard asks the agent's eye color; a correct claim wins the episode, a wrong one
is a fair death. This is the "lethal check" half of the slice loop and the
first `contextual`/`social` death class (GDD §5.4, §5.7, §7).

## References

- GDD §5.4 (Social Checks — closed-palette keyword matching; no LLM on the
  lethal path), §5.7 (Death & the Post-Mortem struct + epitaphs), §7 (Gate:
  win = inside the walls), P3 (fair death — legal, understood, wrong).
- ADR-000 D2 (`died{report}` event), D4 (death report is the terminal packet),
  D5.3 (seeded variant selection via sub-seeding).

## Spec

### The interrogation

- At `gate`, the guard interrogates via `talk` on `voice`. The question this
  tier is fixed: **eye color** (GDD §7).
- Judgment is closed-palette keyword matching against the same palette as the
  pond fact (`blue`, `green`, `brown`, `grey`), per GDD §5.4:
  - Reply contains **exactly one** palette word → that word is the claim.
  - **Zero or several** palette words → "Speak plainly, stranger." Costs one
    round (one tick), **never kills**.
  - Exactly one, and it **matches** the per-seed eye color → **won**; the
    episode ends successfully.
  - Exactly one, and it is **wrong** → `died{report}` with cause
    `social.claim_wrong`.
- No LLM judges anything on this path — matching is deterministic string
  containment against the ordered palette (GDD §5.4, P1).

### Death report struct

Emit the full struct per GDD §5.7 as the terminal packet:

```json
{
  "cause": "social.claim_wrong",
  "detail": {"npc": "gate_guard", "asked": "eye_color", "claimed": "green", "truth": "grey"},
  "round": 61,
  "telegraphs_ignored": [],
  "ritual_progress": null,
  "epitaph": "He was sure his eyes were green. The pond, unbothered, remains grey."
}
```

- **Epitaph templates live in content**, not code. Selection among variants is
  seeded and deterministic (`rng.Subseed(seed, "narration.epitaph")` or a
  documented domain), so the same seed always yields the same epitaph
  (ADR-000 D5.3). Epitaphs are allowed to be the best writing in the game
  (GDD §5.7).
- The final packet carries **rounds elapsed** and the **outcome**
  (won / died-with-cause).

### Win path

On a correct claim, emit a terminal `won` event/packet carrying rounds elapsed
and outcome. "Win: inside the walls" (GDD §7) — no further zones this slice.

## Definition of done

- A scripted agent that inspected the pond and claims the correct color wins;
  the terminal packet reports `outcome=won` and rounds elapsed.
- A wrong single-palette claim dies with `social.claim_wrong` and a fully
  populated death report (npc, asked, claimed, truth, round, epitaph).
- Zero-or-multiple palette words yields "Speak plainly, stranger.", costs one
  round, and never kills — covered by a test.
- Epitaph selection is deterministic per seed and sourced from content.
- Determinism guard passes; palette matching uses an ordered slice, not map
  iteration.

## Out of scope

- Bribes, suspicion, or any guard behavior beyond the eye-color question this
  tier (later tiers add structural variance, GDD §3).
- Procedural (ritual) death classes and the magic system (Phase 1).
- Freeform parsing of the reply (backlog 04) — this issue judges canonical
  `talk` claims; the reply text is matched against the palette directly.
