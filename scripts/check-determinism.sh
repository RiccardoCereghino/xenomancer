#!/usr/bin/env bash
#
# Determinism guard (ADR-000 D5, CI enforcement (c)).
#
# Fails if any banned token appears anywhere under engine/:
#   - math/rand        stdlib PRNG whose behavior drifts across Go versions
#                      (also matches math/rand/v2)
#   - time.            wall-clock / time access
#   - float32 / float64  floating-point types
#
# The engine must be integer-only, time-free, and use the vendored splitmix64
# in engine/internal/rng instead of a stdlib generator.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE="${ROOT}/engine"

# \btime\. avoids matching substrings like runtime. or lifetime.
PATTERN='math/rand|\btime\.|float32|float64'

if matches="$(grep -rEn --include='*.go' "${PATTERN}" "${ENGINE}")"; then
	echo "determinism guard: FAIL — banned token(s) found under engine/:"
	echo "${matches}"
	exit 1
fi

echo "determinism guard: OK — no banned tokens under engine/"
