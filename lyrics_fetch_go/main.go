package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	playerctlTimeout = 2 * time.Second
	globalTimeout    = 20 * time.Second
)

var (
	fetchLyricsFn          = fetchLyrics
	findLocalLRCFn         = findLocalLRC
	saveLocalLyricsFn      = saveLocalLyrics
	recordSearchOutcomeFn  = recordSearchOutcome
	recordFailureEventFn   = recordFailureEvent
	printStatsFn           = printStats
	printFailureAnalysisFn = printFailureAnalysis
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("lyrics-fetch-go", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	debug := fs.Bool("debug", false, "enable debug logging")
	clearCache := fs.Bool("clear-cache", false, "remove cache directory")
	statsMode := fs.Bool("stats", false, "show provider and cache statistics")
	analyzeFailures := fs.Bool("analyze-failures", false, "analyze cached failure causes")
	dryRun := fs.Bool("dry-run", false, "fetch candidates without saving cache")
	noSpotify := fs.Bool("no-spotify", false, "use provided artist/title without playerctl")
	ignoreLocalCache := fs.Bool("ignore-local-cache", false, "skip local cache shortcut")
	artist := fs.String("artist", "", "track artist")
	title := fs.String("title", "", "track title")
	album := fs.String("album", "", "track album")
	duration := fs.Float64("duration", 0, "track duration in seconds")
	trackID := fs.String("track-id", "", "track id")
	deepSearch := fs.Bool("deep-search", false, "try extra provider queries")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *clearCache {
		if err := os.RemoveAll(cacheDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Fprintln(os.Stderr, "lyrics-fetch-go: cache limpo")
		return 0
	}
	if *statsMode {
		if err := printStatsFn(os.Stdout, loadIndex()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	if *analyzeFailures {
		if err := printFailureAnalysisFn(os.Stdout, loadIndex(), loadFailureEvents(), localDir, cacheDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	debugLog(*debug, "startup", map[string]any{"mode": "run", "debug": *debug, "args": args})
	debugLog(*debug, "paths", debugPaths())
	track, ok, err := resolveTrack(*noSpotify, *artist, *title, *album, *duration, *trackID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !ok {
		debugLog(*debug, "track", "missing")
		return 1
	}
	debugLog(*debug, "track", fmt.Sprintf("%s - %s", track.Artist, track.Title))

	if !*ignoreLocalCache && !*dryRun {
		if existingPath, _, ok := findLocalLRCFn(track); ok {
			debugLog(*debug, "local_exists", existingPath)
			debugLog(*debug, "cache_hit", existingPath)
			if err := recordSearchOutcomeFn(track, "found", "local-cache", existingPath, "", []string{existingPath}); err != nil {
				debugLog(*debug, "index_error", err)
			}
			return 0
		}
		debugLog(*debug, "cache_miss", fmt.Sprintf("%s - %s", track.Artist, track.Title))
	}

	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	cand, err := fetchLyricsFn(ctx, track, *debug, *deepSearch)
	if err != nil {
		if err == errNotFound {
			debugLog(*debug, "fetch_failure", map[string]any{"provider": "pipeline", "reason": "not found"})
			if !*dryRun {
				if recErr := recordSearchOutcomeFn(track, "not_found", "", "", "", nil); recErr != nil {
					debugLog(*debug, "index_error", recErr)
				}
				if failErr := recordFailureEventFn(FailureEvent{
					Artist:     track.Artist,
					Title:      track.Title,
					Provider:   "pipeline",
					Category:   "letra inexistente",
					Reason:     "no provider returned synced lyrics",
					Status:     "not_found",
					TrackID:    track.TrackID,
					DurationMs: track.DurationMs,
					Source:     "pipeline",
				}); failErr != nil {
					debugLog(*debug, "failure_log_error", failErr)
				}
			}
			fmt.Fprintln(os.Stderr, "result: not found")
			return 0
		}
		if err == errTimeout {
			debugLog(*debug, "lrclib", "timeout, not caching negative result")
			debugLog(*debug, "fetch_failure", map[string]any{"provider": "pipeline", "reason": "timeout"})
			if !*dryRun {
				if recErr := recordSearchOutcomeFn(track, "timeout", "", "", "", nil); recErr != nil {
					debugLog(*debug, "index_error", recErr)
				}
				if failErr := recordFailureEventFn(FailureEvent{
					Artist:     track.Artist,
					Title:      track.Title,
					Provider:   "pipeline",
					Category:   "timeout",
					Reason:     "global timeout reached",
					Status:     "timeout",
					TrackID:    track.TrackID,
					DurationMs: track.DurationMs,
					Source:     "pipeline",
				}); failErr != nil {
					debugLog(*debug, "failure_log_error", failErr)
				}
			}
			fmt.Fprintln(os.Stderr, "result: not found")
			return 0
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if cand == nil || cand.Text == "" {
		if !*dryRun {
			if recErr := recordSearchOutcomeFn(track, "not_found", "", "", "", nil); recErr != nil {
				debugLog(*debug, "index_error", recErr)
			}
		}
		fmt.Fprintln(os.Stderr, "result: not found")
		return 0
	}
	debugLog(*debug, "fetch_success", map[string]any{"provider": cand.Provider, "source_id": cand.SourceID})
	if *dryRun {
		debugLog(*debug, "dry_run_winner", map[string]any{
			"provider":  cand.Provider,
			"source_id": cand.SourceID,
			"status":    "found",
		})
		fmt.Fprintf(os.Stderr, "dry-run winner: provider=%s source_id=%s\n", cand.Provider, cand.SourceID)
		fmt.Fprintln(os.Stderr, "dry-run: not saving .lrc")
		return 0
	}
	saved, err := saveLocalLyricsFn(track, cand.Text, cand.Provider, cand.SourceID)
	if err != nil {
		debugLog(*debug, "local_save_error", err)
		return 1
	}
	debugLog(*debug, "saved_files", saved)
	return 0
}

func resolveTrack(noSpotify bool, artist, title, album string, duration float64, trackID string) (Track, bool, error) {
	provided := artist != "" || title != "" || album != "" || duration != 0 || trackID != ""
	if provided {
		if artist == "" || title == "" {
			return Track{}, false, fmt.Errorf("artist and title are required together for test mode")
		}
		return Track{
			Artist:     artist,
			Title:      title,
			Album:      album,
			DurationMs: int(duration * 1000),
			TrackID:    trackID,
		}, true, nil
	}
	if noSpotify {
		return Track{}, false, fmt.Errorf("--no-spotify requires --artist and --title")
	}
	return playerctlTrack()
}

func playerctlTrack() (Track, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), playerctlTimeout)
	defer cancel()
	fmtStr := "{{artist}}|||{{title}}|||{{album}}|||{{mpris:length}}|||{{mpris:trackid}}"
	cmd := exec.CommandContext(ctx, "playerctl", "-p", "spotify", "metadata", "--format", fmtStr)
	out, err := cmd.Output()
	if err != nil {
		return Track{}, false, nil
	}
	parts := splitFields(string(out), "|||")
	if len(parts) != 5 {
		return Track{}, false, nil
	}
	track := Track{
		Artist:  parts[0],
		Title:   parts[1],
		Album:   parts[2],
		TrackID: parts[4],
	}
	if track.Artist == "" || track.Title == "" {
		return Track{}, false, nil
	}
	if parts[3] != "" {
		if v, parseErr := strconv.Atoi(strings.TrimSpace(parts[3])); parseErr == nil {
			track.DurationMs = v / 1000
		}
	}
	return track, true, nil
}

func splitFields(s, sep string) []string {
	return strings.Split(strings.TrimSpace(s), sep)
}

func debugLog(debug bool, label string, value any) {
	logEvent(label, value)
	if !debug {
		return
	}
	fmt.Fprintf(os.Stderr, "[lyrics:debug] %s: %v\n", label, value)
}
