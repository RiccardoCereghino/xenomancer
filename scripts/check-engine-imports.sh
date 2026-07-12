#!/usr/bin/env bash
#
# Quarantine guard (ADR-000 D3/D4, GDD §5.2).
#
# The engine must never import the freeform parser: only canonical actions cross
# into the engine and the replay log, so parser evolution can never invalidate a
# replay. This asserts that invariant on the actual import graph — the automated
# form of the "verifiable via go list imports" DoD in the parser backlog.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

MODULE="$(go list -m)"
PARSER="${MODULE}/parser"

# Transitive import closure of every package under engine/.
if go list -deps ./engine/... | grep -qx "${PARSER}"; then
	echo "quarantine guard: FAIL — engine/ transitively imports ${PARSER}"
	echo "the parser must live outside the replay path (ADR-000 D3/D4)."
	exit 1
fi

echo "quarantine guard: OK — engine/ does not import the parser"
