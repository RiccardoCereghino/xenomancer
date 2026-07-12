package engine

import (
	"encoding/json"
	"testing"
)

// guardContent is factsContent plus the gate guard NPC and the epitaph
// templates, mirroring content/zone1/map.json. Built in-process so these tests
// need no filesystem.
func guardContent(t *testing.T) Content {
	t.Helper()
	c, err := ParseContent([]byte(`{
		"start_location": "clearing",
		"eye_color_palette": ["blue", "green", "brown", "grey"],
		"epitaphs": [
			"He was sure his eyes were {claimed}. The pond, unbothered, remains {truth}.",
			"variant one: {claimed} / {truth}",
			"variant two: {claimed} / {truth}",
			"variant three: {claimed} / {truth}"
		],
		"locations": [
			{"id": "clearing", "exits": ["forest_path"], "inspectables": [{"id": "self", "reveals": "self"}]},
			{"id": "forest_path", "exits": ["clearing", "still_pond", "gate"]},
			{"id": "still_pond", "exits": ["forest_path"], "inspectables": [{"id": "reflection", "reveals": "eye_color"}]},
			{"id": "gate", "exits": ["forest_path"], "npcs": [{"id": "gate_guard", "asks": "eye_color"}]}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	return c
}

// claim builds a talk-on-voice round addressing the gate guard with the reply
// text carried in Args.
func claim(word string) RoundSubmission {
	return sub(Action{
		Resource: "voice",
		Verb:     VerbTalk,
		Target:   "gate_guard",
		Args:     json.RawMessage(`{"say":"` + word + `"}`),
	})
}

// atGate drives an agent from the clearing to the gate (no pond detour) and
// returns the state standing before the guard.
func atGate(t *testing.T, c Content, seed uint64) State {
	t.Helper()
	s, _ := drive(t, Init(seed, c), move("forest_path"), move("gate"))
	if s.Location != "gate" {
		t.Fatalf("expected to be at the gate, got %q", s.Location)
	}
	return s
}

// A correct single-palette claim wins: outcome won, the round ticks, and no
// died event is emitted.
func TestGuardCorrectClaimWins(t *testing.T) {
	c := guardContent(t)
	s := atGate(t, c, 0) // seed 0 truth = grey
	before := s.Round

	ns, events, err := Reduce(s, claim("grey"))
	if err != nil {
		t.Fatalf("Reduce: %v", err)
	}
	if ns.Outcome != OutcomeWon {
		t.Errorf("outcome = %q, want %q", ns.Outcome, OutcomeWon)
	}
	if ns.Cause != "" {
		t.Errorf("cause = %q, want empty on a win", ns.Cause)
	}
	if ns.Round != before+1 || ns.Tick != s.Tick+1 {
		t.Errorf("round/tick did not advance once: round %d->%d tick %d->%d", before, ns.Round, s.Tick, ns.Tick)
	}
	if !hasEvent(events, EventWon) {
		t.Error("expected a won event")
	}
	if hasEvent(events, EventDied) {
		t.Error("a correct claim must not die")
	}
	won := firstEvent(events, EventWon)
	if won.Round != ns.Round {
		t.Errorf("won event round = %d, want rounds elapsed %d", won.Round, ns.Round)
	}
}

// A wrong single-palette claim dies with a fully populated death report.
func TestGuardWrongClaimDies(t *testing.T) {
	c := guardContent(t)
	s := atGate(t, c, 0) // seed 0 truth = grey

	ns, events, err := Reduce(s, claim("green"))
	if err != nil {
		t.Fatalf("Reduce: %v", err)
	}
	if ns.Outcome != OutcomeDied {
		t.Errorf("outcome = %q, want %q", ns.Outcome, OutcomeDied)
	}
	if ns.Cause != CauseClaimWrong {
		t.Errorf("cause = %q, want %q", ns.Cause, CauseClaimWrong)
	}
	if !hasEvent(events, EventDied) {
		t.Fatal("expected a died event")
	}
	if hasEvent(events, EventWon) {
		t.Error("a wrong claim must not win")
	}
	ev := firstEvent(events, EventDied)
	if ev.Report == nil {
		t.Fatal("died event carries no report")
	}
	r := ev.Report
	if r.Cause != CauseClaimWrong {
		t.Errorf("report.cause = %q, want %q", r.Cause, CauseClaimWrong)
	}
	if r.Detail.NPC != "gate_guard" || r.Detail.Asked != FactEyeColor {
		t.Errorf("report.detail npc/asked = %q/%q, want gate_guard/eye_color", r.Detail.NPC, r.Detail.Asked)
	}
	if r.Detail.Claimed != "green" || r.Detail.Truth != "grey" {
		t.Errorf("report.detail claimed/truth = %q/%q, want green/grey", r.Detail.Claimed, r.Detail.Truth)
	}
	if r.Round != ns.Round {
		t.Errorf("report.round = %d, want rounds elapsed %d", r.Round, ns.Round)
	}
	if r.Epitaph == "" {
		t.Error("report.epitaph is empty")
	}
	if r.TelegraphsIgnored == nil || len(r.TelegraphsIgnored) != 0 {
		t.Errorf("report.telegraphs_ignored = %v, want empty non-nil slice", r.TelegraphsIgnored)
	}
	if r.RitualProgress != nil {
		t.Error("report.ritual_progress must be nil this sprint")
	}
}

// Zero or several palette words: "speak plainly", costs one round, never kills.
func TestGuardUnclearClaimNeverKills(t *testing.T) {
	c := guardContent(t)
	cases := []struct {
		name string
		word string // the reply text
	}{
		{"zero palette words", "i would rather not say"},
		{"empty reply", ""},
		{"two palette words", "blue or maybe green"},
		{"all palette words", "blue green brown grey"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := atGate(t, c, 0)
			ns, events, err := Reduce(s, claim(tc.word))
			if err != nil {
				t.Fatalf("Reduce: %v", err)
			}
			if ns.Outcome != "" {
				t.Errorf("outcome = %q, want ongoing (never kills)", ns.Outcome)
			}
			if ns.Round != s.Round+1 || ns.Tick != s.Tick+1 {
				t.Error("an unclear claim must still cost one round")
			}
			if hasEvent(events, EventDied) || hasEvent(events, EventWon) {
				t.Error("an unclear claim must not end the episode")
			}
			if reasons := rejectedReasons(events); !equalStrings(reasons, []string{ReasonUnclearClaim}) {
				t.Errorf("reasons = %v, want [%s]", reasons, ReasonUnclearClaim)
			}
		})
	}
}

// An empty Args (no reply at all) also yields "speak plainly", not a panic.
func TestGuardEmptyArgsIsUnclear(t *testing.T) {
	c := guardContent(t)
	s := atGate(t, c, 0)
	ns, events, err := Reduce(s, sub(Action{Resource: "voice", Verb: VerbTalk, Target: "gate_guard"}))
	if err != nil {
		t.Fatalf("Reduce: %v", err)
	}
	if ns.Outcome != "" {
		t.Errorf("outcome = %q, want ongoing", ns.Outcome)
	}
	if reasons := rejectedReasons(events); !equalStrings(reasons, []string{ReasonUnclearClaim}) {
		t.Errorf("reasons = %v, want [%s]", reasons, ReasonUnclearClaim)
	}
}

// Talking to the guard anywhere but the gate, or naming an NPC that is not here,
// is an unknown target — not an interrogation.
func TestGuardOnlyAtGate(t *testing.T) {
	c := guardContent(t)

	// The guard is not in the clearing.
	_, events, err := Reduce(Init(0, c), claim("grey"))
	if err != nil {
		t.Fatalf("Reduce: %v", err)
	}
	if reasons := rejectedReasons(events); !equalStrings(reasons, []string{ReasonUnknownTarget}) {
		t.Errorf("clearing reasons = %v, want [%s]", reasons, ReasonUnknownTarget)
	}

	// A wrong NPC id at the gate is unknown, too.
	s := atGate(t, c, 0)
	_, events, err = Reduce(s, sub(Action{Resource: "voice", Verb: VerbTalk, Target: "innkeeper", Args: []byte(`{"say":"grey"}`)}))
	if err != nil {
		t.Fatalf("Reduce: %v", err)
	}
	if reasons := rejectedReasons(events); !equalStrings(reasons, []string{ReasonUnknownTarget}) {
		t.Errorf("wrong-npc reasons = %v, want [%s]", reasons, ReasonUnknownTarget)
	}
}

// Once terminal, Reduce is a sticky no-op: no tick, identical state hash, no
// events — a dead or winning agent takes no further rounds.
func TestReduceTerminalIsStickyNoOp(t *testing.T) {
	c := guardContent(t)
	for _, tc := range []struct {
		name  string
		seed  uint64
		reply string
	}{
		{"after win", 0, "grey"},
		{"after death", 0, "green"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := atGate(t, c, tc.seed)
			terminal, _, err := Reduce(s, claim(tc.reply))
			if err != nil {
				t.Fatalf("Reduce: %v", err)
			}
			if terminal.Outcome == "" {
				t.Fatalf("expected a terminal outcome, got ongoing")
			}
			before := terminal.StateHash()

			// Any further submission is ignored entirely.
			after, events, err := Reduce(terminal, move("forest_path"))
			if err != nil {
				t.Fatalf("Reduce (post-terminal): %v", err)
			}
			if len(events) != 0 {
				t.Errorf("post-terminal Reduce emitted %d events, want 0", len(events))
			}
			if after.StateHash() != before {
				t.Error("post-terminal Reduce changed the state")
			}
			if after.Location != terminal.Location {
				t.Errorf("post-terminal Reduce moved the agent to %q", after.Location)
			}
		})
	}
}

// matchClaim is closed-palette containment over an ordered slice: exactly one
// distinct palette word is a claim; zero or several are not.
func TestMatchClaim(t *testing.T) {
	palette := []string{"blue", "green", "brown", "grey"}
	cases := []struct {
		reply     string
		wantWord  string
		wantCount int
	}{
		{"brown", "brown", 1},
		{"my eyes are GREY", "grey", 1}, // case-insensitive
		{"", "", 0},
		{"purple", "", 0},
		{"blue and green", "green", 2},
		{"blue green brown grey", "grey", 4},
	}
	for _, tc := range cases {
		word, n := matchClaim(palette, tc.reply)
		if n != tc.wantCount {
			t.Errorf("matchClaim(%q) count = %d, want %d", tc.reply, n, tc.wantCount)
		}
		if n == 1 && word != tc.wantWord {
			t.Errorf("matchClaim(%q) word = %q, want %q", tc.reply, word, tc.wantWord)
		}
	}
}

// Epitaph selection is deterministic per seed and, for the canonical seed 0
// death, reproduces the GDD §5.7 reference line exactly.
func TestEpitaphDeterministicAndCanonical(t *testing.T) {
	c := guardContent(t)

	// Stable across repeated calls for a fixed seed.
	a := c.epitaph(7, "blue", "brown")
	b := c.epitaph(7, "blue", "brown")
	if a != b {
		t.Errorf("epitaph not stable for seed 7: %q vs %q", a, b)
	}

	// Canonical seed 0 death (claimed green, truth grey) — the GDD §5.7 example.
	got := c.epitaph(0, "green", "grey")
	const want = "He was sure his eyes were green. The pond, unbothered, remains grey."
	if got != want {
		t.Errorf("canonical seed-0 epitaph = %q, want %q", got, want)
	}
}

func firstEvent(events []Event, kind string) Event {
	for i := 0; i < len(events); i++ {
		if events[i].Kind == kind {
			return events[i]
		}
	}
	return Event{}
}
