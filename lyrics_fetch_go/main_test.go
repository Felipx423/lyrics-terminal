package main

import (
	"context"
	"testing"
)

func TestBuildLRCLibQuery(t *testing.T) {
	got := buildQuery("Samurai (feat. Stevie Wonder)", "Djavan")
	want := "Samurai Djavan"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidateLRCLIBCandidate(t *testing.T) {
	track := Track{Artist: "Djavan", Title: "Samurai (feat. Stevie Wonder)", Album: "Luz", DurationMs: 288000}
	cand := lrclibCandidate{
		TrackName:    "Samurai",
		ArtistName:   "Luz - Djavan feat. Stevie Wonder",
		AlbumName:    "Luz",
		Duration:     288,
		SyncedLyrics: "[00:01.00]a",
	}
	ok, reason, details := validateLRCLIBCandidate(cand, track)
	if !ok {
		t.Fatalf("expected accept, got reason=%s details=%v", reason, details)
	}
}

func TestResolveTrackTestMode(t *testing.T) {
	track, ok, err := resolveTrack(true, "Djavan", "Samurai", "Luz", 288, "spotify:track:123")
	if err != nil || !ok {
		t.Fatalf("unexpected err=%v ok=%v", err, ok)
	}
	if track.Artist != "Djavan" || track.Title != "Samurai" || track.DurationMs != 288000 {
		t.Fatalf("unexpected track %+v", track)
	}
}

func TestRunDryRunDoesNotSaveOrRecord(t *testing.T) {
	origFetch := fetchLyricsFn
	origSave := saveLocalLyricsFn
	origFind := findLocalLRCFn
	origRecord := recordSearchOutcomeFn
	defer func() {
		fetchLyricsFn = origFetch
		saveLocalLyricsFn = origSave
		findLocalLRCFn = origFind
		recordSearchOutcomeFn = origRecord
	}()

	calledFind := false
	calledSave := false
	calledRecord := false
	fetchLyricsFn = func(ctx context.Context, track Track, debug bool, deepSearch bool) (*Candidate, error) {
		return &Candidate{Text: "[00:00.000]line", Provider: "lrclib", SourceID: "123"}, nil
	}
	findLocalLRCFn = func(track Track) (string, string, bool) {
		calledFind = true
		return "", "", false
	}
	saveLocalLyricsFn = func(track Track, lrcText string, provider string, sourceID string) ([]string, error) {
		calledSave = true
		return nil, nil
	}
	recordSearchOutcomeFn = func(track Track, status, provider, sourceID, reason string, files []string) error {
		calledRecord = true
		return nil
	}

	exitCode := run([]string{"--dry-run", "--no-spotify", "--artist", "Artist", "--title", "Song"})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code %d", exitCode)
	}
	if calledFind {
		t.Fatalf("dry-run should not inspect local cache")
	}
	if calledSave {
		t.Fatalf("dry-run should not save lyrics")
	}
	if calledRecord {
		t.Fatalf("dry-run should not record search outcome")
	}
}
