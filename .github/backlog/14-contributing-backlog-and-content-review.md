---
title: "docs: CONTRIBUTING — backlog-file convention + agent-content review + branch naming"
labels: [docs, content]
---

## Summary

Fold three process rules into CONTRIBUTING so the workflow that produced this cycle's audit
divergences can't recur. It **absorbs #13** (content-review workflow) and adds the backlog-file-
origin rule and the branch-naming rule. The deliverable is process, not tooling — no new bot,
label automation, or CI gate.

## References

- GDD §11 (AI-disclosure posture: the shipped game contains no un-reviewed AI-generated
  player-facing content), §5.7 (epitaphs are "the best writing in the game" — they earn review),
  §5.2 & §13 (the dictionary is AI-authored and shipped as data; the rejection log is its
  backlog).
- ADR-000 Context 2 (phone-only development via GHA — review must be doable from a phone),
  D5.5 (content is inert, hash-addressed data authored separately from the engine).
- `.github/workflows/repo-sync.yml` (backlog-as-code: one `.github/backlog/NN-*.md` spec → one
  issue, create-only by exact title). CONTRIBUTING "The loop: issue → session → PR" and
  "Content & repo policy" (the new rules slot beside these).
- Absorbs **#13**; closes audit **Divergence 02**.

## Pillar

**P5 — goofy surface, rigorous core** (surface prose is a deliberate craft object) and **P3 —
fair doom** (telegraph prose and epitaphs are the legible half of a fair death).

## Spec

Add/adjust three CONTRIBUTING rules:

### 1. Backlog-file origin (closes Divergence 02)

Every issue that produces code **must** originate from — or be back-filled as — a
`.github/backlog/NN-*.md` spec. Issues filed live in a session get **retro-filed** as backlog
specs before build, so the backlog stays the complete source of truth (repo-sync materializes
specs by exact title; retro-files whose title matches an existing issue create no duplicate).

### 2. Reviewing agent-drafted content (absorbs #13)

- **Scope.** Player-facing, agent-drafted strings: dictionary entries, telegraph prose,
  location/observation narration, epitaph templates. Engine code and pure data plumbing are out.
- **Batch size.** A small, phone-reviewable cap (starting value: **≤ 20 entries or ~40 lines of
  prose per content-review PR**), larger drafts split across PRs.
- **Accept/reject convention.** A phone-friendly sign-off (per-entry checklist or an explicit
  `Reviewed-by:` line); rejected entries are struck and sent back to the drafting agent, not
  silently edited in review.
- **The merge rule (non-negotiable).** No agent-authored narration or epitaph merges without an
  **explicit human approval**, line-by-line or batch-by-batch. Silence is not approval; an
  un-reviewed player-facing string is a blocking review comment, not a nit (enforces GDD §11).

### 3. Branch naming (kills the task-4/#4 trap)

Branches are keyed to the **issue** number, **never** the backlog-file number. Backlog-file
ordinals and issue numbers deliberately diverge (repo-sync assigns issue numbers on
materialization); naming a branch after the file number reintroduces the concurrent-batch
collision this rule exists to prevent.

## Definition of done

- CONTRIBUTING carries the three rules above, slotted beside the existing "issue → session → PR"
  loop and "Content & repo policy" without contradicting them.
- The content-review section cross-references GDD §11 and §5.7 and stays lightweight (in-PR
  sign-off, no external tool).
- A dated DEVLOG entry records the decision.

## Determinism impact

**none.** A documentation/process change to CONTRIBUTING. It touches no `/engine`,
`CanonicalBytes`, PRNG, or content-pack bytes.

## Anti-scope check

Checked against GDD §12. A contributor-process doc — not a doom clock, combat, inventory, player
UX, an LLM in the rules path, or repo-secrecy. Explicitly stays lightweight to avoid tooling
scope creep.
