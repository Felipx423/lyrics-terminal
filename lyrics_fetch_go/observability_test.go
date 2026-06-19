package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readCandidateEvents(t *testing.T) []CandidateEvaluationEvent {
	t.Helper()

	data, err := os.ReadFile(candidateLogPath)
	if err != nil {
		t.Fatalf("read candidate log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	events := make([]CandidateEvaluationEvent, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event CandidateEvaluationEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("unmarshal candidate event: %v", err)
		}
		events = append(events, event)
	}
	return events
}

func TestRecordCandidateEvaluationRejectedWritesStructuredEvent(t *testing.T) {
	setupCacheTestEnv(t)

	err := recordCandidateEvaluation(CandidateEvaluationEvent{
		Event:               "candidate_evaluated",
		Provider:            "lrclib",
		SourceID:            "123",
		TargetTrackID:       "spotify:track:1",
		TargetArtist:        "Ryxn Pablo",
		TargetTitle:         "Ainda",
		TargetAlbum:         "Ainda",
		TargetDurationMs:    intPtr(165000),
		CandidateArtist:     "Outra pessoa",
		CandidateTitle:      "Ainda",
		CandidateAlbum:      "Ainda",
		CandidateDurationMs: intPtr(164000),
		TitleMatchType:      "exact",
		ArtistMatchType:     "none",
		DurationDeltaMs:     intPtr(1000),
		Accepted:            boolPtr(false),
		RejectionReasons:    []string{"artist mismatch"},
	})
	if err != nil {
		t.Fatalf("recordCandidateEvaluation failed: %v", err)
	}

	events := readCandidateEvents(t)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	event := events[0]
	if event.Event != "candidate_evaluated" || event.Provider != "lrclib" || event.Accepted == nil || *event.Accepted {
		t.Fatalf("unexpected event: %+v", event)
	}
	if event.TargetTrackID != "spotify:track:1" || event.TargetArtist != "Ryxn Pablo" || event.TargetTitle != "Ainda" {
		t.Fatalf("unexpected target metadata: %+v", event)
	}
	if event.CandidateArtist != "Outra pessoa" || event.CandidateTitle != "Ainda" || event.CandidateAlbum != "Ainda" {
		t.Fatalf("unexpected candidate metadata: %+v", event)
	}
	data, err := os.ReadFile(candidateLogPath)
	if err != nil {
		t.Fatalf("read candidate log: %v", err)
	}
	if strings.Contains(string(data), "secret lyric") {
		t.Fatalf("candidate diagnostics must not contain lyric content:\n%s", string(data))
	}
}

func TestSaveLocalLyricsPersistsProvenanceAndAcceptedEvent(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Aimar", Title: "LINGERIE", Album: "Single", DurationMs: 165000, TrackID: "spotify:track:38YZseF2ALmg58eVQ9r2mZ"}
	cand := Candidate{
		Text:              "[00:01.00]linha 1\n[00:02.00]linha 2",
		Provider:          "lrclib",
		SourceID:          "999",
		Artist:            "Aimar",
		Title:             "LINGERIE",
		Album:             "Single",
		DurationMs:        165000,
		Score:             4,
		MetadataAvailable: true,
		ProvenanceStatus:  "complete",
	}

	if err := recordCandidateEvaluation(CandidateEvaluationEvent{
		Event:                      "candidate_evaluated",
		Provider:                   "lrclib",
		SourceID:                   "999",
		TargetTrackID:              track.TrackID,
		TargetArtist:               track.Artist,
		TargetTitle:                track.Title,
		TargetAlbum:                track.Album,
		TargetDurationMs:           intPtr(track.DurationMs),
		CandidateArtist:            cand.Artist,
		CandidateTitle:             cand.Title,
		CandidateAlbum:             cand.Album,
		CandidateDurationMs:        intPtr(cand.DurationMs),
		TitleMatchType:             "exact",
		ArtistMatchType:            "exact",
		DurationDeltaMs:            intPtr(0),
		Score:                      intPtr(cand.Score),
		Accepted:                   boolPtr(true),
		CandidateMetadataAvailable: boolPtr(true),
		ProvenanceStatus:           "complete",
		ValidationVersion:          validationVersion,
	}); err != nil {
		t.Fatalf("recordCandidateEvaluation failed: %v", err)
	}

	saved, err := saveLocalLyrics(track, cand)
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
	if entry.Provider != "lrclib" || entry.SourceID != "999" {
		t.Fatalf("unexpected provenance: %+v", entry)
	}
	if entry.CandidateArtist != "Aimar" || entry.CandidateTitle != "LINGERIE" || entry.CandidateAlbum != "Single" {
		t.Fatalf("missing candidate provenance: %+v", entry)
	}
	if entry.CandidateDurationMs != 165000 || entry.Score != 4 || entry.ValidationVersion == "" || entry.AcceptedAt == 0 {
		t.Fatalf("missing provenance fields: %+v", entry)
	}

	events := readCandidateEvents(t)
	if len(events) != 1 {
		t.Fatalf("expected 1 accepted event, got %d", len(events))
	}
	if events[0].Accepted == nil || !*events[0].Accepted || events[0].Provider != "lrclib" || events[0].Score == nil || *events[0].Score != 4 {
		t.Fatalf("unexpected accepted event: %+v", events[0])
	}
	if strings.Contains(string(mustReadFile(t, candidateLogPath)), "linha 1") {
		t.Fatalf("accepted candidate diagnostics must not contain lyric content")
	}
}

