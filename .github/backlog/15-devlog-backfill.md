---
title: "docs: DEVLOG backfill — 2026-07-18 design session + 2026-07-20 direction"
labels: [docs]
---

## Summary

Backfill the DEVLOG so its record matches what actually happened (closes audit **Divergence
01**). One entry covers the 2026-07-18 design session that originated #21–24 (and re-scoped #21),
plus the 2026-07-20 direction decisions this cycle locked in. The deliverable is a DEVLOG entry,
authored to the log's convention (one entry per working day, newest-first, decisions and golden
hash not optional).

## References

- `docs/DEVLOG.md` header (convention: newest-first, appended as part of a PR; decisions and
  hashes mandatory).
- Issues #21 (parser dictionary v2, re-scoped in the session), #22 (NPC interaction state
  machines), #23 (movement verb), #24 (content-authoring + telemetry webapp) — the design
  session's output.
- The 2026-07-20 direction decisions: obstructive-only interactions, verb set v2, lethal-dialogue
  deferral (Track E, #12), the dashboard boundary clause, and Mac-mini / Tailscale deploy.

## Pillar

**P1 — the referee is public/auditable.** The DEVLOG is part of the public record; a gap in it is
a gap in the audit trail.

## Spec

Append one dated DEVLOG entry covering:

- **The 2026-07-18 design session.** Why #22 became the headline ("the guard is a robot"), why
  #21 was re-scoped from "the parser is the problem" to a narrower parser-coverage companion, and
  the origin of #23 (movement-verb cleanup) and #24 (authoring + telemetry webapp).
- **The 2026-07-20 direction decisions.** Obstructive-only interactions (no lethal dialogue this
  bump), verb set v2 (`move | talk | inspect | perform | wait`), lethal-dialogue deferral blocked
  on the telegraph contract (#12), the dashboard boundary clause (no shadow engine), and the
  Mac-mini / localhost / Tailscale deploy with local Ollama.

Golden hash: **unchanged** (docs only).

## Definition of done

- `docs/DEVLOG.md` has the backfill entry, dated and newest-first-consistent, listing the session
  origins and the decisions of record.
- The entry cross-references the issues/specs it covers and states the golden hash is unchanged.

## Determinism impact

**none.** A documentation change. No `/engine`, `CanonicalBytes`, PRNG, or content-pack bytes.

## Anti-scope check

Checked against GDD §12. A record-keeping doc — not a doom clock, combat, inventory, player UX,
an LLM in the rules path, or repo-secrecy.
