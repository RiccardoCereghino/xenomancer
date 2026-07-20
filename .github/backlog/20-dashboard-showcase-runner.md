---
title: "feature: dashboard showcase runner (shells out to cmd/run + local ollama)"
labels: [feature]
---

## Summary

The second dashboard view (backlog 11): a showcase runner that runs episodes by **shelling out to
`cmd/run`** against a local Ollama, and streams the resulting events into the telemetry view
(backlog 19). It is the local showcase surface — Mac mini, localhost, reached over Tailscale. It
runs the engine as a subprocess; it never embeds it.

## References

- GDD §11 (results/telemetry & local showcases), §7 (Zone 1 content plan the showcase exercises).
- ADR-000 D8 (shells host the core; the runner owns wall-clock, the reducer never sees it), D6
  (replays are the artifact the runner produces), ADR-001 (engine-exposure contract — shell out,
  don't embed).
- Part of: backlog 11 (dashboard). Feeds: backlog 19 (telemetry view). Local model: Ollama on the
  same host.

## Pillar

**P4 — legible to pilots** and **P5 — goofy surface, rigorous core.** A one-click local showcase
whose rigor is guaranteed because it runs the real engine subprocess.

## Spec

- **Run episodes by shelling out to `cmd/run`** with a chosen seed and a local Ollama model as the
  agent; capture the canonical log / replay it produces (ADR-000 D6/D8).
- **Stream events into the telemetry view** (backlog 19) as episodes run.
- **Boundary clause (from backlog 11).** The runner may run `cmd/run` and render its event stream;
  it may **not** compute, judge, or score anything the reducer computes. Wall-clock/timeouts stay
  in the runner (ADR-000 D8); the reducer only ever sees canonical `wait`s injected into the log.
- **Local only.** Localhost + Tailscale; Ollama on the same host. No public surface, no engine
  dependency added.

## Definition of done

- The runner runs a Zone 1 episode via `cmd/run` + local Ollama and streams its events into the
  telemetry view.
- Its output events are byte-identical to a direct `cmd/run` of the same seed + log (parity).
- The boundary-audit CI grep passes (no resolution logic in runner code).

## Determinism impact

**none.** Outside `/engine`. It invokes `cmd/run` as a subprocess and reads its output; nothing
enters `CanonicalBytes` or the state hash. Timeout-injected waits follow ADR-000 D8 (log stays
byte-identical).

## Anti-scope check

Checked against GDD §12. A local showcase runner — not a mechanic, doom clock, combat, inventory,
game UX, an LLM in the rules path, or repo-secrecy. It respects the public/sealed content split.
