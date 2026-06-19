package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

var (
	homeDir          = mustHomeDir()
	localDir         = filepath.Join(homeDir, ".local", "share", "lyrics")
	cacheDir         = filepath.Join(homeDir, ".cache", "lyrics-terminal")
	indexPath        = filepath.Join(cacheDir, "index.json")
	failureLogPath   = filepath.Join(cacheDir, "failures.jsonl")
	candidateLogPath = filepath.Join(cacheDir, "candidate_evaluations.jsonl")
	mapPaths         = []string{filepath.Join(homeDir, "Music", "lyrics", "lyrics_map.json"), filepath.Join(homeDir, ".config", "spicetify", "CustomApps", "lyrics-plus", "lyrics_map.json")}
)

const validationVersion = "v1"

var candidateEvaluationWriter = appendCandidateEvaluation

func mustHomeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		return "."
	}
	return dir
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func appendJSONLine(path string, value any) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func loadIndex() map[string]IndexEntry {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return map[string]IndexEntry{}
	}
	var index map[string]IndexEntry
	if err := json.Unmarshal(data, &index); err != nil || index == nil {
		return map[string]IndexEntry{}
	}
	return index
}

func saveIndex(index map[string]IndexEntry) error {
	if err := ensureDir(cacheDir); err != nil {
		return err
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	tmp := indexPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, indexPath)
}

func appendFailureEvent(event FailureEvent) error {
	event.CreatedAt = time.Now().Unix()
	return appendJSONLine(failureLogPath, event)
}

