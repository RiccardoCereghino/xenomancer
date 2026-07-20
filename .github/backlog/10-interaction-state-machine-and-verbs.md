---
title: "engine: NPC interaction state machine + first-class verbs (obstructive-only breaking bump)"
labels: [engine, feature, determinism, adr-needed]
---

## Summary

The single breaking bump that settles the interaction substrate. It consolidates and
**supersedes** the two live-filed issues #22 (data-driven NPC interaction state machines)
and #23 (retire the `perform`/`legs` movement hack) into one spec with the decisions of
record from the 2026-07-20 direction baked in, so they cannot be re-litigated by a future
coding session. The guard stops being a one-shot palette match in the reducer and becomes a
**data-driven interaction state machine**; the verb model gets a real `move` verb. Both
touch `CanonicalBytes`, so they land as **one** golden churn.

This is deliberately **obstructive-only**: no new lethal dialogue transitions. The one
existing lethal path — a wrong eye-color claim (`social.claim_wrong`, backlog 03) — is
preserved exactly. Lethal dialogue is Track E, blocked on the telegraph contract (#12).

## References

- GDD §5.4 (Social Checks — closed-palette keyword matching, no LLM on the lethal path),
  §5.2 (action grammar; parse failure is a free, no-tick rejection), §5.7 (death & the
  post-mortem stay legible), §13 (content ships attached to a mechanic).
- ADR-000 D1 (pure deterministic reducer — interaction state folds deterministically),
  D3 (parser quarantine; only canonical actions drive the machine), D4 (wire protocol /
  canonical action shape — the verb set is part of it), D5 (determinism laws: ordered
  slices, integers, vendored rng, no map iteration in the state path), D5.6 (frozen
  `CanonicalBytes` → breaking version).
- ADR-001 (engine-exposure contract, backlog 13) and the revised verb grammar to be
  recorded in an ADR (`adr-needed`).
- Supersedes: **#22**, **#23**. Companion: **#21** (parser must learn `move`, Track C2).

## Pillar

**P1 — the world is the referee, and the referee is public.** A referee that can only say
"speak plainly" is not refereeing a conversation; a machine that greets, asks, remembers,
and judges is the world doing its job. Secondary **P4** — the canonical action stream should
express intent (`move`), not an internal resource trick (`perform legs`).

## Spec

### Decisions of record (frozen)

- **Obstructive-only.** No lethal dialogue transitions in this bump. The existing lethal
  path (wrong eye-color claim → `social.claim_wrong`) is preserved unchanged in class,
  fairness, and encoding. Lethal dialogue is deferred to Track E, blocked on #12.
- **Verb set v2 (closed, frozen):** `move | talk | inspect | perform | wait`. `move` retires
  the `perform`/`legs` movement hack; `perform` remains for ritual/manipulation only. A sixth
  verb is deferred until a mechanic demands it.
- **Judgment stays closed-palette matching.** No model on the rules path (GDD §5.4, P1).

### A. Interaction schema (the deliverable the game and the editor both consume)

- A per-NPC state machine expressed as **data** (JSON in `content/`), not hardcoded Go:
  ordered states, transitions **keyed on canonical actions** (arrive, talk-with-claim, wait,
  leave), and the line(s) emitted per state/transition. Deterministic: ordered transition
  lists, integer/enum state, no map iteration in any state-affecting path.
- **State shape:** `greet → ask(fact) → await_claim → judge`. The asked fact is persisted in
  NPC state (`asked_fact`) so the guard **will not ask twice** (GDD §5.4); re-entering the
  interaction returns the **same** fact, stably.
- **Pose the question in-world.** The guard actually asks ("what color are your eyes?")
  through narration/state, so a capable agent discovers the ask from play, not from the
  system prompt — this is what keeps the walkthrough-free prompt (PR #20) a fair test.

### B. First-class movement verb

- Add `move` to the closed verb set. No canonical log line uses `perform legs` for movement
  anymore; movement is its own verb. `perform` stays for genuine ritual/manipulation.
- Update every producer/consumer: reducer (`engine/reduce.go`), canonical encoding
  (`engine/canonical.go`, `engine/types.go`), scripted agent + golden, content, docs/examples.
  (The parser dictionary learning `move` is #21 / Track C2, outside `/engine`.)

### C. Reimplement the guard on the schema

- The reducer drives the machine from canonical actions and the closed palette; win/death
  judgment stays exactly as fair as today: one correct palette word → won; one wrong → died
  (`social.claim_wrong`); zero/several → free re-prompt, never a kill (P3). A distinct line
  renders on a second attempt.

### D. One CanonicalBytes churn

- Interaction state (incl. `asked_fact`) enters `CanonicalBytes` and the verb-set encoding
  changes: **encoding v3 → v4, engine `0.3.0` → `0.4.0`**. Regenerate goldens, update the
  frozen-layout test, and mark the PR superseded with the reason. Keep it a pure deterministic
  fold (no floats/`time`/`math/rand`/map iteration).

## Definition of done (V1–V5)

- **V1 — state machine drives the flow.** A transcript shows greet → ask → (agent answers)
  → judge, with a distinct line for a second attempt.
- **V2 — asked-fact stable across re-entry.** Leaving and re-entering returns the same fact;
  the guard does not ask twice.
- **V3 — wrong-claim death identical in class.** A wrong single-palette claim still dies with
  `social.claim_wrong`; fairness and the death report are unchanged.
- **V4 — `move` works and legacy phrasings map forward.** Movement is its own verb; no
  canonical line uses `perform legs`.
- **V5 — determinism.** Determinism lint + parser quarantine guard + golden-twice + byte-exact
  `cmd`/`run` reproduction all pass; the interaction fold is pure (engine purity via
  `go list` imports).

## Determinism impact

**state-affecting → breaking engine version.** Interaction state and the new verb enter
`CanonicalBytes`; encoding v3 → v4, engine `0.3.0` → `0.4.0`, goldens superseded. Pure
deterministic fold only. This is the **single** golden churn that #22 and #23 were told to
land together.

## Anti-scope check

Checked against GDD §12. Not on the list: the guard mechanic + its content and a verb-grammar
cleanup — not a doom clock, combat, inventory, UX polish, an LLM in the rules path, or
repo-secrecy. Explicitly out of scope: **any** new lethal dialogue transition (Track E, #12),
new zones/NPCs beyond the guard, the parser dictionary (#21), and the authoring tool
(backlog 11). `CanonicalBytes` changes deliberately under a recorded breaking version; the
vendored PRNG stays frozen.
