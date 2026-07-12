package parser

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

// embeddedDictionary is the default synonym table, shipped as data with the
// binary so runtime lookup is zero-I/O and always consistent (GDD §5.2). It is
// AI-authored offline and checked into the repo; at runtime it is a pure table.
//
//go:embed dictionary.json
var embeddedDictionary []byte

// VerbEntry is a canonical verb plus the body resource it claims. The resource
// is carried as data (not derived by code) so the parser holds no game rules:
// movement is a perform on legs, inspection an inspect on attention, and so on
// (see agent/scripted for the canonical conventions).
type VerbEntry struct {
	Verb     string `json:"verb"`
	Resource string `json:"resource"`
}

// Dictionary is the versioned synonym table (ADR-000 Follow-ups / future
// ADR-003). Verbs maps a normalized freeform verb phrase to its canonical verb
// and resource; Targets maps a normalized freeform target phrase to a canonical
// target id. Both are consulted by exact key lookup only — the parser never
// ranges these maps, so map iteration order can never affect a result.
type Dictionary struct {
	Version int                  `json:"version"`
	Verbs   map[string]VerbEntry `json:"verbs"`
	Targets map[string]string    `json:"targets"`
}

// canonicalVerbs is the closed verb set every dictionary entry must resolve to
// (GDD §5.2, engine/types.go). Kept local so the parser validates against the
// engine's constants without the engine ever depending on the parser.
var canonicalVerbs = map[string]bool{
	engine.VerbInspect: true,
	engine.VerbPerform: true,
	engine.VerbTalk:    true,
	engine.VerbWait:    true,
}

// canonicalResources mirrors the engine's closed resource set (engine/types.go).
// A dictionary that names a resource the engine does not know would only ever
// produce unknown_resource rejections, so we reject such a dictionary at load.
var canonicalResources = map[string]bool{
	"voice":      true,
	"hand_left":  true,
	"hand_right": true,
	"legs":       true,
	"attention":  true,
}

// LoadDictionary parses and validates a dictionary from its JSON bytes. It fails
// loudly on a malformed table (unknown canonical verb or resource, empty target
// id) rather than silently shipping entries that could never resolve.
func LoadDictionary(data []byte) (Dictionary, error) {
	var d Dictionary
	if err := json.Unmarshal(data, &d); err != nil {
		return Dictionary{}, fmt.Errorf("parser: dictionary parse: %w", err)
	}
	for phrase, e := range d.Verbs {
		if !canonicalVerbs[e.Verb] {
			return Dictionary{}, fmt.Errorf("parser: verb %q maps to non-canonical verb %q", phrase, e.Verb)
		}
		if !canonicalResources[e.Resource] {
			return Dictionary{}, fmt.Errorf("parser: verb %q maps to unknown resource %q", phrase, e.Resource)
		}
	}
	for phrase, id := range d.Targets {
		if id == "" {
			return Dictionary{}, fmt.Errorf("parser: target %q maps to an empty id", phrase)
		}
	}
	return d, nil
}
