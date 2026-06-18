package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	errNotFound = errors.New("not found")
	errTimeout  = errors.New("timeout")
)

type lrclibCandidate struct {
	ID           any     `json:"id"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	SyncedLyrics string  `json:"syncedLyrics"`
	SyncedAlt    string  `json:"synced_lyrics"`
}

type neteaseSearchResponse struct {
	Result struct {
		Songs []neteaseSong `json:"songs"`
	} `json:"result"`
}

type neteaseSong struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	DT   int64  `json:"dt"`
	AR   []struct {
		Name string `json:"name"`
	} `json:"ar"`
	AL struct {
		Name string `json:"name"`
	} `json:"al"`
	Alias []string `json:"alia"`
}

type neteaseLyricResponse struct {
	LRC struct {
		Lyric string `json:"lyric"`
	} `json:"lrc"`
	Songs []struct {
		Name string `json:"name"`
		DT   int64  `json:"dt"`
		AR   []struct {
			Name string `json:"name"`
		} `json:"ar"`
	} `json:"songs"`
}

func fetchLyrics(ctx context.Context, track Track, debug bool, deepSearch bool) (*Candidate, error) {
	if cand, err := fetchLRCLIB(ctx, track, debug, deepSearch); err == nil && cand != nil {
		return cand, nil
	} else if err != nil && !errors.Is(err, errNotFound) {
		if errors.Is(err, errTimeout) {
			debugLog(debug, "lrclib", "timeout, not caching negative result")
			_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "lrclib", Category: "timeout", Reason: "lrclib timeout", Status: "timeout", TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
		} else {
			debugLog(debug, "lrclib_error", err)
			_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "lrclib", Category: "provider indisponível", Reason: err.Error(), Status: "error", TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
		}
	}

	if cand, err := fetchNetEaseMap(ctx, track, debug); err == nil && cand != nil {
		return cand, nil
	} else if err != nil && !errors.Is(err, errNotFound) {
		debugLog(debug, "netease_map_error", err)
		_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "netease-map", Category: failureCategoryFromReason(strings.ToLower(err.Error())), Reason: err.Error(), Status: "error", TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
	}

	if cand, err := fetchNetEaseSearch(ctx, track, debug); err == nil && cand != nil {
		return cand, nil
	} else if err != nil && !errors.Is(err, errNotFound) {
		debugLog(debug, "netease_search_error", err)
		_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "netease-search", Category: failureCategoryFromReason(strings.ToLower(err.Error())), Reason: err.Error(), Status: "error", TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
	}

	if cand, err := fetchSyncedLyricsCLI(ctx, track, debug, deepSearch); err == nil && cand != nil {
		return cand, nil
	} else if err != nil && !errors.Is(err, errNotFound) {
		debugLog(debug, "syncedlyrics_error", err)
		_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "syncedlyrics", Category: failureCategoryFromReason(strings.ToLower(err.Error())), Reason: err.Error(), Status: "error", TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
	}

	return nil, errNotFound
}

func fetchLRCLIB(ctx context.Context, track Track, debug bool, deepSearch bool) (*Candidate, error) {
	queries := buildLRCLibQueries(track, deepSearch)
	var sawTimeout bool
	for _, query := range queries {
		if deadlineExceeded(ctx) {
			return nil, errTimeout
		}
		debugLog(debug, "lrclib_query", query.Encode())
		data, err := queryLRCLIB(ctx, query)
		if err != nil {
			if errors.Is(err, errTimeout) || errors.Is(err, context.DeadlineExceeded) {
				sawTimeout = true
				debugLog(debug, "lrclib_timeout", err)
				continue
			}
			if errors.Is(err, errNotFound) {
				continue
			}
			debugLog(debug, "lrclib_error", err)
			continue
		}
		var candidates []lrclibCandidate
		if err := json.Unmarshal(data, &candidates); err != nil {
			return nil, err
		}
		debugLog(debug, "lrclib_candidates", summarizeLRCLIBCandidates(candidates))
		for _, cand := range candidates {
			sourceID := candidateSourceID(cand.ID)
			accepted, reason, details := validateLRCLIBCandidate(cand, track)
			debugLog(debug, "lrclib_candidate", map[string]any{
				"track":           cand.TrackName,
				"artist":          cand.ArtistName,
				"album":           cand.AlbumName,
				"duration":        cand.Duration,
				"title_match":     details["title_match"],
				"artist_contains": details["artist_contains"],
				"duration_diff":   details["duration_diff"],
				"synced":          candText(cand) != "",
				"accepted":        accepted,
				"reason":          reason,
			})
			if !accepted {
				debugLog(debug, "provider_rejected", map[string]any{"provider": "lrclib", "reason": reason, "source_id": sourceID})
				_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "lrclib", Category: failureCategoryFromReason(strings.ToLower(reason)), Reason: reason, Status: "invalid", SourceID: sourceID, TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
				continue
			}
			text := candText(cand)
			if text == "" || !hasSyncedLines(text) {
				debugLog(debug, "provider_rejected", map[string]any{"provider": "lrclib", "reason": "not synced", "source_id": sourceID})
				_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "lrclib", Category: "resultado não sincronizado", Reason: "lrclib candidate without synced lines", Status: "invalid", SourceID: sourceID, TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
				continue
			}
			debugLog(debug, "provider_selected", map[string]any{"provider": "lrclib", "source_id": sourceID})
			return &Candidate{Text: text, Provider: "lrclib", SourceID: sourceID}, nil
		}
	}
	if sawTimeout {
		return nil, errTimeout
	}
	return nil, errNotFound
}

func buildLRCLibQueries(track Track, deepSearch bool) []url.Values {
	queries := []url.Values{}
	first := url.Values{}
	first.Set("q", buildQuery(track.Title, track.Artist))
	queries = append(queries, first)

	second := url.Values{}
	second.Set("artist_name", cleanArtistName(track.Artist))
	second.Set("track_name", cleanTrackTitle(track.Title))
	queries = append(queries, second)

	if deepSearch {
		third := url.Values{}
		third.Set("q", strings.TrimSpace(cleanTrackTitle(track.Title)+" "+track.Artist))
		queries = append(queries, third)
	}

	return dedupeURLValues(queries)
}

func queryLRCLIB(ctx context.Context, params url.Values) ([]byte, error) {
	endpoint := "https://lrclib.net/api/search?" + params.Encode()
	timeout := 15 * time.Second
	if remain, ok := ctx.Deadline(); ok {
		if d := time.Until(remain); d < timeout {
			timeout = d
		}
	}
	if timeout <= 0 {
		return nil, errTimeout
	}
	if curl, err := exec.LookPath("curl"); err == nil {
		cmdCtx, cancel := context.WithTimeout(ctx, timeout+time.Second)
		defer cancel()
		cmd := exec.CommandContext(cmdCtx, curl, "-4", "-L", "--http1.1", "--connect-timeout", "5", "--max-time", "15", "-A", "Mozilla/5.0", endpoint)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if stdout.Len() > 0 {
			return stdout.Bytes(), nil
		}
		if err != nil {
			if cmdCtx.Err() == context.DeadlineExceeded || strings.Contains(strings.ToLower(stderr.String()), "timed out") {
				return nil, errTimeout
			}
			return nil, fmt.Errorf("curl lrclib: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, errNotFound
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(err.Error()), "timeout") {
			return nil, errTimeout
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("lrclib status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func fetchNetEaseMap(ctx context.Context, track Track, debug bool) (*Candidate, error) {
	_ = ctx
	mapData := loadLyricsMap()
	neteaseID := lookupNetEaseID(mapData, track)
	debugLog(debug, "netease_map_id", neteaseID)
	if neteaseID == 0 {
		return nil, errNotFound
	}
	return fetchNetEaseLyric(track, neteaseID, "netease-map", debug)
}

func fetchNetEaseSearch(ctx context.Context, track Track, debug bool) (*Candidate, error) {
	query := buildQuery(track.Title, track.Artist)
	if query == "" {
		return nil, errNotFound
	}
	debugLog(debug, "netease_search_query", query)
	params := url.Values{}
	params.Set("s", query)
	params.Set("type", "1")
	params.Set("offset", "0")
	params.Set("limit", "3")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://music.163.com/api/search/get?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	setNetEaseHeaders(req)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("netease search status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed neteaseSearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	var scored []struct {
		song  neteaseSong
		score int
	}
	for _, song := range parsed.Result.Songs {
		score, details := scoreNetEaseCandidate(song, track)
		debugLog(debug, "netease_search_candidate", map[string]any{
			"id":            song.ID,
			"name":          song.Name,
			"score":         score,
			"title_match":   details["title_match"],
			"artist_match":  details["artist_match"],
			"duration_diff": details["duration_diff"],
		})
		scored = append(scored, struct {
			song  neteaseSong
			score int
		}{song: song, score: score})
	}
	sortNetEaseScored(scored)
	for i, item := range scored {
		if i >= 3 {
			break
		}
		if item.score <= 0 {
			continue
		}
		cand, err := fetchNetEaseLyric(track, item.song.ID, "netease-search", debug)
		if err == nil && cand != nil {
			return cand, nil
		}
		if err != nil && !errors.Is(err, errNotFound) {
			return nil, err
		}
	}
	return nil, errNotFound
}

func fetchNetEaseLyric(track Track, neteaseID int64, provider string, debug bool) (*Candidate, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://music.163.com/api/song/lyric?id=%d&lv=1&kv=1&tv=-1", neteaseID), nil)
	if err != nil {
		return nil, err
	}
	setNetEaseHeaders(req)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("netease lyric status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed neteaseLyricResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	lrc := parsed.LRC.Lyric
	if strings.TrimSpace(lrc) == "" {
		debugLog(debug, "netease_lyric", fmt.Sprintf("%d no lrc", neteaseID))
		debugLog(debug, "provider_rejected", map[string]any{"provider": provider, "reason": "empty lyric", "source_id": neteaseID})
		_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: provider, Category: "letra inexistente", Reason: "netease returned empty lyric", Status: "not_found", SourceID: strconv.FormatInt(neteaseID, 10), TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
		return nil, errNotFound
	}
	candidateTitle := track.Title
	candidateArtist := track.Artist
	candidateDurationMs := track.DurationMs
	if len(parsed.Songs) > 0 {
		candidateTitle = parsed.Songs[0].Name
		candidateArtist = joinArtists(parsed.Songs[0].AR)
		candidateDurationMs = int(parsed.Songs[0].DT)
	}
	accepted, reason, details := validateGenericCandidate(candidateTitle, candidateArtist, "", lrc, track, candidateDurationMs)
	debugLog(debug, "netease_candidate", map[string]any{
		"id":            neteaseID,
		"title":         candidateTitle,
		"artist":        candidateArtist,
		"accepted":      accepted,
		"reason":        reason,
		"title_match":   details["title_match"],
		"artist_match":  details["artist_match"],
		"duration_diff": details["duration_diff"],
	})
	if !accepted {
		debugLog(debug, "provider_rejected", map[string]any{"provider": provider, "reason": reason, "source_id": neteaseID})
		_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: provider, Category: failureCategoryFromReason(strings.ToLower(reason)), Reason: reason, Status: "invalid", SourceID: strconv.FormatInt(neteaseID, 10), TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
		return nil, errNotFound
	}
	debugLog(debug, "provider_selected", map[string]any{"provider": provider, "source_id": neteaseID})
	return &Candidate{Text: lrc, Provider: provider, SourceID: strconv.FormatInt(neteaseID, 10)}, nil
}

func fetchSyncedLyricsCLI(ctx context.Context, track Track, debug bool, deepSearch bool) (*Candidate, error) {
	command, err := exec.LookPath("syncedlyrics")
	if err != nil {
		debugLog(debug, "syncedlyrics", "skipped: command not found")
		return nil, errNotFound
	}
	query := track.Artist + " - " + track.Title
	attempts := []string{query}
	if deepSearch {
		attempts = append(attempts, track.Title+" - "+track.Artist, buildQuery(track.Title, track.Artist))
	}
	attempts = dedupeStrings(attempts)
	for _, attempt := range attempts {
		if deadlineExceeded(ctx) {
			return nil, errTimeout
		}
		debugLog(debug, "syncedlyrics_query", attempt)
		subctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		cmd := exec.CommandContext(subctx, command, attempt)
		out, runErr := cmd.Output()
		cancel()
		if runErr != nil {
			if subctx.Err() == context.DeadlineExceeded {
				debugLog(debug, "syncedlyrics_timeout", attempt)
				debugLog(debug, "provider_rejected", map[string]any{"provider": "syncedlyrics", "reason": "timeout"})
				_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "syncedlyrics", Category: "timeout", Reason: "syncedlyrics timeout", Status: "timeout", TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
				return nil, errTimeout
			}
			debugLog(debug, "syncedlyrics_exit", runErr)
			debugLog(debug, "provider_rejected", map[string]any{"provider": "syncedlyrics", "reason": runErr.Error()})
			_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "syncedlyrics", Category: "provider indisponível", Reason: runErr.Error(), Status: "error", TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
			continue
		}
		text := strings.TrimSpace(string(out))
		if text == "" {
			debugLog(debug, "provider_rejected", map[string]any{"provider": "syncedlyrics", "reason": "empty output"})
			_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "syncedlyrics", Category: "letra inexistente", Reason: "syncedlyrics returned empty output", Status: "not_found", TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
			continue
		}
		accepted, reason, _ := validateGenericCandidate(track.Title, track.Artist, "", text, track, track.DurationMs)
		debugLog(debug, "syncedlyrics_candidate", map[string]any{"accepted": accepted, "reason": reason})
		if !accepted || !hasSyncedLines(text) {
			category := failureCategoryFromReason(strings.ToLower(reason))
			if !hasSyncedLines(text) {
				category = "resultado não sincronizado"
				reason = "syncedlyrics output without timestamps"
			}
			debugLog(debug, "provider_rejected", map[string]any{"provider": "syncedlyrics", "reason": reason})
			_ = recordFailureEvent(FailureEvent{Artist: track.Artist, Title: track.Title, Provider: "syncedlyrics", Category: category, Reason: reason, Status: "invalid", TrackID: track.TrackID, DurationMs: track.DurationMs, Source: "provider"})
			continue
		}
		debugLog(debug, "provider_selected", map[string]any{"provider": "syncedlyrics"})
		return &Candidate{Text: text, Provider: "syncedlyrics"}, nil
	}
	return nil, errNotFound
}

func validateLRCLIBCandidate(cand lrclibCandidate, track Track) (bool, string, map[string]any) {
	return validateGenericCandidate(cand.TrackName, cand.ArtistName, cand.AlbumName, candText(cand), track, int(cand.Duration*1000))
}

func validateGenericCandidate(candidateTitle, candidateArtist, candidateAlbum, lyricsText string, track Track, candidateDurationMs int) (bool, string, map[string]any) {
	cleanTitle := normalizeText(cleanTrackTitle(track.Title))
	candidateTitleNorm := normalizeText(cleanTrackTitle(candidateTitle))
	candidateBlob := normalizeText(strings.Join([]string{candidateTitle, candidateArtist, candidateAlbum}, " "))
	trackArtistNorm := normalizeText(cleanArtistName(track.Artist))
	titleMatch := candidateTitleNorm == cleanTitle || strings.HasPrefix(candidateTitleNorm, cleanTitle) || strings.Contains(candidateTitleNorm, cleanTitle)
	artistMatch := trackArtistNorm != "" && strings.Contains(candidateBlob, trackArtistNorm)
	durationDiff := -1
	if track.DurationMs > 0 && candidateDurationMs > 0 {
		durationDiff = absInt(candidateDurationMs - track.DurationMs)
	}
	details := map[string]any{
		"title_match":   titleMatch,
		"artist_match":  artistMatch,
		"duration_diff": durationDiff,
	}
	if !titleMatch {
		return false, "title mismatch", details
	}
	if !artistMatch {
		return false, "artist not present in candidate", details
	}
	if durationDiff >= 0 && durationDiff > 15000 {
		return false, fmt.Sprintf("duration differs by %d ms", durationDiff), details
	}
	if !hasSyncedLines(lyricsText) {
		return false, "not synced", details
	}
	return true, "ok", details
}

func scoreNetEaseCandidate(song neteaseSong, track Track) (int, map[string]any) {
	titleNorm := normalizeText(cleanTrackTitle(song.Name))
	artistNorm := normalizeText(joinArtists(song.AR))
	trackTitleNorm := normalizeText(cleanTrackTitle(track.Title))
	trackArtistNorm := normalizeText(cleanArtistName(track.Artist))
	titleMatch := titleNorm == trackTitleNorm || strings.Contains(titleNorm, trackTitleNorm) || strings.Contains(trackTitleNorm, titleNorm)
	artistMatch := trackArtistNorm != "" && strings.Contains(normalizeText(song.Name+" "+artistNorm), trackArtistNorm)
	durationDiff := -1
	if track.DurationMs > 0 && song.DT > 0 {
		durationDiff = absInt(int(song.DT) - track.DurationMs)
	}
	score := 0
	if titleMatch {
		score += 2
	}
	if artistMatch {
		score += 2
	}
	if durationDiff >= 0 && durationDiff <= 15000 {
		score += 1
	}
	return score, map[string]any{
		"title_match":   titleMatch,
		"artist_match":  artistMatch,
		"duration_diff": durationDiff,
	}
}

func lookupNetEaseID(m map[string]any, track Track) int64 {
	if len(m) == 0 {
		return 0
	}
	keys := []string{
		track.Artist + " - " + track.Title,
		trackKey(track),
		cleanArtistName(track.Artist) + " - " + cleanTrackTitle(track.Title),
	}
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if id := extractNetEaseID(v); id != 0 {
				return id
			}
		}
	}
	norm := normalizeText(track.Artist + " - " + track.Title)
	for key, v := range m {
		if normalizeText(key) == norm {
			if id := extractNetEaseID(v); id != 0 {
				return id
			}
		}
	}
	return 0
}

func extractNetEaseID(v any) int64 {
	m, ok := v.(map[string]any)
	if !ok {
		return 0
	}
	backend, _ := m["backend"].(string)
	if backend != "NetEase" {
		return 0
	}
	switch id := m["neteaseId"].(type) {
	case float64:
		return int64(id)
	case int64:
		return id
	case int:
		return int64(id)
	case string:
		n, _ := strconv.ParseInt(id, 10, 64)
		return n
	default:
		return 0
	}
}

func summarizeLRCLIBCandidates(cands []lrclibCandidate) []map[string]any {
	out := make([]map[string]any, 0, len(cands))
	for i, cand := range cands {
		if i >= 5 {
			break
		}
		out = append(out, map[string]any{
			"track":    cand.TrackName,
			"artist":   cand.ArtistName,
			"album":    cand.AlbumName,
			"duration": cand.Duration,
			"synced":   candText(cand) != "",
		})
	}
	return out
}

func candText(c lrclibCandidate) string {
	if strings.TrimSpace(c.SyncedLyrics) != "" {
		return c.SyncedLyrics
	}
	return c.SyncedAlt
}

func candidateSourceID(v any) string {
	switch x := v.(type) {
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	case string:
		return x
	default:
		return ""
	}
}

func joinArtists(items []struct {
	Name string `json:"name"`
}) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) != "" {
			parts = append(parts, item.Name)
		}
	}
	return strings.Join(parts, " ")
}

func setNetEaseHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://music.163.com")
	req.Header.Set("Origin", "https://music.163.com")
}

func sortNetEaseScored(items []struct {
	song  neteaseSong
	score int
}) {
	for i := range items {
		for j := i + 1; j < len(items); j++ {
			if items[j].score > items[i].score {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func deadlineExceeded(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func dedupeURLValues(items []url.Values) []url.Values {
	seen := make(map[string]struct{}, len(items))
	out := make([]url.Values, 0, len(items))
	for _, item := range items {
		key := item.Encode()
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}
