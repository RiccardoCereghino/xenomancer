package main

import (
	"encoding/json"
	"strings"
)

// narration holds the plain templates loaded from narration.json (GDD §9).
// The narrator is a pure consumer of engine events and location state
// (ADR-000 D2); it lives in the shell, never in the engine.
type narration struct {
	Locations  map[string]string `json:"locations"`
	OnEnter    string            `json:"on_enter"`
	Wait       string            `json:"wait"`
	Idle       string            `json:"idle"`
	Rejections map[string]string `json:"rejections"`
}

func loadNarration(data []byte) (narration, error) {
	var n narration
	if err := json.Unmarshal(data, &n); err != nil {
		return narration{}, err
	}
	return n, nil
}

// render composes deterministic narration for a resolved round. Templates are
// looked up by key (never iterated), so output order is fixed.
func (n narration) render(location string, moved, waited bool, rejections []Rejection) string {
	var parts []string

	if moved {
		line := strings.ReplaceAll(n.OnEnter, "{location}", location)
		line = strings.ReplaceAll(line, "{description}", n.Locations[location])
		parts = append(parts, line)
	} else if waited {
		parts = append(parts, n.Wait)
	}

	for i := 0; i < len(rejections); i++ {
		if msg, ok := n.Rejections[rejections[i].Reason]; ok {
			parts = append(parts, msg)
		}
	}

	if len(parts) == 0 {
		if desc, ok := n.Locations[location]; ok {
			parts = append(parts, desc)
		} else {
			parts = append(parts, n.Idle)
		}
	}

	return strings.Join(parts, " ")
}
