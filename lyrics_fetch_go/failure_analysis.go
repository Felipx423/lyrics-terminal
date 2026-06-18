package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type FailureAnalysisSummary struct {
	Events         []FailureEvent
	Counts         map[string]int
	ProviderCounts map[string]int
}

func printFailureAnalysis(w io.Writer, index map[string]IndexEntry, logEvents []FailureEvent, localRoot, cacheRoot string) error {
	summary := buildFailureAnalysis(index, logEvents, localRoot, cacheRoot)

	fmt.Fprintln(w, "lyrics-fetch-go failure analysis")
	fmt.Fprintf(w, "total_failures: %d\n", len(summary.Events))
	fmt.Fprintln(w, "category_counts:")
	for _, category := range sortedKeys(summary.Counts) {
		fmt.Fprintf(w, "  %s: %d\n", category, summary.Counts[category])
	}
	fmt.Fprintln(w, "provider_counts:")
	for _, provider := range sortedKeys(summary.ProviderCounts) {
		fmt.Fprintf(w, "  %s: %d\n", provider, summary.ProviderCounts[provider])
	}
	fmt.Fprintln(w, "failed_songs:")
	for _, event := range summary.Events {
		timestamp := "unknown"
		if event.CreatedAt > 0 {
			timestamp = time.Unix(event.CreatedAt, 0).Format(time.RFC3339)
		}
		fmt.Fprintf(w, "  - %s | %s - %s | provider=%s | category=%s | reason=%s | status=%s\n",
			timestamp, event.Artist, event.Title, event.Provider, event.Category, event.Reason, event.Status)
	}
	fmt.Fprintln(w, "resolution_hints:")
	fmt.Fprintln(w, "  musixmatch: letra inexistente, resultado não sincronizado")
	fmt.Fprintln(w, "  ranking_improvements: mismatch de duração, mismatch de artista, mismatch de título, cache inválido")
	fmt.Fprintln(w, "  network_problems: timeout, provider indisponível")
	fmt.Fprintln(w, "  no_practical_solution: outros e casos em que nenhum provider sincronizado existe")
	return nil
}

func buildFailureAnalysis(index map[string]IndexEntry, logEvents []FailureEvent, localRoot, cacheRoot string) FailureAnalysisSummary {
	events := make([]FailureEvent, 0, len(logEvents)+len(index)+8)
	events = append(events, logEvents...)

	for _, entry := range sortedIndexEntries(index) {
		status := inferIndexStatus(entry)
		if status == "" || status == "found" {
			continue
		}
		event := FailureEvent{
			Artist:     entry.Artist,
			Title:      entry.Title,
			Provider:   entry.Provider,
			Reason:     entry.RejectionReason,
			Status:     status,
			SourceID:   entry.SourceID,
			DurationMs: entry.DurationMs,
			CreatedAt:  entry.CreatedAt,
			Source:     "index",
		}
		if event.Provider == "" {
			event.Provider = "pipeline"
		}
		if event.Category == "" {
			event.Category = classifyFailureEvent(event)
		}
		events = append(events, event)
	}

	events = append(events, quarantineFailureEvents(localRoot)...)
	events = append(events, negativeCacheFailureEvents(cacheRoot)...)

	deduped := dedupeFailureEvents(events)
	sort.Slice(deduped, func(i, j int) bool {
		if deduped[i].CreatedAt == deduped[j].CreatedAt {
			if deduped[i].Artist == deduped[j].Artist {
				return deduped[i].Title < deduped[j].Title
			}
			return deduped[i].Artist < deduped[j].Artist
		}
		return deduped[i].CreatedAt > deduped[j].CreatedAt
	})

	summary := FailureAnalysisSummary{
		Events:         deduped,
		Counts:         map[string]int{},
		ProviderCounts: map[string]int{},
	}
	for _, event := range summary.Events {
		category := classifyFailureEvent(event)
		summary.Counts[category]++
		provider := event.Provider
		if provider == "" {
			provider = "pipeline"
		}
		summary.ProviderCounts[provider]++
	}
	return summary
}

func dedupeFailureEvents(events []FailureEvent) []FailureEvent {
	seen := map[string]struct{}{}
	out := make([]FailureEvent, 0, len(events))
	for _, event := range events {
		event.Category = classifyFailureEvent(event)
		key := strings.Join([]string{
			strings.TrimSpace(strings.ToLower(event.Artist)),
			strings.TrimSpace(strings.ToLower(event.Title)),
			strings.TrimSpace(strings.ToLower(event.Provider)),
			strings.TrimSpace(strings.ToLower(event.Category)),
			strings.TrimSpace(strings.ToLower(event.Reason)),
			strings.TrimSpace(strings.ToLower(event.Status)),
		}, "|")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, event)
	}
	return out
}

func quarantineFailureEvents(localRoot string) []FailureEvent {
	badDir := filepath.Join(localRoot, "bad")
	entries, err := os.ReadDir(badDir)
	if err != nil {
		return nil
	}
	out := make([]FailureEvent, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bad") {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), ".bad")
		base = strings.TrimSuffix(base, filepath.Ext(base))
		artist, title := parseTrackFromFileStem(base)
		event := FailureEvent{
			Artist:    artist,
			Title:     title,
			Provider:  "local-cache",
			Category:  "cache inválido",
			Reason:    "quarantined local file",
			Status:    "invalid",
			Path:      filepath.Join(badDir, entry.Name()),
			Source:    "quarantine",
			CreatedAt: entryTimeFromName(entry.Name()),
		}
		out = append(out, event)
	}
	return out
}

func negativeCacheFailureEvents(cacheRoot string) []FailureEvent {
	negativeDir := filepath.Join(cacheRoot, "negative")
	entries, err := os.ReadDir(negativeDir)
	if err != nil {
		return nil
	}
	out := make([]FailureEvent, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		out = append(out, FailureEvent{
			Provider:  "negative-cache",
			Category:  "letra inexistente",
			Reason:    "negative cache entry",
			Status:    "not_found",
			Path:      filepath.Join(negativeDir, entry.Name()),
			Source:    "cache",
			CreatedAt: entryTimeFromName(entry.Name()),
		})
	}
	return out
}

func parseTrackFromFileStem(stem string) (string, string) {
	stem = strings.TrimSpace(stem)
	if stem == "" {
		return "", ""
	}
	if idx := strings.LastIndex(stem, ".lrc"); idx >= 0 {
		stem = stem[:idx]
	}
	if idx := strings.Index(stem, " - "); idx >= 0 {
		return strings.TrimSpace(stem[:idx]), strings.TrimSpace(stem[idx+3:])
	}
	return "", stem
}

func entryTimeFromName(name string) int64 {
	name = strings.TrimSuffix(name, ".bad")
	name = strings.TrimSuffix(name, ".json")
	if idx := strings.LastIndex(name, "."); idx >= 0 && idx+1 < len(name) {
		if n, err := strconv.ParseInt(name[idx+1:], 10, 64); err == nil {
			return n
		}
	}
	return 0
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
