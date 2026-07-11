# lacheisis

Lachesis measures each thread and decides how much is allotted, Atropos cuts it.

A zero-dependency Windows productivity tracker in Go. It samples your foreground
window every few seconds, measures where your time actually goes, and flags the
apps that exceed their daily allotment.

## Usage

```
go build -o lacheisis.exe .

lacheisis track                    # run the collector (detach with Start-Process)
lacheisis report [YYYY-MM-DD]      # the day's thread, per app
lacheisis allot chrome 2h          # give chrome 2h/day; report shows overruns
```

Run it in the background:

```powershell
Start-Process lacheisis.exe -ArgumentList track -WindowStyle Hidden
```

Example report:

```
Lacheisis — 2026-07-12
Thread measured: 3h42m active · 25m idle

  code                     2h10m  ████████████████████
  chrome                   1h05m  ██████████  (54% of 2h allotted)
  windowsterminal            27m  ████

✂ Atropos: slack exceeded its allotment (1h30m / 1h)
```

## How it works

- Win32 `GetForegroundWindow` → process image name, sampled every 5s
- `GetLastInputInfo` marks you idle after 60s without input; locked screen counts as away
- Samples land in append-only JSONL at `%LOCALAPPDATA%\lacheisis\data\`, one file per day
- Config (poll interval, idle threshold, allotments) at `%LOCALAPPDATA%\lacheisis\config.json`
