---
title: "docs: define "Zone 1 at 90%" — measurable definition of done for the content push"
labels: [docs]
---

## Summary

Write the measurable definition of done for the Zone 1 content push, so "get Zone 1 from ~15%
to a defined 90%" is a target with edges rather than a vibe. This doc anchors the whole content
iteration Tracks B–D exist to serve; it blocks nothing but everything downstream measures
against it. Deliverable: `docs/zone1-definition-of-90.md` + a dated DEVLOG entry.

## References

- GDD §3 (difficulty is per-seed values and authored narration knobs, not obscurity), §5.3
  (observation & recall — the pond/eye-color archetype), §5.7 (death & the post-mortem;
  epitaphs are allowed to be the best writing in the game), §7 (Zone 1 content plan), §11
  (results/telemetry).
- The 2026-07-18 showcase post-mortem (DEVLOG) — the "40-round timeout" and walkthrough-in-the-
  prompt failure modes this definition must rule out.

## Pillar

**P3 — fair doom is legible** and **P2 — knowledge is the only progression.** A 90% bar that a
stranger can read off the artifacts is the pillar claim made measurable.

## Spec

Fix a measurable definition of done. Pick/edit from these candidate criteria (the doc decides
the exact bands):

- **Termination.** Naive-LLM showcase on 3 seeds: every episode terminates (win or death) — no
  40-round timeouts like the July post-mortem.
- **Legible death.** Every death is legible: report + epitaph read well; a stranger could tell
  you what the agent did wrong.
- **Verbosity knob bites.** The verbosity knob demonstrably degrades recall — a measured
  baseline-vs-verbose deaths-to-clear delta on the canonical seed.
- **Deaths-to-clear band.** Deaths-to-clear (canonical seed, naive agent) sits inside a target
  band the doc chooses.
- **Epitaph coverage.** Epitaph library ≥ N templates per cause class (the doc sets N).

Each criterion states how it is measured (which artifact, which seeds, which command) so the
content iteration can check itself against the definition without re-interpreting it.

## Definition of done

- `docs/zone1-definition-of-90.md` exists, stating each criterion with its measurement method
  and chosen target band/number.
- The definition is checkable from showcase artifacts + `cmd/trace` output, without human taste
  where a number will do.
- A dated DEVLOG entry records the decision.

## Determinism impact

**none.** A documentation/decision doc. It touches no `/engine`, no `CanonicalBytes`, no PRNG,
and no content-pack bytes.

## Anti-scope check

Checked against GDD §12. A definition-of-done doc, not a mechanic — no doom clock, combat,
inventory, UX polish, LLM in the rules path, or repo-secrecy.
