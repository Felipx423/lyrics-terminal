package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func withDefaultTransport(t *testing.T, rt http.RoundTripper) {
	t.Helper()

	original := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() {
		http.DefaultTransport = original
	})
}

func TestScoreNetEaseCandidatePreservesTargetContainsCandidateMatch(t *testing.T) {
	track := Track{Artist: "Ryxn Pablo", Title: "Ainda", DurationMs: 165000}
	song := neteaseSong{
		ID:   1,
		Name: "Ain",
		DT:   165000,
		AR: []struct {
			Name string `json:"name"`
		}{{Name: "Ryxn Pablo"}},
	}

	score, details := scoreNetEaseCandidate(song, track)

	if score != 5 {
		t.Fatalf("expected legacy score 5, got %d details=%v", score, details)
	}
	if got := details["title_match_type"]; got != "target_contains_candidate" {
		t.Fatalf("expected target_contains_candidate, got %v", got)
	}
	if got := details["artist_match_type"]; got != "artist" {
		t.Fatalf("expected artist match by artist field, got %v", got)
	}
}

func TestEmitCandidateEvaluationHidesMetadataWhenUnavailable(t *testing.T) {
	setupCacheTestEnv(t)

	err := emitCandidateEvaluation(true, CandidateEvaluationEvent{
		Event:                      "candidate_evaluated",
		Provider:                   "syncedlyrics",
		TargetTrackID:              "spotify:track:1",
		TargetArtist:               "Artist",
		TargetTitle:                "Song",
		TargetAlbum:                "Album",
		TargetDurationMs:           intPtr(180000),
		EvaluationStage:            "validation",
		Decision:                   "rejected",
		CandidateMetadataAvailable: boolPtr(false),
		CandidateArtist:            "Should be hidden",
		CandidateTitle:             "Should be hidden",
		CandidateAlbum:             "Should be hidden",
		TitleMatchType:             "exact",
		ArtistMatchType:            "exact",
		DurationDeltaMs:            intPtr(0),
		RejectionReasons:           []string{"provider metadata unavailable"},
	})
	if err != nil {
		t.Fatalf("emitCandidateEvaluation failed: %v", err)
	}

	events := readCandidateEvents(t)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	event := events[0]
	if event.CandidateMetadataAvailable == nil || *event.CandidateMetadataAvailable {
		t.Fatalf("candidate metadata availability must be explicit false: %+v", event)
	}
	if event.CandidateArtist != "" || event.CandidateTitle != "" || event.CandidateAlbum != "" {
		t.Fatalf("candidate metadata must be hidden when unavailable: %+v", event)
	}
	if event.CandidateDurationMs != nil || event.DurationDeltaMs != nil {
		t.Fatalf("candidate duration and delta must be omitted when metadata is unavailable: %+v", event)
	}
	if event.TitleMatchType != "unverified" || event.ArtistMatchType != "unverified" || event.ProvenanceStatus != "partial" {
		t.Fatalf("unexpected metadata-unavailable semantics: %+v", event)
	}
	raw := string(mustReadFile(t, candidateLogPath))
	if !strings.Contains(raw, `"candidate_metadata_available":false`) {
		t.Fatalf("expected explicit false candidate metadata flag:\n%s", raw)
	}
	if strings.Contains(raw, `"candidate_duration_ms"`) || strings.Contains(raw, `"duration_delta_ms"`) {
		t.Fatalf("candidate duration fields must not serialize when metadata is unavailable:\n%s", raw)
	}
}

