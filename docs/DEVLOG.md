# XENOMANCER — Dev Log

Convention: one entry per working day, newest first. Entries are appended by the human or by the coding agent as part of a PR ("Append a dated entry to docs/DEVLOG.md summarizing what shipped, decisions made, and the golden hash if it changed"). Terse is fine; decisions and hashes are not optional. This log is also the one place `sprint-N` vocabulary is allowed to persist (see CONTRIBUTING "Phase & sprint vocabulary").

---

## 2026-07-18 — Issue 08: naive LLM agent + gated showcase

By the time this ran, the guard (03), parser (04), and the `cmd/run` bidirectional runner (05, #18) had all merged to `main`, so issue 08 reduced to its one genuinely-missing piece: the naive LLM player + the gated showcase (backlog 06). Built on the merged runner + parser + `shell/wire`; no engine/content/runner changes.

**Shipped**
- **Naive LLM agent `agent/llm` (06).** Runs as an `--agent` subprocess under `cmd/run`. A **stdlib-only** (`net/http`) Anthropic Messages API client — **no third-party SDK**, `go.mod` still dependency-free. Each turn it reads the observation packet, asks the model, takes the model's **freeform** reply, and routes it through `parser.New().Parse` to a canonical envelope emitted on stdout (the runner reads canonical envelopes). A misparse becomes a free `wait` and the model is told next turn — the quarantine on genuinely sloppy input (GDD P3). Recognizes the terminal won/died packet by its `outcome` field and writes it to `--report` for the showcase to upload. Model via `--model` / `ANTHROPIC_MODEL`; `ANTHROPIC_BASE_URL` supported for gateways/tests. Covered by an `httptest` unit suite **and** an end-to-end run of the agent binary against a stub server — CI stays hermetic and zero-token.
- **Parser wiring for guard replies (06, additive).** The merged parser (04) handled navigation only — a freeform reply could never reach the guard (no NPC target, no `args`). Added additively: guard synonyms (`guard`/`gate guard`/`gatekeeper`/`guardian` → `gate_guard`) + talk-verb synonyms, and a talk action now carries `Args:{"say":"<normalized line>"}` so the engine matches the palette. Misparse-never-kills preserved — a claim still needs an exact talk-verb **and** NPC-target dictionary hit (property test unchanged; one reject-case flipped to a positive guard-claim test). Without this the LLM could reach the gate but never claim, so the showcase could never demonstrate the guard death.
- **`showcase.yml` (06).** `workflow_dispatch` only, best-of-3 on seeds 0/1/2 (0 = canonical death world), allowed to flake, uploads replay + death report. `ci.yml`: one `CGO_ENABLED=0` line so the new `net/http` binary links internally on the macOS runner (pure-Go project; state hashes unaffected). Owner must add `ANTHROPIC_API_KEY` (documented in the workflow + CONTRIBUTING "Live showcase").

**Decisions of record**
- Scope reconciled after concurrent merges of 03/04/05: issue 08 shipped as 06 only, on top of the merged runner/parser/wire — no duplication.
- The parser runs **inside the agent** (model freeform → parser → canonical envelope), because the merged `cmd/run` reads canonical envelopes; the parser stays quarantined outside `/engine`.
- The guard-reply parser support is the one additive bit 06 needs; kept minimal so the merged parser's navigation behavior and property test are untouched.

**Golden hash — unchanged.** No `/engine` or content change; the committed goldens and the `cmd/run` determinism-fixture check are untouched.

---

## 2026-07-12 — Issue 05: gate guard interrogation, win condition & death reports

Closes the slice loop. Implements the "lethal check" half (GDD §7): the gate guard interrogates the agent's eye color, a correct claim wins, a wrong one is a fair death (P3) — the first `contextual`/`social` death class. Rebased onto the parser-v1 merge (backlog 04); the shell now routes both freeform lines (parser) and the terminal died/won packets.

**Shipped**
- **Gate guard NPC** at the gate (`npcs: [{"id":"gate_guard","asks":"eye_color"}]`). `talk` on `voice` triggers the interrogation. Judgment is closed-palette keyword matching by ordered string containment against the same frozen palette as the pond — **no LLM on the lethal path** (GDD §5.4, P1). The claim rides in the action `args` (`{"say":"..."}`); freeform parsing stays backlog 04. Exactly one palette word is a claim; zero or several → "Speak plainly, stranger." (costs one round, **never kills**); one and correct → win; one and wrong → death.
- **Terminal outcome in State.** New `Outcome` (`""`/`won`/`died`) + `Cause` fields, **appended to the frozen `CanonicalBytes` (encoding v2)** so a win and a death at the same gate/round hash differently — replay-as-proof can tell the endings apart. Once set, `Reduce` is a sticky no-op. New event kinds `died{report}` / `won`; new rejection reason `unclear_claim`.
- **Death report** (GDD §5.7) as the terminal packet: `cause`, `detail{npc,asked,claimed,truth}`, `round`, `telegraphs_ignored: []`, `ritual_progress: null`, `epitaph`. **Epitaph templates live in engine content** (`epitaphs` in `map.json`), selected per-seed via `SplitMix64(Subseed(seed, "narration.epitaph")).Next() % len` over a frozen-order slice, `{claimed}`/`{truth}` filled by the reducer. The win packet carries `outcome` + rounds elapsed; the stdio shell emits both terminal packets and stops.
- **Two scripted goldens** — the win path (seed 1: inspect pond → gate → claim brown → won, 6 rounds) and the death path (seed 0: straight to the gate → claim green while truth is grey → died, 3 rounds). `verifygolden` now verifies both; the death golden reproduces the canonical GDD §5.7 line exactly.
- Tests: correct-claim win; wrong-claim death with a fully populated report; zero/several palette words never kill; guard only at the gate; post-terminal `Reduce` is a no-op; `matchClaim` ordered-containment; epitaph determinism + canonical seed-0 line; win/death state-hash distinguishability; shell win/death/speak-plainly packets. Determinism guard still green (ordered slices, integer-only).

**Decisions of record**
- **Engine `Version` 0.1.0 → 0.2.0.** The terminal outcome is part of the replay proof, so it belongs in `CanonicalBytes`; appending it is a frozen-encoding change and bumps the version (ADR-000 D5.6). Death/win are terminal in the engine (sticky), not just a shell concern.
- Epitaphs are **engine content**, not shell narration — the reducer emits them inside the structured `died{report}` (ADR-000 D2/D4), so their selection must be deterministic and seed-derived like any other per-seed fact.
- "Speak plainly" reuses the rejection→narration pipeline (`unclear_claim`): it ticks like any resolved round, so it costs one round and never kills, with no new event surface.

**Golden hash — superseded (rules + content change).** Terminal outcome in the encoding (rules) and the guard NPC + epitaphs in `map.json` (content) both move the hashes; engine version bumped. Both goldens regenerated and reproduced identically across two `verifygolden` runs:
- engine_version: `0.1.0` → `0.2.0`
- content_hash: `sha256:12378e265ea91cea2ada8b575bf951fadcf1e048cb5ee3a2494c514c2cd97eed` → `sha256:737d5c995fbdb7dfbb9a0e2ddcacd8510052bc6376ac69d59b76392b74a7b029`
- win final_state_hash (seed 1): `sha256:964206448bc0af164f7dfc3c32568ba12362330f3e87c6bfe76e5eaf2054aaca` → `sha256:7de18c17673be07208ea4c8f4600e3d69e15c72cddd94a0f82e2052273bc74aa`
- death final_state_hash (seed 0, new): `sha256:7044ef6b788fbd70b8f28b837dc640ee84f50678166208ee81dfb146610dafab`

---

## 2026-07-11 — Backlog: telemetry, fairness, and process specs

Specs-and-docs session (zero engine code). Added three new backlog specs, amended one existing spec, and reconciled the phase/sprint vocabulary.

**Shipped**
- **Backlog 07 — `engine: rejection telemetry in death reports`** (P3). A derived *friction block* on the death report distinguishing reducer rejections (`rejected.conflict` &c., re-derived from the log) from parse rejections (shell-side, in replay-header `meta`, never in the log per ADR-000 D3) plus timeout-injected waits. No new `State` field; derived at report time; the DoD test asserts the golden hash is provably unchanged. Its first task defines the previously-undefined replay-header `meta` contract (see V2).
- **Backlog 08 — `docs: ADR-00X — narration & telegraph rendering contract`** (P3/P1). An ADR-shaped deliverable: what counts as a telegraph in *rendered* narration (guaranteed at every verbosity, never dropped by seeded variant selection, a fixed sentence/clause unit), the auditable "attention-under-noise vs. untelegraphed" criterion, and a CI narrator check proving every `telegraph{stage}` renders to a detectable string at every verbosity. Without it, "no fuse without a telegraph ladder" is untestable and `engine.unfair = 0` is unverifiable.
- **Backlog 09 — `docs: content review workflow for agent-drafted narration`** (P5/P3). A future CONTRIBUTING section: batch size per review PR, accept/reject convention, and the non-negotiable rule that no agent-authored narration or epitaph merges without explicit human approval — phone-reviewable, process not tooling.
- **Amended backlog 06** (naive LLM agent / showcase, issue #8): the first frontier-model run is a **data-gathering run, not the publicity artifact**. DoD now requires capturing the full event stream + friction telemetry, states that tuning iterations (verbosity, fuse numbers, dictionary coverage) are expected before a postable death, and gates the publicity post on a `recall.*`/`social.*` (game-failure) cause — never parser friction. Issue #8's live body was mirror-updated via an addendum comment (repo-sync only creates, never updates bodies — see V4).
- **CONTRIBUTING — phase/sprint vocabulary table**: maps GDD phases 0–4/Someday to the `sprint-N` labels as actually used, and deprecates `sprint-N` outside historical DEVLOG entries in favor of GDD phase numbers. No retroactive renumbering.

**Verification findings (V1–V5)**
- **V1 — backlog spec format/location.** `.github/backlog/NN-slug.md`. Front matter between the first two `---` fences: `title:` (double-quoted), `labels:` (inline `[a, b, c]`), optional `milestone:` (unquoted); body is everything after the second fence. Consumed by `repo-sync.yml`, which is idempotent by **exact title** and **creates only — it never updates an existing issue's body**. Labels must exist in `labels.json`; a `milestone:` must be pre-ensured (`sprint-1 — woods to gate`, `sprint-2 — magic`) or the sync fails. The three new specs **omit `milestone:`** (forward-consistent with the vocabulary decision) and cite their GDD phase in-body instead.
- **V2 — replay-header `meta` contract.** **None exists.** `ReplayHeader.Meta` is an untyped `json.RawMessage` (`engine/replay.go`), shown as `"meta":{}` in ADR-000 D6, and `BuildReplay` never populates it. So backlog 07's **first task is to define that contract** (parse-rejection count + timeout-wait round list, both outside the state hash).
- **V3 — non-death terminal packet.** **None exists.** The death report is the only formal terminal packet (ADR-000 D4); a `won` packet is spec'd in backlog 03 but not formalized in ADR-000/GDD, and neither `died` nor `won` is implemented yet (the engine emits `moved`/`waited`/`rejected` and, since the content-schema-v2 merge, `observed` — but no terminal packet). Per the task's conditional, backlog 07's **friction block is scoped to death reports only**, with attaching it to a future `won` packet noted as follow-up.
- **V4 — issue-06 materialized?** **Yes.** All six backlog specs 01–06 are both files *and* live issues #3–#8 (issue #8 = backlog 06). Because `repo-sync.yml` never updates an existing issue's body, editing the file does **not** propagate to issue #8. Per the backlog-as-code convention the **file is source of truth** and was amended; issue #8's body was mirrored via an addendum comment. There is no addendum-file convention in the repo.
- **V5 — phase/sprint vocabulary.** Genuine **off-by-one drift**: `sprint-0` (README) = the pre-slice walking skeleton; `sprint-1` (milestone "woods to gate") = GDD **Phase 0** (vertical slice); `sprint-2` (milestone "magic") = GDD **Phase 1**. GDD itself mixes "Sprint 1" (§7) with "Phase 2/3" (§7, §12). Resolved by the CONTRIBUTING mapping table, not by renumbering.

**Decisions of record**
- **New specs carry no `sprint-N` milestone.** Consistent with the vocabulary decision and safe against `repo-sync.yml` (an unknown milestone would fail the sync); each spec names its GDD phase in-body.
- **Friction block is descriptive only.** No auto-flagging threshold that would *reclassify* a death — that needs live friction distributions from issue 06; deferred as a follow-up inside backlog 07.
- **Existing milestones left untouched.** `sprint-1`/`sprint-2` are in flight; renaming would break the sync and orphan open issues.

**Determinism impact.** All five items: **none.** No engine code, no `/engine` change, no `CanonicalBytes` or vendored-PRNG change, nothing on the anti-scope list (GDD §12).

**Golden hash — unchanged.** No engine changes were permitted or made this session. (This branch was merged up onto the content-schema-v2 work, which independently superseded the goldens; this session added nothing to that.)

---

## 2026-07-11 — Issue 01: content schema v2 — pond & per-seed facts

First sprint-1 issue. One cloud session, phone-only. Implements the "plant" half of the slice loop (GDD §7).

**Shipped**
- Zone-1 map extended from the two-room stub to the full topology: `clearing → forest_path → gate`, with the optional `still_pond` branch off `forest_path` (GDD §7). Movement stays a `perform` on `legs`; illegal moves stay a structured `illegal_move` rejection.
- First per-seed hidden fact — **eye color**. Drawn from a closed, frozen-order palette (`blue, green, brown, grey`) that lives in content data. Selection is `SplitMix64(Subseed(seed, "facts.eye_color")).Next() % 4` — the vendored rng only, integer arithmetic, no reordering (ADR-000 D5.3). The value is a pure function of the seed, re-derived on demand and **never stored in State**, so `CanonicalBytes` stays frozen and untouched.
- New engine event kind `observed{fact,value}`. `inspect reflection` at the pond emits `observed{eye_color, <word>}`; `inspect self` in the clearing observes species and hair, **never eyes** (GDD §7). Narrated from the content pack; the color reaches narration and packets only on the pond round.
- Scripted agent now walks the whole slice (clearing → forest_path → still_pond → inspect → forest_path → gate); stdio shell surfaces observations.
- Tests: eye color stable across repeated Init; ≥2 distinct colors across seeds; canonical seed-0 = grey; observed only via the pond reflection; the value leaks into no event or packet before the pond; clearing→forest_path→gate leaves the fact unobserved.

**Decisions of record**
- **Seed 0 is the canonical world; its eye color is `grey`** (GDD §3, §5.3) — documented in `content/zone1/README.md` as the memorizable reference. Fittingly, "the pond, unbothered, remains grey" (GDD §5.7).
- The engine tracks no observations — recall is the agent's job, not the referee's. This keeps the frozen canonical encoding untouched by a content change.
- Palette order is frozen (append-only if it ever grows): the index is the answer, so reordering would silently move every seed's value.
- Golden replay kept on **seed 1** (existing) to minimize churn; seed 1's eye color is `brown`.

**Golden hash — superseded (content change).** New map + palette change the content hash and the world, so the golden replays are regenerated (Contributing "How goldens supersede"):
- content_hash: `sha256:ff4e528bfc4d087650e91f14e1ca8fbd1ea906da697e2ba533a6c7136ccf5f97` → `sha256:12378e265ea91cea2ada8b575bf951fadcf1e048cb5ee3a2494c514c2cd97eed`
- final_state_hash: `sha256:e5d2a27270ff2d396a1203a3e628b797d0a46b2516c1ff6f25946b154c8fc27d` → `sha256:964206448bc0af164f7dfc3c32568ba12362330f3e87c6bfe76e5eaf2054aaca` (reproduced identically across two separate `verifygolden` runs).

---

## 2026-07-11 — Day 0 (Saturday)

Everything below happened in one day, phone-only, via Claude Code cloud sessions. No local machine was involved at any point. Design conversation and the sprint-0 launch happened from the beach, between open-water swim intervals.

**Design finalized**
- GDD v0.1 → v0.2: added §11 Release, Licensing & Business Model (open-core Apache-2.0 born open; training grounds vs sealed seasons with the rotation law; replay-as-proof leaderboard; Steam as confirmed Phase 4 target; publicity aimed at the evals/AI community, minimum publicity unit = vertical slice + multi-model results post).
- ADR-000 drafted and committed: pure deterministic reducer, event-sourced output as the single seam, state = seed + canonical action log, wire protocol v1, determinism laws (integer-only, no map iteration, vendored splitmix64, frozen CanonicalBytes), replay format v1, sealed season packs, shells host the core.

**Shipped**
- Repo created: `xenomancer`, public, Apache-2.0. Name collision check done (no shipped game with the title; soft collisions only — D&D creature, soundtrack track, usernames). Formal trademark screening deferred to before the Phase 2 publicity post.
- **PR #1 (merged)** — sprint-0 walking skeleton: `/engine` pure reducer with closed resource/verb sets, movement + wait + tick, structured rejections as Events; vendored splitmix64 + FNV sub-seeding; `CanonicalBytes()` + SHA-256 state hash; replay v1 encode/decode/verify; stdio JSONL shell; scripted agent (clearing → forest_path, wait ×2); golden replay test; CI matrix ubuntu/macos/windows + cross-process determinism job + grep guard on `/engine`.
- Hardening pass on PR #1: import audit clean; map-iteration audit clean; `CanonicalBytes` FROZEN comment added; `Verify()` now fails loudly on content-hash mismatch before folding (with test); `.gitattributes` LF policy added — **root cause of the Windows-only CI failure was autocrlf corrupting hash-addressed content**. The determinism machinery caught a real environment-dependent bug on the project's first PR.
- Golden replay hash: `sha256:e5d2a27270ff2d396a1203a3e628b797d0a46b2516c1ff6f25946b154c8fc27d` — unchanged through the hardening pass, as required.
- **PR #2 (in flight)** — process: YAML issue forms (feature / bug / unfair-death / tuning), PR checklist template enforcing the determinism laws, labels-as-code, `repo-sync.yml` (bash+curl, idempotent) materializing backlog files into issues on merge, six sprint-1 backlog specs (content schema v2 + pond/eye-color; wolf hazard framework; gate guard + death reports; parser v1; bidirectional runner; naive LLM agent + gated showcase), CONTRIBUTING with the publication policy and the ADR-001 deferral. Final pre-merge items: `workflow_dispatch` trigger, pagination/state=all check on the title-existence query, recovery-procedure comment.

**Decisions of record**
- Monorepo retained; engine/content split deferred to ADR-001, trigger = first sealed season (Phase 3). Multi-repo cloud sessions now exist, which removes the tooling objection but not the architectural ones (golden replays couple engine tests to content; sprint-1 schema churn; no atomic cross-repo commit).
- Public issues cover engine, mechanics, and training-grounds content only; season/narrative-surprise content is designed privately and ships only as sealed packs.
- Project board set to private (Projects visibility is independent of repo visibility).
- Donations expected ≈ €0, written down on purpose. The publicity asset is the leaderboard + death reports, audience = evals community.

**Next**
- Merge PR #2 → watch the first live `repo-sync` run in Actions (its first real execution) → six issues appear.
- Issues 01–05, one cloud session each, in order. Add `ANTHROPIC_API_KEY` repo secret before 06.
- Issue 06 milestone: first frontier model meets the guard. Save the death report — it seeds the results post.
