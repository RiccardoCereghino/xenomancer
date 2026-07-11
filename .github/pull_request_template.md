<!--
One issue per PR (see CONTRIBUTING.md). Complete every checklist item — an
unchecked box is a claim you have not made. Delete a section only if it is
genuinely not applicable, and say why.
-->

**Linked issue:** #

### Scope
- [ ] Changes are limited to the linked issue.
- [ ] The anti-scope list (GDD §12) is untouched.

### Determinism (ADR-000 D5)
- [ ] The determinism guard passes.
- [ ] No floats, `time.`, `math/rand`, or map iteration added under `/engine`.
- [ ] Frozen surfaces — `CanonicalBytes` encoding and the vendored rng — are untouched. If touched, I linked the ADR and marked this PR **breaking**.

### Golden hash
- [ ] Unchanged, **or**
- [ ] Superseded — reason: _content vs rules change_; goldens regenerated.

### Tests
- [ ] New behavior is covered, including every new rejection or death class.

### Docs
- [ ] GDD/ADR updates or `[OPEN]` items resolved, if any (or n/a).

### Evidence
<!--
If engine imports changed, paste the output of:
  go list -f '{{.ImportPath}}: {{.Imports}}' ./engine/...
-->
