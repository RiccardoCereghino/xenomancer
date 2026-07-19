// Package engine is the pure, seeded, deterministic reducer at the heart of
// XENOMANCER. It performs no I/O, reads no wall-clock, spawns no goroutines,
// holds no globals, and depends on no shell or LLM (ADR-000 D1).
//
// Determinism laws it obeys (ADR-000 D5), enforced by CI:
//   - Integer arithmetic only; no floating-point types anywhere in this tree.
//   - No map iteration in any state- or event-affecting path; ordered slices
//     or explicit scans only.
//   - Randomness (when needed) derives from the run seed via the vendored
//     splitmix64 in engine/internal/rng; the stdlib generators are banned.
//   - No time, environment, filesystem, or network access.
//   - State.CanonicalBytes defines a frozen field-order encoding; the state
//     hash is SHA-256 over it.
package engine

import "encoding/json"

// Version is the engine version stamped into replay headers (ADR-000 D6).
// Changing CanonicalBytes or the vendored PRNG is a breaking change and must
// bump this. 0.2.0 added the terminal outcome (Outcome/Cause) to the frozen
// CanonicalBytes encoding — the gate guard's win/death (GDD §5.7, §7). 0.3.0
// appended the hazard fuse and grapple counters (Fuse, GrappleRoundsLeft,
// GrappleStruggles) — the wolf's local doom clock (GDD §5.6).
const Version = "0.3.0"

// ProtocolVersion is the wire-protocol version. Every line carries "v": 1
// (ADR-000 D4).
const ProtocolVersion = 1

// Resource is one of the closed set of exclusive body resources a round
// submission may claim (GDD §5.1).
type Resource = string

// The closed resource set. Kept as an ordered slice (not a map) so membership
// checks never depend on iteration order (ADR-000 D5.2).
var resourceSet = []Resource{
	"voice",
	"hand_left",
	"hand_right",
	"legs",
	"attention",
}

// The closed verb set (GDD §5.2, ADR-000 D4). Movement is a perform on legs.
const (
	VerbInspect = "inspect"
	VerbPerform = "perform"
	VerbTalk    = "talk"
	VerbWait    = "wait"
)

// Rejection reason codes. In-game rejection is always an Event, never an error
// (ADR-000 D1); error is reserved for programmer misuse.
const (
	ReasonResourceConflict = "resource_conflict"
	ReasonUnknownVerb      = "unknown_verb"
	ReasonUnknownResource  = "unknown_resource"
	ReasonUnknownTarget    = "unknown_target"
	ReasonIllegalMove      = "illegal_move"
	// ReasonUnclearClaim is the guard's response to a talk claim that names
	// zero or several palette words (GDD §5.4). It costs one round and NEVER
	// kills — the round still resolves and ticks.
	ReasonUnclearClaim = "unclear_claim"
)

// Death causes (GDD §5.7 cause taxonomy). CauseClaimWrong is the first
// contextual/social class: a legal, understood, wrong eye-color claim at the
// gate — a fair death (P3). CauseHazardWolf is the first hazard class: a fuse
// that ran out with the agent still grappled (GDD §5.6). The concrete cause a
// hazard emits is content-driven (Hazard.Cause); this constant is the canonical
// value the shipped wolf uses.
const (
	CauseClaimWrong = "social.claim_wrong"
	CauseHazardWolf = "hazard.wolf"
)

// Outcome is the terminal state of an episode. The empty string means the
// episode is ongoing; once set, the episode has ended (GDD §5.7, §7) and Reduce
// is a no-op. Outcome is part of CanonicalBytes so a win and a death that end at
// the same location/round hash differently (ADR-000 D5.6).
const (
	OutcomeWon  = "won"
	OutcomeDied = "died"
)

