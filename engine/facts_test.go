package engine

import "testing"

// factsContent is the full Zone-1 slice map (clearing, forest_path, still_pond,
// gate) with the eye-color palette, built in-process so these tests need no
// filesystem. It mirrors content/zone1/map.json.
func factsContent(t *testing.T) Content {
	t.Helper()
	c, err := ParseContent([]byte(`{
		"start_location": "clearing",
		"eye_color_palette": ["blue", "green", "brown", "grey"],
		"locations": [
			{"id": "clearing", "exits": ["forest_path"], "inspectables": [{"id": "self", "reveals": "self"}]},
			{"id": "forest_path", "exits": ["clearing", "still_pond", "gate"]},
			{"id": "still_pond", "exits": ["forest_path"], "inspectables": [{"id": "reflection", "reveals": "eye_color"}]},
			{"id": "gate", "exits": ["forest_path"]}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	return c
}

func move(target string) RoundSubmission {
	return sub(Action{Resource: "legs", Verb: VerbPerform, Target: target})
}

func inspect(target string) RoundSubmission {
	return sub(Action{Resource: "attention", Verb: VerbInspect, Target: target})
}

// drive folds a sequence of submissions from state s, returning the final state
// and every event emitted along the way, in order.
func drive(t *testing.T, s State, subs ...RoundSubmission) (State, []Event) {
	t.Helper()
	var all []Event
	for i := 0; i < len(subs); i++ {
		ns, evs, err := Reduce(s, subs[i])
		if err != nil {
			t.Fatalf("Reduce step %d: %v", i, err)
		}
		s = ns
		all = append(all, evs...)
	}
	return s, all
}

// observeEyeColor walks clearing -> forest_path -> still_pond, inspects the
// reflection, and returns the observed eye-color value for the given seed.
func observeEyeColor(t *testing.T, c Content, seed uint64) string {
	t.Helper()
	_, events := drive(t, Init(seed, c), move("forest_path"), move("still_pond"), inspect("reflection"))
	for i := 0; i < len(events); i++ {
		if events[i].Kind == EventObserved && events[i].Fact == FactEyeColor {
			return events[i].Value
		}
	}
	t.Fatalf("seed %d: no observed{eye_color} event at the pond", seed)
	return ""
}

func inPalette(c Content, v string) bool {
	for i := 0; i < len(c.EyeColorPalette); i++ {
		if c.EyeColorPalette[i] == v {
			return true
		}
	}
	return false
}

// Same seed -> identical eye color across repeated Init (determinism).
func TestEyeColorStableAcrossInit(t *testing.T) {
	c := factsContent(t)
	for _, seed := range []uint64{0, 1, 2, 42, 1000, 1 << 40} {
		a := observeEyeColor(t, c, seed)
		b := observeEyeColor(t, c, seed)
		if a != b {
			t.Errorf("seed %d: eye color differs across Init: %q vs %q", seed, a, b)
		}
		if !inPalette(c, a) {
			t.Errorf("seed %d: eye color %q is not in the closed palette", seed, a)
		}
	}
}

// Canonical world: seed 0 is documented as grey (GDD §3; content README).
func TestEyeColorCanonicalSeedZero(t *testing.T) {
	c := factsContent(t)
	if got := observeEyeColor(t, c, 0); got != "grey" {
		t.Errorf("canonical seed 0 eye color = %q, want grey", got)
	}
}

// Several distinct seeds -> at least two distinct colors (per-seed variance is
// the difficulty, not obscurity; GDD §3).
func TestEyeColorVariesAcrossSeeds(t *testing.T) {
	c := factsContent(t)
	var seen []string
	for seed := uint64(0); seed < 16; seed++ {
		col := observeEyeColor(t, c, seed)
		isNew := true
		for i := 0; i < len(seen); i++ {
			if seen[i] == col {
				isNew = false
				break
			}
		}
		if isNew {
			seen = append(seen, col)
		}
	}
	if len(seen) < 2 {
		t.Errorf("expected at least two distinct eye colors across seeds, got %v", seen)
	}
}

// observed{eye_color} is emitted only after inspecting the pond reflection —
// inspecting the reflection anywhere else, or inspecting anything else at the
// pond, does not observe it.
func TestEyeColorOnlyViaPondReflection(t *testing.T) {
	c := factsContent(t)

	// Exactly one observed{eye_color} when inspecting the reflection at the pond.
	_, events := drive(t, Init(1, c), move("forest_path"), move("still_pond"), inspect("reflection"))
	n := 0
	for i := 0; i < len(events); i++ {
		if events[i].Kind == EventObserved && events[i].Fact == FactEyeColor {
			n++
		}
	}
	if n != 1 {
		t.Errorf("observed{eye_color} count at pond = %d, want exactly 1", n)
	}

	// Inspecting "reflection" from the clearing (no such inspectable there) is a
	// rejection, not an observation.
	_, events = drive(t, Init(1, c), inspect("reflection"))
	assertNoEyeColorEvent(t, c, 1, events)
	if !hasEvent(events, EventRejected) {
		t.Error("inspecting reflection where it does not exist should reject")
	}

	// Self-inspection in the clearing reveals a fact, but never eye_color.
	_, events = drive(t, Init(1, c), inspect("self"))
	if !hasObservedFact(events, "self") {
		t.Error("self-inspection in the clearing should observe the 'self' fact")
	}
	assertNoEyeColorEvent(t, c, 1, events)
}

// The eye-color value appears in no event before the pond observation: walking
// the world and self-inspecting never leaks it.
func TestEyeColorNotLeakedBeforePond(t *testing.T) {
	c := factsContent(t)
	const seed = 1
	color := observeEyeColor(t, c, seed)

	// Everything an agent can do before reaching the pond: self-inspect, walk
	// around, wait, try the gate — none may carry the eye color.
	_, events := drive(t, Init(seed, c),
		inspect("self"),
		move("forest_path"),
		sub(Action{Resource: "attention", Verb: VerbWait}),
		move("clearing"),
		move("forest_path"),
		move("gate"),
		move("forest_path"),
	)
	for i := 0; i < len(events); i++ {
		if events[i].Value == color {
			t.Errorf("event %d (kind %q) leaked the eye color %q before the pond", i, events[i].Kind, color)
		}
		if events[i].Kind == EventObserved && events[i].Fact == FactEyeColor {
			t.Errorf("event %d observed eye_color before the pond was inspected", i)
		}
	}
}

// Moving clearing -> forest_path -> gate without visiting the pond leaves the
// fact unobserved.
func TestEyeColorUnobservedWithoutPond(t *testing.T) {
	c := factsContent(t)
	final, events := drive(t, Init(1, c), move("forest_path"), move("gate"))
	if final.Location != "gate" {
		t.Fatalf("final location = %q, want gate", final.Location)
	}
	assertNoEyeColorEvent(t, c, 1, events)
}

func assertNoEyeColorEvent(t *testing.T, c Content, seed uint64, events []Event) {
	t.Helper()
	for i := 0; i < len(events); i++ {
		if events[i].Kind == EventObserved && events[i].Fact == FactEyeColor {
			t.Errorf("unexpected observed{eye_color} event: %+v", events[i])
		}
		if events[i].Value != "" && inPalette(c, events[i].Value) {
			t.Errorf("event %d carries a palette word %q where no eye color should appear", i, events[i].Value)
		}
	}
}

func hasObservedFact(events []Event, fact string) bool {
	for i := 0; i < len(events); i++ {
		if events[i].Kind == EventObserved && events[i].Fact == fact {
			return true
		}
	}
	return false
}
