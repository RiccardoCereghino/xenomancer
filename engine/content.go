package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/RiccardoCereghino/xenomancer/engine/internal/rng"
)

// Location is a node in the zone's location graph. Exits is an ordered slice
// of destination location IDs reachable by a perform on legs. Inspectables is
// an ordered slice of things an inspect can perceive here (GDD §5.3). NPCs is an
// ordered slice of characters a talk can address here (GDD §5.4).
type Location struct {
	ID           string        `json:"id"`
	Exits        []string      `json:"exits"`
	Inspectables []Inspectable `json:"inspectables,omitempty"`
	NPCs         []NPC         `json:"npcs,omitempty"`
}

// NPC is a character an agent can talk to at a location (GDD §5.4). Asks names
// the fact the NPC interrogates: the gate guard asks "eye_color" and judges the
// claim against the per-seed truth (see Content.eyeColor). The set is an ordered
// slice so lookup never iterates a map.
type NPC struct {
	ID   string `json:"id"`
	Asks string `json:"asks"`
}

// Inspectable is a perceivable object at a location. Reveals names the fact an
// inspect on it observes: "eye_color" yields the per-seed palette word at the
// still pond; "self" is a decorative self-inspection (species, hair — never
// eyes; GDD §7). The set is an ordered slice so lookup never iterates a map.
type Inspectable struct {
	ID      string `json:"id"`
	Reveals string `json:"reveals"`
}

