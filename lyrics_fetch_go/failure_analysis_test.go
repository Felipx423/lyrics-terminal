package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildFailureAnalysisClassifiesQuarantineAndIndex(t *testing.T) {
	tempDir := t.TempDir()
	localRoot := filepath.Join(tempDir, "local")
	cacheRoot := filepath.Join(tempDir, "cache")
	if err := os.MkdirAll(filepath.Join(localRoot, "bad"), 0o755); err != nil {
		t.Fatalf("mkdir bad: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cacheRoot, "negative"), 0o755); err != nil {
		t.Fatalf("mkdir negative: %v", err)
	}

	badPath := filepath.Join(localRoot, "bad", "Aimar - LINGERIE.lrc.1781805950.bad")
	if err := os.WriteFile(badPath, []byte("[00:00.00] 作词 : doriko"), 0o644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}

	index := map[string]IndexEntry{
		"pipeline": {
			Artist: "Aimar", Title: "LINGERIE", Provider: "pipeline", Status: "not_found", RejectionReason: "no provider returned synced lyrics", CreatedAt: 10,
		},
		"timeout": {
			Artist: "Artist", Title: "Song", Provider: "lrclib", Status: "timeout", RejectionReason: "lrclib timeout", CreatedAt: 20,
		},
		"duration": {
			Artist: "Artist", Title: "Song 2", Provider: "netease-search", Status: "invalid", RejectionReason: "duration differs by 20000 ms", CreatedAt: 30,
		},
	}

	summary := buildFailureAnalysis(index, nil, localRoot, cacheRoot)

	if summary.Counts["letra inexistente"] == 0 {
		t.Fatalf("expected letra inexistente count, got %+v", summary.Counts)
	}
	if summary.Counts["timeout"] == 0 {
		t.Fatalf("expected timeout count, got %+v", summary.Counts)
	}
	if summary.Counts["mismatch de duração"] == 0 {
		t.Fatalf("expected mismatch de duração count, got %+v", summary.Counts)
	}
	if summary.Counts["cache inválido"] == 0 {
		t.Fatalf("expected cache inválido count, got %+v", summary.Counts)
	}

	if summary.ProviderCounts["pipeline"] == 0 || summary.ProviderCounts["lrclib"] == 0 || summary.ProviderCounts["netease-search"] == 0 || summary.ProviderCounts["local-cache"] == 0 {
		t.Fatalf("unexpected provider counts: %+v", summary.ProviderCounts)
	}

	var out bytes.Buffer
	if err := printFailureAnalysis(&out, index, nil, localRoot, cacheRoot); err != nil {
		t.Fatalf("printFailureAnalysis failed: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"lyrics-fetch-go failure analysis",
		"failed_songs:",
		"resolution_hints:",
		"musixmatch:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in output:\n%s", want, text)
		}
	}
}
