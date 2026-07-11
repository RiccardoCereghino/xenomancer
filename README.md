# XENOMANCER

A text-mode infiltration game where an alien agent — usually a literal AI — must
observe, remember, and follow the rules of a magical medieval society perfectly,
because the first mistake is usually the last. The world is a pure, seeded,
deterministic reducer; the player is an LLM agent, a script, or a human-piloted
hybrid. The game lives in the gap between a rule-bound world and a forgetful
agent.

This repository is **sprint 0: the engine walking skeleton plus determinism
CI** — a two-location zone, a canonical reducer, replay-as-proof, a JSONL shell,
a scripted agent, and the CI that keeps it all bit-reproducible. It is not the
game yet (no pond, wolf, guard, or parser — those are later sprints); it is the
frozen foundation everything else folds onto.

## Layout

| Path | What |
|---|---|
| `engine/` | Pure, seeded, deterministic reducer: `Init`, `Reduce`, `CanonicalBytes`/`StateHash`, replay format v1. No I/O, no time, no floats, no goroutines (ADR-000 D1/D5). |
| `engine/internal/rng/` | Vendored splitmix64 + `subseed(domain) = seed ⊕ fnv64(domain)`. The stdlib generators are banned in `/engine`. |
| `content/zone1/` | Inert, hash-addressed content: the location graph (`map.json`) and narration templates (`narration.json`). |
| `shell/stdio/` | JSONL loop: round envelopes in, observation packets out (ADR-000 D4). Hosts the core; owns all I/O and narration. |
| `agent/scripted/` | Deterministic scripted-bracket agent: walk clearing → forest_path, wait twice, exit. |
| `scripts/` | Determinism guard + golden-replay verifier used by CI. |
| `docs/` | [GDD v0.2](docs/design/GDD-v0.2.md) · [ADR-000](docs/adr/ADR-000-deterministic-reducer-and-protocol.md) |

## Build & test

Requires a Go toolchain ≥ 1.22. No external dependencies (stdlib only).

```sh
go build ./...
go test ./...
```

## Run the scripted agent

The scripted agent emits canonical round envelopes; the stdio shell folds them
through the engine and narrates the result. Pipe one into the other:

```sh
go run ./agent/scripted/main | go run ./shell/stdio
```

Expected output (one observation packet per round — walk to the forest path,
then two waits):

```json
{"v":1,"round":2,"narration":"You move to the forest_path. A forest path threads north ...","holds":[],"result":{"ok":true,"rejections":[]}}
{"v":1,"round":3,"narration":"You wait. Six in-world seconds turn without you.","holds":[],"result":{"ok":true,"rejections":[]}}
{"v":1,"round":4,"narration":"You wait. Six in-world seconds turn without you.","holds":[],"result":{"ok":true,"rejections":[]}}
```

## Protocol (v1)

Every line carries `"v": 1` (ADR-000 D4). A **round envelope** (agent → engine):

```json
{"v":1,"round":1,"actions":[{"resource":"legs","verb":"perform","target":"forest_path"}]}
```

The engine replies with an **observation packet** (engine → agent):

```json
{"v":1,"round":2,"narration":"You move to the forest_path. ...","holds":[],"result":{"ok":true,"rejections":[]}}
```

Resources are the closed set `voice | hand_left | hand_right | legs | attention`;
verbs are the closed set `inspect | perform | talk | wait` (movement is a
`perform` on `legs`). In-game refusals — resource conflicts, unknown
verbs/targets, illegal moves — come back as structured rejections, never as
errors.

## Replay-as-proof

A run is persisted as its seed plus its canonical action log; the final state
hash makes it third-party verifiable (ADR-000 D3/D6). Replay the committed
golden log and print its reproduced hash:

```sh
go run ./scripts/verifygolden
```

CI runs this twice, on separate processes, and asserts the hashes are identical;
the OS matrix (`ubuntu`, `macos`, `windows`) asserts the same committed hash on
every platform. A lint guard fails the build if `math/rand`, `time.`, or float
types appear anywhere under `/engine`.

## License

Apache-2.0. See [LICENSE](LICENSE).
