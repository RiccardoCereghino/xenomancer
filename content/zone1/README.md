# Zone 1 — Woods to Gate (content pack)

The training-grounds content for the vertical slice (GDD §7). Inert,
hash-addressed data (ADR-000 D5.5); the engine loads it but owns none of it.

## Map

```
clearing → forest_path → gate
                 └────→ still_pond   (optional branch, secretly survival-critical)
```

- `clearing` — start. No hazard. Self-inspection (`inspect self`) reveals
  species and hair — **not** eyes (GDD §7).
- `forest_path` — hub; exits to `clearing`, `still_pond`, `gate`. (The wolf
  fuse is backlog 02.)
- `still_pond` — `inspect reflection` observes the per-seed **eye color**
  (GDD §5.3). This is the "plant" the gate guard checks at the gate.
- `gate` — exits to `forest_path`; home of the **gate guard** NPC. `talk` on
  `voice` triggers the lethal eye-color interrogation (GDD §5.4, §7).

## The gate guard (lethal social check)

The `gate_guard` NPC (`npcs: [{"id":"gate_guard","asks":"eye_color"}]`) asks the
agent's eye color. Judgment is closed-palette keyword matching against the same
frozen palette — **no LLM on the lethal path** (GDD §5.4, P1). The reply text
rides in the `talk` action's `args` (`{"say":"..."}`); freeform parsing is
backlog 04.

| Reply names… | Result |
|---|---|
| exactly one palette word == the per-seed truth | **win** — inside the walls (GDD §7) |
| exactly one palette word, wrong | **death** — `social.claim_wrong` + death report |
| zero or several palette words | "Speak plainly, stranger." — costs one round, **never kills** |

## Epitaphs (death-report prose)

The `epitaphs` list holds the death-report templates (GDD §5.7). Selection is
per-seed and deterministic — `SplitMix64(Subseed(seed, "narration.epitaph")).Next()
% len(epitaphs)` indexes the **frozen-order** slice — with `{claimed}`/`{truth}`
filled from the death detail. Epitaphs live in engine content (not narration)
because the reducer emits them inside the structured `died{report}`.

**Canonical:** seed 0's death (claimed `green`, truth `grey`) selects index 0 and
renders the reference line *"He was sure his eyes were green. The pond,
unbothered, remains grey."* (GDD §5.7).

## Per-seed fact: eye color

Drawn once per run from the closed, **frozen-order** palette:

| index | 0 | 1 | 2 | 3 |
|---|---|---|---|---|
| color | `blue` | `green` | `brown` | `grey` |

Selection derives **exclusively** from the vendored rng and integer arithmetic
(ADR-000 D5.3): `SplitMix64(Subseed(seed, "facts.eye_color")).Next() % 4`
indexes the palette. No `math/rand`, no floats, no map iteration, no reordering
— the index is stable forever, so a seed's answer never moves.

## Canonical world (GDD §3, §5.3)

**Seed 0 is the canonical world.** Its eye color is **`grey`** — the fixed,
memorizable reference for knowledge runs and goldens. ("The pond, unbothered,
remains grey.")

The committed golden replay runs on **seed 1**, whose eye color is **`brown`**.
