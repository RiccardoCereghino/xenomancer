---
title: "parser: dictionary v1 and the misparse-never-kills property"
labels: [parser, feature]
milestone: sprint-1 — woods to gate
---

## Summary

Add the quarantined freeform parser: a package **outside `/engine`** that maps
freeform agent text to canonical `RoundSubmission`s via a versioned dictionary,
by deterministic lookup only. Its defining guarantee is that a misparse can
never kill (GDD P3): unknown input is a free rejection, and no freeform string
can ever produce a state-affecting action that was not an exact dictionary hit.

## References

- GDD §5.2 (Actions & the Parser — quarantined, synonym dictionary as data,
  parse failure returns "I don't understand" with no tick cost, misparse never
  kills), P3 (misunderstanding never kills).
- ADR-000 D3 (the parser is **not** in the replay path — only canonical actions
  enter the engine and the log, so parser evolution can never invalidate a
  replay), D4 (freeform is accepted only by the parser package), Follow-ups
  (ADR-003 parser dictionary format & rejection telemetry).

## Spec

### Location & purity

- New package **outside `/engine`** (e.g. `/parser`), consistent with ADR-000
  D3. The engine never imports it. Only canonical `RoundSubmission`s cross into
  the engine and the log.
- Deterministic **lookup only** — no fuzzy matching, no model, no randomness.
  The same input always maps to the same canonical action (or the same
  rejection).

### Normalization + dictionary

- Normalize input: lowercase, strip punctuation, collapse whitespace.
- A dictionary (JSON, checked in as data) maps normalized freeform verbs and
  targets to the canonical verb set (`inspect | perform | talk | wait`) and
  canonical target ids. AI-authored offline, shipped as data; at runtime it is
  a pure table lookup (GDD §5.2, ADR-000 D3).

### Rejections are free

- Input with no dictionary hit returns a **parse rejection** ("I don't
  understand") that **costs nothing** — no tick, no state change. It never
  reaches the engine. The rejection log is the dictionary's backlog (GDD §13).

### Property (the load-bearing test)

- Property test: **no freeform input can ever produce a state-affecting action
  that was not an exact dictionary hit.** Any input that does not resolve to a
  dictionary entry must yield a rejection, not a canonical action — therefore a
  misparse can never advance a fuse, make a claim, or kill (P3).

### Shell integration

- The stdio shell accepts **either** canonical JSON round envelopes **or**
  freeform lines. Freeform lines go through the parser; canonical JSON goes
  straight to the engine. Only canonical actions are written to the log.

## Definition of done

- A `/parser` (or equivalently out-of-engine) package exists; `/engine` does
  not import it (verifiable via `go list` imports).
- Known synonyms map to the correct canonical actions; unknown input returns a
  no-cost rejection.
- The property test passes: non-dictionary input never produces a
  state-affecting canonical action.
- The stdio shell accepts both canonical JSON and freeform lines, logging only
  canonical actions.
- The determinism guard is unaffected (parser lives outside `/engine`).

## Out of scope

- Any LLM or fuzzy fallback in the parser (explicitly rejected; quarantine
  only — ADR-000 Alternatives).
- Seeded synonym variance / structural variance (GDD §3, a later tier).
- Rejection telemetry format beyond a simple log (deferred to ADR-003).
