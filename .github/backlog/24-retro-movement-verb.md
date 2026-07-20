---
title: "feature: retire the `perform`/`legs` hack for a first-class movement verb (breaking)"
labels: [engine, feature, adr-needed]
---

> **Retro-filed 2026-07-20 for issue #23** (live-filed 2026-07-18) — closes audit
> Divergence 02. This file mirrors the existing issue so the backlog is complete; repo-sync
> skips it (the title already matches #23) and creates no duplicate.
>
> **Superseded by backlog spec 10 (A2 — obstructive-only interaction state machine + first-class
> verbs).** Spec 10 freezes verb set v2 (`move | talk | inspect | perform | wait`) and lands this
> cleanup in the same single CanonicalBytes churn as #22. Build from spec 10; close #23 as
> superseded once spec 10 materializes.

## Summary

Movement is currently modeled as `perform` on the `legs` resource — there is no first-class
movement verb; the closed verb set is `{inspect, perform, talk, wait}` (`engine/types.go`,
`engine/reduce.go`). "Go to the gate" becomes `perform legs -> gate`, which leaks the engine's
resource plumbing into every canonical log line and every replay. Introduce a proper movement
verb (`move`) and retire the overload, so the action grammar reads cleanly.

## References

- GDD §5.2 (action grammar — verbs/resources; a grammar cleanup).
- ADR-000 D4 (wire protocol / canonical action shape — the verb set is part of it), D5 (frozen
  `CanonicalBytes` — changing the verb set is a breaking engine version), ADR-003 (planned;
  record the revised verb grammar, `adr-needed`).

## Pillar

**P4 — built for machines, legible to pilots.** The canonical action stream should express intent
(`move`), not an internal resource trick (`perform legs`).

## Spec

- Add a first-class movement verb (`move`) to the closed set; `perform` remains for genuine
  non-movement (ritual/manipulation) actions — decided in spec 10 / the ADR.
- Update every producer/consumer: parser dictionary (#21), reducer (`engine/reduce.go`), scripted
  agent + golden, content, and docs/examples.
- Bump the engine `Version` (and `ProtocolVersion` if the envelope changes).

## Definition of done

- No canonical log line uses `perform legs` for movement; movement is its own verb.
- Determinism guard passes; the scripted agent still wins the slice; golden regenerated.
- **Golden hash superseded** (canonical action bytes change), marked in the PR with the reason.

## Determinism impact

**touches-frozen-surface → breaking engine version.** The verb set is part of the canonical
action encoding. Do this in the **same breaking bump** as the interaction rework (#22) so the
golden is superseded once, not twice — see spec 10.

## Anti-scope check

Checked against GDD §12. A verb-grammar cleanup — not a new mechanic, doom clock, or repo-secrecy.
The vendored PRNG stays frozen; `CanonicalBytes` changes deliberately under a recorded breaking
version.
