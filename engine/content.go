package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Location is a node in the zone's location graph. Exits is an ordered slice
// of destination location IDs reachable by a perform on legs.
type Location struct {
	ID    string   `json:"id"`
	Exits []string `json:"exits"`
}

// Content is inert, hash-addressed world data (ADR-000 D5.5). This sprint it
// carries only the location graph loaded from content/zone1/map.json. It is
// parsed into typed structs and ordered slices — never maps — so no rule ever
// depends on iteration order.
//
// The hash is the SHA-256 of the exact plaintext the content was parsed from;
// it identifies the pack independently of the engine binary (ADR-000 D5.5).
type Content struct {
	StartLocation string     `json:"start_location"`
	Locations     []Location `json:"locations"`

	hash [32]byte
}

// ParseContent validates and loads a content pack from its plaintext bytes.
// The engine performs no filesystem access itself (ADR-000 D5.4); the caller
// (a shell or a test) reads the file and passes the bytes here.
func ParseContent(data []byte) (Content, error) {
	var c Content
	if err := json.Unmarshal(data, &c); err != nil {
		return Content{}, fmt.Errorf("engine: content parse: %w", err)
	}
	if c.StartLocation == "" {
		return Content{}, fmt.Errorf("engine: content has no start_location")
	}
	if _, ok := c.location(c.StartLocation); !ok {
		return Content{}, fmt.Errorf("engine: start_location %q is not a defined location", c.StartLocation)
	}
	// Every exit must reference a defined location.
	for i := 0; i < len(c.Locations); i++ {
		loc := c.Locations[i]
		for j := 0; j < len(loc.Exits); j++ {
			if _, ok := c.location(loc.Exits[j]); !ok {
				return Content{}, fmt.Errorf("engine: location %q has exit to undefined location %q", loc.ID, loc.Exits[j])
			}
		}
	}
	c.hash = sha256.Sum256(data)
	return c, nil
}

// HashString is the content identity used in replay headers: "sha256:<hex>".
func (c Content) HashString() string {
	return "sha256:" + hex.EncodeToString(c.hash[:])
}

// location returns the location with the given ID by ordered linear scan.
func (c Content) location(id string) (Location, bool) {
	for i := 0; i < len(c.Locations); i++ {
		if c.Locations[i].ID == id {
			return c.Locations[i], true
		}
	}
	return Location{}, false
}

// isExit reports whether target is a direct exit from the location from.
func (c Content) isExit(from, target string) bool {
	loc, ok := c.location(from)
	if !ok {
		return false
	}
	for i := 0; i < len(loc.Exits); i++ {
		if loc.Exits[i] == target {
			return true
		}
	}
	return false
}
