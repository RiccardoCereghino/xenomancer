---
title: "shell: bidirectional runner and agent harness"
labels: [shell, agent, feature]
milestone: sprint-1 — woods to gate
---

## Summary

Add a runner (`cmd/run`) that hosts the engine and speaks JSONL in both
directions with an agent subprocess. This is the harness the CI showcase and
the scripted/centaur agents run on, and the seam where the wall-clock deadline
lives — outside the reducer (ADR-000 D8).

## References

- ADR-000 D8 (shells host the core; the wall-clock stays outside — a timeout is
  resolved by injecting a canonical `wait` into the log), D2/D4 (JSONL wire
  protocol; events are the seam), D6 (replay file format).
- GDD §9 (the protocol is the UI: JSON lines over stdio, round envelope in,
  observation packet out), §10 (CI: hermetic scripted agents).

## Spec

### The runner

- New `cmd/run` binary that:
  - `Init`s the engine from a seed + content pack and folds submissions via
    `Reduce`, emitting observation packets (ADR-000 D4) after each round.
  - Launches an **agent subprocess** via `--agent <cmd>` and speaks **JSONL
    both directions**: observation packets to the agent's stdin, round
    envelopes from the agent's stdout.
  - Writes the canonical action log and, at episode end, a replay file
    (ADR-000 D6 header: engine version, protocol version, content hash, seed,
    bracket; plus the log and final state hash).

### Wall-clock, off by default

- A **wall-clock deadline flag is present but defaults off** (ADR-000 D8,
  GDD §9 — off in tests; 60–90s live). When enabled and an agent misses the
  deadline, the runner injects a canonical `wait` into the log for that round
  (never touches world state directly), so replays stay exact.

### Port the scripted agents

- Port the existing scripted agent(s) to run under `cmd/run` as an `--agent`
  subprocess, so the same harness drives both scripted and (later) LLM agents.

### Regenerate goldens

- Regenerate the golden replays on top of the backlog **01–03** content (new
  map, pond fact, wolf, guard). The regenerated replay is the CI determinism
  fixture. Note the supersession in the PR's golden-hash box.

## Definition of done

- `cmd/run --agent <cmd> --seed <n>` plays a full episode end to end, exchanging
  JSONL with the subprocess and emitting a valid replay file (ADR-000 D6).
- The wall-clock flag exists and is **off by default**; when on, a missed
  deadline injects a canonical `wait` and the replay still reproduces.
- The scripted agent runs under the runner and clears the slice green.
- Golden replays are regenerated on top of 01–03 and the determinism CI passes.

## Out of scope

- The LLM agent and the gated live showcase workflow (backlog 06).
- The HTTP shell and spectator client (ADR-000 D8, Phase 3/4).
- Any change to `/engine` rules — this is a shell/harness that hosts the
  existing reducer unchanged.
