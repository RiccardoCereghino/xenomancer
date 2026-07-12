// Package parser is the quarantined freeform text parser (GDD §5.2, ADR-000
// D3/D4). It maps freeform agent text to canonical engine round submissions via
// a versioned synonym dictionary, by deterministic lookup only — no fuzzy
// matching, no model, no randomness. The same input always maps to the same
// canonical action (or the same rejection).
//
// It lives OUTSIDE /engine on purpose: the parser is not in the replay path
// (ADR-000 D3). Only canonical actions cross into the engine and the log, so
// parser evolution can never invalidate a replay. The engine never imports this
// package; this package imports engine only for the canonical Action /
// RoundSubmission types and the closed verb set.
//
// The defining guarantee is that a misparse can never kill (GDD P3): input with
// no dictionary hit is a free rejection ("I don't understand") that costs
// nothing — no tick, no state change, it never reaches the engine — and no
// freeform string can ever produce a state-affecting action that was not an
// exact dictionary hit. That invariant holds by construction (every emitted
// action is built solely from exact dictionary lookups) and is pinned by the
// property test in parser_test.go.
package parser