func TestSaveLocalLyricsMarksCompleteProvenanceWhenOptionalMetadataMissing(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Aimar", Title: "LINGERIE", Album: "Single", DurationMs: 165000, TrackID: "spotify:track:partial"}
	cand := Candidate{
		Text:              "[00:01.00]linha 1\n[00:02.00]linha 2",
		Provider:          "lrclib",
		SourceID:          "999",
		Artist:            "Aimar",
		Title:             "LINGERIE",
		MetadataAvailable: true,
		ProvenanceStatus:  "complete",
		ValidationVersion: validationVersion,
	}

	if _, err := saveLocalLyrics(track, cand); err != nil {
		t.Fatalf("saveLocalLyrics failed: %v", err)
	}

	index := loadIndex()
	entry, ok := index[trackKey(track)]
	if !ok {
		t.Fatalf("missing index entry")
	}
	if entry.ProvenanceStatus != "complete" {
		t.Fatalf("expected complete provenance when required fields exist, got %+v", entry)
	}
	if !entry.CandidateMetadataAvailable {
		t.Fatalf("expected candidate metadata flag to persist: %+v", entry)
	}
	if entry.CandidateAlbum != "" || entry.CandidateDurationMs != 0 {
		t.Fatalf("optional metadata should still be empty when provider does not supply it: %+v", entry)
	}
}

func TestSaveLocalLyricsRejectsManualCompleteWithoutCentralFields(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Aimar", Title: "LINGERIE", Album: "Single", DurationMs: 165000, TrackID: "spotify:track:partial"}
	cand := Candidate{
		Text:              "[00:01.00]linha 1\n[00:02.00]linha 2",
		Provider:          "lrclib",
		SourceID:          "999",
		MetadataAvailable: true,
		ProvenanceStatus:  "complete",
		ValidationVersion: validationVersion,
	}

	if _, err := saveLocalLyrics(track, cand); err != nil {
		t.Fatalf("saveLocalLyrics failed: %v", err)
	}

	entry := loadIndex()[trackKey(track)]
	if entry.ProvenanceStatus != "partial" {
		t.Fatalf("manual complete without central fields must fall back to partial: %+v", entry)
	}
}

func TestCacheReuseWithoutIndexLogsMissingProvenanceWithoutAcceptedFalse(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Legacy Artist", Title: "Legacy Song", Album: "Legacy Album", DurationMs: 120000, TrackID: "spotify:track:legacy"}
	lrcPath := filepath.Join(localDir, exactBaseName(track)+".lrc")
	if err := os.MkdirAll(filepath.Dir(lrcPath), 0o755); err != nil {
		t.Fatalf("mkdir local dir: %v", err)
	}
	if err := os.WriteFile(lrcPath, []byte("[00:01.00]linha\n[00:02.00]mais"), 0o644); err != nil {
		t.Fatalf("write local lrc: %v", err)
	}

	moduleRun := run([]string{"--no-spotify", "--artist", track.Artist, "--title", track.Title})
	if moduleRun != 0 {
		t.Fatalf("unexpected exit code %d", moduleRun)
	}

	events := readCandidateEvents(t)
	if len(events) != 1 {
		t.Fatalf("expected 1 candidate event, got %+v", events)
	}
	event := events[0]
	if event.Event != "cache_provenance_missing" || !event.CacheReused || event.Decision != "cache_reused" || event.ProvenanceStatus != "missing" {
		t.Fatalf("unexpected provenance-missing event: %+v", event)
	}
	raw := string(mustReadFile(t, candidateLogPath))
	if strings.Contains(raw, `"accepted":false`) {
		t.Fatalf("cache provenance missing must not serialize accepted=false:\n%s", raw)
	}
}

