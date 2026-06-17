package main

import "testing"

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
