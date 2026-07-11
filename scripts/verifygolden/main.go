// Command verifygolden replays the committed golden log against the zone-1
// content, verifies it reproduces the recorded final_state_hash, and prints
// that hash to stdout (ADR-000 D6). The determinism CI job runs it twice, in
// separate processes, and asserts the two printed hashes are identical — the
// machine-checkable form of "replay-as-proof".
package main

import (
	"fmt"
	"os"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

func main() {
	mapBytes, err := os.ReadFile("content/zone1/map.json")
	if err != nil {
		fatal("read map.json: %v", err)
	}
	content, err := engine.ParseContent(mapBytes)
	if err != nil {
		fatal("parse content: %v", err)
	}

	replayBytes, err := os.ReadFile("agent/scripted/testdata/golden_replay.json")
	if err != nil {
		fatal("read golden replay: %v", err)
	}
	replay, err := engine.Decode(replayBytes)
	if err != nil {
		fatal("decode golden replay: %v", err)
	}

	ok, err := engine.Verify(replay, content)
	if err != nil {
		fatal("verify: %v", err)
	}
	if !ok {
		fatal("golden replay failed verification: final_state_hash mismatch")
	}

	// Print the reproduced hash for the CI job to compare across runs.
	fmt.Println(replay.FinalStateHash)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "verifygolden: "+format+"\n", args...)
	os.Exit(1)
}
