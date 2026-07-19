package wire

import (
	"os"
	"strings"
	"testing"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

// loadZone1 reads the real zone-1 content pack and narration (a test may do I/O;
// the engine may not).
func loadZone1(t *testing.T) (engine.Content, Narration) {
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
	n, err := LoadNarration(narBytes)
	if err != nil {
		t.Fatalf("LoadNarration: %v", err)
	}
	return c, n
}

func reduceRound(t *testing.T, s engine.State, in engine.RoundSubmission) (engine.State, []engine.Event) {
	t.Helper()
	ns, events, err := engine.Reduce(s, in)
	if err != nil {
		t.Fatalf("Reduce: %v", err)
	}
	return ns, events
}

func moveTo(target string) engine.RoundSubmission {
	return engine.RoundSubmission{V: engine.ProtocolVersion, Actions: []engine.Action{{Resource: "legs", Verb: engine.VerbPerform, Target: target}}}
}

func waitAttention() engine.RoundSubmission {
	return engine.RoundSubmission{V: engine.ProtocolVersion, Actions: []engine.Action{{Resource: "attention", Verb: engine.VerbWait}}}
}

// The shell weaves the wolf's telegraph and grapple prose into the round's
// narration, and the final grapple death is delivered as a terminal death packet
// carrying cause hazard.wolf (GDD §5.6, ADR-000 D4). This exercises the whole
// shell-facing hazard surface on the real content pack.
func TestWolfNarrationAndTerminalPacket(t *testing.T) {
	c, n := loadZone1(t)

	// Enter the forest path (fuse 1), then wait toward the grapple, checking the
	// packets the shell would emit each round.
	s := engine.Init(0, c)
	s, events := reduceRound(t, s, moveTo("forest_path"))

	sawStage1, sawGrappled := false, false
	// Round after entry: 11 waits climb the fuse 2..12; the grapple springs at 12.
	for i := 0; i < 11; i++ {
		s, events = reduceRound(t, s, waitAttention())
		pkt := BuildPacket(s, events, n)
		if strings.Contains(pkt.Narration, "almost decorative") {
			sawStage1 = true
		}
		if strings.Contains(pkt.Narration, "wolf breaks from the treeline") {
			sawGrappled = true
			// On the grapple round the packet also reports the sustained holds.
			if len(pkt.Holds) != 3 {
				t.Errorf("grapple packet holds = %d, want 3", len(pkt.Holds))
			}
		}
	}
	if !sawStage1 {
		t.Error("stage-1 telegraph prose never appeared in a packet")
	}
	if !sawGrappled {
		t.Error("grapple prose never appeared in a packet")
	}
	if s.GrappleRoundsLeft == 0 {
		t.Fatalf("expected grappled after climbing the fuse")
	}

	// Fail the escape: two waits, and the second is a terminal wolf death.
	s, events = reduceRound(t, s, waitAttention())
	if _, ok := TerminalPacket(s, events, n); ok {
		t.Fatal("episode ended after only one grapple round")
	}
	s, events = reduceRound(t, s, waitAttention())

	terminal, ok := TerminalPacket(s, events, n)
	if !ok {
		t.Fatal("expected a terminal packet on the wolf death")
	}
	death, ok := terminal.(DeathPacket)
	if !ok {
		t.Fatalf("terminal packet is %T, want DeathPacket", terminal)
	}
	if death.Outcome != engine.OutcomeDied {
		t.Errorf("outcome = %q, want died", death.Outcome)
	}
	if death.Cause != engine.CauseHazardWolf {
		t.Errorf("cause = %q, want %q", death.Cause, engine.CauseHazardWolf)
	}
	if len(death.TelegraphsIgnored) != 3 {
		t.Errorf("telegraphs_ignored = %v, want 3 stages", death.TelegraphsIgnored)
	}
}
