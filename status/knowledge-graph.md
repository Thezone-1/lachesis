# Lacheisis — Knowledge Graph

Updated: 2026-07-12 (evening)

## What it is
Windows productivity tracker CLI in Go (stdlib only, no deps). Lachesis measures the
thread (time per app), Atropos flags what exceeds its allotment. Core stance:
productivity is **yield** (outcomes per focused hour), never volume — volume
metrics get farmed (Goodhart). See `planning/v0.2-yield.md` for the full argument.

## Status
- v0.1.0 MVP built and dogfooding on Somoprovo's machine since 2026-07-12
- Tracker runs as a detached hidden process: `Start-Process lacheisis.exe track -WindowStyle Hidden`
  (does NOT survive reboot yet — restart manually)
- Demo allotment set: `code` → 6h/day

## ➡ NEXT SESSION: build v0.2 per `planning/v0.2-yield.md`
Four layers, in order: (1) category rules engine `categories.go` + `classify`
command, (2) focus-block sessionizer `blocks.go` (deep-work ≥25min, switch
rate, title entropy), (3) `gityield.go` commit-to-block attribution from
configured repos, (4) ratio-based daily score 0–100 + report v2. Anti-gaming:
score only ratios/outcomes, flag low-entropy "farmer" blocks. Test fixtures:
farmer vs deep worker must diverge. All computable from existing JSONL — no
schema change.

## Architecture (flat `package main`)
| File | Role |
|------|------|
| `main.go` | CLI dispatch: `track`, `report [day]`, `allot <app> <dur>`, `version` |
| `win.go` | Win32 via `syscall.NewLazyDLL`: GetForegroundWindow → pid → QueryFullProcessImageNameW for app name; GetLastInputInfo for idle |
| `store.go` | Append-only JSONL per local day: `%LOCALAPPDATA%\lacheisis\data\YYYY-MM-DD.jsonl`, one `Sample{ts,app,title,idle}` per poll |
| `config.go` | `%LOCALAPPDATA%\lacheisis\config.json`: poll_seconds (5), idle_seconds (60), allotments map app→duration |
| `report.go` | Aggregates samples (each = poll_seconds of attention) per app, bar chart, allotment % + "✂ Atropos" overrun lines |

## Design decisions
- **JSONL over SQLite** for v0.1 — zero deps, torn lines skipped on read; revisit if querying grows
- App key = lowercased exe basename without `.exe` (e.g. `chrome`, `code`)
- Locked screen / secure desktop / idle ≥60s → sample counted as idle
- Titles are stored per sample (drill-down exists in data, not yet in report UI)

## Later (v0.3+ candidates, deliberately out of v0.2)
- Live Atropos alert (toast/beep) from the tracker when allotment crossed, not just in report
- Weekly report + trends; `lacheisis top` for current-day live view
- Auto-start on login (Task Scheduler); single-instance lock
- Claude/token usage ingestion via ccyield data (same yield-not-spend framing)
