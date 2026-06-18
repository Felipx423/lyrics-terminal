#!/usr/bin/env python3

from __future__ import annotations

import atexit
import errno
import json
import os
import re
import select
import shutil
import signal
import subprocess
import sys
import textwrap
import time
import unicodedata
from dataclasses import dataclass
from difflib import SequenceMatcher
from pathlib import Path
from typing import Any


HOME = Path.home()
LOCAL_DIR = HOME / ".local" / "share" / "lyrics"
CACHE_DIR = HOME / ".cache" / "lyrics-terminal"
NEGATIVE_DIR = CACHE_DIR / "negative"
INDEX_PATH = CACHE_DIR / "index.json"
LOG_PATH = CACHE_DIR / "lyrics.log"
BUILD_INFO_PATH = CACHE_DIR / "build-info.json"
BUILD_INFO_PATHS = (CACHE_DIR / "build-info.json", HOME / ".local" / "share" / "lyrics" / "build-info.json")
SPOTIFY_PLAYER = "spotify"
FONT_FAMILY = "Monocraft"
FONT_SIZE = "32"
MAX_LOG_SIZE = 5 * 1024 * 1024
MAX_LOG_FILES = 5

OFFSET_TAG_RE = re.compile(r"^\[offset:([+-]?\d+)\]\s*$", re.IGNORECASE)
LRC_LINE_RE = re.compile(r"\[(\d{1,2}):(\d{2})(?:[.:](\d{1,3}))?\](.*)")
META_LINE_RE = re.compile(r"^\[(ar|ti|al|by|offset|re|ve|length|la|language):", re.IGNORECASE)
CJK_CHAR_RE = re.compile(r"[\u3040-\u30ff\u3400-\u4dbf\u4e00-\u9fff\uf900-\ufaff\uac00-\ud7af]")
NOISE_PATTERNS = [
    r"\((?:feat\.?|ft\.?|with|remaster(?:ed)?(?:\s+\d{4})?|live|ao vivo|deluxe|edit|version|mono|stereo|radio edit|instrumental|acoustic|acústico|karaoke|demo)[^)]*\)",
    r"\[(?:feat\.?|ft\.?|with|remaster(?:ed)?(?:\s+\d{4})?|live|ao vivo|deluxe|edit|version|mono|stereo|radio edit|instrumental|acoustic|acústico|karaoke|demo)[^\]]*\]",
    r"\s+-\s+(?:ao vivo|live|remaster(?:ed)?(?:\s+\d{4})?|deluxe.*|single version|radio edit|edit|version|instrumental|acoustic|acústico)$",
]


@dataclass
class TrackInfo:
    artist: str
    title: str
    album: str
    duration_ms: int
    track_id: str


DEBUG = False
LAST_RENDER: tuple[str, tuple[int, int], str] | None = None


def debug_log(label: str, value: Any) -> None:
    log_event(label, value)
    if DEBUG:
        print(f"[lyrics:debug] {label}: {value}", file=sys.stderr)


def _log_path() -> Path:
    return Path(CACHE_DIR) / "lyrics.log"


def _ensure_log_dir() -> None:
    try:
        Path(CACHE_DIR).mkdir(parents=True, exist_ok=True)
    except Exception:
        pass


def _rotate_logs() -> None:
    path = _log_path()
    if not path.exists():
        return
    try:
        if path.stat().st_size < MAX_LOG_SIZE:
            return
    except Exception:
        return
    try:
        oldest = path.with_name(f"{path.name}.{MAX_LOG_FILES - 1}")
        if oldest.exists():
            oldest.unlink()
        for index in range(MAX_LOG_FILES - 2, 0, -1):
            src = path.with_name(f"{path.name}.{index}")
            dst = path.with_name(f"{path.name}.{index + 1}")
            if src.exists():
                src.replace(dst)
        rotated = path.with_name(f"{path.name}.1")
        if rotated.exists():
            rotated.unlink()
        path.replace(rotated)
    except Exception:
        pass


def _format_log_value(value: Any) -> str:
    if isinstance(value, BaseException):
        return str(value)
    if isinstance(value, Path):
        return str(value)
    try:
        return json.dumps(value, ensure_ascii=False, default=str)
    except Exception:
        return str(value)


