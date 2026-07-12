package engine

import (
	"bytes"
	"testing"
)

func TestCanonicalBytesFieldOrderIsFrozen(t *testing.T) {
	// A hand-built State with a known layout must produce exactly these bytes.
	// If this test breaks, CanonicalBytes changed — a breaking engine version
	// that orphans existing replays (ADR-000 D5.6). Do not "fix" it lightly.
	s := State{
		Seed:     1,
		Tick:     2,
		Round:    3,
		Location: "clearing",
		Holds: []Hold{
			{Resource: "hand_right", Tag: "mana_hold", Since: 12},
		},
	}
	want := []byte{
		0, 0, 0, 0, 0, 0, 0, 1, // Seed
		0, 0, 0, 0, 0, 0, 0, 2, // Tick
		0, 0, 0, 0, 0, 0, 0, 3, // Round
		0, 0, 0, 8, // len("clearing")
		'c', 'l', 'e', 'a', 'r', 'i', 'n', 'g',
		0, 0, 0, 1, // Holds count
		0, 0, 0, 10, // len("hand_right")
		'h', 'a', 'n', 'd', '_', 'r', 'i', 'g', 'h', 't',
		0, 0, 0, 9, // len("mana_hold")
		'm', 'a', 'n', 'a', '_', 'h', 'o', 'l', 'd',
		0, 0, 0, 0, 0, 0, 0, 12, // Since
		0, 0, 0, 0, // len(Outcome) = 0 (ongoing) — appended in encoding v2
		0, 0, 0, 0, // len(Cause) = 0 — appended in encoding v2
	}
	got := s.CanonicalBytes()
	if !bytes.Equal(got, want) {
		t.Errorf("CanonicalBytes layout drifted.\n got: %v\nwant: %v", got, want)
	}
}

// A won state and a died state that are otherwise identical (same seed, tick,
// round, location) must hash differently — the whole reason the terminal outcome
// is part of CanonicalBytes (ADR-000 D5.6). Without it, replay-as-proof could not
// tell a win from a death at the gate.
func TestStateHashDistinguishesOutcomes(t *testing.T) {
	base := State{Seed: 1, Tick: 6, Round: 6, Location: "gate"}
	won := base
	won.Outcome = OutcomeWon
	died := base
	died.Outcome = OutcomeDied
	died.Cause = CauseClaimWrong

	if base.StateHash() == won.StateHash() {
		t.Error("won outcome must change the state hash")
	}
	if won.StateHash() == died.StateHash() {
		t.Error("won and died must hash differently")
	}
}

func TestStateHashDistinguishesStates(t *testing.T) {
	a := State{Seed: 1, Location: "clearing"}
	b := State{Seed: 1, Location: "forest_path"}
	if a.StateHash() == b.StateHash() {
		t.Error("different locations must hash differently")
	}
	if a.StateHash() != a.StateHash() {
		t.Error("StateHash must be stable for a fixed state")
	}
	// Content is excluded from the hash: two states differing only in Content
	// must hash identically (ADR-000 D5.5 / D6).
	withContent := a
	withContent.Content = Content{StartLocation: "clearing"}
	if a.StateHash() != withContent.StateHash() {
		t.Error("Content must not affect the state hash")
	}
}
