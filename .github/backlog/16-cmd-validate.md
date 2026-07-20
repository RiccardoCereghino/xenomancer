---
title: "content: cmd/validate — content-pack validator CLI (editor save-time backend)"
labels: [content, feature]
---

## Summary

Add `cmd/validate`: a CLI that validates a content pack against the schema and reports
diagnostics. It is the editor's **save-time backend** (backlog 21's author view saves through it),
and it is the first of the engine-exposure CLI consumers (ADR-001). Golden-neutral: it lives
outside `/engine` and changes no engine bytes. **New tool** — today only `cmd/run` exists.

## References

- GDD §5.2 & §13 (content is inert data; the parser/dictionary and content packs are authored as
  data and loaded by the engine), §11 (public/sealed content policy).
- ADR-000 D1/D3 (the reducer loads content via `ParseContent`; validation must accept exactly
  what the engine accepts), ADR-001 (engine-exposure contract — `cmd/validate` is a CLI consumer,
  not a shadow engine).
- Companion: backlog 18 (the published JSON Schema this validates against), backlog 21 (author
  view save path), backlog 10 (the interaction schema this grows to cover).

## Pillar

**P4 — legible to pilots** and **P1 — the referee is singular.** Content is validated against the
engine's own acceptance, so the editor can never save something the engine won't load.

## Spec

- **Input:** a path to a content pack (map/facts/hazards/narration JSON, per backlog 18's schema).
- **Output:** exit code + **JSON diagnostics on stdout** — a machine-readable list of findings
  (path, rule class, message), so the author view can surface them inline. Design the diagnostic
  format to allow **new rule classes** to be added later (each finding carries a `class`), because
  Track E adds a lethal-telegraph check (refuse any pack where a lethal transition lacks an
  authored telegraph ladder on its approach).
- **Acceptance parity.** A pack that validates must be one `engine.ParseContent` accepts, and vice
  versa — validation is not a second, drifting schema. Prefer validating against backlog 18's
  published JSON Schema plus the same structural checks the engine performs on load.
- **Determinism-neutral.** Outside `/engine`; no floats/rng/model needed. Deterministic output
  order (stable diagnostic ordering) so diffs and the author view are stable.

## Definition of done

- `cmd/validate <pack>` exits non-zero with JSON diagnostics on a bad pack and zero on a good one.
- A pack that passes `cmd/validate` loads under `engine.ParseContent` unchanged (parity test).
- The diagnostic format carries a `class` per finding, documented so Track E's lethal-telegraph
  rule slots in without a format change.
- `/engine` does not import the validator (verifiable via `go list` imports); golden unchanged.

## Determinism impact

**none.** Outside `/engine`. It reads content JSON and emits diagnostics; nothing enters
`CanonicalBytes` or the state hash.

## Anti-scope check

Checked against GDD §12. A content-validator CLI — not a mechanic, doom clock, combat, inventory,
player UX, an LLM in the rules path, or repo-secrecy. It respects the public/sealed content split.