def log_event(label: str, value: Any = None) -> None:
    _ensure_log_dir()
    path = _log_path()
    _rotate_logs()
    timestamp = time.strftime("%Y-%m-%dT%H:%M:%S%z", time.localtime())
    payload = f"{timestamp} {label}"
    if value is not None:
        payload += f": {_format_log_value(value)}"
    payload += "\n"
    try:
        with path.open("a", encoding="utf-8") as handle:
            handle.write(payload)
    except OSError as exc:
        if exc.errno not in {errno.ENOENT, errno.EACCES}:
            pass


def _read_build_info() -> dict[str, Any]:
    for path in BUILD_INFO_PATHS:
        if not path.exists():
            continue
        try:
            data = json.loads(path.read_text(encoding="utf-8"))
        except Exception:
            continue
        if isinstance(data, dict):
            return data
    return {}


def _git_metadata() -> dict[str, str]:
    start = Path(__file__).resolve()
    candidates = [start.parent, *start.parents]
    repo_root = None
    for candidate in candidates:
        if (candidate / ".git").exists():
            repo_root = candidate
            break
    if repo_root is None:
        return {}
    try:
        version = subprocess.check_output(
            ["git", "-C", str(repo_root), "describe", "--tags", "--always", "--dirty"],
            text=True,
            stderr=subprocess.DEVNULL,
        ).strip()
    except Exception:
        version = ""
    try:
        commit = subprocess.check_output(
            ["git", "-C", str(repo_root), "rev-parse", "HEAD"],
            text=True,
            stderr=subprocess.DEVNULL,
        ).strip()
    except Exception:
        commit = ""
    try:
        build_date = subprocess.check_output(
            ["git", "-C", str(repo_root), "show", "-s", "--format=%cI", "HEAD"],
            text=True,
            stderr=subprocess.DEVNULL,
        ).strip()
    except Exception:
        build_date = ""
    return {"version": version, "commit": commit, "build_date": build_date}


def version_info() -> tuple[str, str, str]:
    info = {
        "version": os.environ.get("LYRICS_VERSION", ""),
        "commit": os.environ.get("LYRICS_COMMIT", ""),
        "build_date": os.environ.get("LYRICS_BUILD_DATE", ""),
    }
    build_info = _read_build_info()
    for key in info:
        if not info[key]:
            value = build_info.get(key)
            if isinstance(value, str):
                info[key] = value
    git_info = _git_metadata()
    for key in info:
        if not info[key]:
            info[key] = git_info.get(key, "")
    return (
        info["version"] or "dev",
        info["commit"] or "unknown",
        info["build_date"] or "unknown",
    )


def setup_terminal() -> None:
    sys.stdout.write("\033[?25l")
    sys.stdout.flush()
    atexit.register(restore_terminal)
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            signal.signal(sig, lambda *_: sys.exit(0))
        except Exception:
            pass


def restore_terminal() -> None:
    try:
        sys.stdout.write("\033[?25h")
        sys.stdout.flush()
    except Exception:
        pass


def clear_screen() -> None:
    sys.stdout.write("\033[2J\033[H")
    sys.stdout.flush()


def display_width(text: str) -> int:
    return len(text)


