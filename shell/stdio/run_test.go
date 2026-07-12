package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/RiccardoCereghino/xenomancer/engine"
	"github.com/RiccardoCereghino/xenomancer/parser"
)

// loadZone1Dict reads the real zone-1 parser dictionary from disk.
func loadZone1Dict(t *testing.T) parser.Dictionary {
	t.Helper()
	data, err := os.ReadFile("../../content/zone1/dictionary.json")
	if err != nil {
		t.Fatalf("read dictionary.json: %v", err)
	}
	d, err := parser.LoadDictionary(data)
	if err != nil {
		t.Fatalf("LoadDictionary: %v", err)
	}
	return d
}

// decodePackets splits the shell's JSONL output into packets.
func decodePackets(t *testing.T, out string) []ObservationPacket {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(out))
	var packets []ObservationPacket
	for dec.More() {
		var p ObservationPacket
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("decode packet: %v", err)
		}
		packets = append(packets, p)
	}
	return packets
}

// The shell accepts canonical JSON envelopes AND freeform lines through the same
// loop, feeds only canonical actions to the engine, and a freeform miss is a free
// rejection that does not advance the round (DoD: accepts both; misparse never
// kills, P3).
func TestRunAcceptsCanonicalAndFreeform(t *testing.T) {
	content, nar := loadZone1(t)
	dict := loadZone1Dict(t)
	state := engine.Init(1, content)

	// canonical self-inspect, then three freeform lines: a move that parses, an
	// unrecognized line (free rejection), and another move that parses.
	input := strings.Join([]string{
		`{"v":1,"round":1,"actions":[{"resource":"attention","verb":"inspect","target":"self"}]}`,
		"walk to the forest",
		"frobnicate the gate",
		"walk to the gate",
	}, "\n") + "\n"

	var out bytes.Buffer
	if err := run(strings.NewReader(input), &out, state, nar, dict); err != nil {
		t.Fatalf("run: %v", err)
	}

	packets := decodePackets(t, out.String())
	if len(packets) != 4 {
		t.Fatalf("got %d packets, want 4:\n%s", len(packets), out.String())
	}

	// Packet 0: canonical inspect resolved cleanly, round advanced to 1 (the
	// packet reports the next round to submit, 2).
	if !packets[0].Result.OK || packets[0].Round != 2 {
		t.Errorf("canonical inspect packet = %+v, want ok round 2", packets[0])
	}

	// Packet 1: freeform "walk to the forest" parsed to a canonical move; round
	// advanced to 2 (next round 3).
	if !packets[1].Result.OK || packets[1].Round != 3 {
		t.Errorf("freeform move packet = %+v, want ok round 3", packets[1])
	}

	// Packet 2: unrecognized freeform is a free rejection — the narration is the
	// parser's message, no engine rejection is reported, and the round does NOT
	// advance (same next-round as the previous packet).
	rej := packets[2]
	if rej.Result.OK {
		t.Errorf("rejection packet reported ok, want not ok: %+v", rej)
	}
	if rej.Narration != parser.RejectMessage {
		t.Errorf("rejection narration = %q, want %q", rej.Narration, parser.RejectMessage)
	}
	if len(rej.Result.Rejections) != 0 {
		t.Errorf("free rejection carried engine rejections: %+v", rej.Result.Rejections)
	}
	if rej.Round != packets[1].Round {
		t.Errorf("rejection advanced the round: got %d, want unchanged %d", rej.Round, packets[1].Round)
	}

	// Packet 3: the next freeform move parses and advances again — the rejected
	// round was truly handed back, not consumed.
	if !packets[3].Result.OK || packets[3].Round != 4 {
		t.Errorf("freeform move-to-gate packet = %+v, want ok round 4", packets[3])
	}
}
