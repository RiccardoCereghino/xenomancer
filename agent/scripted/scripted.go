// Package scripted is a deterministic agent for the scripted bracket
// (GDD §3). It emits a fixed sequence of canonical round envelopes: walk from
// the clearing to the forest path, wait twice, then exit. Being fully scripted
// it does not depend on observations, so it drives the stdio shell over a
// one-way pipe.
package scripted

import "github.com/RiccardoCereghino/xenomancer/engine"

// Script returns the canonical action log this agent submits, in order. The
// same slice every call: this is the save file and the proof object
// (ADR-000 D3).
func Script() []engine.RoundSubmission {
	return []engine.RoundSubmission{
		// Round 1: walk clearing -> forest_path (movement is perform on legs).
		{
			V:     engine.ProtocolVersion,
			Round: 1,
			Actions: []engine.Action{
				{Resource: "legs", Verb: engine.VerbPerform, Target: "forest_path"},
			},
		},
		// Round 2: wait.
		{
			V:     engine.ProtocolVersion,
			Round: 2,
			Actions: []engine.Action{
				{Resource: "attention", Verb: engine.VerbWait},
			},
		},
		// Round 3: wait.
		{
			V:     engine.ProtocolVersion,
			Round: 3,
			Actions: []engine.Action{
				{Resource: "attention", Verb: engine.VerbWait},
			},
		},
	}
}
