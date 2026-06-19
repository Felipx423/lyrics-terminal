# Lyrics Terminal

<p align="center">
  <img src="assets/demo.gif" alt="Lyrics Terminal demo">
</p>

[English](README.md) | [Português](README.pt-BR.md)

Lyrics Terminal is a Linux application that shows synchronized Spotify lyrics in the terminal. It detects the current track, reuses local cache entries when they are valid, falls back across providers when needed, and exposes health, stats, and failure-analysis tools for real-world use.

It is designed for people who want a light, terminal-first lyrics experience without browser overlays or heavyweight desktop apps.

## Current Status

Project status: Real World Testing

Current release: `v0.5.0 Observability`

Current milestone: `v0.6.0 Real World Testing`

The project is functional, but it is not being presented as fully stable yet. Real-world usage data is still being collected before larger feature work.

Feedback is welcome, especially for:

- wrong lyrics
- tracks without lyrics
- provider quality
- cache validation and quarantine
- installation flow

## Demo

<p align="center">
  <img src="assets/screenshot.png" alt="Lyrics Terminal screenshot">
</p>

## What It Does

- Shows synchronized lyrics for the current Spotify track
- Detects track changes automatically
- Handles pause and resume
- Runs in a dedicated Kitty window or in the current terminal
- Reuses valid local `.lrc` files
- Validates local cache entries before reuse
- Quarantines invalid or suspicious `.lrc` files
- Falls back across multiple providers
- Exposes health, stats, and failure-analysis commands
- Writes runtime logs for troubleshooting

## Requirements

- Linux
- Spotify visible through MPRIS
- `python3`
- `playerctl`
- `kitty` for the default windowed mode and `lyrics --kitty`
- Kitty is not required for `lyrics --current`
- `sptlrx` for live fallback rendering
- Go toolchain to build `lyrics-fetch-go` during installation

## Installation

Clone the repository:

```bash
git clone https://github.com/Felipx423/lyrics-terminal.git
cd lyrics-terminal
```

Install the runtime:

```bash
chmod +x install.sh
./install.sh
```

The installer builds `lyrics-fetch-go` and installs the runtime pieces into `~/.local/bin`.

## Basic Usage

Start the default lyrics view:

```bash
lyrics
```

Run in the current terminal:

```bash
lyrics --current
```

Open the explicit Kitty mode:

```bash
lyrics --kitty
```

Show version information:

```bash
lyrics --version
```

Run an environment check:

```bash
lyrics --health
```

## Execution Modes

### `lyrics`

Default launcher.

Behavior:

- opens the dedicated Kitty window by default
- checks Spotify playback through `playerctl`
- tries the local cache first
- starts background fetching and live fallback rendering when needed

### `lyrics --kitty`

Explicit alias for the default Kitty-based behavior.

Use this when you want the dedicated window mode to be obvious in scripts or notes.

### `lyrics --current`

Runs the runtime in the current terminal instead of opening Kitty.

Use this when you want to keep output in the active terminal session.
It does not require Kitty.

### `lyrics --health`

Checks whether the environment is ready.

It verifies:

- `playerctl`
- `kitty`
- `sptlrx`
- `lyrics-fetch-go`
- cache directories
- `index.json`
- Spotify status through `playerctl`

If Kitty is missing, `lyrics --health` reports it as `WARN` because `lyrics --current` can still work.

### `lyrics --version`

Prints the build metadata exposed by the Python runtime.

## Diagnostics

Use the Go fetcher diagnostics when you need cache and provider visibility.

```bash
lyrics-fetch-go --stats
lyrics-fetch-go --analyze-failures
```

`lyrics-fetch-go --stats` summarizes:

- local `.lrc` files
- quarantined files
- negative cache entries
- provider and status counts
- recent search activity

`lyrics-fetch-go --analyze-failures` inspects:

- persisted failure events
- the search index
- quarantine contents
- negative cache contents

Destructive cache reset:

```bash
lyrics-fetch-go --clear-cache
```

This removes the fetcher cache directory at `~/.cache/lyrics-terminal/`. Use it only when you really want to delete persisted cache data.

## Providers

Current provider order:

1. LRCLIB
2. NetEase Map
3. NetEase Search
4. syncedlyrics

Providers are validated before a result is accepted. A technically valid response is still rejected if the metadata does not match the current track or if the synced lyrics look suspicious.

## Cache, Validation, and Quarantine

Valid local `.lrc` files are reused after validation.

Current cache and quarantine paths:

- cache and diagnostics data: `~/.cache/lyrics-terminal/`
- local lyrics cache: `~/.local/share/lyrics/`
- invalid local files: `~/.local/share/lyrics/bad/`

If a local `.lrc` file is empty, missing timestamps, missing usable lyric lines, or otherwise suspicious, it is moved to quarantine instead of being reused.

This prevents broken or wrong lyrics from being picked up again on later runs.

## Known Limitations

- The project is focused on Linux
- Spotify access depends on MPRIS and `playerctl`
- Provider coverage is limited by the current upstream sources
- Kitty mode requires Kitty to be installed
- Current-terminal mode does not provide the same windowed experience as Kitty
- The project is still in real-world testing

## Report Bugs and Contribute

If you find wrong lyrics, missing lyrics, cache problems, installation issues, or provider regressions, please open an issue with enough detail to reproduce the case.

Useful evidence usually includes:

- artist
- title
- provider used
- whether the result came from cache or a live fetch
- relevant logs from `~/.cache/lyrics-terminal/lyrics.log`
- failure analysis output

Before larger changes, read the maintainer docs:

- [Development Guide](docs/DEVELOPMENT.md)
- [Real World Testing Exit Criteria](docs/REAL_WORLD_TESTING.md)

[Open an issue](https://github.com/Felipx423/lyrics-terminal/issues)
