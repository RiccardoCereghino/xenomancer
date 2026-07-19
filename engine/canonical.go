package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// CanonicalBytes serializes State into a fixed field-order binary encoding
// (ADR-000 D5.6).
//
// This encoding is FROZEN. The field order and byte layout below must never
// change: changing them changes every state hash, which is a breaking engine
// version that orphans all existing replays (ADR-000 D5.6). Any such change
// must bump Version and keep the old tag buildable.
//
// Encoding v3 (all integers big-endian):
//
//   1. Seed              : uint64                      (8 bytes)
//   2. Tick              : uint64                      (8 bytes)
//   3. Round             : uint64                      (8 bytes)
//   4. Location          : uint32 length + UTF-8 bytes
//   5. Holds             : uint32 count, then for each hold in slice order:
//      Resource          : uint32 length + UTF-8 bytes
//      Tag               : uint32 length + UTF-8 bytes
//      Since             : uint64                      (8 bytes)
//   6. Outcome           : uint32 length + UTF-8 bytes  (empty while ongoing)
//   7. Cause             : uint32 length + UTF-8 bytes  (empty unless died)
//   8. Fuse              : uint64                      (8 bytes)
//   9. GrappleRoundsLeft : uint64                      (8 bytes)
//  10. GrappleStruggles  : uint64                      (8 bytes)
//
// Fields 6-7 were appended in v2 (engine 0.2.0) for the terminal outcome
// (GDD §5.7, §7). Fields 8-10 were appended in v3 (engine 0.3.0) for the hazard
// fuse and grapple state — the wolf (GDD §5.6). Appending is still a
// frozen-encoding change — it moves every hash — so each bumped Version and
// superseded the goldens (ADR-000 D5.6).
//
// Content is intentionally NOT encoded: content identity is tracked separately
// by content hash in the replay header (ADR-000 D5.5 / D6).
func (s State) CanonicalBytes() []byte {
	b := make([]byte, 0, 88)
	b = appendUint64(b, s.Seed)
	b = appendUint64(b, s.Tick)
	b = appendUint64(b, s.Round)
	b = appendString(b, s.Location)
	b = appendUint32(b, uint32(len(s.Holds)))
	for i := 0; i < len(s.Holds); i++ {
		b = appendString(b, s.Holds[i].Resource)
		b = appendString(b, s.Holds[i].Tag)
		b = appendUint64(b, s.Holds[i].Since)
	}
	b = appendString(b, s.Outcome)
	b = appendString(b, s.Cause)
	b = appendUint64(b, s.Fuse)
	b = appendUint64(b, s.GrappleRoundsLeft)
	b = appendUint64(b, s.GrappleStruggles)
	return b
}

// StateHash is the SHA-256 over CanonicalBytes, formatted as "sha256:<hex>"
// (ADR-000 D5.6). It is the value compared for replay verification.
func (s State) StateHash() string {
	sum := sha256.Sum256(s.CanonicalBytes())
	return "sha256:" + hex.EncodeToString(sum[:])
}

func appendUint64(b []byte, v uint64) []byte {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], v)
	return append(b, tmp[:]...)
}

func appendUint32(b []byte, v uint32) []byte {
	var tmp [4]byte
	binary.BigEndian.PutUint32(tmp[:], v)
	return append(b, tmp[:]...)
}

func appendString(b []byte, s string) []byte {
	b = appendUint32(b, uint32(len(s)))
	return append(b, s...)
}
