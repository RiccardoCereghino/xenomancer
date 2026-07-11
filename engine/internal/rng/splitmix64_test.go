package rng

import "testing"

func TestSplitMix64FrozenSequence(t *testing.T) {
	// splitmix64 from seed 0 has a well-known reference sequence. These values
	// are frozen forever (ADR-000 D5.3): a mismatch means the generator drifted
	// and every seeded replay is invalidated.
	want := []uint64{
		0xE220A8397B1DCDAF,
		0x6E789E6AA1B965F4,
		0x06C45D188009454F,
		0xF88BB8A8724C81EC,
		0x1B39896A51A8749B,
	}
	r := New(0)
	for i, w := range want {
		if got := r.Next(); got != w {
			t.Errorf("Next[%d] = %#016x, want %#016x", i, got, w)
		}
	}
}

func TestSubseedIsSeedXorFnv(t *testing.T) {
	// subseed(domain) = seed XOR fnv64(domain) (ADR-000 D5.3).
	const seed = uint64(0xDEADBEEF)
	if got := Subseed(seed, "world-gen"); got != seed^fnv64("world-gen") {
		t.Errorf("Subseed must be seed XOR fnv64(domain): got %#x", got)
	}
	// Distinct domains give distinct streams.
	if Subseed(seed, "world-gen") == Subseed(seed, "npc-values") {
		t.Error("distinct domains must derive distinct sub-seeds")
	}
	// Deterministic.
	if Subseed(seed, "narration") != Subseed(seed, "narration") {
		t.Error("Subseed must be deterministic")
	}
}

func TestFnv64Known(t *testing.T) {
	// FNV-1a 64 of the empty string is the offset basis; of "a" is a known
	// constant. Guards the in-repo hash against silent drift.
	if got := fnv64(""); got != 14695981039346656037 {
		t.Errorf("fnv64(\"\") = %d, want offset basis", got)
	}
	if got := fnv64("a"); got != 0xAF63DC4C8601EC8C {
		t.Errorf("fnv64(\"a\") = %#x, want 0xAF63DC4C8601EC8C", got)
	}
}
