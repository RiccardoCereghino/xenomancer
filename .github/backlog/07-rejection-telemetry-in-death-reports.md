---
title: "engine: rejection telemetry in death reports"
labels: [engine, feature, adr-needed]
---

## Summary

Add a **friction block** to the structured death report (GDD §5.7): a derived,
report-time summary of how much the run fought the referee before it died —
structured (reducer) rejections, parse rejections, and timeout-injected waits.
This makes "was this death earned, or did the agent drown in friction?" a
first-class, auditable question on the post-mortem the centaur loop already runs
on (P3 — fair doom is legible, not asserted).

Two rejection populations are deliberately distinguished:

- **`rejected.conflict` and siblings** — *reducer* events (`EventRejected`,
  e.g. `resource_conflict`, `illegal_move`), re-derivable by folding the
  canonical log. These already exist (`engine/types.go`).
- **Parse rejections** — *shell-side*. A parse failure never reaches the reducer
  (ADR-000 D3: only canonical actions enter the engine and the log), so it
  cannot be a reducer event and is not in the log. It lives in the
  **replay-header `meta`** (ADR-000 D6), not the action log.

## References

- GDD §5.2 (parser: parse failure is a free, no-tick rejection — never in the
  log), §5.7 (Death & the Post-Mortem: the report is a first-class feature the
  centaur loop runs on), §13 (Risks: the rejection log is the dictionary's
  backlog; thin coverage means agents fight the parser, not the game).
- ADR-000 D2 (`rejected{reason}` is a reducer event; events are the only seam),
  D3 (parse failures never reach the reducer or the log), D6 (replay header
  `meta`), D8 (a wall-clock timeout is resolved by the **shell** injecting a
  canonical `wait` into the log — the reducer never sees the wall-clock),
  Follow-ups (**ADR-003** parser dictionary format & rejection telemetry — this
  spec is the concrete realization of its telemetry half).

## Pillar

**P3 — Fair doom.** Every death is attributable and reported; the friction block
adds the missing axis (was the run starved of ticks by friction it never chose?)
without which "fair" is asserted, not shown.

## Dependencies

- Verification **V2 (this session): the replay-header `meta` field has no
  defined contract** — `ReplayHeader.Meta` is an untyped `json.RawMessage`
  (`engine/replay.go`), rendered as `"meta":{}` in ADR-000 D6 and never
  populated by `BuildReplay`. Therefore **task 1 of this spec is to define that
  contract** (below); the friction block cannot be built until it exists.
- Verification **V3 (this session): no non-death terminal packet exists.** The
  death report is the only formal terminal packet (ADR-000 D4). A `won` packet is
  spec'd in backlog 03 but not yet formalized in ADR-000/GDD. **The friction
  block is therefore scoped to death reports only** this iteration; when backlog
  03's `won` terminal packet is formalized, the same derived block should be
  attached there too (noted as a follow-up, not built here).
- The reducer's `rejected` events already exist (backlog 04/earlier). The
  timeout-injected `wait` path is specified by backlog 05 (the runner owns the
  wall-clock, ADR-000 D8); this spec defines the *meta record* that makes those
  injected waits countable.

## Spec

### Task 1 — Define the replay-header `meta` contract (prerequisite)

`meta` is shell/runner-authored telemetry *about the run*, never consumed by the
reducer and never part of `CanonicalBytes` or `state_hash`. It carries only what
the reducer structurally cannot know because it lives outside the log:

```json
"meta": {
  "parse_rejections": 0,
  "timeout_waits": [ ]
}
```

- `parse_rejections` (integer): count of freeform lines the shell/parser rejected
  ("I don't understand") over the run. These never reached the reducer (D3), so
  the reducer cannot count them; the shell must. Zero is written explicitly.
- `timeout_waits` (ordered array of integers): the **round numbers** whose `wait`
  was **injected by the shell on a wall-clock timeout** (ADR-000 D8), not chosen
  by the agent. The canonical log stores an ordinary `wait` for these rounds and
  **stays byte-identical** — the "this wait was a timeout" fact lives only here.
  Ordered slice (not a map) so the encoding is stable (ADR-000 D5.2).

Contract rules:
- `meta` is optional and **outside** the state hash. A replay with `meta:{}` (or
  absent) is exactly as valid as one with a populated `meta`; verification
  (`Verify`) must not read it. **This is what keeps the golden hash provably
  unchanged.**