func loadFailureEvents() []FailureEvent {
	f, err := os.Open(failureLogPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	events := make([]FailureEvent, 0, 64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event FailureEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events
}

func recordFailureEvent(event FailureEvent) error {
	if event.Artist == "" && event.Title == "" && event.Provider == "" && event.Category == "" {
		return nil
	}
	return appendFailureEvent(event)
}

func appendCandidateEvaluation(event CandidateEvaluationEvent) error {
	event.CreatedAt = time.Now().Unix()
	return appendJSONLine(candidateLogPath, event)
}

func recordCandidateEvaluation(event CandidateEvaluationEvent) error {
	if event.Event == "" {
		event.Event = "candidate_evaluated"
	}
	if event.Event == "candidate_evaluated" && event.Provider == "" && event.TargetArtist == "" && event.TargetTitle == "" {
		return nil
	}
	return candidateEvaluationWriter(event)
}

func failureCategoryFromReason(reason string) string {
	switch {
	case reason == "":
		return "outros"
	case strings.Contains(reason, "timeout"):
		return "timeout"
	case strings.Contains(reason, "duration"):
		return "mismatch de duração"
	case strings.Contains(reason, "artist"):
		return "mismatch de artista"
	case strings.Contains(reason, "title"):
		return "mismatch de título"
	case strings.Contains(reason, "not synced"):
		return "resultado não sincronizado"
	case strings.Contains(reason, "cjk") || strings.Contains(reason, "empty") || strings.Contains(reason, "no_timestamp") || strings.Contains(reason, "no_usable_lyric_lines"):
		return "cache inválido"
	case strings.Contains(reason, "status") || strings.Contains(reason, "command not found") || strings.Contains(reason, "curl") || strings.Contains(reason, "network") || strings.Contains(reason, "connect"):
		return "provider indisponível"
	case strings.Contains(reason, "not found"):
		return "letra inexistente"
	default:
		return "outros"
	}
}

func classifyFailureEvent(event FailureEvent) string {
	if event.Category != "" {
		return event.Category
	}
	if event.Status == "timeout" {
		return "timeout"
	}
	if event.Reason != "" {
		category := failureCategoryFromReason(strings.ToLower(event.Reason))
		if category != "outros" {
			return category
		}
	}
	if event.Status == "invalid" {
		return "cache inválido"
	}
	if event.Status == "not_found" {
		return "letra inexistente"
	}
	if event.Provider == "pipeline" && event.Status == "not_found" {
		return "letra inexistente"
	}
	return "outros"
}

func upsertIndexEntry(track Track, patch IndexEntry) error {
	index := loadIndex()
	now := time.Now().Unix()
	key := trackKey(track)
	entry := index[key]
	if entry.CreatedAt == 0 {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now
	if patch.Artist != "" {
		entry.Artist = patch.Artist
	}
	if patch.Title != "" {
		entry.Title = patch.Title
	}
	if patch.Provider != "" {
		entry.Provider = patch.Provider
	}
	if patch.SourceID != "" {
		entry.SourceID = patch.SourceID
	}
	if patch.DurationMs != 0 {
		entry.DurationMs = patch.DurationMs
	}
	if patch.Status != "" {
		entry.Status = patch.Status
	}
	if patch.RejectionReason != "" {
		entry.RejectionReason = patch.RejectionReason
	}
	if patch.Files != nil {
		entry.Files = patch.Files
	}
	if patch.CandidateArtist != "" {
		entry.CandidateArtist = patch.CandidateArtist
	}
	if patch.CandidateTitle != "" {
		entry.CandidateTitle = patch.CandidateTitle
	}
	if patch.CandidateAlbum != "" {
		entry.CandidateAlbum = patch.CandidateAlbum
	}
	if patch.CandidateDurationMs != 0 {
		entry.CandidateDurationMs = patch.CandidateDurationMs
	}
	if patch.Score != 0 {
		entry.Score = patch.Score
	}
	if patch.CandidateMetadataAvailable {
		entry.CandidateMetadataAvailable = patch.CandidateMetadataAvailable
	}
	if patch.ProvenanceStatus != "" {
		entry.ProvenanceStatus = patch.ProvenanceStatus
	}
	if patch.ValidationVersion != "" {
		entry.ValidationVersion = patch.ValidationVersion
	}
	if patch.AcceptedAt != 0 {
		entry.AcceptedAt = patch.AcceptedAt
	}
	index[key] = entry
	return saveIndex(index)
}

func atomicWriteFile(path string, data []byte) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func localLyricsPaths(track Track) []string {
	exact := filepath.Join(localDir, exactBaseName(track)+".lrc")
	normalized := filepath.Join(localDir, normalizedBaseName(track)+".lrc")
	if exact == normalized {
		return []string{exact}
	}
	return []string{exact, normalized}
}

func countCJKChars(text string) int {
	count := 0
	for _, r := range text {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
			count++
		}
	}
	return count
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func trackPrefersPortuguese(track Track) bool {
	text := normalizeText(track.Artist + " " + track.Title)
	hints := []string{
		"djavan",
		"chico buarque",
		"caetano veloso",
		"gilberto gil",
		"skank",
		"lagum",
		"liniker",
		"djonga",
		"baco exu do blues",
		"vanessa da mata",
		"jorge vercillo",
		"marisa monte",
		"gal costa",
		"tim maia",
	}
	for _, hint := range hints {
		if strings.Contains(text, hint) {
			return true
		}
	}
	if strings.ContainsAny(track.Artist+track.Title, "ãõáéíóúçâêôà") {
		return true
	}
	ptMarkers := []string{" ao vivo", " feat ", " participacao", " participação", " pra ", " nao ", " não ", " teu ", " tua ", " seu ", " sua "}
	for _, marker := range ptMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func trackLooksLatinScript(track Track) bool {
	label := track.Artist + track.Title
	if countCJKChars(label) > 0 {
		return false
	}
	for _, r := range label {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func localLRCInvalidReason(track Track, text string) string {
	if strings.TrimSpace(text) == "" {
		return "empty"
	}
	lines, _ := parseLRCText(text)
	if len(lines) == 0 {
		return "no_timestamp"
	}
	useful := make([]string, 0, len(lines))
	for _, line := range lines {
		if isUsefulLyricLine(line[1]) {
			useful = append(useful, strings.TrimSpace(line[1]))
		}
	}
	if len(useful) == 0 {
		return "no_usable_lyric_lines"
	}
	combined := strings.Join(useful, " ")
	cjkChars := countCJKChars(combined)
	alphaChars := 0
	for _, r := range combined {
		if unicode.IsLetter(r) {
			alphaChars++
		}
	}
	if cjkChars >= 4 && cjkChars >= maxInt(4, alphaChars*15/100) && (trackPrefersPortuguese(track) || trackLooksLatinScript(track)) {
		return "cjk_suspect"
	}
	return ""
}

func quarantineBadLRC(path string) string {
	badDir := filepath.Join(localDir, "bad")
	if err := ensureDir(badDir); err != nil {
		return ""
	}
	target := filepath.Join(badDir, filepath.Base(path)+"."+fmt.Sprintf("%d", time.Now().Unix())+".bad")
	if err := os.Rename(path, target); err != nil {
		return ""
	}
	return target
}

func inspectLocalLRC(track Track) (string, string, bool) {
	for _, path := range localLyricsPaths(track) {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			debugLog(false, "cache_invalid", "unreadable")
			if target := quarantineBadLRC(path); target != "" {
				debugLog(false, "quarantine", target)
			}
			continue
		}
		if reason := localLRCInvalidReason(track, string(data)); reason != "" {
			debugLog(false, "cache_invalid", reason)
			if target := quarantineBadLRC(path); target != "" {
				debugLog(false, "quarantine", target)
			}
			continue
		}
		debugLog(false, "cache_hit", path)
		return path, string(data), true
	}
	targetKey := trackKey(track)
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return "", "", false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".lrc" {
			continue
		}
		stem := strings.TrimSuffix(name, ".lrc")
		if normalizeText(stem) == targetKey {
			path := filepath.Join(localDir, name)
			data, err := os.ReadFile(path)
			if err != nil {
				debugLog(false, "cache_invalid", "unreadable")
				if target := quarantineBadLRC(path); target != "" {
					debugLog(false, "quarantine", target)
				}
				continue
			}
			if reason := localLRCInvalidReason(track, string(data)); reason != "" {
				debugLog(false, "cache_invalid", reason)
				if target := quarantineBadLRC(path); target != "" {
					debugLog(false, "quarantine", target)
				}
				continue
			}
			debugLog(false, "cache_hit", path)
			return path, string(data), true
		}
	}
	debugLog(false, "cache_miss", trackKey(track))
	return "", "", false
}

func findLocalLRC(track Track) (string, string, bool) {
	return inspectLocalLRC(track)
}

func saveLocalLyrics(track Track, cand Candidate) ([]string, error) {
	if err := ensureDir(localDir); err != nil {
		return nil, err
	}
	exact := filepath.Join(localDir, exactBaseName(track)+".lrc")
	normalized := filepath.Join(localDir, normalizedBaseName(track)+".lrc")
	saved := []string{exact}
	if err := atomicWriteFile(exact, []byte(cand.Text)); err != nil {
		return nil, err
	}
	if normalized != exact {
		if err := atomicWriteFile(normalized, []byte(cand.Text)); err != nil {
			return nil, err
		}
		saved = append(saved, normalized)
	}
	now := time.Now().Unix()
	validation := cand.ValidationVersion
	if validation == "" {
		validation = validationVersion
	}
	provenance := provenanceStatusForCandidate(cand, now)
	if err := upsertIndexEntry(track, IndexEntry{
		Artist:                     track.Artist,
		Title:                      track.Title,
		Provider:                   cand.Provider,
		SourceID:                   cand.SourceID,
		DurationMs:                 track.DurationMs,
		Status:                     "found",
		Files:                      saved,
		CandidateArtist:            cand.Artist,
		CandidateTitle:             cand.Title,
		CandidateAlbum:             cand.Album,
		CandidateDurationMs:        cand.DurationMs,
		Score:                      cand.Score,
		CandidateMetadataAvailable: cand.MetadataAvailable,
		ProvenanceStatus:           provenance,
		ValidationVersion:          validation,
		AcceptedAt:                 now,
	}); err != nil {
		return nil, err
	}
	return saved, nil
}

func recordSearchOutcome(track Track, status, provider, sourceID, reason string, files []string) error {
	index := loadIndex()
	key := trackKey(track)
	now := time.Now().Unix()
	entry := index[key]
	if entry.CreatedAt == 0 {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now
	entry.Artist = track.Artist
	entry.Title = track.Title
	entry.DurationMs = track.DurationMs
	entry.Status = status
	entry.RejectionReason = reason
	if status == "found" {
		if provider != "" && provider != "local-cache" {
			entry.Provider = provider
		} else if entry.Provider == "" && provider == "local-cache" {
			entry.Provider = provider
		}
		if sourceID != "" && provider != "local-cache" {
			entry.SourceID = sourceID
		} else if entry.SourceID == "" && provider == "local-cache" {
			entry.SourceID = sourceID
		}
		if files != nil {
			entry.Files = files
		}
	} else {
		entry.Provider = ""
		entry.SourceID = ""
		entry.Files = nil
	}
	index[key] = entry
	return saveIndex(index)
}

func loadLyricsMap() map[string]any {
	for _, p := range mapPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var out map[string]any
		if json.Unmarshal(data, &out) == nil {
			return out
		}
	}
	return map[string]any{}
}

func debugPaths() string {
	return fmt.Sprintf("local=%s cache=%s index=%s", localDir, cacheDir, indexPath)
}

func sortedIndexEntries(index map[string]IndexEntry) []IndexEntry {
	out := make([]IndexEntry, 0, len(index))
	for _, entry := range index {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt == out[j].CreatedAt {
			return out[i].UpdatedAt > out[j].UpdatedAt
		}
		return out[i].CreatedAt > out[j].CreatedAt
	})
	return out
}

func provenanceStatusForCandidate(cand Candidate, acceptedAt int64) string {
	if !cand.MetadataAvailable {
		return "partial"
	}
	if cand.Provider != "" && cand.Artist != "" && cand.Title != "" && cand.ValidationVersion != "" && acceptedAt != 0 {
		return "complete"
	}
	return "partial"
}

func indexEntryProvenanceStatus(entry IndexEntry) string {
	if entry.ProvenanceStatus == "complete" {
		if entry.Provider != "" && entry.CandidateMetadataAvailable && entry.CandidateArtist != "" && entry.CandidateTitle != "" && entry.ValidationVersion != "" && entry.AcceptedAt != 0 {
			return "complete"
		}
		return "partial"
	}
	if entry.ProvenanceStatus == "partial" {
		return "partial"
	}
	if entry.ProvenanceStatus == "missing" {
		return "missing"
	}
	if entry.ValidationVersion == "" && entry.AcceptedAt == 0 && entry.CandidateArtist == "" && entry.CandidateTitle == "" && entry.CandidateAlbum == "" && entry.CandidateDurationMs == 0 && entry.Score == 0 && !entry.CandidateMetadataAvailable {
		return "missing"
	}
	if entry.Provider != "" && entry.CandidateMetadataAvailable && entry.CandidateArtist != "" && entry.CandidateTitle != "" && entry.ValidationVersion != "" && entry.AcceptedAt != 0 {
		return "complete"
	}
	if entry.ValidationVersion == "" && entry.AcceptedAt == 0 && !entry.CandidateMetadataAvailable {
		return "missing"
	}
	return "partial"
}

func indexEntryHasProvenance(entry IndexEntry) bool {
	return indexEntryProvenanceStatus(entry) != "missing"
}

func intPtr(v int) *int {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func recordCacheProvenanceMissing(track Track) error {
	index := loadIndex()
	entry, ok := index[trackKey(track)]
	if ok && indexEntryProvenanceStatus(entry) != "missing" {
		return nil
	}
	reasons := []string{"legacy cache entry lacks provenance"}
	sourceID := ""
	if ok {
		sourceID = entry.SourceID
	}
	return recordCandidateEvaluation(CandidateEvaluationEvent{
		Event:            "cache_provenance_missing",
		Provider:         "local-cache",
		SourceID:         sourceID,
		TargetTrackID:    track.TrackID,
		TargetArtist:     track.Artist,
		TargetTitle:      track.Title,
		TargetAlbum:      track.Album,
		TargetDurationMs: intPtr(track.DurationMs),
		EvaluationStage:  "cache",
		Decision:         "cache_reused",
		CacheReused:      true,
		ProvenanceStatus: "missing",
		RejectionReasons: reasons,
	})
}
