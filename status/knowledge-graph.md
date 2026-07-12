# Lacheisis — Knowledge Graph

Updated: 2026-07-12 (v0.2 shipped)

## What it is
Windows productivity tracker CLI in Go (stdlib only, no deps). Lachesis measures the
thread (time per app), Atropos flags what exceeds its allotment. Core stance:
productivity is **yield** (outcomes per focused hour), never volume — volume
metrics get farmed (Goodhart). Full argument in `planning/v0.2-yield.md`.

## Status
- v0.2.0 built 2026-07-12: yield-based scoring live, all 4 layers from the plan
- Dogfooding: tracker running as detached hidden process on this machine
  (`Start-Process lacheisis.exe track -WindowStyle Hidden`; does NOT survive
  reboot — restart manually)
- Config now includes `rules` (defaults ship in code), `repos`
  (lacheisis, ccyield, trainpulse, somoprovo portfolio), `weights`
- First real-day score: 13/100 (fragmented day, no deep blocks yet — honest)
- 7 unit tests pass incl. farmer-vs-deep-worker divergence acceptance test

## Architecture (flat `package main`)
| File | Role |
|------|------|
| `main.go` | CLI dispatch: `track`, `report [day]`, `allot`, `classify`, `version` |
| `win.go` | Win32 via `syscall.NewLazyDLL`: foreground window → app name; GetLastInputInfo for idle |
| `store.go` | Append-only JSONL per local day: `%LOCALAPPDATA%\lacheisis\data\YYYY-MM-DD.jsonl` |
| `config.go` | config.json: poll/idle seconds, allotments, rules, repos, score weights |
| `categories.go` | Rule engine: title rules before app rules, first match wins, unmatched→neutral; `classify` cmd |
| `blocks.go` | Sessionizer: segments split at idle/gaps (>3 ticks+jitter), same-app runs, ≤2-sample glances absorbed; Block entropy = unique titles/samples; deep = ≥25min productive |
| `gityield.go` | Read-only `git log --since/--until` per configured repo; commit → containing deep block, else nearest ended ≤30min before; yieldRatio = linked deep time / deep time |
| `score.go` | Score 0–100: focus 35 + depth 25 + cohesion 15 + yield 25 (weights in config); verdict line; farmer warnings (entropy < 0.05) |
| `report.go` | Report v2: score header, category split, per-app bars w/ [+]/[·]/[!] tags, deep-block timeline w/ commits, orphan commits, Atropos cuts, warnings |
| `v2_test.go` | Rule precedence, glance/gap/idle boundaries, entropy, attribution, farmer-vs-deep, score bounds |

## Design decisions
- **Never score volume** — every score component is a ratio; commit counts shown but not scored
- Block category = majority vote of its samples' categories (chrome-on-youtube blocks classify by title)
- Glance time counts toward block duration but not titles; glances still count as switches (fragmentation stays honest)
- Empty day scores 0 (cohesion isn't free when nothing was measured)
- App key = lowercased exe basename; JSONL over SQLite still (no schema change needed for v0.2)
- Test fixtures built in code (time-controlled), not testdata files — deviation from plan, simpler

## Later (v0.3+ candidates)
- Live Atropos alert (toast/beep) from tracker when allotment crossed
- Weekly report + trends; `lacheisis top` live view
- Auto-start on login (Task Scheduler); single-instance lock
- Metronome-uniform switching detection (second farmer signature — only entropy implemented)
- Claude/token usage ingestion via [[ccyield]] data (same yield-not-spend framing)
