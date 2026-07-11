package engine

import "testing"

// testContent is the two-location graph from content/zone1/map.json, built
// in-process so the reducer tests need no filesystem.
func testContent(t *testing.T) Content {
	t.Helper()
	c, err := ParseContent([]byte(`{
		"start_location": "clearing",
		"locations": [
			{"id": "clearing", "exits": ["forest_path"]},
			{"id": "forest_path", "exits": ["clearing"]}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	return c
}

func sub(actions ...Action) RoundSubmission {
	return RoundSubmission{V: ProtocolVersion, Round: 1, Actions: actions}
}

func TestReduceLegality(t *testing.T) {
	tests := []struct {
		name string
		// from is the starting location for the round.
		from string
		in   RoundSubmission
		// wantLoc is the expected location after the round.
		wantLoc string
		// wantTicked is whether the round advanced tick/round.
		wantTicked bool
		// wantReasons is the ordered rejection reasons expected (nil = none).
		wantReasons []string
		// wantMoved is whether a moved event is expected.
		wantMoved bool
	}{
		{
			name:       "legal move clearing to forest_path",
			from:       "clearing",
			in:         sub(Action{Resource: "legs", Verb: VerbPerform, Target: "forest_path"}),
			wantLoc:    "forest_path",
			wantTicked: true,
			wantMoved:  true,
		},
		{
			name:       "legal move back forest_path to clearing",
			from:       "forest_path",
			in:         sub(Action{Resource: "legs", Verb: VerbPerform, Target: "clearing"}),
			wantLoc:    "clearing",
			wantTicked: true,
			wantMoved:  true,
		},
		{
			name:       "wait ticks and does not move",
			from:       "clearing",
			in:         sub(Action{Resource: "attention", Verb: VerbWait}),
			wantLoc:    "clearing",
			wantTicked: true,
		},
		{
			name:       "empty round is a pass",
			from:       "clearing",
			in:         sub(),
			wantLoc:    "clearing",
			wantTicked: true,
		},
		{
			name:        "illegal move to non-adjacent/unknown location",
			from:        "clearing",
			in:          sub(Action{Resource: "legs", Verb: VerbPerform, Target: "nowhere"}),
			wantLoc:     "clearing",
			wantTicked:  true,
			wantReasons: []string{ReasonIllegalMove},
		},
		{
			name:        "unknown verb",
			from:        "clearing",
			in:          sub(Action{Resource: "legs", Verb: "dance", Target: "jig"}),
			wantLoc:     "clearing",
			wantTicked:  true,
			wantReasons: []string{ReasonUnknownVerb},
		},
		{
			name:        "unknown target via inspect (no inspectables this sprint)",
			from:        "clearing",
			in:          sub(Action{Resource: "attention", Verb: VerbInspect, Target: "pond"}),
			wantLoc:     "clearing",
			wantTicked:  true,
			wantReasons: []string{ReasonUnknownTarget},
		},
		{
			name:        "unknown target via talk (no NPCs this sprint)",
			from:        "clearing",
			in:          sub(Action{Resource: "voice", Verb: VerbTalk, Target: "guard"}),
			wantLoc:     "clearing",
			wantTicked:  true,
			wantReasons: []string{ReasonUnknownTarget},
		},
		{
			name:        "unknown resource",
			from:        "clearing",
			in:          sub(Action{Resource: "tail", Verb: VerbWait}),
			wantLoc:     "clearing",
			wantTicked:  true,
			wantReasons: []string{ReasonUnknownResource},
		},
		{
			name: "resource conflict rejects whole round with no tick",
			from: "clearing",
			in: sub(
				Action{Resource: "legs", Verb: VerbPerform, Target: "forest_path"},
				Action{Resource: "legs", Verb: VerbWait},
			),
			wantLoc:     "clearing",
			wantTicked:  false,
			wantReasons: []string{ReasonResourceConflict},
		},
		{
			name: "non-conflicting multi-action round resolves and ticks once",
			from: "clearing",
			in: sub(
				Action{Resource: "legs", Verb: VerbPerform, Target: "forest_path"},
				Action{Resource: "attention", Verb: VerbInspect, Target: "treeline"},
			),
			wantLoc:     "forest_path",
			wantTicked:  true,
			wantMoved:   true,
			wantReasons: []string{ReasonUnknownTarget},
		},
	}

	c := testContent(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Init(0, c)
			s.Location = tt.from
			startTick, startRound := s.Tick, s.Round

			ns, events, err := Reduce(s, tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ns.Location != tt.wantLoc {
				t.Errorf("location = %q, want %q", ns.Location, tt.wantLoc)
			}

			ticked := ns.Tick == startTick+1 && ns.Round == startRound+1
			notTicked := ns.Tick == startTick && ns.Round == startRound
			if tt.wantTicked && !ticked {
				t.Errorf("expected tick/round to advance once: tick %d->%d round %d->%d", startTick, ns.Tick, startRound, ns.Round)
			}
			if !tt.wantTicked && !notTicked {
				t.Errorf("expected no tick/round advance: tick %d->%d round %d->%d", startTick, ns.Tick, startRound, ns.Round)
			}

			gotReasons := rejectedReasons(events)
			if !equalStrings(gotReasons, tt.wantReasons) {
				t.Errorf("rejection reasons = %v, want %v", gotReasons, tt.wantReasons)
			}

			if gotMoved := hasEvent(events, EventMoved); gotMoved != tt.wantMoved {
				t.Errorf("moved event = %v, want %v", gotMoved, tt.wantMoved)
			}
		})
	}
}

func TestReduceErrorIsProgrammerMisuseOnly(t *testing.T) {
	c := testContent(t)
	s := Init(0, c)

	// Missing verb is a malformed struct -> error, not a rejection Event.
	if _, _, err := Reduce(s, sub(Action{Resource: "legs"})); err == nil {
		t.Error("expected error for action with empty verb")
	}
	// Missing resource is a malformed struct -> error.
	if _, _, err := Reduce(s, sub(Action{Verb: VerbWait})); err == nil {
		t.Error("expected error for action with empty resource")
	}
	// A well-formed but in-game-illegal action must NOT error.
	if _, _, err := Reduce(s, sub(Action{Resource: "legs", Verb: VerbPerform, Target: "nowhere"})); err != nil {
		t.Errorf("illegal move must be a rejection Event, not an error: %v", err)
	}
}

func TestReduceIsPure(t *testing.T) {
	c := testContent(t)
	s := Init(7, c)
	in := sub(Action{Resource: "legs", Verb: VerbPerform, Target: "forest_path"})

	before := s.StateHash()
	ns1, _, _ := Reduce(s, in)
	ns2, _, _ := Reduce(s, in)

	if s.StateHash() != before {
		t.Error("Reduce mutated the input state")
	}
	if ns1.StateHash() != ns2.StateHash() {
		t.Error("Reduce is not deterministic across calls")
	}
}

func rejectedReasons(events []Event) []string {
	var out []string
	for i := 0; i < len(events); i++ {
		if events[i].Kind == EventRejected {
			out = append(out, events[i].Reason)
		}
	}
	return out
}

func hasEvent(events []Event, kind string) bool {
	for i := 0; i < len(events); i++ {
		if events[i].Kind == kind {
			return true
		}
	}
	return false
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
