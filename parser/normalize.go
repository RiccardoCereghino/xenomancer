package parser

import "strings"

// normalize turns a freeform line into its canonical token sequence: lowercase,
// every character outside [a-z0-9] treated as a separator, and whitespace
// collapsed (GDD §5.2). It is a pure function — same input, same tokens — and
// deliberately ASCII-only this sprint. Dictionary keys are stored already in
// this normalized form, so lookup is an exact match against joined windows of
// these tokens.
func normalize(line string) []string {
	var b strings.Builder
	b.Grow(len(line))
	for _, r := range line {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			// Any punctuation, whitespace, or non-ASCII rune becomes a break.
			b.WriteByte(' ')
		}
	}
	return strings.Fields(b.String())
}
