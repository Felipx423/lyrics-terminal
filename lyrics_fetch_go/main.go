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

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("lyrics-fetch-go", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	debug := fs.Bool("debug", false, "")
	clearCache := fs.Bool("clear-cache", false, "")
	noSpotify := fs.Bool("no-spotify", false, "")
	ignoreLocalCache := fs.Bool("ignore-local-cache", false, "")
	artist := fs.String("artist", "", "")
	title := fs.String("title", "", "")
	album := fs.String("album", "", "")
	duration := fs.Float64("duration", 0, "")
	trackID := fs.String("track-id", "", "")
	deepSearch := fs.Bool("deep-search", false, "")
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

	if !*ignoreLocalCache {
		if existingPath, _, ok := findLocalLRC(track); ok {
			debugLog(*debug, "local_exists", existingPath)
			return 0
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	cand, err := fetchLyrics(ctx, track, *debug, *deepSearch)
	if err != nil {
		if err == errNotFound {
			fmt.Fprintln(os.Stderr, "result: not found")
			return 0
		}
		if err == errTimeout {
			debugLog(*debug, "lrclib", "timeout, not caching negative result")
			fmt.Fprintln(os.Stderr, "result: not found")
			return 0
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if cand == nil || cand.Text == "" {
		fmt.Fprintln(os.Stderr, "result: not found")
		return 0
	}
	saved, err := saveLocalLyrics(track, cand.Text, cand.Provider, cand.SourceID)
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
	if !debug {
		return
	}
	fmt.Fprintf(os.Stderr, "[lyrics:debug] %s: %v\n", label, value)
}