// Event kinds emitted by Reduce. Events are the only seam between the reducer
// and every downstream consumer — narrator, scoring, spectator, replay
// (ADR-000 D2). This sprint emits moved, waited, rejected, observed, the hazard
// ladder (telegraph, grappled, struggled, freed), and the terminal died/won; the
// remaining kind (ritual_step) arrives with later content.
const (
	EventMoved    = "moved"
	EventWaited   = "waited"
	EventRejected = "rejected"
	// EventTelegraph is a fuse warning woven into narration at a content-defined
	// fuse point (GDD §5.6). Its Stage is the ladder rung (1..N); the shell maps
	// the stage to prose. A telegraph is a free signal — it never ticks or kills;
	// missing it under noise is the attention test, and every telegraph that
	// fired before a hazard death is listed in the death report (P3).
	EventTelegraph = "telegraph"
	// EventGrappled fires the round a hazard's fuse reaches its length and the
	// hazard seizes the agent, creating the sustained Holds (GDD §5.1/§5.6).
	EventGrappled = "grappled"
	// EventStruggled reports a successful struggle against a grapple (a perform on
	// a held resource with the hazard's struggle target). Enough struggles inside
	// the window break the grapple.
	EventStruggled = "struggled"
	// EventFreed fires when the required struggles land in time: the Holds are
	// released and the fuse resets (GDD §5.6).
	EventFreed = "freed"
	// EventObserved reports a fact the agent perceived this round (GDD §5.3).
	// The eye-color observation at the still pond is the first instance: its
	// Value is the per-seed palette word (see Content.eyeColor). Recall of an
	// observed fact is the agent's responsibility, never the engine's — the
	// reducer stores no observation in State, only emits the event.
	EventObserved = "observed"
	// EventDied is the terminal death event: it carries a full DeathReport in
	// Event.Report and ends the episode (GDD §5.7, ADR-000 D2 died{report}).
	EventDied = "died"
	// EventWon is the terminal win event: the agent is inside the walls
	// (GDD §7). Its Round stamps the rounds elapsed.
	EventWon = "won"
)

// Action is a single resource claim within a round (ADR-000 D4). Args is
// carried verbatim as raw JSON; the reducer decodes it into a fixed struct, never
// a map, so no rule depends on iteration order. The gate-guard talk is the first
// user: it reads the claim from Args ({"say":"..."}), matched against the palette.
type Action struct {
	Resource Resource        `json:"resource"`
	Verb     string          `json:"verb"`
	Target   string          `json:"target"`
	Args     json.RawMessage `json:"args,omitempty"`
}

// RoundSubmission is the round envelope (agent -> engine), ADR-000 D4:
//
//	{"v":1,"round":17,"actions":[ ... ]}
//
// Only canonical submissions enter the engine and the replay log; freeform
// text is mapped to this shape by the (out-of-engine) parser, so the replay
// path never depends on parser behavior (ADR-000 D3).
type RoundSubmission struct {
	V       int      `json:"v"`
	Round   int      `json:"round"`
	Actions []Action `json:"actions"`
}

// Hold is a sustained resource claim spanning rounds (GDD §5.1). No verbs
// create holds this sprint, but the field is part of the canonical encoding.
type Hold struct {
	Resource Resource `json:"resource"`
	Tag      string   `json:"tag"`
	Since    uint64   `json:"since"`
}

// Event is an ordered, structured record of something that happened in a
// round. Fields are flat and typed (no maps) so the encoding order is fixed.
// Fact/Value carry an observation (EventObserved): Fact is the fact key
// (e.g. "eye_color"), Value its per-seed word. They stay empty on other kinds.
// Report carries the death report on EventDied and is nil otherwise. Events are
// re-derived on replay and never enter CanonicalBytes, so the nested Report does
// not affect the state hash (ADR-000 D3).
type Event struct {
	Kind     string       `json:"kind"`
	To       string       `json:"to,omitempty"`
	Reason   string       `json:"reason,omitempty"`
	Resource Resource     `json:"resource,omitempty"`
	Verb     string       `json:"verb,omitempty"`
	Target   string       `json:"target,omitempty"`
	Fact     string       `json:"fact,omitempty"`
	Value    string       `json:"value,omitempty"`
	// Stage is the telegraph ladder rung on EventTelegraph (1..N) and zero on
	// every other kind (GDD §5.6). Like all Event fields it is re-derived on
	// replay and never enters CanonicalBytes.
	Stage    int          `json:"stage,omitempty"`
	Report   *DeathReport `json:"report,omitempty"`
	Tick     uint64       `json:"tick"`
	Round    uint64       `json:"round"`
}

