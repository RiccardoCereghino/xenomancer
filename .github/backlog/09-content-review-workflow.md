---
title: "docs: content review workflow for agent-drafted narration"
labels: [docs, content]
---

## Summary

Add a lightweight **CONTRIBUTING section** (the deliverable is process, not
tooling) defining how agent-drafted content — dictionary entries, telegraph
prose, epitaphs — gets human review, and specifically how it gets reviewed *from
a phone*, since development is phone-only via GitHub Actions (ADR-000 Context 2).
The load-bearing rule: **no agent-authored narration or epitaph merges without an
explicit human approval**, line-by-line or batch-by-batch. This keeps the "best
writing in the game" (GDD §5.7, epitaphs) and every player-facing string under a
human's eye, which the AI-disclosure posture (GDD §11: the shipped game contains
no un-reviewed AI-generated player-facing content) depends on.

## References

- GDD §5.7 (epitaphs are template-generated and "allowed to be the best writing
  in the game" — they earn review), §5.2 & §13 (the dictionary is AI-authored
  offline and shipped as data; the rejection log is its backlog — those
  additions need review too), §11 (AI-disclosure posture: no un-reviewed
  AI-generated player-facing content ships), §3 (narration verbosity/variants are
  authored difficulty, so their prose is design-critical).
- ADR-000 Context 2 (phone-only development via GHA — review must be doable on a
  phone), D5.5 (content is inert, hash-addressed data, authored separately from
  the engine).
- CONTRIBUTING "The loop: issue → session → PR" and "Content & repo policy"
  (this section slots beside them).

## Pillar

**P5 — Goofy surface, rigorous core** (the surface prose is a deliberate craft
object, not filler) and **P3 — Fair doom** (telegraph prose and epitaphs are the
legible half of a fair death; sloppy prose breaks the attention-under-noise
contract and the post-mortem's teaching value).

## Spec

The future issue's deliverable is a new CONTRIBUTING section, **"Reviewing
agent-drafted content,"** defining:

- **What it covers.** Player-facing, agent-drafted strings: dictionary entries
  (freeform→canonical synonyms), telegraph prose, location/observation
  narration, and epitaph templates. Engine code and pure data plumbing are out of
  scope — this is about *words a player reads*.
- **Batch size per review PR.** A small, phone-reviewable cap (propose a concrete
  number, e.g. **≤ 20 entries or ≤ ~40 lines of prose per content-review PR**) so
  a reviewer can actually read every line on a phone. Larger drafts split across
  PRs. The number is a starting value, tunable once real review load exists.
- **Accept/reject convention.** A simple, phone-friendly convention for signing
  off — e.g. a per-entry checklist in the PR body, or an explicit
  `Reviewed-by:`/approval line, with rejected entries struck and sent back to the
  drafting agent rather than silently edited in review. Reviewer taste is the
  bar for prose; the dictionary's bar is "correct canonical mapping."
- **The merge rule (non-negotiable).** No agent-authored narration or epitaph
  merges without an **explicit human approval line-by-line or batch-by-batch.**
  Silence is not approval; an un-reviewed player-facing string is a blocking
  review comment, not a nit. This is the concrete enforcement of GDD §11's
  disclosure posture.

Keep it **lightweight — process, not tooling.** No new bot, label automation, or
CI gate is required by this issue (a reviewer's explicit approval is the gate). If
a future issue wants to *enforce* the merge rule with a check, that is a separate,
later decision.

## Definition of done

- CONTRIBUTING has a "Reviewing agent-drafted content" section covering: scope,
  batch size per review PR, the accept/reject convention, and the explicit-
  human-approval merge rule.
- The section is phone-reviewable in spirit (small batches, in-PR sign-off, no
  external tool needed) and cross-references GDD §11 and §5.7.
- It slots cleanly beside the existing CONTRIBUTING "issue → session → PR" loop
  and "Content & repo policy" sections without contradicting them.

## Determinism impact

**none.** This is a documentation/process change to CONTRIBUTING. It touches no
code, no `/engine`, no `CanonicalBytes`, no PRNG, and no content pack bytes — it
governs how content PRs are *reviewed*, not what the engine does with content.

## Anti-scope check

Checked against GDD §12. Not on the list: this is a review-process doc, not a
global doom clock, upkeep, combat, human-facing UX *product* polish (this is
contributor process, not player UX), inventory, an LLM in the rules path, or
repo-secrecy. Explicitly stays lightweight to avoid tooling scope creep.
