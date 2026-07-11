# ADR-000 — Deterministic Reducer & Protocol Contract

| | |
|---|---|
| **Status** | Proposed — becomes Accepted when merged by the owner |
| **Date** | 2026-07-11 |
| **Deciders** | Riccardo |
| **Informs / informed by** | GDD v0.2 (pillars P1–P4, §9–§11) |
| **Supersedes** | — (founding record) |

## Context

XENOMANCER is a machine-first text game whose value depends on properties most
games never need:

1. **Replay-as-proof.** A submitted run (seed + action log) must be
   re-executable by any third party — or by a trusted CI verifier for sealed
   season content — and byte-reproduce the outcome. The leaderboard has zero
   infrastructure only if this holds.
2. **Hermetic, free CI.** Development is phone-only via GitHub Actions; the
   engine's own tests must run deterministically at zero token cost, across
   runner OSes and Go versions.
3. **Open referee, sealed content.** The engine is public (Apache-2.0);
   leaderboard seasons ship as encrypted data packs in the same repo. Rules and
   content must therefore be strictly separable.
4. **Many shells, one core.** stdio today; HTTP and a Steam spectator client
   later. None of them may fork game logic.
5. **LLM strictly outside the rules path.** The player's agent is an LLM; the
   parser may one day use one; the narrator may one day be decorated by one.
   None of this may influence state transitions or replay (GDD P1).
6. **Fair-doom auditability.** "Misparse never kills" and "no fuse without
   telegraph" are testable claims only if the rules engine is inspectable and
   its behavior reproducible.

The forces conflict in one specific way: maximum expressiveness for content
authors and agent inputs versus bit-exact reproducibility across machines and
years. This ADR resolves that conflict by construction.

## Decision

### D1 — Pure reducer core

`/engine` is a Go library with no I/O, no wall-clock, no goroutines, no globals,
no LLM, and no dependency on any shell:

```go
func Init(seed uint64, content Content) State
func Reduce(s State, sub RoundSubmission) (State, []Event, error)
```

Every world rule — legality, hidden facts, fuses, ritual steps, death — lives
behind `Reduce`. `error` is reserved for programmer misuse (malformed
submission struct); in-game rejection is an **Event**, not an error.

### D2 — Event-sourced output; events are the only seam

`Reduce` emits an ordered `[]Event` (e.g. `moved`, `observed{fact}`,
`telegraph{stage}`, `ritual_step{k}`, `rejected{reason}`, `died{report}`).
Every other component is a pure consumer of events:

- **Narrator:** `Narrate(events, seed, verbosity) → text` (templates + seeded
  variants; separate package).
- **Scoring:** extracted from events of a verified replay, never self-reported.
- **Spectator client (Phase 4):** renders the event stream; needs nothing else.
- **Replays:** store actions; events are re-derived on replay.

Nothing consumes engine internals. If a consumer needs data, the reducer emits
an event for it.

### D3 — State is seed + canonical action log

Persistence is the append-only log of canonical `RoundSubmission`s. Save = the
log; load = replay from `Init`. Snapshots are permitted as a cache, never as a
source of truth. Consequence: the parser is not in the replay path — replays
contain only canonical actions, so parser evolution (including a future LLM
fallback on the agent side) can never invalidate a replay.

### D4 — Wire protocol v1 (JSON lines)

Every line carries `"v": 1`. Closed verb set: `inspect | perform | talk | wait`.

**Round envelope (agent → engine):**

```json
{"v":1,"round":17,"actions":[
  {"resource":"legs","verb":"perform","target":"path_north","args":{}},
  {"resource":"attention","verb":"inspect","target":"treeline","args":{}}
]}
```

**Observation packet (engine → agent):**

```json
{"v":1,"round":18,"narration":"...","holds":[{"resource":"hand_right","tag":"mana_hold","since":12}],"result":{"ok":true,"rejections":[]}}
```

Death report: schema per GDD §5.7, delivered as the terminal packet. Freeform
text is accepted only by the parser package, which maps it to this schema via
the versioned dictionary; only canonical actions enter the engine and the log.

### D5 — Determinism laws (non-negotiable, CI-enforced)

1. **Integer arithmetic only** in state and rules (ticks, fuses, damage,
   prices). No floats anywhere in `/engine`.
2. **No map iteration** in any state- or event-affecting path. Ordered slices
   or explicitly sorted keys only.