func TestFetchNetEaseSearchLogsRankedOutAndNotAttemptedCandidates(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Ryxn Pablo", Title: "Ainda", Album: "Ainda", DurationMs: 165000, TrackID: "spotify:track:netease"}
	searchResponse := `{
		"result": {
			"songs": [
				{"id": 1, "name": "Ainda", "dt": 165000, "ar": [{"name": "Ryxn Pablo"}], "al": {"name": "Ainda"}},
				{"id": 2, "name": "Ainda", "dt": 165000, "ar": [{"name": "Ryxn Pablo"}], "al": {"name": "Ainda"}},
				{"id": 3, "name": "Ainda", "dt": 165000, "ar": [{"name": "Ryxn Pablo"}], "al": {"name": "Ainda"}},
				{"id": 4, "name": "Ainda", "dt": 165000, "ar": [{"name": "Outro"}], "al": {"name": "Ainda"}},
				{"id": 5, "name": "Outro", "dt": 1000, "ar": [{"name": "Alguém"}], "al": {"name": "Outro"}}
			]
		}
	}`
	emptyLyricResponse := `{
		"lrc": {"lyric": ""},
		"songs": [{"name": "Ainda", "dt": 165000, "ar": [{"name": "Ryxn Pablo"}]}]
	}`
	withDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Path, "/api/search/get"):
			return jsonResponse(200, searchResponse), nil
		case strings.Contains(req.URL.Path, "/api/song/lyric"):
			return jsonResponse(200, emptyLyricResponse), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s", req.URL.String())
		}
	}))

	cand, err := fetchNetEaseSearch(context.Background(), track, false)
	if !errors.Is(err, errNotFound) || cand != nil {
		t.Fatalf("expected no result, got cand=%+v err=%v", cand, err)
	}

	events := readCandidateEvents(t)
	if !containsCandidateDecision(events, "ranked_out") {
		t.Fatalf("expected ranked_out event, got %+v", events)
	}
	if !containsCandidateDecision(events, "not_attempted") {
		t.Fatalf("expected not_attempted event, got %+v", events)
	}
	if containsAcceptedFalseForStage(events, "ranking") {
		t.Fatalf("ranking events must not serialize accepted=false: %+v", events)
	}
	if !hasDecisionForSource(events, "ranked_out", "4") {
		t.Fatalf("expected ranked_out for source_id 4, got %+v", events)
	}
	if !hasDecisionForSource(events, "not_attempted", "5") {
		t.Fatalf("expected not_attempted for source_id 5, got %+v", events)
	}
}

func TestFetchNetEaseSearchLogsProviderErrorWhenLyricFetchFails(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Ryxn Pablo", Title: "Ainda", Album: "Ainda", DurationMs: 165000, TrackID: "spotify:track:netease"}
	searchResponse := `{
		"result": {
			"songs": [
				{"id": 11, "name": "Ainda", "dt": 165000, "ar": [{"name": "Ryxn Pablo"}], "al": {"name": "Ainda"}}
			]
		}
	}`
	withDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Path, "/api/search/get"):
			return jsonResponse(200, searchResponse), nil
		case strings.Contains(req.URL.Path, "/api/song/lyric"):
			return jsonResponse(500, `{"error":"boom"}`), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s", req.URL.String())
		}
	}))

	cand, err := fetchNetEaseSearch(context.Background(), track, false)
	if err == nil {
		t.Fatalf("expected provider error, got cand=%+v", cand)
	}

	events := readCandidateEvents(t)
	if !containsCandidateDecision(events, "provider_error") {
		t.Fatalf("expected provider_error event, got %+v", events)
	}
	if !hasDecisionForSource(events, "provider_error", "11") {
		t.Fatalf("expected provider_error for source_id 11, got %+v", events)
	}
}

