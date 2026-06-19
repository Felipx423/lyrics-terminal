package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupCacheTestEnv(t *testing.T) {
	t.Helper()

	origHomeDir := homeDir
	origLocalDir := localDir
	origCacheDir := cacheDir
	origIndexPath := indexPath
	origFailureLogPath := failureLogPath
	origCandidateLogPath := candidateLogPath

	root := t.TempDir()
	homeDir = root
	localDir = filepath.Join(root, "local")
	cacheDir = filepath.Join(root, "cache")
	indexPath = filepath.Join(cacheDir, "index.json")
	failureLogPath = filepath.Join(cacheDir, "failures.jsonl")
	candidateLogPath = filepath.Join(cacheDir, "candidate_evaluations.jsonl")

	t.Cleanup(func() {
		homeDir = origHomeDir
		localDir = origLocalDir
		cacheDir = origCacheDir
		indexPath = origIndexPath
		failureLogPath = origFailureLogPath
		candidateLogPath = origCandidateLogPath
	})
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func requireNoBadQuarantine(t *testing.T) {
	t.Helper()

	badDir := filepath.Join(localDir, "bad")
	if _, err := os.Stat(badDir); err == nil {
		entries, readErr := os.ReadDir(badDir)
		if readErr != nil {
			t.Fatalf("read bad dir: %v", readErr)
		}
		if len(entries) != 0 {
			t.Fatalf("expected no quarantine files, got %d entries", len(entries))
		}
		return
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat bad dir: %v", err)
	}
}

func requireSingleBadFileWithContent(t *testing.T, want string) string {
	t.Helper()

	badDir := filepath.Join(localDir, "bad")
	entries, err := os.ReadDir(badDir)
	if err != nil {
		t.Fatalf("read bad dir: %v", err)
	}
	var badFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".bad") {
			badFiles = append(badFiles, filepath.Join(badDir, entry.Name()))
		}
	}
	if len(badFiles) != 1 {
		t.Fatalf("expected 1 quarantined file, got %d", len(badFiles))
	}
	data, err := os.ReadFile(badFiles[0])
	if err != nil {
		t.Fatalf("read quarantined file: %v", err)
	}
	if string(data) != want {
		t.Fatalf("unexpected quarantined content:\nwant %q\ngot  %q", want, string(data))
	}
	return badFiles[0]
}

func TestInspectLocalLRCValidCacheHit(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Djavan", Title: "Samurai"}
	path := filepath.Join(localDir, exactBaseName(track)+".lrc")
	content := "[00:01.00]No mundo\n[00:02.00]todo\n[00:03.00]seu"
	writeTestFile(t, path, content)

	gotPath, gotText, ok := inspectLocalLRC(track)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if gotPath != path {
		t.Fatalf("unexpected path: got %s want %s", gotPath, path)
	}
	if gotText != content {
		t.Fatalf("unexpected content: got %q want %q", gotText, content)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected cache file to remain: %v", err)
	}
	requireNoBadQuarantine(t)
}

func TestInspectLocalLRCEmptyCacheQuarantinesFile(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Artist", Title: "Empty"}
	path := filepath.Join(localDir, exactBaseName(track)+".lrc")
	writeTestFile(t, path, "")

	gotPath, gotText, ok := inspectLocalLRC(track)
	if ok {
		t.Fatalf("expected cache miss")
	}
	if gotPath != "" || gotText != "" {
		t.Fatalf("expected empty return values, got %q %q", gotPath, gotText)
	}

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected original cache file to be moved")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat original cache file: %v", err)
	}
	requireSingleBadFileWithContent(t, "")
}

func TestInspectLocalLRCWithoutTimestampIsQuarantined(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Artist", Title: "No Timestamp"}
	path := filepath.Join(localDir, exactBaseName(track)+".lrc")
	content := "just plain lyrics\nwith no timestamp markers"
	writeTestFile(t, path, content)

	gotPath, gotText, ok := inspectLocalLRC(track)
	if ok {
		t.Fatalf("expected cache miss")
	}
	if gotPath != "" || gotText != "" {
		t.Fatalf("expected empty return values, got %q %q", gotPath, gotText)
	}

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected original cache file to be moved")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat original cache file: %v", err)
	}
	requireSingleBadFileWithContent(t, content)
}

