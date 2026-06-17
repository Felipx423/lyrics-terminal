package main

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	offsetTagRe = regexp.MustCompile(`^\[offset:([+-]?\d+)\]\s*$`)
	lrcLineRe   = regexp.MustCompile(`^\[(\d{1,2}):(\d{2})(?:[.:](\d{1,3}))?\](.*)$`)
	metaLineRe  = regexp.MustCompile(`^\[(ar|ti|al|by|offset|re|ve|length|la|language):`)

	noisePatterns = []*regexp.Regexp{
		regexp.MustCompile(`\((?:feat\.?|ft\.?|with|remaster(?:ed)?(?:\s+\d{4})?|live|ao vivo|deluxe|edit|version|mono|stereo|radio edit|instrumental|acoustic|acústico|karaoke|demo)[^)]*\)`),
		regexp.MustCompile(`\[(?:feat\.?|ft\.?|with|remaster(?:ed)?(?:\s+\d{4})?|live|ao vivo|deluxe|edit|version|mono|stereo|radio edit|instrumental|acoustic|acústico|karaoke|demo)[^\]]*\]`),
		regexp.MustCompile(`\s+-\s+(?:ao vivo|live|remaster(?:ed)?(?:\s+\d{4})?|deluxe.*|single version|radio edit|edit|version|instrumental|acoustic|acústico)$`),
	}
	splitArtistRe = regexp.MustCompile(`\s*(?:feat\.?|ft\.?|with|x|&|/|,)\s*`)
)

func safeFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	value = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		}
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, " .")
	if value == "" {
		return "unknown"
	}
	return value
}

func stripDiacritics(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch r {
		case 'á', 'à', 'â', 'ã', 'ä', 'å', 'Á', 'À', 'Â', 'Ã', 'Ä', 'Å':
			b.WriteByte('a')
		case 'é', 'è', 'ê', 'ë', 'É', 'È', 'Ê', 'Ë':
			b.WriteByte('e')
		case 'í', 'ì', 'î', 'ï', 'Í', 'Ì', 'Î', 'Ï':
			b.WriteByte('i')
		case 'ó', 'ò', 'ô', 'õ', 'ö', 'Ó', 'Ò', 'Ô', 'Õ', 'Ö':
			b.WriteByte('o')
		case 'ú', 'ù', 'û', 'ü', 'Ú', 'Ù', 'Û', 'Ü':
			b.WriteByte('u')
		case 'ç', 'Ç':
			b.WriteByte('c')
		case 'ñ', 'Ñ':
			b.WriteByte('n')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeText(value string) string {
	value = strings.ToLower(stripDiacritics(value))
	var b strings.Builder
	b.Grow(len(value))
	lastSpace := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func cleanNoise(value string) string {
	text := value
	for _, pattern := range noisePatterns {
		text = pattern.ReplaceAllStringFunc(text, func(match string) string {
			return ""
		})
	}
	text = strings.Join(strings.Fields(text), " ")
	return strings.Trim(text, " -–—_")
}

func cleanArtistName(artist string) string {
	artist = cleanNoise(artist)
	parts := splitArtistRe.Split(artist, 2)
	if len(parts) > 0 {
		if trimmed := strings.TrimSpace(parts[0]); trimmed != "" {
			return trimmed
		}
	}
	return strings.TrimSpace(artist)
}

func cleanTrackTitle(title string) string {
	return cleanNoise(title)
}

func trackKey(track Track) string {
	return normalizeText(cleanArtistName(track.Artist) + " - " + cleanTrackTitle(track.Title))
}

func exactBaseName(track Track) string {
	return safeFilename(track.Artist) + " - " + safeFilename(track.Title)
}

func normalizedBaseName(track Track) string {
	return safeFilename(normalizeText(track.Artist)) + " - " + safeFilename(normalizeText(track.Title))
}

func parseLRCText(text string) ([][2]string, int) {
	type line struct {
		ts   int
		text string
	}
	var lines []line
	offsetMs := 0
	for _, raw := range strings.Split(text, "\n") {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff"))
		if raw == "" {
			continue
		}
		if m := offsetTagRe.FindStringSubmatch(raw); len(m) == 2 {
			if v, err := strconv.Atoi(m[1]); err == nil {
				offsetMs = v
			}
			continue
		}
		if metaLineRe.MatchString(raw) {
			continue
		}
		m := lrcLineRe.FindStringSubmatch(raw)
		if len(m) != 5 {
			continue
		}
		minute, err1 := strconv.Atoi(m[1])
		second, err2 := strconv.Atoi(m[2])
		if err1 != nil || err2 != nil {
			continue
		}
		fraction := m[3]
		if fraction == "" {
			fraction = "0"
		}
		fraction = (fraction + "000")[:3]
		textValue := strings.TrimSpace(m[4])
		if textValue == "" {
			continue
		}
		ms, err := strconv.Atoi(fraction)
		if err != nil {
			continue
		}
		ts := (minute*60+second)*1000 + ms
		lines = append(lines, line{ts: ts, text: textValue})
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].ts < lines[j].ts })
	out := make([][2]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, [2]string{strconv.Itoa(line.ts), line.text})
	}
	return out, offsetMs
}

func hasSyncedLines(text string) bool {
	lines, _ := parseLRCText(text)
	return len(lines) > 0
}

func buildQuery(title, artist string) string {
	return strings.TrimSpace(cleanTrackTitle(title) + " " + cleanArtistName(artist))
}
