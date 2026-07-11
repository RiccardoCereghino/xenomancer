# XENOMANCER — Development Log

Newest entries first. This log is narrative and historical; it is the one place
`sprint-N` vocabulary is allowed to persist (see CONTRIBUTING "Phase & sprint
vocabulary"). It is not a source of truth — the GDD, ADRs, and the backlog are.

---

## 2026-07-11 — Backlog: telemetry, fairness, and process specs

Specs-and-docs session (zero engine code). Added three new backlog specs,
amended one existing spec, and reconciled the phase/sprint vocabulary.

### What shipped

- **Backlog 07 — `engine: rejection telemetry in death reports`** (P3). A derived
  *friction block* on the death report distinguishing reducer rejections
  (`rejected.conflict` &c., re-derived from the log) from parse rejections
  (shell-side, in replay-header `meta`, never in the log per ADR-000 D3) plus
  timeout-injected waits. No new `State` field; derived at report time; the DoD
  test asserts the golden hash is provably unchanged. Its first task defines the
  previously-undefined replay-header `meta` contract (see V2).
- **Backlog 08 — `docs: ADR-00X — narration & telegraph rendering contract`**
  (P3/P1). An ADR-shaped deliverable: what counts as a telegraph in *rendered*
  narration (guaranteed at every verbosity, never dropped by seeded variant
  selection, a fixed sentence/clause unit), the auditable "attention-under-noise
  vs. untelegraphed" criterion, and a CI narrator check proving every
  `telegraph{stage}` renders to a detectable string at every verbosity. Without
  it, "no fuse without a telegraph ladder" is untestable and `engine.unfair = 0`
  is unverifiable.
- **Backlog 09 — `docs: content review workflow for agent-drafted narration`**
  (P5/P3). A future CONTRIBUTING section: batch size per review PR, accept/reject
  convention, and the non-negotiable rule that no agent-authored narration or
  epitaph merges without explicit human approval — phone-reviewable, process not
  tooling.
- **Amended backlog 06** (naive LLM agent / showcase, issue #8): the first
  frontier-model run is a **data-gathering run, not the publicity artifact**.
  DoD now requires capturing the full event stream + friction telemetry, states
  that tuning iterations (verbosity, fuse numbers, dictionary coverage) are
  expected before a postable death, and gates the publicity post on a
  `recall.*`/`social.*` (game-failure) cause — never parser friction.
- **CONTRIBUTING — phase/sprint vocabulary table**: maps GDD phases 0–4/Someday
  to the `sprint-N` labels as actually used, and deprecates `sprint-N` outside
  historical DEVLOG entries in favor of GDD phase numbers. No retroactive
  renumbering.

### Verification findings (V1–V5)

- **V1 — backlog spec format/location.** `.github/backlog/NN-slug.md`. Front
  matter between the first two `---` fences: `title:` (double-quoted),
  `labels:` (inline `[a, b, c]`), optional `milestone:` (unquoted); body is
  everything after the second fence. Consumed by `repo-sync.yml`, which is
  idempotent by **exact title** and **creates only — it never updates an
  existing issue's body**. Labels must exist in `labels.json`; a `milestone:`
  must be pre-ensured (`sprint-1 — woods to gate`, `sprint-2 — magic`) or the
  sync fails. The three new specs **omit `milestone:`** (forward-consistent with
  the vocabulary decision) and cite their GDD phase in-body instead.
- **V2 — replay-header `meta` contract.** **None exists.** `ReplayHeader.Meta`
  is an untyped `json.RawMessage` (`engine/replay.go`), shown as `"meta":{}` in
  ADR-000 D6, and `BuildReplay` never populates it. So backlog 07's **first
  task is to define that contract** (parse-rejection count + timeout-wait round
  list, both outside the state hash).
- **V3 — non-death terminal packet.** **None exists.** The death report is the
  only formal terminal packet (ADR-000 D4); a `won` packet is spec'd in backlog
  03 but not formalized in ADR-000/GDD, and neither death nor win is implemented
  yet (sprint-0 is a walking skeleton emitting only `moved`/`waited`/`rejected`).
  Per the task's conditional, backlog 07's **friction block is scoped to death
  reports only**, with attaching it to a future `won` packet noted as follow-up.
- **V4 — issue-06 materialized?** **Yes.** All six backlog specs 01–06 are both
  files *and* live issues #3–#8 (issue #8 = backlog 06). Because `repo-sync.yml`
  never updates an existing issue's body, editing the file does **not** propagate
  to issue #8. Per the backlog-as-code convention the **file is source of truth**
  and was amended; **issue #8's body needs a manual edit** (or an addendum
  comment) to match — flagged in the PR. There is no addendum-file convention in
  the repo.
- **V5 — phase/sprint vocabulary.** Genuine **off-by-one drift**: `sprint-0`
  (README) = the pre-slice walking skeleton; `sprint-1` (milestone "woods to
  gate") = GDD **Phase 0** (vertical slice); `sprint-2` (milestone "magic") =
  GDD **Phase 1**. GDD itself mixes "Sprint 1" (§7) with "Phase 2/3" (§7, §12).
  Resolved by the CONTRIBUTING mapping table, not by renumbering.

### Decisions

- **New specs carry no `sprint-N` milestone.** Consistent with the vocabulary
  decision and safe against `repo-sync.yml` (an unknown milestone would fail the
  sync); each spec names its GDD phase in-body.
- **Friction block is descriptive only.** No auto-flagging threshold that would
  *reclassify* a death — that needs live friction distributions from issue 06;
  deferred as a follow-up inside backlog 07.
- **Existing milestones left untouched.** `sprint-1`/`sprint-2` are in flight;
  renaming would break the sync and orphan open issues.

### Determinism impact

All five items: **none.** No engine code, no `/engine` change, no `CanonicalBytes`
or vendored-PRNG change, nothing on the anti-scope list (GDD §12).

### Golden hash

**Unchanged** — no engine changes were permitted or made this session.
