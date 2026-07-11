// Package rng is the vendored pseudo-random generator for the engine
// (ADR-000 D5.3). The stdlib math generators are banned from /engine: their
// behavior has changed across Go versions and would silently break replays on
// a frozen-forever surface. This ~30-line splitmix64 lives in-repo so its
// output is fixed forever.
//
// All randomness derives from the run seed via documented sub-seeding:
//
//	subseed(domain) = seed XOR fnv64(domain)
//
// giving one independent stream per system (world-gen, NPC values,
// narration-variant selection). No stream is consumed this sprint; the
// generator is vendored now so later systems have a frozen source ready.
package rng

// SplitMix64 is a deterministic 64-bit generator. Its zero value is not
// usable; construct one with New.
type SplitMix64 struct {
	state uint64
}

// New returns a generator seeded with the given value.
func New(seed uint64) *SplitMix64 {
	return &SplitMix64{state: seed}
}

// Next returns the next 64-bit value and advances the state. The constants are
// the reference splitmix64 constants and must never change.
func (r *SplitMix64) Next() uint64 {
	r.state += 0x9E3779B97F4A7C15
	z := r.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// Subseed derives a per-domain seed from the run seed (ADR-000 D5.3):
// subseed(domain) = seed XOR fnv64(domain).
func Subseed(seed uint64, domain string) uint64 {
	return seed ^ fnv64(domain)
}

// fnv64 is the FNV-1a 64-bit hash over the bytes of s, implemented in-repo so
// the sub-seeding derivation has no external or version-sensitive dependency.
func fnv64(s string) uint64 {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211
	h := uint64(offset64)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	return h
}
