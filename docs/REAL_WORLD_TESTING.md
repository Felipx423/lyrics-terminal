# Real World Testing Exit Criteria

## Purpose of v0.6.0

The `v0.6.0 - Real World Testing` phase exists to validate the project under normal daily use, with real Spotify playback, real provider behavior, real cache reuse, and real user mistakes.

The goal is not to promise stability. The goal is to observe the system in production-like conditions, record what actually fails, and confirm that the cache, quarantine, diagnostics, and provider selection behave well enough to move to the next milestone.

## Current Status

Project status: Real World Testing

Current release: v0.5.0 Observability

Current milestone: v0.6.0 Real World Testing

The project is considered functional but not yet stable. Real-world usage data is still being collected before broader feature expansion.

## What to Observe During Real Use

During the testing window, observe:

- whether the correct lyrics appear for the current track
- whether the track changes are detected reliably
- whether pause and resume stay in sync with playback
- whether the cache is reused correctly for valid `.lrc` files
- whether invalid `.lrc` files are rejected and quarantined
- whether provider fallback behaves sensibly when the first source fails
- whether false positives appear for tracks with the wrong language, artist, or title
- whether the diagnostics reflect the current state of the system
- whether logs are sufficient to reproduce or triage a failure later

## Minimum Exit Criteria

This phase can be considered complete only after all of the following are true:

- at least 14 days of real usage have been observed
- at least 100 tracks have been observed in real sessions
- no critical crash is reproducible in normal usage
- wrong-lyrics cases have been investigated or explicitly recorded
- provider metrics have been reviewed
- invalid cache and quarantine behavior has been validated
- critical issues have been triaged
- the README still makes it clear that the project is in real-world testing

These are minimum exit criteria, not a guarantee of final stability.

## What Counts as a Critical Bug

A bug is critical if it blocks real-world use or makes the output unreliable in a way that cannot be ignored.

Treat the following as critical:

- a reproducible crash in `lyrics` or `lyrics-fetch-go`
- a reproducible failure to detect or update the current track
- a reproducible failure to recover from pause, resume, or track change
- invalid local cache entries being reused after validation
- quarantine failing to move invalid `.lrc` files out of the cache path
- repeated wrong lyrics for the same track with a clear reproduction path
- diagnostics or analysis commands failing in a way that prevents triage

If a bug can corrupt cache state or cause the user to trust wrong lyrics repeatedly, treat it as critical.

## What Counts as Acceptable for Future Versions

Some issues are acceptable to carry into future versions if they are documented, understood, and do not block real-world usage.

Examples:

- a provider occasionally returning no result
- a provider being slower than desired but still functional
- a rare false negative that falls back to another provider
- a track without available synced lyrics from any current provider
- a locale-specific ranking improvement that is useful but not urgent
- better filtering, ranking, or observability that would improve quality but not correctness

Acceptable future issues should be recorded, not ignored.

## How to Record Wrong Lyrics

When the app shows wrong lyrics, record the case with enough detail to reproduce it later.

Capture:

- artist
- title
- provider used
- whether the issue came from cache reuse or a live fetch
- the observed wrong lyric behavior
- the expected lyric behavior
- whether the track is a one-off or reproducible
- any relevant logs from `~/.cache/lyrics-terminal/lyrics.log`
- any related failure entries in `~/.cache/lyrics-terminal/failures.jsonl`
- any quarantined file in `~/.local/share/lyrics/bad/`

Also record whether the cache entry was invalidated, quarantined, or left in place. If a wrong lyric comes from a cached `.lrc`, treat that as a cache regression until proven otherwise.

## How to Record Tracks Without Lyrics

When a track has no lyrics, record it as a discovery result, not automatically as a bug.

Capture:

- artist
- title
- duration
- provider order tried
- whether the result was a true empty result or a provider failure
- whether the track is known to have lyrics elsewhere
- whether the absence is consistent across multiple attempts

Track-with-no-lyrics cases should be triaged as one of the following:

- legitimate missing coverage
- provider failure
- bad metadata match
- cache or quarantine regression

Only the last two categories should be treated as correctness problems.

## Logs and Diagnostic Commands

Use the built-in diagnostics first.

Commands:

```bash
lyrics --health
lyrics-fetch-go --stats
lyrics-fetch-go --analyze-failures
```

What each command is for:

- `lyrics --health` checks the runtime prerequisites and basic environment health
- `lyrics-fetch-go --stats` summarizes cache and provider behavior
- `lyrics-fetch-go --analyze-failures` inspects persisted failures, quarantine, and failure categories

Useful files for triage:

- `~/.cache/lyrics-terminal/lyrics.log`
- `~/.cache/lyrics-terminal/index.json`
- `~/.cache/lyrics-terminal/failures.jsonl`
- `~/.local/share/lyrics/bad/`

Use logs to answer three questions:

- did the app see the right track
- did it choose the right source
- did it preserve or reject cache entries correctly

## Final Checklist Before Closing the Milestone

Before closing the milestone, confirm all of the following:

- the testing period reached at least 14 real-world days
- at least 100 tracks were observed
- no reproducible critical crash remains open
- wrong-lyrics incidents were investigated or recorded with enough detail
- missing-lyrics incidents were classified correctly
- provider metrics were reviewed from `lyrics-fetch-go --stats`
- failure analysis was reviewed with `lyrics-fetch-go --analyze-failures`
- quarantine behavior was validated on invalid `.lrc` files
- cache reuse was validated on valid `.lrc` files
- critical issues were triaged with a clear next action
- the README still states that the project is in real-world testing
- no new feature work has been merged in place of the validation work

If any item is missing, the milestone is not ready to close.