func TestFetchNetEaseSearchAndSaveLocalLyricsPersistProvenance(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Ryxn Pablo", Title: "Ainda", Album: "Ainda", DurationMs: 165000, TrackID: "spotify:track:38YZseF2ALmg58eVQ9r2mZ"}
	searchResponse := `{
		"result": {
			"songs": [
				{"id": 38, "name": "Ainda", "dt": 165000, "ar": [{"name": "Ryxn Pablo"}], "al": {"name": "Ainda"}}
			]
		}
	}`
	lyricResponse := `{
		"lrc": {"lyric": "[00:01.00]Linha 1\n[00:02.00]Linha 2"},
		"songs": [{"name": "Ainda", "dt": 165000, "ar": [{"name": "Ryxn Pablo"}]}]
	}`
	withDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Path, "/api/search/get"):
			return jsonResponse(200, searchResponse), nil
		case strings.Contains(req.URL.Path, "/api/song/lyric"):
			return jsonResponse(200, lyricResponse), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s", req.URL.String())
		}
	}))

	cand, err := fetchNetEaseSearch(context.Background(), track, false)
	if err != nil {
		t.Fatalf("fetchNetEaseSearch failed: %v", err)
	}
	if cand == nil {
		t.Fatalf("expected candidate")
	}
	if cand.Provider != "netease-search" || cand.SourceID != "38" {
		t.Fatalf("unexpected candidate provenance: %+v", cand)
	}
	if !cand.MetadataAvailable || cand.ProvenanceStatus != "complete" {
		t.Fatalf("expected complete provenance on candidate: %+v", cand)
	}

	saved, err := saveLocalLyrics(track, *cand)
	if err != nil {
		t.Fatalf("saveLocalLyrics failed: %v", err)
	}
	if len(saved) == 0 {
		t.Fatalf("expected saved files")
	}

	index := loadIndex()
	entry, ok := index[trackKey(track)]
	if !ok {
		t.Fatalf("missing index entry")
	}
	if entry.Provider != "netease-search" || entry.SourceID != "38" {
		t.Fatalf("unexpected provenance in index: %+v", entry)
	}
	if entry.ProvenanceStatus != "complete" || !entry.CandidateMetadataAvailable {
		t.Fatalf("expected complete provenance in index: %+v", entry)
	}
	if entry.CandidateArtist != "Ryxn Pablo" || entry.CandidateTitle != "Ainda" {
		t.Fatalf("unexpected candidate provenance in index: %+v", entry)
	}

	events := readCandidateEvents(t)
	if !containsCandidateDecision(events, "accepted") {
		t.Fatalf("expected accepted candidate event, got %+v", events)
	}
	if strings.Contains(string(mustReadFile(t, candidateLogPath)), "Linha 1") {
		t.Fatalf("candidate diagnostics must not contain full lyric text")
	}
}

func TestEmitCandidateEvaluationLogsDiagnosticWriteFailureInDebug(t *testing.T) {
	setupCacheTestEnv(t)

	originalWriter := candidateEvaluationWriter
	candidateEvaluationWriter = func(event CandidateEvaluationEvent) error {
		return fmt.Errorf("boom")
	}
	t.Cleanup(func() {
		candidateEvaluationWriter = originalWriter
	})

	err := emitCandidateEvaluation(true, CandidateEvaluationEvent{
		Event:                      "candidate_evaluated",
		Provider:                   "lrclib",
		TargetTrackID:              "track-1",
		TargetArtist:               "Artist",
		TargetTitle:                "Song",
		CandidateMetadataAvailable: boolPtr(true),
		CandidateArtist:            "Artist",
		CandidateTitle:             "Song",
		RejectionReasons:           []string{"boom"},
	})
	if err == nil {
		t.Fatalf("expected diagnostic write error")
	}
	data := string(mustReadFile(t, filepath.Join(cacheDir, "lyrics.log")))
	if !strings.Contains(data, "candidate_diagnostic_error") {
		t.Fatalf("expected debug diagnostic error log, got:\n%s", data)
	}
}

func containsCandidateDecision(events []CandidateEvaluationEvent, decision string) bool {
	for _, event := range events {
		if event.Decision == decision {
			return true
		}
	}
	return false
}

func containsAcceptedFalseForStage(events []CandidateEvaluationEvent, stage string) bool {
	for _, event := range events {
		if event.EvaluationStage == stage && event.Accepted != nil && !*event.Accepted {
			return true
		}
	}
	return false
}

func hasDecisionForSource(events []CandidateEvaluationEvent, decision, sourceID string) bool {
	for _, event := range events {
		if event.Decision == decision && event.SourceID == sourceID {
			return true
		}
	}
	return false
}
