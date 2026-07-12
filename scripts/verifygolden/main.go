// Command verifygolden replays the committed golden logs against the zone-1
// content, verifies each reproduces its recorded final_state_hash, and prints
// those hashes to stdout in order (ADR-000 D6). The determinism CI job runs it
// twice, in separate processes, and asserts the two printed outputs are
// identical — the machine-checkable form of "replay-as-proof". Both slice
// endings are pinned: the win golden (seed 1) and the death golden (seed 0).
package main

import (
	"fmt"
	"os"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

// goldens are the committed replay fixtures, verified in this fixed order.
var goldens = []string{
	"agent/scripted/testdata/golden_replay.json",       // win path, seed 1
	"agent/scripted/testdata/golden_death_replay.json", // death path, seed 0
}

func main() {
	mapBytes, err := os.ReadFile("content/zone1/map.json")
	if err != nil {
		fatal("read map.json: %v", err)
	}
	content, err := engine.ParseContent(mapBytes)
	if err != nil {
		fatal("parse content: %v", err)
	}

	for _, path := range goldens {
		replayBytes, err := os.ReadFile(path)
		if err != nil {
			fatal("read %s: %v", path, err)
		}
		replay, err := engine.Decode(replayBytes)
		if err != nil {
			fatal("decode %s: %v", path, err)
		}
		ok, err := engine.Verify(replay, content)
		if err != nil {
			fatal("verify %s: %v", path, err)
		}
		if !ok {
			fatal("%s failed verification: final_state_hash mismatch", path)
		}
		// Print the reproduced hash for the CI job to compare across runs.
		fmt.Println(replay.FinalStateHash)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "verifygolden: "+format+"\n", args...)
	os.Exit(1)
}
