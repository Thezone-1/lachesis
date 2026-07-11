# Lacheisis — Knowledge Graph

Updated: 2026-07-12

## What it is
Windows productivity tracker CLI in Go (stdlib only, no deps). Lachesis measures the
thread (time per app), Atropos flags what exceeds its allotment.

## Status
- v0.1.0 MVP built and dogfooding on Somoprovo's machine since 2026-07-12
- Tracker runs as a detached hidden process: `Start-Process lacheisis.exe track -WindowStyle Hidden`

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

## Next candidates
- Title/category grouping (e.g. classify browser tabs by title)
- Live Atropos alert (toast/beep) from the tracker when allotment crossed, not just in report
- Weekly report + trends; `lacheisis top` for current-day live view
- Auto-start on login (Task Scheduler); single-instance lock
