package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type StatsSummary struct {
	LocalLRCFiles       int
	ProviderCounts      map[string]int
	StatusCounts        map[string]int
	NegativeCacheFiles  int
	QuarantinedFiles    int
	RecentEntries       []IndexEntry
	ApproxSuccessRate   float64
	ApproxSuccessSource int
}

func inferIndexStatus(entry IndexEntry) string {
	if entry.Status != "" {
		return entry.Status
	}
	if entry.Provider != "" || len(entry.Files) > 0 {
		return "found"
	}
	return ""
}

func summarizeIndex(index map[string]IndexEntry) StatsSummary {
	summary := StatsSummary{
		ProviderCounts: map[string]int{},
		StatusCounts:   map[string]int{},
	}
	entries := sortedIndexEntries(index)
	summary.RecentEntries = entries
	for _, entry := range entries {
		if entry.Provider != "" {
			summary.ProviderCounts[entry.Provider]++
		}
		if status := inferIndexStatus(entry); status != "" {
			summary.StatusCounts[status]++
		}
	}
	considered := summary.StatusCounts["found"] + summary.StatusCounts["not_found"] + summary.StatusCounts["timeout"]
	summary.ApproxSuccessSource = considered
	if considered > 0 {
		summary.ApproxSuccessRate = float64(summary.StatusCounts["found"]) / float64(considered)
	}
	return summary
}

func countFilesWithSuffix(root string, suffix string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if filepath.Base(path) == "bad" && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), suffix) {
			count++
		}
		return nil
	})
	return count
}

func countQuarantinedFiles() int {
	badDir := filepath.Join(localDir, "bad")
	return countFilesWithSuffix(badDir, ".bad")
}

func countLocalLRCFiles() int {
	return countFilesWithSuffix(localDir, ".lrc")
}

func countNegativeCacheFiles() int {
	return countFilesWithSuffix(filepath.Join(cacheDir, "negative"), ".json")
}

func printStats(w io.Writer, index map[string]IndexEntry) error {
	summary := summarizeIndex(index)
	summary.LocalLRCFiles = countLocalLRCFiles()
	summary.NegativeCacheFiles = countNegativeCacheFiles()
	summary.QuarantinedFiles = countQuarantinedFiles()

	fmt.Fprintln(w, "lyrics-fetch-go stats")
	fmt.Fprintf(w, "local_lrc_files: %d\n", summary.LocalLRCFiles)
	fmt.Fprintf(w, "quarantined_files: %d\n", summary.QuarantinedFiles)
	fmt.Fprintf(w, "negative_cache_files: %d\n", summary.NegativeCacheFiles)
	fmt.Fprintln(w, "status_counts:")
	for _, status := range []string{"found", "not_found", "invalid", "timeout"} {
		fmt.Fprintf(w, "  %s: %d\n", status, summary.StatusCounts[status])
	}
	fmt.Fprintln(w, "provider_counts:")
	providers := make([]string, 0, len(summary.ProviderCounts))
	for provider := range summary.ProviderCounts {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	for _, provider := range providers {
		fmt.Fprintf(w, "  %s: %d\n", provider, summary.ProviderCounts[provider])
	}
	if summary.ApproxSuccessSource >= 5 {
		fmt.Fprintf(w, "approx_success_rate: %.1f%% (%d/%d)\n", summary.ApproxSuccessRate*100, summary.StatusCounts["found"], summary.ApproxSuccessSource)
	} else {
		fmt.Fprintf(w, "approx_success_rate: insufficient_data (%d entries)\n", summary.ApproxSuccessSource)
	}
	fmt.Fprintln(w, "recent_searches:")
	for i, entry := range summary.RecentEntries {
		if i >= 5 {
			break
		}
		timestamp := "unknown"
		if entry.CreatedAt > 0 {
			timestamp = time.Unix(entry.CreatedAt, 0).Format(time.RFC3339)
		}
		fmt.Fprintf(w, "  - %s | %s - %s | status=%s | provider=%s | source_id=%s | reason=%s\n",
			timestamp, entry.Artist, entry.Title, inferIndexStatus(entry), entry.Provider, entry.SourceID, entry.RejectionReason)
	}
	return nil
}
