# XENOMANCER — Dev Log

Convention: one entry per working day, newest first. Entries are appended by the human or by the coding agent as part of a PR ("Append a dated entry to docs/DEVLOG.md summarizing what shipped, decisions made, and the golden hash if it changed"). Terse is fine; decisions and hashes are not optional.

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
