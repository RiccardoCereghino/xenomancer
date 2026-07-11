package engine

import (
	"encoding/json"
	"fmt"
)

// ReplayHeader identifies exactly what a replay was produced against
// (ADR-000 D6). A replay is valid only against its precise
// {engine_version, content_hash}; old engine versions therefore remain tagged
// and buildable forever.
type ReplayHeader struct {
	EngineVersion   string          `json:"engine_version"`
	ProtocolVersion int             `json:"protocol_version"`
	ContentHash     string          `json:"content_hash"`
	Seed            uint64          `json:"seed"`
	Bracket         string          `json:"bracket"`
	Meta            json.RawMessage `json:"meta,omitempty"`
}

// Replay is the replay file format v1 (ADR-000 D6):
//
//	{"header":{...}, "log":[ -canonical round envelopes- ], "final_state_hash":"sha256:-"}
//
// The log stores actions only; events are re-derived on replay (ADR-000 D3).
type Replay struct {
	Header         ReplayHeader      `json:"header"`
	Log            []RoundSubmission `json:"log"`
	FinalStateHash string            `json:"final_state_hash"`
}

// BuildReplay folds a canonical action log from Init and returns the resulting
// replay plus the final State. It is the shared driver used by shells, agents,
// and tests to turn a run into a proof object.
func BuildReplay(seed uint64, content Content, log []RoundSubmission, bracket string) (Replay, State, error) {
	s := Init(seed, content)
	for i := 0; i < len(log); i++ {
		ns, _, err := Reduce(s, log[i])
		if err != nil {
			return Replay{}, State{}, fmt.Errorf("engine: replay build at round %d: %w", i, err)
		}
		s = ns
	}
	r := Replay{
		Header: ReplayHeader{
			EngineVersion:   Version,
			ProtocolVersion: ProtocolVersion,
			ContentHash:     content.HashString(),
			Seed:            seed,
			Bracket:         bracket,
		},
		Log:            log,
		FinalStateHash: s.StateHash(),
	}
	return r, s, nil
}

// Encode serializes a replay to JSON.
func Encode(r Replay) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// Decode parses a replay from JSON.
func Decode(data []byte) (Replay, error) {
	var r Replay
	if err := json.Unmarshal(data, &r); err != nil {
		return Replay{}, fmt.Errorf("engine: replay decode: %w", err)
	}
	return r, nil
}

// Verify re-executes a replay against a content pack and reports whether it
// reproduces the recorded final_state_hash (ADR-000 D6). Verification fails
// (ok == false) if the content does not match the replay's content_hash or if
// the folded state hash differs. A non-nil error indicates programmer misuse
// surfaced during the fold, not an invalid-but-well-formed replay.
func Verify(r Replay, content Content) (bool, error) {
	if r.Header.ContentHash != content.HashString() {
		return false, nil
	}
	s := Init(r.Header.Seed, content)
	for i := 0; i < len(r.Log); i++ {
		ns, _, err := Reduce(s, r.Log[i])
		if err != nil {
			return false, fmt.Errorf("engine: replay verify at round %d: %w", i, err)
		}
		s = ns
	}
	return s.StateHash() == r.FinalStateHash, nil
}
