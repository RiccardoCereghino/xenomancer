// Package scripted is a deterministic agent for the scripted bracket
// (GDD §3). It emits a fixed sequence of canonical round envelopes. Being fully
// scripted it does not read observations, so it drives the stdio shell over a
// one-way pipe — its knowledge of the world (which color to claim) is baked in.
//
// Two scripts cover the two endings of the Sprint-1 slice:
//   - Script (seed 1): the WIN path — inspect the pond, walk to the gate, and
//     claim the correct color (seed 1's eye color is brown; content README).
//   - DeathScript (seed 0): the DEATH path — walk straight to the gate and claim
//     a wrong color, reproducing the canonical GDD §5.7 death (claimed green,
//     truth grey). "Walking straight to the gate and dying there is the game
//     teaching its thesis" (GDD §7).
package scripted

import (
	"encoding/json"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

// say builds the Args for a talk claim: {"say":"<word>"}. The reply text rides
// in Args because freeform parsing is backlog 04; the guard matches it against
// the palette directly.
func say(word string) json.RawMessage {
	return json.RawMessage(`{"say":"` + word + `"}`)
}

// Script returns the canonical action log for the WIN path (seed 1), in order.
// The same slice every call: this is the save file and the proof object
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
		// Round 2: take the branch forest_path -> still_pond.
		{
			V:     engine.ProtocolVersion,
			Round: 2,
			Actions: []engine.Action{
				{Resource: "legs", Verb: engine.VerbPerform, Target: "still_pond"},
			},
		},
		// Round 3: inspect the reflection — observe the per-seed eye color.
		{
			V:     engine.ProtocolVersion,
			Round: 3,
			Actions: []engine.Action{
				{Resource: "attention", Verb: engine.VerbInspect, Target: "reflection"},
			},
		},
		// Round 4: back off the branch still_pond -> forest_path.
		{
			V:     engine.ProtocolVersion,
			Round: 4,
			Actions: []engine.Action{
				{Resource: "legs", Verb: engine.VerbPerform, Target: "forest_path"},
			},
		},
		// Round 5: walk forest_path -> gate.
		{
			V:     engine.ProtocolVersion,
			Round: 5,
			Actions: []engine.Action{
				{Resource: "legs", Verb: engine.VerbPerform, Target: "gate"},
			},
		},
		// Round 6: answer the guard with the recalled color — the lethal check.
		// Seed 1's eye color is brown, so this claim wins.
		{
			V:     engine.ProtocolVersion,
			Round: 6,
			Actions: []engine.Action{
				{Resource: "voice", Verb: engine.VerbTalk, Target: "gate_guard", Args: say("brown")},
			},
		},
	}
}

// DeathScript returns the canonical action log for the DEATH path (seed 0). It
// skips the pond, walks straight to the gate, and confidently claims green —
// while seed 0's eye color is grey. A fair death: legal, understood, wrong.
func DeathScript() []engine.RoundSubmission {
	return []engine.RoundSubmission{
		// Round 1: walk clearing -> forest_path.
		{
			V:     engine.ProtocolVersion,
			Round: 1,
			Actions: []engine.Action{
				{Resource: "legs", Verb: engine.VerbPerform, Target: "forest_path"},
			},
		},
		// Round 2: walk forest_path -> gate (never visiting the pond).
		{
			V:     engine.ProtocolVersion,
			Round: 2,
			Actions: []engine.Action{
				{Resource: "legs", Verb: engine.VerbPerform, Target: "gate"},
			},
		},
		// Round 3: claim green to the guard — wrong (truth is grey) — and die.
		{
			V:     engine.ProtocolVersion,
			Round: 3,
			Actions: []engine.Action{
				{Resource: "voice", Verb: engine.VerbTalk, Target: "gate_guard", Args: say("green")},
			},
		},
	}
}