func TestLegacyCacheHitLogsCacheProvenanceMissing(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Legacy Artist", Title: "Legacy Song", Album: "Legacy Album", DurationMs: 120000, TrackID: "spotify:track:legacy"}
	lrcPath := filepath.Join(localDir, exactBaseName(track)+".lrc")
	if err := os.MkdirAll(filepath.Dir(lrcPath), 0o755); err != nil {
		t.Fatalf("mkdir local dir: %v", err)
	}
	if err := os.WriteFile(lrcPath, []byte("[00:01.00]linha\n[00:02.00]mais"), 0o644); err != nil {
		t.Fatalf("write local lrc: %v", err)
	}
	if err := saveIndex(map[string]IndexEntry{
		trackKey(track): {
			Artist:   track.Artist,
			Title:    track.Title,
			Provider: "lrclib",
			SourceID: "123",
			Status:   "found",
			Files:    []string{lrcPath},
		},
	}); err != nil {
		t.Fatalf("save legacy index: %v", err)
	}

	exitCode := run([]string{"--no-spotify", "--artist", track.Artist, "--title", track.Title})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code %d", exitCode)
	}

	events := readCandidateEvents(t)
	if len(events) != 1 || events[0].Event != "cache_provenance_missing" {
		t.Fatalf("expected cache_provenance_missing event, got %+v", events)
	}
	if !events[0].CacheReused || events[0].Decision != "cache_reused" || events[0].ProvenanceStatus != "missing" {
		t.Fatalf("cache provenance missing event has wrong semantics: %+v", events[0])
	}
	raw := mustReadFile(t, candidateLogPath)
	if strings.Contains(string(raw), `"accepted":false`) {
		t.Fatalf("cache provenance missing event must not serialize accepted=false:\n%s", string(raw))
	}
	if _, err := os.Stat(lrcPath); err != nil {
		t.Fatalf("expected legacy cache file to remain: %v", err)
	}
}

func TestOldIndexFormatRemainsReadable(t *testing.T) {
	setupCacheTestEnv(t)

	raw := `{
		"key": {
			"artist": "Artist",
			"title": "Song",
			"provider": "lrclib",
			"source_id": "42",
			"created_at": 10,
			"updated_at": 20,
			"duration_ms": 123000,
			"status": "found",
			"files": ["/tmp/song.lrc"]
		}
	}`
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	if err := os.WriteFile(indexPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	index := loadIndex()
	entry, ok := index["key"]
	if !ok {
		t.Fatalf("expected legacy index entry")
	}
	if entry.Artist != "Artist" || entry.Title != "Song" || entry.Provider != "lrclib" || entry.SourceID != "42" {
		t.Fatalf("unexpected legacy entry: %+v", entry)
	}
	summary := summarizeIndex(index)
	if summary.ProviderCounts["lrclib"] != 1 || summary.StatusCounts["found"] != 1 {
		t.Fatalf("unexpected summary from legacy index: %+v", summary)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return data
}
