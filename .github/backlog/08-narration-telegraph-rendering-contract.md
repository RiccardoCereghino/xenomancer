---
title: "docs: ADR-00X — narration & telegraph rendering contract"
labels: [docs, adr-needed]
---

## Summary

Write an ADR (the deliverable is a **decision record, not code**) that defines
what a *telegraph* is at the rendered-narration layer and how its presence is
mechanically proven. Today the fairness law "no fuse without a telegraph ladder"
(GDD P3, §5.6) is stated at the **event** layer — the reducer emits
`telegraph{stage}` events — but there is **no contract** that those events
survive into the text an agent actually reads. The narrator is a seeded,
verbosity-controlled template layer (ADR-000 D2: `Narrate(events, seed,
verbosity) → text`); a telegraph that is elided by low verbosity or dropped by a
seeded variant is, in play, an untelegraphed fuse — an `engine.unfair` death that
the current tests cannot catch. This ADR closes that gap by making telegraph
presence an auditable property of *rendered* output.

## References

- GDD §5.6 (Hazards: "no fuse without a telegraph ladder"; telegraphs woven into
  narration where a skimming agent will miss them — attention-under-noise),
  §5.7 (`engine.unfair` must stay at zero; untelegraphed hazard = engine bug),
  §3 (narration verbosity and seeded synonym variance are difficulty knobs),
  §9 (narration is verbosity-knob controlled, telegraphs woven in).
- ADR-000 D2 (`telegraph{stage}` is a reducer event; the narrator is a pure
  event consumer, `Narrate(events, seed, verbosity) → text`), D5.3 (seeded
  variant selection via sub-seeding — the mechanism that must never drop a
  telegraph).
- Backlog 02 / issue #4 (the wolf telegraph ladder emits stages 1–3) — the first
  concrete producer this contract governs.

## Pillar

**P3 — Fair doom** (primary) and **P1 — the referee is public/auditable.**
Without this contract, `engine.unfair = 0` is unverifiable and the ladder law is
untestable — both are load-bearing pillar claims.

## Spec

The ADR (proposed number ADR-00X — assign the next free number at authoring
time; ADR-001/002/003 are already earmarked in ADR-000 Follow-ups) must define,
at minimum:

### 1. What counts as a telegraph in rendered narration

- **Guaranteed presence at every verbosity level.** A telegraph string must
  render at *every* verbosity setting the game ships. Verbosity may change a
  telegraph's *prominence, length, or surrounding noise* — that is the
  attention-under-noise difficulty (GDD §3, §5.6) — but must **never** reduce it
  to zero characters. Terser verbosity makes a telegraph *easier to miss*, never
  *absent*.
- **Never elided by seeded variant selection.** Seeded synonym/variant selection
  (ADR-000 D5.3) may choose *which phrasing* of a telegraph renders, but every
  variant in a telegraph's palette must itself be a telegraph. A variant set
  where some entries carry the signal and some do not is a spec violation.
- **Identifiable unit.** The ADR must fix the granularity at which a telegraph is
  a detectable unit — a sentence or a clause (pick one and justify it), so that
  "the telegraph rendered" is a decidable string-level fact, not a vibe. This is
  what a CI check can assert against.

### 2. The auditable "attention-under-noise" vs. "untelegraphed" criterion

Define the line between a *fair* telegraph (present but easy to miss under noise
— legal difficulty) and an *effectively untelegraphed* fuse (an engine bug). The
criterion must be **auditable**, i.e. checkable without human taste. Suggested
form (the ADR decides the exact rule): a fuse stage is *telegraphed* iff, for
every shipped verbosity and every seed in a determinism test corpus, the rendered
narration for that round contains the stage's telegraph unit as a detectable
string. Anything short of "present at every (verbosity, seed)" is untelegraphed
by definition — noise is allowed to *bury* the signal, never to *delete* it.

### 3. How narrator tests prove telegraph presence mechanically

- A **CI check** that, for every `telegraph{stage}` event a content pack can
  emit, asserts the narrator renders it to a **detectable string at every
  verbosity setting** (and across the seed corpus for variant selection). This is
  a narrator/consumer test — it lives outside `/engine` (the narrator is a
  separate package, ADR-000 D2) and spends no tokens.
- The check must fail loudly if any (stage, verbosity, seed) combination renders
  narration with the telegraph unit missing — that failure *is* the mechanical
  detector for an `engine.unfair` regression, caught in CI rather than in a
  death report.
- The ADR should specify how content declares the detectable unit per telegraph
  (e.g. a tagged span / a per-stage marker in the content pack) so the test has a
  ground-truth target to search for, rather than guessing.

## Definition of done

The ADR is the deliverable. Done when the ADR is written and Proposed, and it:

- fixes the three definitions above (telegraph unit, guaranteed-presence rule,
  no-elision-by-variant rule);
- states the auditable attention-under-noise vs. untelegraphed criterion in a
  form a test can decide;
- specifies the CI narrator check (every `telegraph{stage}` → detectable string
  at every verbosity, across the seed corpus) precisely enough that a later
  coding session can implement it from the ADR alone;
- names how content declares each telegraph's detectable unit;
- records the consequence: with this contract, "no fuse without a telegraph
  ladder" is mechanically testable and `engine.unfair = 0` is verifiable — and
  without it, both are unenforceable.

## Determinism impact

**none.** The ADR governs the *narrator* (a pure consumer of the event stream,
ADR-000 D2) and a CI test, not the reducer. It adds no `State`, touches no
`CanonicalBytes` or the vendored PRNG, and changes no rule. Seeded variant
selection it constrains already exists (D5.3); this only forbids variant sets
that drop the signal. (If authoring reveals the contract requires a reducer or
frozen-surface change, stop and revise — it should not.)

## Anti-scope check

Checked against GDD §12. Not on the list: this is a fairness-auditability ADR
plus a narrator CI check — not a global doom clock, upkeep, combat, human-facing
UX polish, inventory, an LLM in the rules path, or repo-secrecy. The narration
*inflation* LLM layer (GDD §10) stays out of scope and off by default; this
contract governs the deterministic template narrator only.