// FactEyeColor is the fact key an inspect of the pond reflection reveals
// (GDD §5.3). It is the only fact whose value is drawn per-seed this sprint.
const FactEyeColor = "eye_color"

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

	// EyeColorPalette is the closed, ordered palette the per-seed eye-color
	// fact is drawn from (GDD §3, §5.3). Order is FROZEN: selection is an
	// integer index into this slice, so reordering it changes every seed's
	// answer and orphans knowledge runs. Append-only if it ever grows.
	EyeColorPalette []string `json:"eye_color_palette,omitempty"`

	// Epitaphs is the ordered slice of death-report epitaph templates (GDD §5.7).
	// Selection is a per-seed integer index into this slice (see Content.epitaph),
	// so — like the palette — order is FROZEN: reordering moves every seed's
	// epitaph. Templates carry {claimed} and {truth} placeholders. Epitaphs live
	// in engine content because the reducer emits them inside the structured
	// died{report} (ADR-000 D2/D4).
	Epitaphs []string `json:"epitaphs,omitempty"`

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
	// Every exit must reference a defined location; every inspectable must name
	// a fact, and an eye-color inspectable requires a non-empty palette to
	// index (a divide-by-zero otherwise, and an unanswerable fact).
	hasNPC := false
	for i := 0; i < len(c.Locations); i++ {
		loc := c.Locations[i]
		for j := 0; j < len(loc.Exits); j++ {
			if _, ok := c.location(loc.Exits[j]); !ok {
				return Content{}, fmt.Errorf("engine: location %q has exit to undefined location %q", loc.ID, loc.Exits[j])
			}
		}
		for j := 0; j < len(loc.Inspectables); j++ {
			insp := loc.Inspectables[j]
			if insp.ID == "" {
				return Content{}, fmt.Errorf("engine: location %q has an inspectable with no id", loc.ID)
			}
			if insp.Reveals == "" {
				return Content{}, fmt.Errorf("engine: inspectable %q at %q reveals no fact", insp.ID, loc.ID)
			}
			if insp.Reveals == FactEyeColor && len(c.EyeColorPalette) == 0 {
				return Content{}, fmt.Errorf("engine: inspectable %q at %q reveals eye_color but the pack has an empty eye_color_palette", insp.ID, loc.ID)
			}
		}
		for j := 0; j < len(loc.NPCs); j++ {
			npc := loc.NPCs[j]
			if npc.ID == "" {
				return Content{}, fmt.Errorf("engine: location %q has an npc with no id", loc.ID)
			}
			if npc.Asks == "" {
				return Content{}, fmt.Errorf("engine: npc %q at %q asks for no fact", npc.ID, loc.ID)
			}
			if npc.Asks == FactEyeColor && len(c.EyeColorPalette) == 0 {
				return Content{}, fmt.Errorf("engine: npc %q at %q asks eye_color but the pack has an empty eye_color_palette", npc.ID, loc.ID)
			}
			hasNPC = true
		}
	}
	// An NPC can kill (a wrong claim → died{report}), and the report carries a
	// seeded epitaph; a pack with an interrogator but no epitaphs to index is an
	// unfinishable death (a divide-by-zero, and empty prose).
	if hasNPC && len(c.Epitaphs) == 0 {
		return Content{}, fmt.Errorf("engine: content defines an npc but has an empty epitaphs list")
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

// inspectable returns the inspectable with id target at location from, by
// ordered linear scan (no maps, ADR-000 D5.2).
func (c Content) inspectable(from, target string) (Inspectable, bool) {
	loc, ok := c.location(from)
	if !ok {
		return Inspectable{}, false
	}
	for i := 0; i < len(loc.Inspectables); i++ {
		if loc.Inspectables[i].ID == target {
			return loc.Inspectables[i], true
		}
	}
	return Inspectable{}, false
}

// npc returns the NPC with id target at location from, by ordered linear scan
// (no maps, ADR-000 D5.2).
func (c Content) npc(from, target string) (NPC, bool) {
	loc, ok := c.location(from)
	if !ok {
		return NPC{}, false
	}
	for i := 0; i < len(loc.NPCs); i++ {
		if loc.NPCs[i].ID == target {
			return loc.NPCs[i], true
		}
	}
	return NPC{}, false
}

// eyeColor is the per-seed eye-color fact (GDD §3, §5.3). It derives EXCLUSIVELY
// from the vendored rng via a documented sub-seed and integer arithmetic over
// the frozen-order palette — no other randomness source, no reordering
// (ADR-000 D5.3): one SplitMix64 draw seeded by Subseed(seed, "facts.eye_color"),
// taken mod len(palette) as the index. It is a pure function of (seed, palette):
// same seed always yields the same word, forever.
//
// Canonical world: seed 0 yields "grey" (GDD §3 — the memorizable reference for
// knowledge runs and goldens; see content/zone1/README.md).
func (c Content) eyeColor(seed uint64) string {
	r := rng.New(rng.Subseed(seed, "facts."+FactEyeColor))
	idx := r.Next() % uint64(len(c.EyeColorPalette))
	return c.EyeColorPalette[idx]
}

// epitaph selects a per-seed epitaph template from the frozen-order Epitaphs
// slice and fills its {claimed}/{truth} placeholders. Selection mirrors eyeColor
// exactly — one SplitMix64 draw seeded by Subseed(seed, "narration.epitaph"),
// taken mod len(epitaphs) as the index (ADR-000 D5.3) — so the same seed always
// yields the same epitaph, forever. The rendered epitaph is emitted inside the
// structured died{report}; epitaphs are allowed to be the best writing in the
// game (GDD §5.7).
//
// Canonical world: seed 0's death yields the reference line
// "He was sure his eyes were {claimed}. The pond, unbothered, remains {truth}."
func (c Content) epitaph(seed uint64, claimed, truth string) string {
	r := rng.New(rng.Subseed(seed, "narration.epitaph"))
	idx := r.Next() % uint64(len(c.Epitaphs))
	t := c.Epitaphs[idx]
	t = strings.ReplaceAll(t, "{claimed}", claimed)
	t = strings.ReplaceAll(t, "{truth}", truth)
	return t
}

// deathReport assembles the full GDD §5.7 death report for a wrong social claim.
// telegraphs_ignored is empty (no hazards this sprint) and ritual_progress is
// nil (no procedural deaths), so they serialize to [] and null respectively.
func (c Content) deathReport(seed uint64, npc, asked, claimed, truth string, round uint64) DeathReport {
	return DeathReport{
		Cause:             CauseClaimWrong,
		Detail:            DeathDetail{NPC: npc, Asked: asked, Claimed: claimed, Truth: truth},
		Round:             round,
		TelegraphsIgnored: []string{},
		RitualProgress:    nil,
		Epitaph:           c.epitaph(seed, claimed, truth),
	}
}

// matchClaim scans reply for palette words by ordered string containment
// (no maps, ADR-000 D5.2) and returns the matched word and the number of
// distinct palette words found. Exactly one is a claim; zero or several is
// "speak plainly" (GDD §5.4). Case-insensitive so a capitalized reply still
// matches the lower-case palette.
func matchClaim(palette []string, reply string) (string, int) {
	lower := strings.ToLower(reply)
	claim := ""
	n := 0
	for i := 0; i < len(palette); i++ {
		if strings.Contains(lower, palette[i]) {
			claim = palette[i]
			n++
		}
	}
	return claim, n
}