// DeathReport is the structured post-mortem emitted on EventDied (GDD §5.7). It
// is a first-class packet — the centaur loop runs on it — and is delivered as
// the terminal packet (ADR-000 D4). All fields are typed and ordered (no maps)
// so the encoding is deterministic.
type DeathReport struct {
	Cause             string          `json:"cause"`
	Detail            DeathDetail     `json:"detail"`
	Round             uint64          `json:"round"`
	TelegraphsIgnored []string        `json:"telegraphs_ignored"`
	RitualProgress    *RitualProgress `json:"ritual_progress"`
	Epitaph           string          `json:"epitaph"`
}

// DeathDetail is the cause-specific detail of a death. For social.claim_wrong it
// records the NPC, what it asked, what the agent claimed, and the truth. For a
// hazard death it records the hazard id and the last telegraph stage the agent
// was shown before it died. All fields are omitempty so each cause serializes
// only the detail it fills; the report is re-derived, never hashed, so extending
// this struct does not touch CanonicalBytes.
type DeathDetail struct {
	NPC     string `json:"npc,omitempty"`
	Asked   string `json:"asked,omitempty"`
	Claimed string `json:"claimed,omitempty"`
	Truth   string `json:"truth,omitempty"`
	Hazard  string `json:"hazard,omitempty"`
	Stage   int    `json:"stage,omitempty"`
}

// RitualProgress is the ritual state of a procedural death (Phase 1). No ritual
// deaths exist this sprint, so DeathReport.RitualProgress is always nil and
// serializes to null.
type RitualProgress struct{}

// State is the full world state. It is derived entirely from the seed and the
// canonical action log (ADR-000 D3): Init produces the initial State and each
// Reduce folds one submission into the next.
//
// Content is world data used by the rules; it is deliberately excluded from
// CanonicalBytes because content identity is tracked separately by content
// hash in the replay header (ADR-000 D5.5 / D6).
type State struct {
	Seed     uint64
	Tick     uint64
	Round    uint64
	Location string
	Holds    []Hold

	// Outcome is the terminal state ("" ongoing, OutcomeWon, OutcomeDied); once
	// non-empty the episode has ended and Reduce is a no-op. Cause is the death
	// cause (GDD §5.7 taxonomy), set only on OutcomeDied. Both are part of
	// CanonicalBytes (appended in encoding v2) so the outcome is part of the
	// replay proof — a win and a death do not hash alike.
	Outcome string
	Cause   string

	// Fuse is the current hazard fuse: the count of consecutive rounds the agent
	// has ended in the hazard zone (GDD §5.6). It advances by one per in-zone
	// round and resets to zero the moment the agent ends a round outside the zone
	// ("the wolf will not pursue you past the treeline"). When it reaches the
	// zone hazard's length the agent is grappled.
	//
	// GrappleRoundsLeft is the number of rounds remaining to break a grapple;
	// zero means not grappled. GrappleStruggles counts successful struggles so
	// far in the current grapple. Enough struggles before the rounds run out
	// break free; running out grappled is a hazard death.
	//
	// All three are appended to CanonicalBytes (encoding v3, engine 0.3.0) so the
	// fuse state is part of the replay proof.
	Fuse              uint64
	GrappleRoundsLeft uint64
	GrappleStruggles  uint64

	Content Content
}

// grappled reports whether the agent is currently held by a hazard (GDD §5.6):
// a grapple is active exactly while there are rounds left to break it.
func (s State) grappled() bool {
	return s.GrappleRoundsLeft > 0
}

func isValidResource(r Resource) bool {
	for i := 0; i < len(resourceSet); i++ {
		if resourceSet[i] == r {
			return true
		}
	}
	return false
}