def render_single_line(text: str) -> None:
    global LAST_RENDER
    size = shutil.get_terminal_size((80, 24))
    state = (text, (size.columns, size.lines), "line")
    if state == LAST_RENDER:
        return
    LAST_RENDER = state

    clear_screen()
    cols, rows = size.columns, size.lines
    wrapped = textwrap.wrap(text, width=max(18, min(52, cols - 8)), break_long_words=False, break_on_hyphens=False) or [""]
    top = max(0, (rows - len(wrapped)) // 2)
    for _ in range(top):
        print()
    for part in wrapped:
        pad = max(0, (cols - display_width(part)) // 2)
        print(" " * pad + part)
    sys.stdout.flush()


def render_message(title: str, lines: list[str]) -> None:
    global LAST_RENDER
    size = shutil.get_terminal_size((80, 24))
    state = ("\n".join([title, *lines]), (size.columns, size.lines), "message")
    if state == LAST_RENDER:
        return
    LAST_RENDER = state

    clear_screen()
    cols, rows = size.columns, size.lines
    payload = [title, ""] + lines
    wrapped: list[str] = []
    for line in payload:
        if not line:
            wrapped.append("")
            continue
        wrapped.extend(textwrap.wrap(line, width=max(24, min(72, cols - 10)), break_long_words=False, break_on_hyphens=False) or [""])

    box_width = min(cols - 6, max((display_width(line) for line in wrapped), default=0) + 6)
    box_width = max(30, box_width)
    top = max(0, (rows - len(wrapped) - 4) // 2)
    for _ in range(top):
        print()
    border = "─" * (box_width - 2)
    left_pad = " " * max(0, (cols - box_width) // 2)
    print(left_pad + f"┌{border}┐")
    for line in wrapped:
        pad_inside = max(0, box_width - 2 - display_width(line))
        left = pad_inside // 2
        right = pad_inside - left
        print(left_pad + f"│{' ' * left}{line}{' ' * right}│")
    print(left_pad + f"└{border}┘")
    sys.stdout.flush()


def run_command(cmd: list[str], timeout: float = 2.0) -> str:
    completed = subprocess.run(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        timeout=timeout,
        check=True,
    )
    return completed.stdout


def spotify_status() -> str:
    try:
        return run_command(["playerctl", "-p", SPOTIFY_PLAYER, "status"], timeout=2.0).strip().lower()
    except Exception:
        return ""


def get_track_info() -> TrackInfo | None:
    fmt = "{{artist}}|||{{title}}|||{{album}}|||{{mpris:length}}|||{{mpris:trackid}}"
    try:
        raw = run_command(["playerctl", "-p", SPOTIFY_PLAYER, "metadata", "--format", fmt], timeout=2.0).strip()
    except Exception:
        return None
    parts = raw.split("|||")
    if len(parts) != 5:
        return None
    artist, title, album, duration_raw, track_id = (part.strip() for part in parts)
    if not artist or not title:
        return None
    try:
        duration_ms = int(int(duration_raw) / 1000) if duration_raw else 0
    except ValueError:
        duration_ms = 0
    return TrackInfo(artist=artist, title=title, album=album, duration_ms=duration_ms, track_id=track_id)


def current_position_ms() -> int:
    try:
        value = run_command(["playerctl", "-p", SPOTIFY_PLAYER, "position"], timeout=2.0)
        return max(0, int(float(value.strip()) * 1000))
    except Exception:
        return 0


def safe_filename(value: str) -> str:
    value = unicodedata.normalize("NFKC", value or "").strip()
    value = value.replace("/", "-").replace("\\", "-")
    value = re.sub(r"[\x00-\x1f\x7f<>:\"|?*]", "", value)
    value = re.sub(r"\s{2,}", " ", value).strip(" .")
    return value or "unknown"


def normalize_text(value: str) -> str:
    value = unicodedata.normalize("NFKD", value or "")
    value = "".join(ch for ch in value if not unicodedata.combining(ch))
    value = value.casefold()
    value = re.sub(r"[^a-z0-9]+", " ", value)
    return " ".join(value.split())


def clean_noise(value: str) -> str:
    text = unicodedata.normalize("NFKC", value or "")
    for pattern in NOISE_PATTERNS:
        text = re.sub(pattern, "", text, flags=re.IGNORECASE)
    text = re.sub(r"\s{2,}", " ", text)
    return text.strip(" -–—_").strip()


def clean_artist_name(artist: str) -> str:
    artist = clean_noise(artist)
    parts = re.split(r"\s*(?:feat\.?|ft\.?|with|x|&|/|,)\s*", artist, maxsplit=1, flags=re.IGNORECASE)
    if parts and parts[0].strip():
        return parts[0].strip()
    return artist.strip()


def clean_track_title(title: str) -> str:
    return clean_noise(title)


def track_key(track: TrackInfo) -> str:
    return normalize_text(f"{clean_artist_name(track.artist)} - {clean_track_title(track.title)}")


def exact_base_name(track: TrackInfo) -> str:
    return f"{safe_filename(track.artist)} - {safe_filename(track.title)}"


def normalized_base_name(track: TrackInfo) -> str:
    return f"{safe_filename(normalize_text(track.artist))} - {safe_filename(normalize_text(track.title))}"


def local_lyrics_paths(track: TrackInfo) -> list[Path]:
    exact = LOCAL_DIR / f"{exact_base_name(track)}.lrc"
    normalized = LOCAL_DIR / f"{normalized_base_name(track)}.lrc"
    paths = [exact]
    if normalized != exact:
        paths.append(normalized)
    return paths


def _candidate_local_lrc(track: TrackInfo) -> Path | None:
    for path in local_lyrics_paths(track):
        if path.exists():
            return path
    target_key = track_key(track)
    if LOCAL_DIR.exists():
        for path in LOCAL_DIR.glob("*.lrc"):
            if normalize_text(path.stem) == target_key:
                return path
    return None


def count_cjk_chars(text: str) -> int:
    return len(CJK_CHAR_RE.findall(text or ""))


def track_looks_latin_script(track: TrackInfo) -> bool:
    label = f"{track.artist}{track.title}"
    if count_cjk_chars(label):
        return False
    return any(ch.isalpha() for ch in label)


def local_lrc_invalid_reason(track: TrackInfo, text: str) -> str | None:
    if not text or not text.strip():
        return "empty"

    lines, _ = parse_lrc_text(text)
    if not lines:
        return "no_timestamp"

    useful_lines = [line_text.strip() for _, line_text in lines if is_useful_lyric_line(line_text)]
    if not useful_lines:
        return "no_usable_lyric_lines"

    combined = " ".join(useful_lines)
    cjk_chars = count_cjk_chars(combined)
    alpha_chars = sum(1 for ch in combined if ch.isalpha())
    if cjk_chars >= 4 and cjk_chars >= max(4, int(alpha_chars * 0.15)) and (track_prefers_portuguese(track) or track_looks_latin_script(track)):
        return "cjk_suspect"

    return None


def quarantine_bad_local_lrc(path: Path, reason: str) -> Path | None:
    try:
        bad_dir = LOCAL_DIR / "bad"
        bad_dir.mkdir(parents=True, exist_ok=True)
        target = bad_dir / f"{path.name}.{int(time.time())}.bad"
        path.rename(target)
        debug_log("cache_invalid", reason)
        debug_log("quarantine", str(target))
        return target
    except Exception:
        return None


def inspect_local_lrc(track: TrackInfo) -> tuple[Path | None, str | None, str | None]:
    path = _candidate_local_lrc(track)
    if not path:
        return None, None, None
    try:
        text = path.read_text(encoding="utf-8")
    except Exception:
        quarantine_bad_local_lrc(path, "unreadable")
        return None, None, "unreadable"

    reason = local_lrc_invalid_reason(track, text)
    if reason:
        quarantine_bad_local_lrc(path, reason)
        return None, None, reason
    debug_log("cache_hit", str(path))
    return path, text, None


def find_local_lrc(track: TrackInfo) -> Path | None:
    path, _, _ = inspect_local_lrc(track)
    return path


def parse_lrc_text(text: str) -> tuple[list[tuple[int, str]], int]:
    lines: list[tuple[int, str]] = []
    offset_ms = 0
    for raw in (text or "").splitlines():
        raw = raw.strip("\ufeff").rstrip()
        if not raw:
            continue
        offset_match = OFFSET_TAG_RE.match(raw)
        if offset_match:
            try:
                offset_ms = int(offset_match.group(1))
            except ValueError:
                offset_ms = 0
            continue
        if META_LINE_RE.match(raw):
            continue
        match = LRC_LINE_RE.match(raw)
        if not match:
            continue
        minute = int(match.group(1))
        second = int(match.group(2))
        fraction = (match.group(3) or "0").ljust(3, "0")[:3]
        text_value = match.group(4).strip()
        if not text_value:
            continue
        timestamp = (minute * 60 + second) * 1000 + int(fraction)
        lines.append((timestamp, text_value))
    lines.sort(key=lambda item: item[0])
    return lines, offset_ms


def serialize_lrc(lines: list[tuple[int, str]], offset_ms: int = 0) -> str:
    parts = []
    if offset_ms:
        parts.append(f"[offset:{offset_ms}]")
    for timestamp, text in lines:
        minute = timestamp // 60000
        second = (timestamp % 60000) // 1000
        millis = timestamp % 1000
        parts.append(f"[{minute:02d}:{second:02d}.{millis:03d}]{text}")
    return "\n".join(parts) + "\n"


def current_line(lines: list[tuple[int, str]], position_ms: int) -> str:
    chosen = ""
    for ts, text in lines:
        if position_ms < ts:
            break
        if text.strip():
            chosen = text.strip()
    if chosen:
        return chosen
    for _, text in lines:
        if text.strip():
            return text.strip()
    return ""


def lyrics_language(text: str) -> str:
    words = re.findall(r"[a-zA-ZÀ-ÿ']+", text.casefold())
    if not words:
        return "unknown"
    pt_hits = 0
    en_hits = 0
    pt_tokens = {
        "eu", "voce", "você", "nao", "não", "pra", "pro", "do", "da", "de", "que", "meu", "minha", "te", "tua",
        "teu", "amor", "vida", "saudade", "hoje", "amanha", "amanhã", "vivo", "coracao", "coração", "só", "so",
    }
    en_tokens = {"the", "and", "you", "your", "i", "me", "my", "love", "to", "of", "in", "is", "it", "for", "with"}
    for word in words:
        if word in pt_tokens:
            pt_hits += 1
        if word in en_tokens:
            en_hits += 1
    if any(ch in text for ch in "ãõáéíóúçâêôà"):
        pt_hits += 2
    if pt_hits >= 3 and pt_hits >= en_hits:
        return "pt"
    if en_hits >= 3 and en_hits > pt_hits + 1:
        return "en"
    return "mixed"


def track_prefers_portuguese(track: TrackInfo) -> bool:
    text = normalize_text(f"{track.artist} {track.title}")
    hints = {
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
    if any(hint in text for hint in hints):
        return True
    if any(ch in f"{track.artist}{track.title}" for ch in "ãõáéíóúçâêôà"):
        return True
    pt_markers = (" ao vivo", " feat ", " participacao", " participação", " pra ", " nao ", " não ", " teu ", " tua ", " seu ", " sua ")
    return any(marker in text for marker in pt_markers)


def is_useful_lyric_line(text: str) -> bool:
    if not text or not text.strip():
        return False
    lowered = text.strip().casefold()
    if lowered in {"...", "♪", "♫"}:
        return False
    if lowered.startswith("[") and "]" in lowered:
        return False
    if any(marker in lowered for marker in ("no lyrics", "not found", "offline", "error", "failed")):
        return False
    return any(ch.isalpha() for ch in text)


def similarity(a: str, b: str) -> float:
    return SequenceMatcher(None, normalize_text(a), normalize_text(b)).ratio()


def load_index() -> dict[str, Any]:
    if not INDEX_PATH.exists():
        return {}
    try:
        data = json.loads(INDEX_PATH.read_text(encoding="utf-8"))
        return data if isinstance(data, dict) else {}
    except Exception:
        return {}


def save_index(index: dict[str, Any]) -> None:
    CACHE_DIR.mkdir(parents=True, exist_ok=True)
    INDEX_PATH.write_text(json.dumps(index, ensure_ascii=False, indent=2), encoding="utf-8")


def load_negative(track: TrackInfo) -> dict[str, Any] | None:
    path = NEGATIVE_DIR / f"{track_key(track)}.json"
    if not path.exists():
        return None
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return None
    if not isinstance(data, dict):
        return None
    timestamp = float(data.get("timestamp") or 0)
    if time.time() - timestamp > 6 * 60 * 60:
        try:
            path.unlink()
        except Exception:
            pass
        return None
    return data


def write_negative(track: TrackInfo, reason: str) -> None:
    NEGATIVE_DIR.mkdir(parents=True, exist_ok=True)
    path = NEGATIVE_DIR / f"{track_key(track)}.json"
    payload = {"timestamp": time.time(), "reason": reason}
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")


def clear_cache() -> None:
    if CACHE_DIR.exists():
        shutil.rmtree(CACHE_DIR, ignore_errors=True)


def save_local_lyrics(track: TrackInfo, lrc_text: str, provider: str, source_id: str | None = None) -> list[Path]:
    LOCAL_DIR.mkdir(parents=True, exist_ok=True)
    exact = LOCAL_DIR / f"{exact_base_name(track)}.lrc"
    normalized = LOCAL_DIR / f"{normalized_base_name(track)}.lrc"
    saved: list[Path] = []
    exact.write_text(lrc_text, encoding="utf-8")
    saved.append(exact)
    if normalized != exact:
        normalized.write_text(lrc_text, encoding="utf-8")
        saved.append(normalized)

    index = load_index()
    index[track_key(track)] = {
        "artist": track.artist,
        "title": track.title,
        "provider": provider,
        "source_id": source_id,
        "updated_at": time.time(),
        "files": [str(path) for path in saved],
    }
    save_index(index)
    debug_log("fetch_success", {"provider": provider, "source_id": source_id, "files": [str(path) for path in saved]})
    return saved


def load_local_lrc_text(track: TrackInfo) -> tuple[Path | None, str | None]:
    path, text, _ = inspect_local_lrc(track)
    return path, text


def load_local_lrc_text_with_reason(track: TrackInfo) -> tuple[Path | None, str | None, str | None]:
    return inspect_local_lrc(track)


def terminate_process(proc: subprocess.Popen[Any] | None) -> None:
    if not proc:
        return
    try:
        proc.terminate()
        proc.wait(timeout=1.0)
    except Exception:
        try:
            proc.kill()
        except Exception:
            pass