func TestInspectLocalLRCWithTimestampsButNoUsefulLinesIsQuarantined(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Artist", Title: "Useless Lines"}
	path := filepath.Join(localDir, exactBaseName(track)+".lrc")
	content := strings.Join([]string{
		"[ar:Artist]",
		"[ti:Useless Lines]",
		"[00:01.00]...",
		"[00:02.00][Chorus]",
		"[00:03.00]♪",
	}, "\n")
	writeTestFile(t, path, content)

	gotPath, gotText, ok := inspectLocalLRC(track)
	if ok {
		t.Fatalf("expected cache miss")
	}
	if gotPath != "" || gotText != "" {
		t.Fatalf("expected empty return values, got %q %q", gotPath, gotText)
	}

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected original cache file to be moved")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat original cache file: %v", err)
	}
	requireSingleBadFileWithContent(t, content)
}

func TestInspectLocalLRCRejectsCJKForLatinTrack(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Aimar", Title: "LINGERIE"}
	path := filepath.Join(localDir, exactBaseName(track)+".lrc")
	content := strings.Join([]string{
		"[00:01.00]きらめく夜に",
		"[00:02.00]あなたを待ってる",
		"[00:03.00]終わらない夢",
	}, "\n")
	writeTestFile(t, path, content)

	gotPath, gotText, ok := inspectLocalLRC(track)
	if ok {
		t.Fatalf("expected cache miss")
	}
	if gotPath != "" || gotText != "" {
		t.Fatalf("expected empty return values, got %q %q", gotPath, gotText)
	}

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected original cache file to be moved")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat original cache file: %v", err)
	}
	requireSingleBadFileWithContent(t, content)
}

func TestInspectLocalLRCAcceptsValidCJKTrack(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "宇多田ヒカル", Title: "初恋"}
	path := filepath.Join(localDir, exactBaseName(track)+".lrc")
	content := strings.Join([]string{
		"[00:01.00]あの日の",
		"[00:02.00]鼓動を",
		"[00:03.00]まだ覚えてる",
	}, "\n")
	writeTestFile(t, path, content)

	gotPath, gotText, ok := inspectLocalLRC(track)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if gotPath != path {
		t.Fatalf("unexpected path: got %s want %s", gotPath, path)
	}
	if gotText != content {
		t.Fatalf("unexpected content: got %q want %q", gotText, content)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected cache file to remain: %v", err)
	}
	requireNoBadQuarantine(t)
}

func TestInspectLocalLRCFallsBackToNormalizedPath(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Aimar feat. Someone", Title: "LINGERIE"}
	exactPath := filepath.Join(localDir, exactBaseName(track)+".lrc")
	normalizedPath := filepath.Join(localDir, normalizedBaseName(track)+".lrc")
	exactContent := "plain text without timestamps"
	normalizedContent := strings.Join([]string{
		"[00:01.00]All the right words",
		"[00:02.00]for the normalized cache",
	}, "\n")
	writeTestFile(t, exactPath, exactContent)
	writeTestFile(t, normalizedPath, normalizedContent)

	gotPath, gotText, ok := inspectLocalLRC(track)
	if !ok {
		t.Fatalf("expected fallback hit")
	}
	if gotPath != normalizedPath {
		t.Fatalf("unexpected path: got %s want %s", gotPath, normalizedPath)
	}
	if gotText != normalizedContent {
		t.Fatalf("unexpected content: got %q want %q", gotText, normalizedContent)
	}

	if _, err := os.Stat(exactPath); !os.IsNotExist(err) {
		t.Fatalf("expected invalid exact cache file to be moved, stat err=%v", err)
	}
	if _, err := os.Stat(normalizedPath); err != nil {
		t.Fatalf("expected normalized cache file to remain: %v", err)
	}
	requireSingleBadFileWithContent(t, exactContent)
}

func TestInspectLocalLRCMissingCacheReturnsMissWithoutQuarantine(t *testing.T) {
	setupCacheTestEnv(t)

	track := Track{Artist: "Missing Artist", Title: "Missing Song"}

	gotPath, gotText, ok := inspectLocalLRC(track)
	if ok {
		t.Fatalf("expected cache miss")
	}
	if gotPath != "" || gotText != "" {
		t.Fatalf("expected empty return values, got %q %q", gotPath, gotText)
	}

	badDir := filepath.Join(localDir, "bad")
	if _, err := os.Stat(badDir); !os.IsNotExist(err) {
		t.Fatalf("expected no quarantine directory, stat err=%v", err)
	}
}