- Fields are additive-only and namespaced by purpose; unknown fields are ignored
  by consumers. Record this contract in ADR-000 D6 (or ADR-003 when it lands) so
  it is not re-invented per shell.

### Task 2 — The friction block on the death report

Extend the death report (GDD §5.7) with a `friction` object, **derived at report
time** by folding the run — **no new `State` field, no change to
`CanonicalBytes`**:

```json
"friction": {
  "structured_rejections": 0,
  "structured_rejections_last_5_rounds": 0,
  "timeout_waits": 0,
  "parse_rejections": 0
}
```

- `structured_rejections`: total count of reducer `rejected` events over the
  whole run, obtained by re-deriving events from the log (they are already in the
  event stream — this is a count, not new state).
- `structured_rejections_last_5_rounds`: the same count restricted to the final
  five in-world rounds before death — the "was it flailing at the end?" signal.
- `timeout_waits`: `len(meta.timeout_waits)` — agent-chosen waits are **not**
  counted here (that is the whole point of task 1's separate record).
- `parse_rejections`: `meta.parse_rejections` verbatim.

### Task 3 — The explicit zero block

A run with no friction of any kind emits the block with all four counters at `0`
(never omits `friction`). "Zero friction" is a positive, legible claim on the
post-mortem — a clean death reads as clean, not as missing data.

## Definition of done

- **The golden hash is provably unchanged.** The DoD test: regenerate/verify the
  committed golden replay and assert `final_state_hash` is byte-identical to the
  pre-change value, because `friction` is derived at report time and `meta` is
  excluded from `CanonicalBytes`/`state_hash`. An unchanged golden hash on this
  PR is the *expected* result, not a red flag (contrast the content specs, which
  supersede) — say so in the golden-hash box.
- The replay-header `meta` contract is documented (task 1) and populated by the
  shell/runner: `parse_rejections` counted, `timeout_waits` recording injected-
  wait round numbers, both written explicitly (zero/empty when none).
- A death report carries a `friction` block whose four counters match a hand-
  computed run: total structured rejections, last-5-rounds subset, timeout-wait
  count (distinct from agent waits), and parse-rejection count from `meta`.
- Timeout-injected waits are counted separately from agent-chosen waits **with
  the canonical log unchanged** — proven by a test that runs the same log with
  and without a timeout injection and shows identical log bytes but different
  `friction.timeout_waits`.
- A zero-friction run emits the explicit all-zero block (tested).
- Determinism guard passes; no floats/`time.`/`math/rand`/map iteration added
  under `/engine`; the friction derivation uses ordered scans only.

## Determinism impact

**none.** No new `State` field; nothing enters `CanonicalBytes` or the state
hash. `friction` is derived at report time from re-folded events; `meta` is
explicitly outside the hash and ignored by `Verify`. The canonical log format is
untouched, so replays reproduce byte-for-byte. (If an implementer finds they
*must* add a `State` field or touch `CanonicalBytes` to make timeout-waits
countable, that is a contradiction of this spec — stop and revise the spec, do
not proceed.)

## Anti-scope check

Checked against GDD §12. Not on the list: this is death-report enrichment and
shell-side telemetry, not a global doom clock, upkeep, combat, UX polish,
inventory, an LLM in the rules path, or repo-secrecy. `CanonicalBytes`, the
vendored PRNG, and the anti-scope surfaces are untouched.

## Out of scope / follow-up

- **Auto-flagging thresholds** that would *reclassify* a death (e.g. "N parse
  rejections in the last M rounds ⇒ reclassify as `engine.unfair`/parser
  friction rather than a game death"). That reclassification needs a live-data
  distribution to set a non-arbitrary threshold, which only arrives with **issue
  06** (the naive-LLM-agent run, #8). **Follow-up:** revisit once issue 06 has
  produced real friction distributions; until then the friction block is
  *descriptive only* and never changes a death's `cause`.
- Attaching the same friction block to the `won` terminal packet — blocked on
  that packet being formalized (backlog 03 spec's `won`; see V3 above). Follow-up
  once it exists.
- Any parser/dictionary format work beyond the `parse_rejections` counter
  (that is ADR-003 / backlog 04 territory).
