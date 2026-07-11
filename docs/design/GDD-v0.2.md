# XENOMANCER — Game Design Document — v0.2

| | |
|---|---|
| **Status** | Living document — pre-production |
| **Working title** | XENOMANCER (alternates: Fireball Etiquette, The Guard Will Ask) — name collision/trademark check pending, required before anything public |
| **Genre** | Machine-first text infiltration / knowledge-progression game |
| **Platform** | Headless engine (Go, single static binary). Played over a JSON protocol by LLM agents, scripts, or hybrids. Steam release (scripting-first + spectator client) is a confirmed long-term target. |
| **Owner** | Riccardo |

**Changelog.** v0.2 — added §11 Release, Licensing & Business Model (open-core,
content layers & seasonal rotation, replay-as-proof leaderboard, Steam posture,
publicity plan); roadmap, risks, anti-scope updated; event stream formalized as
the spectator/replay seam. · v0.1 — initial consolidation (July 2026).

> A GDD is a promise about what the game is, not a contract about how it is
> built. Technical decisions live in ADRs (see ADR-000); this document owns
> design intent, mechanics, tone, and scope. Sections marked **[OPEN]** are
> known unknowns.

## 1. Vision

**One-liner.** A text-mode infiltration game where an alien agent — usually a
literal AI — must observe, remember, and follow the rules of a magical medieval
society perfectly, because the first mistake is usually the last.

**Elevator pitch.** Ruby Warrior meets Increlution, administered by a bored gate
guard. You crash-land in a duchy that hunts your kind. Your mission: climb the
social ladder and steal the secret of magic. Nothing here requires reflexes,
graphics, or dexterity — only attention, memory, and flawless procedure. The
world is deterministic and rule-bound; the player agent is probabilistic and
forgetful. The game lives in that gap.

**Why it exists.** LLM agents fail in a characteristic way: they attempt
impossible chess moves, skip step 7 of the runbook, and confidently report facts
they never observed. XENOMANCER turns each of those failure modes into a game
mechanic with lethal stakes and a legible post-mortem. It is a game first — with
pacing, story, and jokes — and an evaluation harness second, without pretending
the two are in conflict: the comedy is the benchmark. An alien being too
literal-minded to pass as human is funny for exactly the same reason it is
measurable.

**Audience honesty.** The players are pilots and their agents: people who write,
steer, and debug LLM agents. Humans cannot comfortably play it directly, by
design. "Fun" therefore means: fair deaths, readable post-mortems, escalating
mastery, and a world worth dying in repeatedly — not menus and polish. The
publicity audience is the AI/evals community (see §11), not the general gaming
public; a spectator-facing Steam product comes later, built on the same event
stream.

## 2. Design Pillars

Every feature must serve at least one pillar. A feature that violates a pillar
is cut, however attractive.

- **P1 — The world is the referee, and the referee is public.** The engine is a
  pure, seeded, deterministic reducer. All rules, hidden facts, hazards, and
  death conditions are enforced structurally, with zero LLM inside the rules
  path. Illegal actions are impossible to land, and attempted illegality is
  recorded, not improvised around. The referee's source is open from day one:
  "fair" is auditable, not asserted.
- **P2 — Knowledge is the only progression.** No XP, no stats, no gear
  treadmill. You advance because you know things: your eye color, the third sign
  of the fireball, which guard accepts bribes. The seed is the world; on a fixed
  seed, knowledge transfers across deaths — carried in the agent's context, the
  pilot's notes, or the script itself. The script is the save file. Corollary:
  obscurity is not difficulty — challenge comes from per-seed values and
  structural variance, never from hiding the rules.
- **P3 — Fair doom.** Every death is attributable, telegraphed, and reported.
  Misunderstanding never kills; being understood and wrong does. No fuse without
  a telegraph ladder. An unfair death (parser error, untelegraphed hazard) is
  classified as an engine bug, not a game event.
- **P4 — Built for machines, legible to pilots.** Text volume and response
  deadlines gate out direct human play. Structured death reports and metrics
  make the human-monitors-agent loop (the intended meta-game) actually workable.
