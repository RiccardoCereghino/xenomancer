---
title: "docs: ADR-001 — local tooling & engine-exposure contract (no shadow engine)"
labels: [docs, adr-needed]
---

## Summary

Write ADR-001 (the deliverable is a **decision record, not code**) fixing the one way the
engine is exposed to tooling. The dashboard (backlog 11) and the CLI consumers (backlog 16/17)
all need the engine's results; without a recorded contract, an agent will eventually argue for
embedding `/engine` as a library inside the dashboard and quietly grow a second, drifting copy
of the rules. This ADR draws the "no shadow engine" line so that argument is already settled.

## References

- ADR-000 D1 (pure deterministic reducer), D3 (parser quarantine — only canonical actions
  reach the reducer/log), D6 (replay format), **D8** (shells host the core; wall-clock stays
  outside — the optional localhost `shell/http` this ADR permits derives from D8).
- CONTRIBUTING "Content & repo policy"; the ADR-000 table + `### DN` structure this ADR reuses.
- Informs: backlog 11 (dashboard boundary clause), 16 (`cmd/validate`), 17 (`cmd/trace`).

## Pillar

**P1 — the referee is public and singular** and **P4 — legible to pilots.** One engine, exposed
one way; tooling consumes it, never reimplements it.

## Spec

ADR-001 records that the engine is exposed to tooling **only** through:

1. **CLI consumers** (`cmd/validate`, `cmd/trace`, `cmd/run`) — pure event-stream consumers /
   content validators. They shell out or read the canonical log; they do not re-derive rules.
2. **(Optional, later) a localhost-only `shell/http`** per ADR-000 D8 — the same reducer, with
   wall-clock outside, never public.

And that it forbids:

- **No library-level embedding of `/engine`** inside the dashboard or any tool. The dashboard
  reads content data and renders the event stream produced by the CLIs; it never links the
  reducer to compute, judge, or score (this is the boundary clause of backlog 11, elevated to
  an ADR so it is not re-argued).

Format: `# ADR-001 — …` heading, the two-column metadata table (Status `Proposed — becomes
Accepted when merged by the owner`, Date, Deciders, Informs / informed by, Supersedes), then
`## Context` (numbered forces) and `## Decision` with `### D1 …` subsections. Filed at
`docs/adr/ADR-001-<slug>.md`.

## Definition of done

- `docs/adr/ADR-001-*.md` exists and is **Proposed**, fixing the two allowed exposure paths and
  the "no library embedding" prohibition precisely enough that a reviewer can reject a shadow
  engine by citing it.
- Cross-referenced from backlog 11's boundary clause and from CONTRIBUTING where relevant.
- A dated DEVLOG entry records the decision.

## Determinism impact

**none.** An ADR governs a boundary, not the reducer. It adds no `State`, touches no
`CanonicalBytes` or the vendored PRNG, and changes no rule.

## Anti-scope check

Checked against GDD §12. A boundary-defining ADR — not a doom clock, combat, inventory, UX
polish, an LLM in the rules path, or repo-secrecy.
