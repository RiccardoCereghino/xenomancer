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
  (GDD §5.3). This is the "plant" the gate guard later checks (backlog 03).
- `gate` — exits to `forest_path`. (The guard interrogation is backlog 03.)

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
