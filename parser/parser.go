package parser

import (
	"encoding/json"
	"strings"

	"github.com/RiccardoCereghino/xenomancer/engine"
)

// Parser maps freeform text to canonical round submissions by deterministic
// dictionary lookup. Construct it once and reuse it; Parse holds no state and is
// safe to call repeatedly with identical results.
type Parser struct {
	dict Dictionary
}

// New returns a Parser backed by the embedded default dictionary. It panics only
// if the checked-in dictionary.json is itself malformed — a build-time bug, not
// a runtime condition — so callers need not handle an error for the default.
func New() *Parser {
	d, err := LoadDictionary(embeddedDictionary)
	if err != nil {
		panic("parser: embedded dictionary is invalid: " + err.Error())
	}
	return &Parser{dict: d}
}

// NewFromBytes returns a Parser backed by a caller-supplied dictionary (for
// tests or content overrides). The bytes are validated the same way as the
// embedded default.
func NewFromBytes(data []byte) (*Parser, error) {
	d, err := LoadDictionary(data)
	if err != nil {
		return nil, err
	}
	return &Parser{dict: d}, nil
}

// Parse maps one freeform line to a canonical single-action RoundSubmission.
//
// It returns ok == true with a canonical submission only when the line yields an
// exact verb dictionary hit (plus an exact target hit for verbs that need one).
// Any other input — no known verb, or a non-wait verb with no known target —
// returns the zero submission and ok == false: the free parse rejection ("I
// don't understand"). Nothing is emitted toward the engine in that case, so a
// misparse costs no tick and can never kill (GDD P3).
//
// Because every field of the returned action is drawn solely from an exact
// dictionary lookup, no freeform string can ever produce a state-affecting
// action that was not an exact dictionary hit — the load-bearing invariant.
func (p *Parser) Parse(line string) (engine.RoundSubmission, bool) {
	tokens := normalize(line)
	if len(tokens) == 0 {
		return engine.RoundSubmission{}, false
	}

	verb, vStart, vLen, ok := p.matchVerb(tokens)
	if !ok {
		return engine.RoundSubmission{}, false
	}

	action := engine.Action{
		Resource: verb.Resource,
		Verb:     verb.Verb,
	}

	// wait is the only verb that resolves without a target this sprint. Every
	// other verb needs an exact target hit or the whole line is rejected.
	if verb.Verb != engine.VerbWait {
		target, ok := p.matchTarget(tokens, vStart, vLen)
		if !ok {
			return engine.RoundSubmission{}, false
		}
		action.Target = target
	}

	// A talk addresses an NPC (Target) and carries the reply text in Args
	// ({"say":"..."}); the guard matches it against the palette (GDD §5.4). The
	// whole normalized line is the reply — the engine extracts the single palette
	// word from it, so a claim needs no palette word in the dictionary. The action
	// still required an exact talk-verb hit AND an exact NPC-target hit, so
	// misparse-never-kills holds: no freeform line reaches the guard without both
	// dictionary hits (GDD P3).
	if verb.Verb == engine.VerbTalk {
		action.Args = sayArgs(tokens)
	}

	return engine.RoundSubmission{
		V:       engine.ProtocolVersion,
		Actions: []engine.Action{action},
	}, true
}

// sayArgs encodes the reply text a talk claim carries: {"say":"<line>"}. The
// engine's closed-palette matching scans the whole reply, so the full normalized
// line is passed and json.Marshal handles escaping.
func sayArgs(tokens []string) json.RawMessage {
	b, _ := json.Marshal(struct {
		Say string `json:"say"`
	}{Say: strings.Join(tokens, " ")})
	return b
}

// matchVerb finds the verb dictionary hit over the token windows, preferring the
// longest phrase and, among equal lengths, the leftmost — a fully deterministic
// order that never ranges the dictionary map. It returns the entry and the
// [start, start+length) span it consumed so the target scan can skip it.
func (p *Parser) matchVerb(tokens []string) (VerbEntry, int, int, bool) {
	for length := len(tokens); length >= 1; length-- {
		for start := 0; start+length <= len(tokens); start++ {
			phrase := strings.Join(tokens[start:start+length], " ")
			if e, ok := p.dict.Verbs[phrase]; ok {
				return e, start, length, true
			}
		}
	}
	return VerbEntry{}, 0, 0, false
}

// matchTarget finds the target dictionary hit over the token windows that do not
// overlap the verb span, longest-then-leftmost. Skipping the verb's own tokens
// keeps a verb word from doubling as its own target.
func (p *Parser) matchTarget(tokens []string, vStart, vLen int) (string, bool) {
	for length := len(tokens); length >= 1; length-- {
		for start := 0; start+length <= len(tokens); start++ {
			if overlaps(start, length, vStart, vLen) {
				continue
			}
			phrase := strings.Join(tokens[start:start+length], " ")
			if id, ok := p.dict.Targets[phrase]; ok {
				return id, true
			}
		}
	}
	return "", false
}

// overlaps reports whether window [aStart, aStart+aLen) intersects
// [bStart, bStart+bLen).
func overlaps(aStart, aLen, bStart, bLen int) bool {
	return aStart < bStart+bLen && bStart < aStart+aLen
}
