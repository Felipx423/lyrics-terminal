package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSummarizeIndex(t *testing.T) {
	index := map[string]IndexEntry{
		"a": {Artist: "Artist A", Title: "Song A", Provider: "lrclib", Status: "found", CreatedAt: 10, UpdatedAt: 20},
		"b": {Artist: "Artist B", Title: "Song B", Provider: "netease-search", Status: "not_found", CreatedAt: 20, UpdatedAt: 30},
		"c": {Artist: "Artist C", Title: "Song C", Provider: "syncedlyrics", Status: "timeout", CreatedAt: 30, UpdatedAt: 40},
	}

	summary := summarizeIndex(index)

	if summary.ProviderCounts["lrclib"] != 1 || summary.ProviderCounts["netease-search"] != 1 || summary.ProviderCounts["syncedlyrics"] != 1 {
		t.Fatalf("unexpected provider counts: %+v", summary.ProviderCounts)
	}
	if summary.StatusCounts["found"] != 1 || summary.StatusCounts["not_found"] != 1 || summary.StatusCounts["timeout"] != 1 {
		t.Fatalf("unexpected status counts: %+v", summary.StatusCounts)
	}
	if summary.ApproxSuccessSource != 3 {
		t.Fatalf("unexpected success source: %d", summary.ApproxSuccessSource)
	}
	if summary.ApproxSuccessRate <= 0 || summary.ApproxSuccessRate >= 1 {
		t.Fatalf("unexpected success rate: %f", summary.ApproxSuccessRate)
	}
	if len(summary.RecentEntries) != 3 {
		t.Fatalf("unexpected recent entries: %d", len(summary.RecentEntries))
	}
}

func TestPrintStatsIncludesKeySections(t *testing.T) {
	var out bytes.Buffer
	index := map[string]IndexEntry{
		"a": {Artist: "Artist A", Title: "Song A", Provider: "lrclib", Status: "found", CreatedAt: 10, UpdatedAt: 20},
	}

	if err := printStats(&out, index); err != nil {
		t.Fatalf("printStats failed: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"lyrics-fetch-go stats",
		"local_lrc_files:",
		"provider_counts:",
		"status_counts:",
		"recent_searches:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in output:\n%s", want, text)
		}
	}
}
