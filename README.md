# Lachesis

Lachesis measures each thread and decides how much is allotted. Atropos cuts it.

A local-first productivity tracker for developers. It samples your foreground
window, finds your deep-work blocks, and scores the day by **yield** (outcomes per
focused hour), not by how long a window stayed open. Time-in-app is easy to farm:
leave an editor focused and look busy. Ratios and artifacts are much harder to fake.

- **One binary, no dependencies, no account, no network.** Everything stays on your machine.
- **Scores outcomes.** Commits are linked to the focus block they came from.
- **Resists gaming.** A block that loops one window title is flagged instead of rewarded.

> **Status: v0.3 (early).** Tracking works on **Windows** and **macOS**. **Linux** support
> (X11 only, needs `xprop` and `xprintidle`) is implemented but experimental: it is
> unit-tested and cross-built in CI, and has not yet had broad testing on real desktops.
> Wayland is not supported yet.

## Install

**Download a release (recommended).** Grab the archive for your OS and CPU from the
[Releases](https://github.com/Thezone-1/lachesis/releases) page, unpack it, and put
`lachesis` somewhere on your `PATH`.

**With Go 1.23+:**

```
go install github.com/Thezone-1/lachesis@latest
```

## Quick start

```powershell
lachesis start --label "deep work"     # begin a session in the background
lachesis status                        # is it running? since when?
lachesis stop                          # end the session
lachesis track                         # or run it in the foreground with the live view
lachesis report                        # today's score, focus blocks, yield
lachesis allot chrome 2h               # give chrome 2h a day; reports show overruns
lachesis classify title youtube distracting
```

`lachesis start` runs the tracker in the background and keeps running after you close the terminal.
Only one session runs at a time. `lachesis stop` ends it cleanly, and `lachesis status` shows when it
began. Sessions are recorded in `sessions.jsonl` and errors from a background session go to
`lachesis.log`, both in the data folder.

**macOS permissions.** The first run asks to control *System Events*. For window titles, also
allow your terminal under *System Settings > Privacy & Security > Accessibility*. Without it
Lachesis still tracks the app name, with an empty title.

**Linux.** Install the helpers once: `sudo apt install x11-utils xprintidle`.

Add the repos whose commits should count toward yield in `config.json`
(see [Configuration](#configuration)):

```json
{ "repos": ["C:\\Users\\you\\code\\my-project"] }
```

### The live view

On a terminal, `lachesis track` shows a tiny runner game. A ball of yarn rolls along the thread:
the thread is solid behind it (already spun) and dashed ahead (still to spin).

```
  L A C H E S I S                          HI 00074  00031
                                                          
                                                          
            (&)                                           
                  [!]                           [!]       
  ~~~~~~~~~~--  --  --  --  --  --  --  --  --  --  --  --
```

- **Focused or neutral:** the yarn runs (fast or at a jog) and jumps each `[!]` distraction.
- **Distracted:** it can't see the obstacle in time, so it bumps `(x)` and slows down.
- **Away:** the yarn rests and the world stops.
- **Over an allotment:** Atropos' scissors drop and snip the thread. `G A M E   O V E R`.
- The numbers in the corner are the Dino-style high score: `HI` is your best day of deep-work minutes, and the second number is today's.

It only animates on a real terminal. In the background, in a pipe or with `--plain`, it prints one
line and stays quiet. `--hide-titles` hides window titles for recordings, `LACHESIS_ASCII=1` uses
plain glyphs and `NO_COLOR=1` turns colors off. `lachesis demo` plays the whole story without
tracking anything, which is a good way to take a first look.

### Example report

```
Lachesis, 2026-07-12
Score 74: a well-measured day
  focus 0.72 · depth 0.61 · cohesion 0.88 · yield 0.67
Thread measured: 3h42m active · 25m idle
  [+] productive 2h40m · [·] neutral 45m · [!] distracting 17m
  3 commits · 11 files touched · 1.4 commits per deep hour

  [+] code                     2h10m  ████████████████████
  [·] chrome                   1h05m  ██████████  (54% of 2h allotted)

Deep work, 2 block(s), 1h55m total, longest 1h20m, 4 switches/h:
  09:12–10:32  code             1h20m  ● a1b2c3f fix sessionizer (lachesis)
```

## How the score works

The 0–100 score is built only from ratios and outcomes:

| Component | Definition | Weight |
|-----------|-----------|--------|
| Focus     | productive time / active time | 35 |
| Depth     | deep-work time (blocks of 25+ min) / productive time | 25 |
| Cohesion  | 1 − min(1, context switches per active hour / 30) | 15 |
| Yield     | commit-linked deep-work hours / deep-work hours | 25 |

- **Rules** label every sample `productive`, `neutral` or `distracting`. Title rules run
  before app rules, so Chrome on YouTube is distracting even if Chrome is neutral.
- **Focus blocks** are built from the sample stream. Short alt-tab glances (2 samples or
  fewer) do not break a block. Idle always does.
- **Yield** reads each configured repo's `git log` (read-only) and attributes each commit
  to the deep-work block that contains it, or the nearest one that ended within 30 minutes.

## Resource use

Measured over 30 seconds on an Apple Silicon Mac, with the window checked every 5 seconds:

| | CPU | Memory |
|---|---|---|
| background (`start`, plain) | about 2% of one core | 26 MB peak |
| live view (`track`) | about 2% of one core | 26 MB peak |

The live view adds almost nothing. Most of the CPU goes on macOS to the system call that reads the
front window (`osascript`, about 0.1 s of CPU per sample). Windows reads it directly and should
use less, but that has not been measured yet. Lower the cost by raising `poll_seconds` in the
config. Data grows by about 1.7 MB a day at the default settings.

## Privacy

Lachesis records the **app name and window title** every 5 seconds, plus an idle flag.
Window titles can contain sensitive text (document names, tab titles).

- Data is written only to local JSONL files, one per day. There is no network code.
- Delete the data folder at any time to erase everything.
- It reads `git log` of the repos you list, nothing else about them.

Data folder: `%LOCALAPPDATA%\lachesis` on Windows, `~/Library/Application Support/lachesis`
on macOS, `$XDG_DATA_HOME/lachesis` (or `~/.local/share/lachesis`) on Linux.
If you used the project before it was renamed (`lacheisis`), your folder is moved
automatically on first run.

## Configuration

`config.json` in the data folder. Every key is optional.

| Key | Meaning | Default |
|-----|---------|---------|
| `poll_seconds` | sampling interval | 5 |
| `idle_seconds` | idle after this long without input | 60 |
| `allotments` | per-app daily limits, e.g. `{"chrome": "2h"}` | none |
| `rules` | classification rules | built-in set |
| `repos` | git repos scanned for commit yield | none |
| `weights` | score weights (focus, depth, cohesion, yield; sum to 100) | 35/25/15/25 |

## Roadmap

- [x] Window tracking, focus blocks, yield scoring, allotments (Windows, macOS)
- [ ] **AI leverage:** link local AI-assistant usage counts (never content) to shipped
      artifacts, giving "tokens per commit"
- [ ] **MCP server:** let an AI assistant ask "how was my week?" and get focus, yield and
      leverage numbers
- [x] macOS tracking
- [ ] Linux: more testing on real desktops, Wayland (sway, Hyprland, GNOME)
- [ ] Homebrew, winget and scoop packages

## Contributing

Issues and PRs are welcome. Run `go vet ./... && go test ./...` before opening one.
CI runs the same checks on Windows, macOS and Linux.

## License

MIT, see [LICENSE](LICENSE).
