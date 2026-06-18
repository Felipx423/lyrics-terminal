package main

import (
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
	homeDir   = mustHomeDir()
	localDir  = filepath.Join(homeDir, ".local", "share", "lyrics")
	cacheDir  = filepath.Join(homeDir, ".cache", "lyrics-terminal")
	indexPath = filepath.Join(cacheDir, "index.json")
	mapPaths  = []string{filepath.Join(homeDir, "Music", "lyrics", "lyrics_map.json"), filepath.Join(homeDir, ".config", "spicetify", "CustomApps", "lyrics-plus", "lyrics_map.json")}
)

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
			quarantineBadLRC(path)
			continue
		}
		if reason := localLRCInvalidReason(track, string(data)); reason != "" {
			quarantineBadLRC(path)
			continue
		}
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
				quarantineBadLRC(path)
				continue
			}
			if reason := localLRCInvalidReason(track, string(data)); reason != "" {
				quarantineBadLRC(path)
				continue
			}
			return path, string(data), true
		}
	}
	return "", "", false
}

func findLocalLRC(track Track) (string, string, bool) {
	return inspectLocalLRC(track)
}

func saveLocalLyrics(track Track, lrcText string, provider string, sourceID string) ([]string, error) {
	if err := ensureDir(localDir); err != nil {
		return nil, err
	}
	exact := filepath.Join(localDir, exactBaseName(track)+".lrc")
	normalized := filepath.Join(localDir, normalizedBaseName(track)+".lrc")
	saved := []string{exact}
	if err := atomicWriteFile(exact, []byte(lrcText)); err != nil {
		return nil, err
	}
	if normalized != exact {
		if err := atomicWriteFile(normalized, []byte(lrcText)); err != nil {
			return nil, err
		}
		saved = append(saved, normalized)
	}
	if err := upsertIndexEntry(track, IndexEntry{
		Artist:     track.Artist,
		Title:      track.Title,
		Provider:   provider,
		SourceID:   sourceID,
		DurationMs: track.DurationMs,
		Status:     "found",
		Files:      saved,
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
		entry.Provider = provider
		entry.SourceID = sourceID
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