- **P5 — Goofy surface, rigorous core.** The tone is bureaucratic lethality — a
  world where paperwork, etiquette, and municipal jurisdiction are matters of
  life and death. Underneath the jokes, every mechanic is deterministic and
  measurable.

## 3. Player, Modes & Brackets

The player is an agent: an LLM loop, a script, or a script supervised by a human
reading death reports and patching behavior (the centaur — expected to be the
winning strategy at high levels).

| Mode | Seed | What it tests | Analogy |
|---|---|---|---|
| Benchmark | Random per run | In-context competence in a single episode: observation, recall under noise, procedure | An exam |
| Knowledge run | Canonical / daily seed | Cross-run learning; mastery of a fixed world | Increlution, roguelike dailies |
| Scripted / Centaur | Either | The pilot's memory architecture and tooling; script robustness against variance knobs | Ruby Warrior |

**Leaderboard brackets** (what the agent may bring): raw context / scaffolded
(notes, scratchpads, tools) / scripted. A scaffolded agent trivially beating a
recall test is not cheating — it is the measurement working. Brackets keep the
comparison honest. (Verification mechanics and their known limits: §11.)

**Difficulty is authored, not emergent.** The knobs below deliberately place the
equilibrium between the three modes per tier:

1. **Narration verbosity** — inflates context; degrades recall; gates humans.
2. **Seeded synonym variance** — same state, many phrasings; breaks regex
   scripts; forces semantic parsing.
3. **Structural variance** — which fact the guard demands, and where it is
   discoverable, varies per seed; forces general observe-and-recall machinery.
4. **Per-seed procedures** — ritual steps vary per seed, learnable only
   in-world; forces agents to implement discovery, not answers. (Late-game
   tier.)

Every knob targets an axis where LLMs and humans genuinely diverge. There are
deliberately no reflex, graphics, or dexterity axes. These knobs — not repo
secrecy — are also the game's durable defense against being "solved by reading
the source" (P2 corollary).

## 4. Core Loop

```
observe (narration packet)
  → hypothesize (what is true? what is legal? what is about to eat me?)
  → act (one round: allocate body resources to verbs)
  → outcome (new observation, or a structured death report)
  → learn (in context, in notes, or in the script)
  → retry (same seed: knowledge compounds / new seed: skill compounds)
```

Session cadence: many short episodes. Early episodes end in instructive deaths
within minutes; mastery is measured in rounds elapsed on a clean run.

## 5. Mechanics

### 5.1 The Round & the Body Economy

Time advances in rounds of six in-world seconds (pure fiction — the engine
stores an integer tick; "6 seconds" appears only in narration). Each prompt is
one round. Within a round, the agent allocates exclusive body resources:

| Resource | Examples |
|---|---|
| voice | talk, chant, scream (inadvisable) |
| hand_left / hand_right | signs, holding, opening, gathering mana |
| legs | movement, fleeing, kicking |
| attention | inspecting, feeling mana, noticing the growl behind you |

A round submission is a set of resource claims. Conflicting claims (two verbs on
one hand) are rejected before resolution — a structured rejection, costing no
tick, resubmittable within the wall-clock. Some verbs create sustained claims
spanning rounds (holding gathered mana, being grappled). Breaking a sustained
claim has system-specific consequences (see 5.5). This single rule generates the
game's composite-legality surface: not just illegal moves, but illegal
combinations — which is where real agents actually fail.

### 5.2 Actions & the Parser

Canonical verbs: `inspect`, `perform`, `talk`, `wait` (movement is a `perform`
on legs). Canonical schema:

```json
{"round": [
  {"resource": "legs", "verb": "perform", "target": "path_north"},
  {"resource": "attention", "verb": "inspect", "target": "treeline"}
]}
```

Agents may also submit freeform text; a quarantined parser maps it to canonical
actions via a versioned synonym dictionary (AI-authored offline, checked into
the repo as data — at runtime it is a deterministic lookup). Parse failure
returns "I don't understand" with no tick cost. Misparse never kills (P3). The
action log stores only canonical actions, so replay is exact regardless of how
sloppily the agent phrased things.

