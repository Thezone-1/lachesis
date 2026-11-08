# Lachesis

> Lachesis measures each thread and decides how much is allotted. Atropos cuts it.

A local-first focus tracker for developers, with a tiny game in your terminal. A ball of yarn runs
along a golden thread while you work, jumps your distractions, and gets snipped by Atropos when you
go over an allotment. When a session ends you get a recap card.

It scores your day by **yield** (outcomes per focused hour), not by how long a window stayed open.
Time-in-app is easy to farm: leave an editor focused and look busy. Ratios and artifacts are not.

- **One binary, no dependencies, no account, no network.** Everything stays on your machine.
- **Scores outcomes.** Commits are linked to the focus block they came from.
- **Resists gaming.** A block that loops one window title is flagged instead of rewarded.
- **Light.** About 1% of one CPU core and 26 MB of memory (see [Resource use](#resource-use)).

```
  L A C H E S I S                          HI 00074  00031
                                          * .   +
            (&)         ' *
                  [!]                           [!]
  ~~~~~~~~~~--  --  --  --  --  --  --  --  --  --  --  --
  ● tracking · session 12m · 140 samples · last 2s ago
  now    [+] productive  code main.go
  today  3h42m active · 25m idle · focus block 1h20m
  score  ███████░░░ 74  a well-measured day
```

## Quick start

```sh
lachesis doctor             # once: checks that everything works on this machine
lachesis start              # begin a session when you start to work
# ... code ...
lachesis stop               # end the session and see your recap
```

That is the whole idea. `start` when you sit down, `stop` when you are done. While a session runs,
Lachesis watches the foreground window every few seconds. Nothing is recorded outside a session.

```
  ╭────────────────────────────────────────────────────────╮
  │ ✦ session complete · "writing the README"              │
  │ 1h 12m on the clock · 1h 05m active                    │
  │ productive 58m · distracting 4m                        │
  │ 2 deep blocks · longest 38m                            │
  │ 3 commits linked to deep work                          │
  │ score ███████░░░ 74  a well-measured day               │
  ╰────────────────────────────────────────────────────────╯
```

## Install

### 1. Get the binary

**Download a release (recommended).** Open the
[Releases](https://github.com/Thezone-1/lachesis/releases) page, download the archive for your OS and
CPU, unpack it, and put `lachesis` somewhere on your `PATH`.

**Or build from source** (needs [Go 1.23+](https://go.dev/dl/)):

```sh
go install github.com/Thezone-1/lachesis@latest
```

Check it: `lachesis version`.

> Release binaries are not code-signed yet. macOS may say the app "cannot be opened" because it was
> downloaded; run `xattr -d com.apple.quarantine ./lachesis` once. Windows SmartScreen may ask you to
> choose "More info > Run anyway". Building from source avoids both.

### 2. Set up your OS, then run `lachesis doctor`

`lachesis doctor` checks everything below and tells you what to fix. Run it after any change.

**macOS**

- Tracking the foreground app works straight away, with no permission.
- To also read **window titles** (so Lachesis can tell a YouTube tab from a docs tab), allow your
  terminal app under *System Settings > Privacy & Security > Accessibility*, and *Automation > your
  terminal > System Events*. macOS asks once the first time. `lachesis doctor --open` jumps to the right pane.
- Titles refresh every 10 s for browsers and 30 s for other apps, and immediately when you switch app.
  Without the permission, Lachesis still works but only sees app names.

**Windows**

- Nothing to install or allow. Windows 10 or 11 in Windows Terminal or PowerShell gives the best colors.

**Linux** (X11 only for now, experimental)

```sh
sudo apt install x11-utils xprintidle      # Debian / Ubuntu; other distros: xprop and xprintidle
```

You need an X11 (Xorg) session. Wayland is not supported yet. Linux is unit-tested and cross-built in CI
but has had less testing on real desktops than macOS and Windows.

### 3. Tell it which repos count toward yield (optional)

Open `config.json` in the data folder (see [Privacy](#privacy)) and list your repos:

```json
{ "repos": ["/Users/you/code/my-project", "/Users/you/code/another"] }
```

Without repos you still get focus, depth and cohesion. Yield needs them.

## Daily use

| Command | What it does |
|---|---|
| `lachesis start [--label NAME]` | Begin a session in the background. It keeps running if you close the terminal. |
| `lachesis stop` | End the session and print the recap card. |
| `lachesis status` | Is a session running? Since when? What was the last one? |
| `lachesis track [--plain] [--hide-titles]` | Run a session in the foreground with the live game view. |
| `lachesis report [YYYY-MM-DD]` | Score, focus blocks, commit yield and per-app time for a day. |
| `lachesis allot <app> <duration>` | Give an app a daily allotment, e.g. `lachesis allot chrome 2h`. |
| `lachesis classify <app\|title> <pattern> <category>` | Add a rule (productive, neutral or distracting). |
| `lachesis doctor [--open]` | Check that everything works here and how to fix what does not. |
| `lachesis demo` | Play the whole story without tracking anything. A good first look. |

Only one session runs at a time. Starting twice just tells you it is already running, and a crash never
blocks the next start.

## The live view

`lachesis track` shows a runner game. The yarn rolls along the thread, which is solid gold behind it
(already spun) and dashed ahead (still to spin).

- **Focused or neutral:** it runs (fast, or at a jog) and jumps each `[!]` distraction.
- **Distracted:** it cannot see the obstacle in time, so it bumps `(x)` and slows down.
- **Away:** the yarn rests and the world stops.
- **A 25-minute deep block:** sparkles. You earned them.
- **Over an allotment:** Atropos' scissors drop and snip the thread. `G A M E   O V E R`.
- The corner shows a Dino-style score: `HI` is your best day of deep-work minutes, then today's.

It only animates on a real terminal. In the background, in a pipe or with `--plain` it prints one line
and stays quiet. `--hide-titles` keeps window titles off the screen for recordings,
`LACHESIS_ASCII=1` uses plain glyphs and `NO_COLOR=1` turns color off.

## How the score works

The 0-100 score is built only from ratios and outcomes:

| Component | Definition | Weight |
|-----------|-----------|--------|
| Focus     | productive time / active time | 35 |
| Depth     | deep-work time (blocks of 25+ min) / productive time | 25 |
| Cohesion  | 1 - min(1, context switches per active hour / 30) | 15 |
| Yield     | commit-linked deep-work hours / deep-work hours | 25 |

- **Rules** label every sample `productive`, `neutral` or `distracting`. Title rules run before app
  rules, so Chrome on YouTube is distracting even if Chrome is neutral.
- **Focus blocks** are built from the sample stream. Short alt-tab glances do not break a block. Idle always does.
- **Yield** reads each configured repo's `git log` (read-only) and attributes each commit to the deep-work
  block that contains it, or the nearest one that ended within 30 minutes.
- **A score needs 5 minutes of activity.** Before that it says "warming up".

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

## Privacy

Lachesis records the **app name and window title** every few seconds during a session, plus an idle flag.
Window titles can contain sensitive text (document names, tab titles).

- Data is written only to local files that only you can read. There is **no network code**.
- Delete the data folder at any time to erase everything.
- It reads `git log` of the repos you list, nothing else about them.

Data folder: `%LOCALAPPDATA%\lachesis` on Windows, `~/Library/Application Support/lachesis` on macOS,
`$XDG_DATA_HOME/lachesis` (or `~/.local/share/lachesis`) on Linux. Inside: one `.jsonl` file per day,
`config.json`, `sessions.jsonl` (start and stop times) and `lachesis.log` (errors from a background session).
If you used the project before it was renamed (`lacheisis`), your folder is moved automatically on first run.

## Resource use

Measured over 30 seconds on an Apple Silicon Mac:

| | CPU | Memory |
|---|---|---|
| background session (`start`) | about 1% of one core | 26 MB peak |
| live view (`track`) | about 1% of one core | 26 MB peak |

The live view adds almost nothing. On macOS the app name comes from a very light system call and
window titles are refreshed slowly, which is what keeps it low. Raise `poll_seconds` to lower it
further. Data grows by about 1.7 MB a day at the default settings. Windows and Linux have not been measured yet.

## Troubleshooting

- **Everything shows as "away".** Your screen is locked, or you have been idle longer than `idle_seconds`.
  Lachesis counts a locked screen as away on purpose.
- **"window titles are off".** macOS Accessibility is not allowed for your terminal. Run `lachesis doctor --open`.
- **`lachesis start` says it did not start.** Read `lachesis.log` in the data folder.
- **"already tracking".** A session is running. `lachesis status` shows it, `lachesis stop` ends it.
- **Colors look flat.** Use a terminal with 256-color support (Terminal.app, iTerm2, Windows Terminal and
  most Linux terminals have it), and make sure `NO_COLOR` is not set.
- **Linux says it is a Wayland session.** Log in with an X11 / Xorg session instead.

## Roadmap

- [x] Window tracking, focus blocks, yield scoring, allotments (Windows, macOS)
- [x] Sessions: start, stop, status, recap
- [ ] **AI leverage:** link local AI-assistant usage counts (never content) to shipped artifacts,
      giving "tokens per commit"
- [ ] **MCP server:** let an AI assistant ask "how was my week?" and get focus, yield and leverage numbers
- [ ] Linux: more testing on real desktops, Wayland (sway, Hyprland, GNOME)
- [ ] Homebrew, winget and scoop packages, signed binaries

## Contributing

Issues and PRs are welcome. Run `go vet ./... && go test ./...` before opening one. CI runs the same
checks on Windows, macOS and Linux.

## License

MIT, see [LICENSE](LICENSE).
