---
title: "feature: parser dictionary v2 — coverage + parse-miss guidance"
labels: [feature, adr-needed]
---

> **Retro-filed 2026-07-20 for issue #21** (live-filed 2026-07-18) — closes audit
> Divergence 02 (every issue that produces code originates from a backlog spec). This file
> mirrors the existing issue so the backlog is the complete source of truth; repo-sync skips
> it (the title already matches #21) and creates no duplicate.
>
> **Status: active — Track C2.** Not superseded. Build it after backlog 10 (so the parser can
> learn the new `move` verb). As showcase content grows, the rejection log is this issue's
> backlog — revisit whenever episodes show parser thrash.

> Re-scoped after the 2026-07-18 design session. The root cause of the first in-character run's
> failures was not the parser — it was that the guard has no conversational depth. That headline
> moved to #22 (now backlog 10). This issue is the narrower, genuine parser-coverage companion.

## Summary

The dictionary is thin, so much of what a model says never becomes a valid action. Broaden
coverage and add context-sensitive target resolution, keeping the parser a **pure deterministic
lookup** (ADR-000 D3/D5) — no fuzzy/LLM matching. Also make an unparseable line's rejection
*guiding* rather than a dead end.

## References

- GDD §13 (the rejection log is the dictionary's backlog; thin coverage means agents fight the
  parser, not the game), §5.2 (parse failure is a free, no-tick rejection; P3 misparse-never-kills).
- ADR-000 D3 (parser quarantine — only canonical actions reach the reducer/log; parser is
  deterministic), D5 (determinism laws — pure lookup, no rng/model), ADR-003 (planned parser
  dictionary format, `adr-needed`).

## Pillar

**P4 — built for machines, legible to pilots.** If natural phrasing can't reach a valid action,
the interface — not the world — is what the agent fights.

## Evidence — first in-character run (`29641428835`, seeds 0/1/2)

1. **"say open the gate" → `unknown_target`.** A talk whose noun is "gate" resolves to the gate
   location, so speech never reaches the guard. Talk should resolve to the NPC present.
2. **"look in the pond / at the water / at myself" → `unknown_target`.** Only `reflection` is
   inspectable; seed 2 stood at the mirror 3× and never learned its eye color.
3. **Filler-led prose → free waits (16–21/episode).** "i point to the gate and say may i pass"
   etc. don't match verb+target and become no-ops.

## Spec

- **Coverage + context-sensitive resolution:** talk + a place where an NPC stands ⇒ that NPC;
  inspect + a place ⇒ its inspectable feature (pond/water/"myself" ⇒ `reflection`). Strip leading
  filler ("i ", "i say", "please", "i approach the … and say") to the operative verb + claim.
- **Learn the new movement verb from backlog 10** (don't hardcode `perform`/`legs`).
- **Guiding parse-miss response (shell-side):** an unparsed line returns a short, non-spoiling
  nudge ("Say it plainly — one short action"), never revealing the puzzle. (In-world NPC dialogue
  is backlog 10.)
- **Stays deterministic and quarantined:** a recognized verb token is still required; a real miss
  is still a free, no-tick rejection; the load-bearing property test (gibberish never yields a
  canonical action) still passes, plus regression cases lifted verbatim from the run above.

## Definition of done

Replaying the three transcripts, the three failure classes above are handled and the
unparsed→wait share drops from ~40–50% to **<15%**. **Golden unchanged** (parser + shell live
outside `/engine`).

## Determinism impact

**none.** Parser and shell responses are outside `/engine`; nothing enters `CanonicalBytes`.
Parser stays a deterministic lookup (no floats/rng/model).

## Anti-scope check

Checked against GDD §12. Parser-coverage + parse-miss prose only — not a mechanic, doom clock, or
an LLM in the parser lethal path. Explicitly out of scope: fuzzy/embedding/LLM matching, NPC
dialogue depth (backlog 10), and any engine rule change.
