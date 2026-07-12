# Contributing to XENOMANCER

XENOMANCER is a machine-first text infiltration game and evaluation harness. The
design lives in [`docs/design/GDD-v0.2.md`](docs/design/GDD-v0.2.md); the
founding technical decisions live in
[`docs/adr/ADR-000`](docs/adr/ADR-000-deterministic-reducer-and-protocol.md).
Read those two before opening a PR — they are the source of truth.

## The loop: issue → session → PR

1. **Issue.** Every change starts as an issue. Use the forms in
   [`.github/ISSUE_TEMPLATE`](.github/ISSUE_TEMPLATE) (feature, bug, unfair
   death, tuning). A feature must name the GDD/ADR section it derives from and
   the pillar it serves (GDD §2); if it has no anchor in the source of truth, it
   does not ship. The backlog itself is code:
   [`.github/backlog`](.github/backlog) files are reconciled into issues by
   `repo-sync.yml` on merge to `main`.
2. **Session.** A coding session implements a single issue. The issue should
   carry enough spec — behavior, edge cases, definition of done, out-of-scope
   lines — that the session can work from the issue alone.
3. **PR.** Open a PR that completes the
   [pull request checklist](.github/pull_request_template.md). **One issue per
   PR** — keep scope limited to the linked issue and leave the anti-scope list
   (GDD §12) untouched.

## Phase & sprint vocabulary

The repo grew two overlapping names for the same timeline — GDD **phases**
(§12 roadmap) and **`sprint-N`** milestones — and they are off by one. This is
the authoritative mapping of what is actually in the repo today:

| GDD phase (§12) | Repo label / milestone | Where it lives |
|---|---|---|
| — (pre-slice foundation) | `sprint-0` | Walking skeleton: engine + determinism CI (README, DEVLOG). Shipped (PR #1). |
| **Phase 0 — Vertical slice** | milestone `sprint-1 — woods to gate` | Zone 1: pond, wolf, guard, parser, runner. Backlog 01–06 / issues #3–#8. |
| **Phase 1 — Magic** | milestone `sprint-2 — magic` | Hermit zone, Sparklight + Fireball. |
| **Phase 2 — Town & contracts** | *(no milestone yet)* | Zone 2, negotiation, structural variance; publicity post. |
| **Phase 3 — Leaderboard & seasons** | *(no milestone yet)* | Replay-as-proof verification, sealed season, HTTP shell. |
| **Phase 4 — Steam** | *(no milestone yet)* | Spectator client, official seasons, opt-in inflation. |
| **Someday** | *(no milestone yet)* | Per-seed procedures tier; MMO shell. |

**Session prompts and issues must use GDD phase numbers; `sprint-N` is
deprecated outside historical DEVLOG entries and the two existing milestones
above.** Do not renumber the existing `sprint-1`/`sprint-2` milestones — they are
in flight and their renaming would break `repo-sync.yml` and orphan open issues.
New backlog specs cite the GDD phase in their body rather than a `sprint-N`
milestone.

## Determinism laws (summary)

The engine is a pure, seeded, deterministic reducer so that replays are proofs
(GDD P1, ADR-000 D1). Inside `/engine`:

- **Integer arithmetic only** — no floats anywhere.
- **No map iteration** in any state- or event-affecting path — ordered slices or
  explicitly sorted keys only.
- **Vendored PRNG only** — the in-repo `splitmix64`; `math/rand` and
  `math/rand/v2` are banned. All randomness derives from the run seed via
  sub-seeding: `subseed(domain) = seed ⊕ fnv64(domain)`.
- **No time, env, filesystem, or network** in `/engine`.
- `State.CanonicalBytes()` and the vendored PRNG are **frozen surfaces** —
  changing either is a breaking engine version.

CI enforces these (a guard lint plus a cross-OS replay-hash matrix). The full,
authoritative statement is **ADR-000 D5** — read it before touching the engine.

## How goldens supersede

Golden replays are the committed determinism fixtures. A **rules** change that
alters state must regenerate them and mark the PR's golden-hash box
**superseded**, with the reason (content vs rules change). A **content** change
that alters the canonical world (new facts, new map, new palette) likewise
supersedes and regenerates them. An unchanged golden hash on a state-affecting
PR is a red flag — say which case applies in the PR.

## Content & repo policy

- **Public vs held-out content.** Public issues cover engine, mechanics, and
  training-grounds content; held-out season and narrative-surprise content is
  designed privately and ships only as sealed packs (GDD §11).
- **Repo structure.** Engine and content remain a monorepo; a private
  seasons-workshop split is deferred to ADR-001, trigger = first sealed season
  (Phase 3).
