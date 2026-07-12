# lacheisis

Lachesis measures each thread and decides how much is allotted, Atropos cuts it.

A zero-dependency Windows productivity tracker in Go. It samples your foreground
window every few seconds, measures where your time actually goes, and scores the
day by **yield** — outcomes per focused hour — never volume. Volume metrics get
farmed (Goodhart); ratios and artifacts don't.

## Usage

```
go build -o lacheisis.exe .

lacheisis track                    # run the collector (detach with Start-Process)
lacheisis report [YYYY-MM-DD]      # score, focus blocks, commit yield, per-app thread
lacheisis allot chrome 2h          # give chrome 2h/day; report shows overruns
lacheisis classify title youtube distracting   # add a classification rule
```

Run it in the background:

```powershell
Start-Process lacheisis.exe -ArgumentList track -WindowStyle Hidden
```

Example report:

```
Lacheisis — 2026-07-12
Score 74 — a well-measured day
  focus 0.72 · depth 0.61 · cohesion 0.88 · yield 0.67
Thread measured: 3h42m active · 25m idle
  [+] productive 2h40m · [·] neutral 45m · [!] distracting 17m
  3 commits · 11 files touched · 1.4 commits per deep hour

  [+] code                     2h10m  ████████████████████
  [·] chrome                   1h05m  ██████████  (54% of 2h allotted)
  [+] windowsterminal            27m  ████

Deep work — 2 block(s), 1h55m total, longest 1h20m, 4 switches/h:
  09:12–10:32  code             1h20m  ● a1b2c3f fix sessionizer (lacheisis)
  11:04–11:39  code               35m

✂ Atropos: slack exceeded its allotment (1h30m / 1h)
```

## Scoring (v0.2): attendance is farmable, yield is not

Foreground hours can be farmed — leave an editor focused and look busy. So the
0–100 score is built only from ratios and outcomes:

| Component | Definition | Weight |
|-----------|-----------|--------|
| Focus     | productive time / active time | 35 |
| Depth     | deep-work time (≥25 min productive blocks) / productive time | 25 |
| Cohesion  | 1 − min(1, context switches per active hour / 30) | 15 |
| Yield     | commit-linked deep-work hours / deep-work hours | 25 |

- **Rules** classify every sample `productive | neutral | distracting`; title
  rules run before app rules (chrome on YouTube is distracting even if chrome
  is neutral). Manage with `lacheisis classify`.
- **Focus blocks** sessionize the sample stream: alt-tab glances (≤2 samples)
  don't break a block, idle always does.
- **Yield** scans the configured `"repos"` (read-only `git log`) and attributes
  each commit to the deep-work block containing it, or the nearest one that
  ended ≤30 min before.
- **Anti-gaming**: farmed activity loops one window title. Deep blocks with
  near-zero title entropy are flagged `⚠ looping?`, and no metric anywhere
  rewards raw volume.

## How it works

- Win32 `GetForegroundWindow` → process image name, sampled every 5s
- `GetLastInputInfo` marks you idle after 60s without input; locked screen counts as away
- Samples land in append-only JSONL at `%LOCALAPPDATA%\lacheisis\data\`, one file per day
- Config (poll interval, idle threshold, allotments, rules, repos, score
  weights) at `%LOCALAPPDATA%\lacheisis\config.json`
