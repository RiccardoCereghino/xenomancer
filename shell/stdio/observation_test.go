package main

import (
	"os"
	"strings"
	"testing"

	"github.com/RiccardoCereghino/xenomancer/engine"
	"github.com/RiccardoCereghino/xenomancer/parser"
)

// loadZone1 reads the real zone-1 content pack and narration from disk (a test
// may do I/O; the engine may not).
func loadZone1(t *testing.T) (engine.Content, narration) {
	t.Helper()
	mapBytes, err := os.ReadFile("../../content/zone1/map.json")
	if err != nil {
		t.Fatalf("read map.json: %v", err)
	}
	c, err := engine.ParseContent(mapBytes)
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	narBytes, err := os.ReadFile("../../content/zone1/narration.json")
	if err != nil {
		t.Fatalf("read narration.json: %v", err)
	}
	nar, err := loadNarration(narBytes)
	if err != nil {
		t.Fatalf("loadNarration: %v", err)
	}
	return c, nar
}

// containsPaletteWord reports whether s mentions any eye-color palette word.
func containsPaletteWord(c engine.Content, s string) (string, bool) {
	lower := strings.ToLower(s)
	for i := 0; i < len(c.EyeColorPalette); i++ {
		if strings.Contains(lower, c.EyeColorPalette[i]) {
			return c.EyeColorPalette[i], true
		}
	}
	return "", false
}

// The eye-color value appears in no PACKET (narration or structured
// observations) before the pond reflection is inspected, and does appear on the
// round it is (GDD §5.3). This exercises the full shell-facing surface, not just
// engine events.
func TestEyeColorAppearsInPacketsOnlyAtPond(t *testing.T) {
	c, nar := loadZone1(t)

	// Round-by-round walk: self-inspect, to forest_path, to still_pond, inspect
	// reflection, back, to gate. The pond inspect is the fourth round (index 3).
	walk := []engine.RoundSubmission{
		{V: 1, Round: 1, Actions: []engine.Action{{Resource: "attention", Verb: engine.VerbInspect, Target: "self"}}},
		{V: 1, Round: 2, Actions: []engine.Action{{Resource: "legs", Verb: engine.VerbPerform, Target: "forest_path"}}},
		{V: 1, Round: 3, Actions: []engine.Action{{Resource: "legs", Verb: engine.VerbPerform, Target: "still_pond"}}},
		{V: 1, Round: 4, Actions: []engine.Action{{Resource: "attention", Verb: engine.VerbInspect, Target: "reflection"}}},
		{V: 1, Round: 5, Actions: []engine.Action{{Resource: "legs", Verb: engine.VerbPerform, Target: "forest_path"}}},
		{V: 1, Round: 6, Actions: []engine.Action{{Resource: "legs", Verb: engine.VerbPerform, Target: "gate"}}},
	}
	const pondInspect = 3 // zero-based index of the reflection inspect

	state := engine.Init(1, c)
	var pondColor string
	for i := 0; i < len(walk); i++ {
		next, events, err := engine.Reduce(state, walk[i])
		if err != nil {
			t.Fatalf("Reduce round %d: %v", i, err)
		}
		state = next
		packet := buildPacket(state, events, nar)

		// Structured observations carry the value only on the pond round.
		for j := 0; j < len(packet.Observations); j++ {
			if packet.Observations[j].Fact == engine.FactEyeColor {
				if i != pondInspect {
					t.Errorf("round %d: eye_color observation before the pond inspect", i)
				}
				pondColor = packet.Observations[j].Value
			}
		}

		if word, ok := containsPaletteWord(c, packet.Narration); ok {
			if i != pondInspect {
				t.Errorf("round %d: narration leaked palette word %q before the pond: %q", i, word, packet.Narration)
			}
		} else if i == pondInspect {
			t.Errorf("pond round: narration did not name the eye color: %q", packet.Narration)
		}
	}

	if pondColor == "" {
		t.Fatal("never observed an eye color at the pond")
	}
	if pondColor != "brown" { // seed 1 golden value
		t.Errorf("seed 1 eye color = %q, want brown", pondColor)
	}
}

// A freeform line that the parser understands advances the round exactly like a
// canonical envelope: the shell folds the parsed submission through the engine
// and the agent moves. Only canonical actions ever reach the engine (GDD §5.2).
func TestFreeformLineAdvancesRound(t *testing.T) {
	c, nar := loadZone1(t)
	p := parser.New()

	sub, ok := p.Parse("walk to the forest path")
	if !ok {
		t.Fatal(`parser rejected "walk to the forest path"; expected a canonical move`)
	}

	state := engine.Init(1, c)
	next, events, err := engine.Reduce(state, sub)
	if err != nil {
		t.Fatalf("Reduce: %v", err)
	}
	state = next

	packet := buildPacket(state, events, nar)
	if state.Location != "forest_path" {
		t.Errorf("after freeform walk, location = %q, want forest_path", state.Location)
	}
	if !packet.Result.OK {
		t.Errorf("freeform move packet not OK: %+v", packet.Result)
	}
	if packet.Round != 2 {
		t.Errorf("round after one accepted freeform line = %d, want 2", packet.Round)
	}
}

// A freeform line the parser cannot map is a free rejection: it never reaches
// the engine, so the round counter does not advance and the packet carries the
// not_understood rejection (GDD P3 — misparse never kills, never even costs a
// tick).
func TestFreeformRejectionCostsNothing(t *testing.T) {
	c, _ := loadZone1(t)
	p := parser.New()

	if _, ok := p.Parse("frobnicate the gate"); ok {
		t.Fatal(`parser accepted "frobnicate the gate"; expected a rejection`)
	}

	// The shell short-circuits before Reduce, so state (and its round) is
	// untouched. The rejection packet still reports the pending round.
	state := engine.Init(1, c)
	packet := parseRejectionPacket(state)

	if packet.Result.OK {
		t.Error("parse-rejection packet reports OK; want not OK")
	}
	if len(packet.Result.Rejections) != 1 || packet.Result.Rejections[0].Reason != "not_understood" {
		t.Errorf("rejection = %+v, want a single not_understood", packet.Result.Rejections)
	}
	if packet.Round != int(state.Round)+1 {
		t.Errorf("rejection packet round = %d, want %d (unchanged)", packet.Round, int(state.Round)+1)
	}
	if !strings.Contains(packet.Narration, "understand") {
		t.Errorf("rejection narration = %q, want an 'I don't understand' message", packet.Narration)
	}
}
