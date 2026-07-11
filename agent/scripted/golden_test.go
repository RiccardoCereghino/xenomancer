package scripted_test

import (
	"os"
	"testing"

	"github.com/RiccardoCereghino/xenomancer/agent/scripted"
	"github.com/RiccardoCereghino/xenomancer/engine"
)

const goldenSeed = 1

// loadContent reads the real zone-1 content pack from disk (a test may do I/O;
// the engine may not).
func loadContent(t *testing.T) engine.Content {
	t.Helper()
	data, err := os.ReadFile("../../content/zone1/map.json")
	if err != nil {
		t.Fatalf("read map.json: %v", err)
	}
	c, err := engine.ParseContent(data)
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	return c
}

// TestGoldenReplay runs the scripted agent, produces a replay, verifies it,
// replays the log twice, and asserts identical final state hashes. It also
// pins the result against the committed golden replay file so any drift in the
// canonical encoding or reducer is caught (ADR-000 D5.6 / D6).
func TestGoldenReplay(t *testing.T) {
	c := loadContent(t)
	log := scripted.Script()

	// 1. Run the scripted agent into a replay.
	replay, final, err := engine.BuildReplay(goldenSeed, c, log, "scripted")
	if err != nil {
		t.Fatalf("BuildReplay: %v", err)
	}

	// The scripted agent must end on the forest path (clearing -> forest_path,
	// then two waits).
	if final.Location != "forest_path" {
		t.Errorf("final location = %q, want forest_path", final.Location)
	}
	if final.Round != 3 || final.Tick != 3 {
		t.Errorf("final round/tick = %d/%d, want 3/3", final.Round, final.Tick)
	}

	// 2. Verify the replay reproduces its own final hash.
	ok, err := engine.Verify(replay, c)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Fatal("freshly built replay failed verification")
	}

	// 3. Replay the log twice from scratch and assert identical hashes.
	h1 := foldHash(t, c, log)
	h2 := foldHash(t, c, log)
	if h1 != h2 {
		t.Errorf("state hash differs across two replays: %s vs %s", h1, h2)
	}
	if h1 != replay.FinalStateHash {
		t.Errorf("folded hash %s != replay final_state_hash %s", h1, replay.FinalStateHash)
	}

	// 4. Encode/decode round trip must still verify.
	encoded, err := engine.Encode(replay)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := engine.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	ok, err = engine.Verify(decoded, c)
	if err != nil {
		t.Fatalf("Verify(decoded): %v", err)
	}
	if !ok {
		t.Error("decoded replay failed verification")
	}

	// 5. Pin against the committed golden file.
	goldenBytes, err := os.ReadFile("testdata/golden_replay.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	golden, err := engine.Decode(goldenBytes)
	if err != nil {
		t.Fatalf("decode golden: %v", err)
	}
	if golden.FinalStateHash != replay.FinalStateHash {
		t.Errorf("golden final_state_hash %s != rebuilt %s (canonical encoding drift?)",
			golden.FinalStateHash, replay.FinalStateHash)
	}
	ok, err = engine.Verify(golden, c)
	if err != nil {
		t.Fatalf("Verify(golden): %v", err)
	}
	if !ok {
		t.Error("committed golden replay failed verification")
	}
}

// TestGoldenReplayRejectsWrongContent proves a replay is valid only against its
// exact content hash (ADR-000 D6).
func TestGoldenReplayRejectsWrongContent(t *testing.T) {
	goldenBytes, err := os.ReadFile("testdata/golden_replay.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	golden, err := engine.Decode(goldenBytes)
	if err != nil {
		t.Fatalf("decode golden: %v", err)
	}
	other, err := engine.ParseContent([]byte(`{
		"start_location": "clearing",
		"locations": [
			{"id": "clearing", "exits": ["forest_path"]},
			{"id": "forest_path", "exits": ["clearing", "clearing"]}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	ok, err := engine.Verify(golden, other)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ok {
		t.Error("replay verified against content with a different hash")
	}
}

func foldHash(t *testing.T, c engine.Content, log []engine.RoundSubmission) string {
	t.Helper()
	s := engine.Init(goldenSeed, c)
	for i := 0; i < len(log); i++ {
		ns, _, err := engine.Reduce(s, log[i])
		if err != nil {
			t.Fatalf("Reduce round %d: %v", i, err)
		}
		s = ns
	}
	return s.StateHash()
}