### 5.3 Observation & Recall

Facts are planted in narration — some decorative, some survival-critical much
later. The archetype: your reflection in the still pond reveals your (per-seed)
eye color; fifty verbose rounds later, the gate guard asks for it. Answering
from a prior instead of an observation is the game's thesis failure. On the
canonical seed the value is fixed and memorizable across runs — that is
generational knowledge, not a bug.

### 5.4 Social Checks

NPCs are state machines with a conversational surface — hidden state (what
they'll accept, what they suspect, what they cost) plus deterministic dialogue
templates. Claims made in `talk` are judged by closed palettes and keyword
matching:

- Reply contains exactly one palette word → that is the claim; judged against
  hidden state.
- Zero or several → "Speak plainly, stranger." Costs one round. Never kills.
- One, and wrong → consequence per NPC. For the gate guard: death. Fair death —
  legal, understood, wrong.

No LLM judges anything on the lethal path.

### 5.5 Magic: Rituals as Protocols

Spells are multi-round procedures with resource holds and interleaving — runbook
adherence wearing a robe. Reference spell, **Fireball**:

| Rounds | Resource | Step |
|---|---|---|
| 1 | attention | Feel the mana in the atmosphere |
| 2 → throw | hand_right | Gather mana and hold (sustained claim) |
| odd rounds 3–11 | hand_left | Signs 1–10, in order |
| even rounds 4–10 | voice | Chant syllables between sign phases (rhythm matters) |
| final | hand_right | Throw at target |

**Failstate gradient** (P3 — each is deterministic and reported):

| Failure | Trigger | Consequence | Class |
|---|---|---|---|
| Fizzle | Gather without feeling first | Round wasted; faint static; mild embarrassment | Procedural |
| Decay | Chant off-rhythm / missed syllable | Hold weakens; ritual extends or fizzles | Procedural |
| Misfire | Wrong sign at step k | Burn scaling with k — singe (k≤3), wound (4–7), maiming (8+). Commitment is priced. | Procedural |
| Detonation | Hand conflict or early release during hold | Death. The mana went somewhere. | Procedural |
| Witnessed | Any casting observed by a hostile | Arrest → execution track | Contextual |

The contextual row is the load-bearing one: a flawless fireball in view of the
gate is still doom. Procedural mastery is not safety; the agent must track social
state and procedure simultaneously. That coupling is what makes this a game
rather than a checklist.

**Fairness requirement:** every hidden procedure has an in-world discovery path
with a cost model — a singed spellbook page (partial), a hermit tutor (paid,
quest-gated), a safe practice clearing (with its own hazard fuse). The
interesting test is the explore/exploit economy of buying knowledge with risk,
never brute-forcing permutations against lethal feedback.

### 5.6 Hazards: Local Doom Clocks

There is no global doom clock. Pressure is a property of places and situations:
zone hazards with hidden fuses. The law (P3): no fuse without a telegraph
ladder — escalating signals woven into narration where a skimming agent will
miss them. Hazards are therefore attention-under-noise tests, the sibling of the
recall test.

Reference hazard, the **Wolf** (Deep Woods, fuse = 12 rounds in-zone):

| Fuse | Signal (embedded in normal narration) |
|---|---|
| 6/12 | A howl, distant, almost decorative. |
| 9/12 | Rustling. The howl again — no longer decorative. |
| 11/12 | A low growl, sourced disturbingly locally. |
| 12/12 | The Maul. Grapple (sustained claim on both hands + legs); 2 rounds to break free, else death. |

Leaving the zone pauses and resets the fuse. In-world: "The wolf, a stickler for
jurisdiction, will not pursue you past the treeline." Mechanically clean;
tonally correct.

The pattern generalizes: later hazards include suspicion (a social fuse fed by
risky acts, not time), curfew bells, and the practice-clearing's resident
problem. **[OPEN]** Whether any hazard should ever be un-pausable.

Early game is deliberately slow — the clearing has no fuse. Pacing is level
design, not a timer.

### 5.7 Death & the Post-Mortem

Death ends the episode and emits a structured death report — a first-class
feature, since the centaur loop runs on it:

```json
{
  "cause": "social.claim_wrong",
  "detail": {"npc": "gate_guard", "asked": "eye_color", "claimed": "green", "truth": "grey"},
  "round": 61,
  "telegraphs_ignored": [],
  "ritual_progress": null,
  "epitaph": "He was sure his eyes were green. The pond, unbothered, remains grey."
}
```

Cause taxonomy: `procedural.*` (ritual step k), `contextual.*` (witnessed,
social), `hazard.*` (which, and which telegraph stage was last acknowledged),
`recall.*`, `resource.*` (conflict consequences), and `engine.unfair` — which
must stay at zero and is tracked as a bug class, not a game event. Epitaphs are
template-generated, deterministic, and are allowed to be the best writing in the
game.

## 6. World, Story & Tone

**Premise.** You are a scout of a stranded alien expedition. Your kind cannot
generate magic but can learn it — which is precisely why the Duchy exterminates
you on sight. Detection is hard, so society has grown paranoid rituals of
identity: the Bureau of Ocular Verification, gate interrogations, etiquette with
teeth. Your mission is to pass as human, climb from woods vagrant to court mage,
and transmit the secret of magic home.

**Story arc = the social ladder.** Each rung grants access to deeper knowledge
and imposes harsher checks: Woods → Gate → Town (citizen) → Guild (apprentice) →
Academy → Court. The narrative is delivered diegetically — NPC dialogue
templates, found documents, overheard proclamations, and your own epitaphs —
never cutscenes.

**Tone.** Bureaucratic lethality, played straight. The world is not wacky; it is
rigorous about absurd things, and the player dies of paperwork. The comedy
engine is that an alien's literal-mindedness and confident confabulation mirror
an LLM's — the joke and the benchmark are the same object (P5). Style reference:
Pratchett's civic institutions, Papers, Please's checkpoints, dwarven
bureaucracy.

**[OPEN]** How much backstory the alien gets (armpit bioluminescence is on the
table, pending dignity review).

## 7. Content Plan

### Zone 1 — Woods to Gate (Sprint 1 — the vertical slice)

| Location | Contents |
|---|---|
| Clearing | Wake. Self-inspection (species, hair — not eyes). No hazards. Tutorial-by-narration. |
| Forest path | Wolf fuse active. Branch to pond is optional — and secretly survival-critical. |
| Still pond | `inspect` reflection → eye color observation (per-seed; fixed on canonical seed). |
| Gate | Guard interrogation (eye color, this tier). Win: inside the walls. |

Walking straight to the gate and dying there is the game teaching its thesis.
Sprint 1 contains the entire loop in miniature: plant → noise → recall → lethal
check, plus one hazard ladder.

### Zone 2 — Town (Phase 2)

Square, tavern, notice board with contracts (lodging, work — gradeable
objectives with terms and deadlines), a shopkeeper with hidden price floors
(negotiation as information extraction).

### Zone 3 — The Hermit (Phase 2/3)

Magic tutorial: spellbook fragment, tutor, practice clearing. First ritual:
Sparklight (3 rounds) before Fireball (12).

### Later

Academy (per-seed procedures tier), Court (endgame checks), structural-variance
rollout to all NPC checks. From Phase 3 onward, new content ships in two layers:
training grounds (open) and official seasons (held out, rotating) — see §11.

## 8. Scoring & Metrics

**Per-run score (the game):** outcome; rounds elapsed (the efficiency axis with
an infinite ceiling — this replaces the global clock's job); knowledge items
discovered; telegraphs heeded; contracts fulfilled.

**Pilot metrics (the harness):** illegal-action rate; recovery-after-rejection
rate; recall accuracy; telegraph latency (rounds from first signal to evasive
action); ritual fidelity; deaths-to-clear on a fixed seed. Reported per bracket
(raw / scaffolded / scripted).

Scores are extracted from the event stream of a verified replay (§11), never
self-reported.

## 9. Interface (the protocol is the UI)

- **Transport:** JSON lines over stdio (engine ⇄ agent). An HTTP shell is a
  later wrapper around the same reducer; nothing in the design assumes it.
- **Inbound:** the round envelope (5.2).
- **Outbound:** observation packet — narration (verbosity-knob controlled,
  telegraphs woven in), round number, active sustained claims, last-action
  result, and rejections.
- **The event stream is the seam.** The reducer emits domain events; narration,
  replays, scoring, and the future spectator client (§11, Steam) are all pure
  consumers of the same stream. Nothing consumes anything else.
- **Two clocks, never coupled:** the in-world tick, and a wall-clock deadline
  per response (config: off in tests; 60–90 s live). The deadline plus narration
  volume is the anti-human gate — a human cannot read 2,000 words and act in
  time; a model can. Timeout resolves to a canonical `wait` written to the log,
  so replay stays exact. Wall-clock never touches world state.

## 10. Technical Overview (pointers only — details live in ADRs, starting with ADR-000)

Pure Go reducer, `(state, action) → (state, events)`, seeded; state = seed +
append-only canonical action log; save = the log, load = replay; a state hash
makes replays third-party verifiable. Quarantined dictionary parser
(deterministic at runtime). Template narrator with seeded variants; an optional
LLM "inflation" layer (e.g. local ollama) may later decorate outbound narration
only, must never feed back into state, and ships off by default forever (this is
also the Steam AI-disclosure posture, §11). Season content ships as sealed
(encrypted) data packs addressed by content hash. Shells: stdio now, HTTP later,
spectator client later — same core. CI: hermetic engine tests with scripted
agents (free, deterministic) + a gated live-agent showcase job (spends tokens,
allowed to flake, best-of-3).

**Repo layout:** `/engine`, `/content/<zone>` (dictionaries, maps, templates as
data), `/shell/stdio`, `/agent`, `/docs/adr`, one GHA workflow.

## 11. Release, Licensing & Business Model

**Open-core, born open.** The engine, protocol, parser, and training-ground
content are public under Apache-2.0 from the first commit. Rationale: (a) P1 — a
benchmark with a closed referee will not be trusted, cited, or adopted; "fair
doom" must be auditable; (b) the repo accretes watchers, issues, and portfolio
signal during development — a closed-then-open release launches to silence twice
and burns launch-day trust ("source?" is the first HN comment); (c) clean solo
IP + Apache keeps every future door open. A lightweight CLA is adopted the day
the first outside contribution arrives, preserving relicensing freedom.
Closed-then-open is explicitly rejected.

**Content layers & the rotation law.** Secrecy is a consumable with a half-life
measured in streams and shared transcripts — so it is spent deliberately, never
relied on:

| Layer | Visibility | Purpose |
|---|---|---|
| Training grounds | Open (repo) | Zone 1 + canonical-seed world; onboarding, scripting mode, knowledge runs. Spoiler-tolerant by genre — walkthroughs are the point. |
| Official seasons | Held out (sealed data packs in the public repo; decryption key lives only in the trusted verifier) | Leaderboard integrity while fresh. |
| Rotation | On retirement, a season's content is decrypted and merged into the training grounds | Secrecy expires on schedule instead of leaking on stream; the open corpus only grows. |

Durable difficulty comes from the variance knobs (§3), never from secrecy —
repo-mining reveals distributions, not answers (P2 corollary).

**Leaderboard: replay-as-proof.** A run submission is a replay file (seed,
content hash, canonical action log, state hash). Open-zone replays are
verifiable by anyone's CI; season replays are submitted by PR and verified by
the trusted GHA workflow holding the season key. Scores are extracted from
events, never self-reported. Known, documented limit: a replay cannot prove
wall-clock compliance, so a patient human puppeteering an "LLM bracket" run is
undetectable — brackets are partly honor-system, as on most public benchmarks; a
future attested-runner bracket may tighten this.

**Steam (confirmed long-term target).** The Steam product is scripting mode
first (the Screeps/Bitburner-shaped offering), a spectator/replay client built
on the event stream, and official seasons — the client and seasons are the
proprietary layer on the open core; precedents: Bitburner (free + OSS on Steam),
Mindustry (GPL, sells), Screeps (open engine, paid client + server
subscription). AI-disclosure posture is deliberately trivial: the shipped game
contains no AI-generated player-facing content (templates + deterministic
referee; the player brings the LLM), and the inflation layer stays off by
default; if ever shipped enabled, it triggers the live-generated content regime
(guardrails + disclosure) and is treated as its own release decision.

**Money & attention.** Donations are a tip jar, not a plan: expected ≈ €0,
written down here so it cannot disappoint later. The primary asset is attention
in the AI/evals community; the marketing engine is the leaderboard plus the
death reports. Minimum publicity unit: vertical slice + a multi-model results
post ("we ran N frontier models; here is the one that forgot its own eye color,
and here are the epitaphs"). Nothing before that milestone is visible;
everything after compounds. The durable moat is the trademark, the leaderboard's
authority, and shipping velocity — not the vault. Action item, blocking any
public artifact: name collision & trademark search for XENOMANCER.

## 12. Scope & Roadmap

| Phase | Deliverable | Definition of done |
|---|---|---|
| 0 — Vertical slice | Zone 1 complete: round system, parser, pond fact, wolf ladder, guard check, death reports, seeds. Public repo (Apache-2) from first commit. | Scripted agent clears it green in CI from a phone; a naive LLM agent dies instructively |
| 1 — Magic | Hermit zone, Sparklight + Fireball, procedural failstate gradient | Ritual fidelity measurable; contextual doom (witnessed) works |
| 2 — Town & contracts | Zone 2, negotiation NPCs, contract objectives, structural variance on the guard. Publicity milestone: multi-model results post. | Scripts require general recall machinery to survive variance; the post ships |
| 3 — Leaderboard & seasons | Replay-as-proof verification on open zones; first sealed season + trusted verifier; HTTP shell; live deadlines | Third parties submit verified runs; season 1 rotates into the open corpus on retirement |
| 4 — Steam | Spectator/replay client on the event stream; scripting-first packaging; official seasons on the storefront; narration inflation as an opt-in | Steam release with a trivial AI disclosure |
| Someday | Per-seed procedures tier; MMO shell (shared world, real datastore) | Explicitly out of scope until a host exists; the reducer seam is the insurance |

**Anti-scope (cut on sight):** global doom clock; survival-sim upkeep (hunger,
sleep); combat system beyond hazard grapples; human-facing UX polish before
Phase 4; inventory sprawl; any LLM inside the rules path;
security-through-repo-secrecy (the vault) and closed-then-open releases.

## 13. Risks & Open Questions

| Risk | Mitigation |
|---|---|
| Dictionary coverage too thin → agents fight the parser, not the game | Parser rejections are free and logged; the rejection log is the dictionary's backlog. AI-expand offline, ship as data. |
| Difficulty tuning without playtesters | Agent playtests are cheap and parallel; deaths-to-clear per seed is the tuning signal. |
| Live-run token costs | Hermetic CI by default; live runs gated and tiny; local-model bracket later. |
| Season content leaks early (streams, transcripts) | Priced in: rotation law treats secrecy as a consumable; difficulty rests on variance knobs, not secrecy. |
| Human-piloted runs polluting LLM brackets | Documented honor-system limitation; attested-runner bracket as a later tightening. |
| Name collision / trademark conflict | Search before any public artifact; the trademark is the moat, so this blocks Phase 0 publication. |
| "Is watching an agent die fun?" | Yes, if the epitaphs are good and the post-mortem teaches. This is a design budget line, not an afterthought. |
| Scope creep via worldbuilding | Pillars + anti-scope list; content ships only attached to a mechanic it exercises. |

**[OPEN]** Exact fuse/telegraph numbers (tune in slice). Whether rejections
should rate-limit within a round. Sparklight design. Epitaph template library
size. Season cadence. Alien armpit question.

---

*End of v0.2. Founding technical decisions: see ADR-000 (deterministic reducer &
protocol contract).*
