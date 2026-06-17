package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func localLyricsPaths(track Track) []string {
	exact := filepath.Join(localDir, exactBaseName(track)+".lrc")
	normalized := filepath.Join(localDir, normalizedBaseName(track)+".lrc")
	if exact == normalized {
		return []string{exact}
	}
	return []string{exact, normalized}
}

func findLocalLRC(track Track) (string, string, bool) {
	for _, path := range localLyricsPaths(track) {
		if _, err := os.Stat(path); err == nil {
			if data, err := os.ReadFile(path); err == nil {
				return path, string(data), true
			}
			return path, "", true
		}
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
			data, _ := os.ReadFile(path)
			return path, string(data), true
		}
	}
	return "", "", false
}

func saveLocalLyrics(track Track, lrcText string, provider string, sourceID string) ([]string, error) {
	if err := ensureDir(localDir); err != nil {
		return nil, err
	}
	exact := filepath.Join(localDir, exactBaseName(track)+".lrc")
	normalized := filepath.Join(localDir, normalizedBaseName(track)+".lrc")
	saved := []string{exact}
	if err := os.WriteFile(exact, []byte(lrcText), 0o644); err != nil {
		return nil, err
	}
	if normalized != exact {
		if err := os.WriteFile(normalized, []byte(lrcText), 0o644); err != nil {
			return nil, err
		}
		saved = append(saved, normalized)
	}
	index := loadIndex()
	index[trackKey(track)] = IndexEntry{
		Artist:    track.Artist,
		Title:     track.Title,
		Provider:  provider,
		SourceID:  sourceID,
		UpdatedAt: time.Now().Unix(),
		Files:     saved,
	}
	if err := saveIndex(index); err != nil {
		return nil, err
	}
	return saved, nil
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