3. **Vendored PRNG.** A ~30-line splitmix64 (or PCG32) lives in-repo. The stdlib
   `math/rand` and `math/rand/v2` are banned from `/engine` — stdlib generator
   behavior has changed across Go versions and would silently break replays. All
   randomness derives from the run seed via documented sub-seeding:
   `subseed(domain) = seed ⊕ fnv64(domain)`, one stream per system (world-gen,
   NPC values, narration-variant selection).
4. **No time, env, filesystem, or network** in `/engine`.
5. **Content is inert, hash-addressed data.** JSON packs validated at load;
   identified by SHA-256 of plaintext. The engine binary and the content pack
   version independently.
6. **Canonical encoding is a frozen API.** `State.CanonicalBytes()` defines a
   fixed field-order binary encoding; `state_hash = SHA-256(CanonicalBytes)`.
   Changing it is a breaking engine version.

**CI enforcement:** (a) a determinism test replaying a corpus of logs twice and
comparing hashes; (b) the same test across an OS matrix (ubuntu / macos /
windows runners) asserting identical hashes; (c) a lint pass rejecting
`math/rand`, `time.`, and float types inside `/engine`.

### D6 — Replay file format v1

```json
{"header":{"engine_version":"v0.3.1","protocol_version":1,
           "content_hash":"sha256:-","seed":1234,
           "bracket":"scripted","meta":{}},
 "log":[ -canonical round envelopes- ],
 "final_state_hash":"sha256:-"}
```

Verification = load content by hash (public, or decrypted by the trusted season
verifier), `Init`, fold the log, compare `final_state_hash`, extract score from
re-derived events. A replay is valid only against its exact
`{engine_version, content_hash}`; old engine versions therefore remain tagged
and buildable forever.

### D7 — Sealed season packs

Season content ships in the public repo as an encrypted blob (age or AES-GCM);
the key exists only in the trusted GHA verifier's secrets. Replay headers
reference the plaintext hash, so when a season retires and its pack is decrypted
into the training grounds (GDD §11 rotation law), all historical season replays
become publicly re-verifiable with no format change.

### D8 — Shells host the core; wall-clock stays outside

`shell/stdio` (now): JSONL over stdin/stdout; the optional wall-clock deadline
is shell configuration; a timeout is resolved by the shell injecting a canonical
`wait` into the log. `shell/http` and the spectator client (later) wrap the
identical reducer and event stream. The wall-clock can never influence the
reducer except through that injected canonical action — replays remain exact
regardless of live-timing behavior.

## Alternatives considered

- **LLM in the referee** (parsing or adjudication at runtime): rejected —
  destroys determinism, replay-as-proof, and fair-doom auditability (GDD P1).
  Quarantined instead (parser outside the log; narration decoration
  outbound-only, off by default).
- **HTTP-first service:** rejected for now — requires a host; violates the
  phone-only constraint; stdio is CI-native. HTTP is a later shell, not a
  foundation.
- **Snapshot persistence as source of truth:** rejected — reintroduces
  state-encoding drift, is larger, is not self-verifying. Logs are smaller,
  diffable, and are the proof object.
- **stdlib `math/rand(/v2)`:** rejected — cross-version behavioral risk on a
  frozen-forever surface.
- **Float math:** rejected — cross-platform reproducibility risk for zero
  modeling benefit in a tick-based world.
- **ECS or an existing game framework:** rejected — KISS; a reducer plus events
  is the entire requirement, and frameworks smuggle in nondeterminism
  (schedulers, float transforms, iteration order).

## Consequences

**Positive.** Hermetic zero-cost engine CI; a leaderboard with no servers
(replays are proofs); the spectator/Steam client reduces to an event renderer;
sealed seasons without engine forks; every fairness claim in the GDD is
mechanically testable; save/load for free.

**Accepted costs.** Reducer-purity discipline — every rule must be expressible
as data + fold, even when a shortcut would be quicker. Two surfaces are frozen
forever (CanonicalBytes encoding, vendored PRNG); changing either is a major
version that orphans old replays, so old tags must stay buildable. All content
is authored as data, which front-loads schema design. The determinism laws need
active enforcement (lint + matrix test), not good intentions.

**Follow-ups.** ADR-001 content-pack schema & sealing mechanics · ADR-002
leaderboard verification workflow & bracket policy · ADR-003 parser dictionary
format & rejection telemetry.
