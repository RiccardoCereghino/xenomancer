---
title: "feature: data-driven NPC interaction state machines + expanded dialogue (the guard is a robot)"
labels: [engine, feature, adr-needed]
---

> **Retro-filed 2026-07-20 for issue #22** (live-filed 2026-07-18) — closes audit
> Divergence 02. This file mirrors the existing issue so the backlog is complete; repo-sync
> skips it (the title already matches #22) and creates no duplicate.
>
> **Superseded by backlog spec 10 (A2 — obstructive-only interaction state machine + first-class
> verbs).** Spec 10 consolidates this issue with #23 and bakes in the 2026-07-20 decisions of
> record (obstructive-only, verb set v2, one CanonicalBytes churn). Build from spec 10; close
> #22 as superseded once spec 10 materializes.

## Summary

The guard has no conversational state. Today the whole "interrogation" is a one-shot, hardcoded
palette match in the reducer (`engine/reduce.go` `VerbTalk` branch): a talk at the gate either
names one palette word (→ won/died) or doesn't (→ "speak plainly"). There is no greeting, no
*asking* the question in-world, no memory that it already asked. This issue rebuilds NPC
interactions as **data-driven state machines** with far more authored lines, and defines the
**interaction schema** the game and the future authoring tool both consume.

## References

- GDD §5.4 (the gate guard / social claim), §5.2 (action grammar & talk; parse failure stays a
  free, no-tick rejection), §5.7 (death & the post-mortem), §13 (content ships attached to a
  mechanic).
- ADR-000 D1 (pure deterministic reducer — interaction state folds deterministically), D3 (parser
  quarantine; no LLM on the rules path), D5 (determinism laws), ADR-003 (planned; the
  interaction/dialogue format half, `adr-needed`).

## Pillar

**P1 — the world is the referee, and the referee is public.** A referee that can only say "speak
plainly" isn't refereeing a conversation. Secondary **P2** — knowledge is the only progression.

## Spec

- **A. Interaction schema.** A per-NPC state machine as **data** (JSON in `content/`): ordered
  states, transitions keyed on canonical actions, lines per state/transition. Deterministic
  (ordered lists, integer/enum state, no map iteration). Guard machine minimum:
  `idle → notices_you → asks_question → awaits_claim → {passed | denied}`, with "I already asked
  once" handling. **Pose the question in-world** so a capable agent discovers the ask from play.
- **B. Expanded lines.** Greetings, the question, re-prompts, the plain-speak nudge, pass/deny
  prose, idle/ambient variants — volume is the point (this is why the webapp, #24, exists).
- **C. Reimplement the guard on the schema.** The reducer drives the machine from canonical
  actions + the closed palette; win/death judgement stays exactly as fair as today (one correct
  word → won; one wrong → died; zero/several → free re-prompt, never a kill — P3).

## Definition of done

- The guard runs from data: greet → ask → (agent answers) → judge, with a distinct line for a
  second attempt.
- Interaction state folds deterministically; guard judgement unchanged in fairness. Determinism
  guard passes.
- The interaction schema is documented (ADR).
- **Golden hash superseded** (rules + content change; interaction state enters `CanonicalBytes`).
  Land in the **same breaking bump** as the verb-model cleanup (#23) — see spec 10.

## Determinism impact

**state-affecting → breaking engine version.** Interaction state enters `CanonicalBytes`; the
engine `Version` bumps and goldens supersede. Pure deterministic fold. Land with #23 (spec 10)
so the golden churns once.

## Anti-scope check

Checked against GDD §12. The guard mechanic + its content — not a doom clock, combat, inventory,
UX polish, an LLM in the rules path, or repo-secrecy. Out of scope: new zones/NPCs beyond the
guard, the parser dictionary (#21), and the authoring tool (#24).
