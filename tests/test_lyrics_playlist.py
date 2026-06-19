from __future__ import annotations

import importlib.util
from importlib.machinery import SourceFileLoader
import contextlib
import io
import types
import unittest
import tempfile
from pathlib import Path

import lyricslib as real_lib


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "scripts" / "lyrics"
LOCAL_SCRIPT_PATH = Path(__file__).resolve().parents[1] / "scripts" / "lyrics-local"


def load_script_module():
    loader = SourceFileLoader("lyrics_script", str(SCRIPT_PATH))
    spec = importlib.util.spec_from_loader(loader.name, loader)
    if spec is None or spec.loader is None:
        raise RuntimeError("unable to load scripts/lyrics")
    module = importlib.util.module_from_spec(spec)
    loader.exec_module(module)
    return module


def load_local_script_module():
    loader = SourceFileLoader("lyrics_local_script", str(LOCAL_SCRIPT_PATH))
    spec = importlib.util.spec_from_loader(loader.name, loader)
    if spec is None or spec.loader is None:
        raise RuntimeError("unable to load scripts/lyrics-local")
    module = importlib.util.module_from_spec(spec)
    loader.exec_module(module)
    return module


class FakeLib:
    def __init__(self, statuses, tracks, local_paths) -> None:
        self.DEBUG = False
        self.statuses = iter(statuses)
        self.tracks = iter(tracks)
        self.local_paths = local_paths
        self.calls: list[tuple[str, object]] = []
        self.CACHE_DIR = real_lib.CACHE_DIR
        self.LOCAL_DIR = real_lib.LOCAL_DIR
        self.INDEX_PATH = real_lib.INDEX_PATH

    def setup_terminal(self) -> None:
        self.calls.append(("setup_terminal", None))

    def restore_terminal(self) -> None:
        self.calls.append(("restore_terminal", None))

    def spotify_status(self) -> str:
        return next(self.statuses, "stopped")

    def get_track_info(self):
        return next(self.tracks, None)

    def find_local_lrc(self, track):
        return self.local_paths.get(track.title)

    def load_local_lrc_text_with_reason(self, track):
        value = self.local_paths.get(track.title)
        if isinstance(value, tuple):
            return value
        if value:
            return value, "[00:00.000]line one\n", None
        return None, None, None

    def local_lrc_invalid_reason(self, track, text):
        return None

    def render_message(self, title: str, lines: list[str]) -> None:
        self.calls.append(("render_message", title))

    def render_single_line(self, text: str) -> None:
        self.calls.append(("render_single_line", text))

    def clear_screen(self) -> None:
        self.calls.append(("clear_screen", None))

    def debug_log(self, label: str, value) -> None:
        self.calls.append(("debug_log", (label, value)))

    def log_event(self, label: str, value=None) -> None:
        self.calls.append(("log_event", (label, value)))

    def normalize_text(self, value: str) -> str:
        return value.lower()

    def is_useful_lyric_line(self, text: str) -> bool:
        return True

    def terminate_process(self, proc) -> None:
        self.calls.append(("terminate_process", getattr(proc, "pid", None)))

    def current_position_ms(self) -> int:
        return 1000

    def current_line(self, lines, pos_ms):
        return lines[0][1] if lines else ""

    def version_info(self):
        return ("1.2.3", "abcdef0", "2024-01-02T03:04:05Z")


def logged_events(fake_lib: FakeLib) -> list[tuple[str, object]]:
    return [value for label, value in fake_lib.calls if label == "log_event"]


def event_payloads(fake_lib: FakeLib, event_name: str) -> list[object]:
    return [payload for name, payload in logged_events(fake_lib) if name == event_name]


class LyricsPlaylistTest(unittest.TestCase):
    def test_track_change_restarts_pipeline(self) -> None:
        module = load_script_module()
        track_one = types.SimpleNamespace(artist="Artist One", title="First Song", album="", duration_ms=180000, track_id="1")
        track_two = types.SimpleNamespace(artist="Artist Two", title="Second Song", album="", duration_ms=200000, track_id="2")
        fake_lib = FakeLib(
            statuses=["playing", "playing", "stopped"],
            tracks=[track_one, track_two, track_two],
            local_paths={"Second Song": "/tmp/second-song.lrc"},
        )
        stream_calls: list[str] = []
        local_calls: list[str] = []

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = lambda *_args, **_kwargs: None
        module.spawn_background_fetch = lambda track, debug=False: fake_lib.debug_log("fetch_spawned", "pid=4321") or 4321
        module.stream_sptlrx = lambda track, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS: stream_calls.append(track.title) or ("track_changed", track_two)
        module.exec_lyrics_local = lambda debug=False: local_calls.append("local") or 0
        module.DEFAULT_NO_OUTPUT_SECONDS = 10.0

        result = module.run_terminal(debug=True)

        self.assertEqual(result, 0)
        self.assertEqual(stream_calls, ["First Song"])
        self.assertEqual(local_calls, ["local"])

        debug_logs = [value for label, value in fake_lib.calls if label == "debug_log"]
        self.assertIn(("pipeline_start", "starting"), debug_logs)
        self.assertIn(("current_track", "Artist One - First Song"), debug_logs)
        self.assertIn(("track_changed", "Artist One - First Song -> Artist Two - Second Song"), debug_logs)
        self.assertIn(("pipeline_restart", "restarting_pipeline"), debug_logs)
        self.assertIn(("fetch_spawned", "pid=4321"), debug_logs)
        self.assertIn(("no_output_timeout", "10s"), debug_logs)
        track_results = event_payloads(fake_lib, "track_result")
        self.assertTrue(any(payload.get("result") == "track_changed_before_result" for payload in track_results))

    def test_runtime_flags_propagate_no_output_timeout(self) -> None:
        module = load_script_module()

        flags = module.runtime_flags(True, 15.0)

        self.assertEqual(flags, ["--debug", "--run", "--no-output-timeout", "15"])

    def test_main_defaults_to_kitty(self) -> None:
        module = load_script_module()
        calls: list[tuple[str, bool, float]] = []

        module.run_terminal = lambda debug=False, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS: calls.append(("run", debug, no_output_timeout)) or 0
        module.launch_kitty = lambda debug=False, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS: calls.append(("kitty", debug, no_output_timeout)) or 0
        original_argv = module.sys.argv
        module.sys.argv = ["lyrics", "--debug"]
        try:
            result = module.main()
        finally:
            module.sys.argv = original_argv

        self.assertEqual(result, 0)
        self.assertEqual(calls, [("kitty", True, module.DEFAULT_NO_OUTPUT_SECONDS)])

    def test_main_routes_to_current_terminal_skips_kitty_launcher(self) -> None:
        module = load_script_module()
        calls: list[tuple[str, bool, float]] = []

        module.run_terminal = lambda debug=False, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS: calls.append(("run", debug, no_output_timeout)) or 0
        def fail_if_kitty_called(*_args, **_kwargs):
            raise AssertionError("launch_kitty must not be called for --current")

        module.launch_kitty = fail_if_kitty_called
        original_argv = module.sys.argv
        module.sys.argv = ["lyrics", "--current", "--debug"]
        try:
            result = module.main()
        finally:
            module.sys.argv = original_argv

        self.assertEqual(result, 0)
        self.assertEqual(calls, [("run", True, module.DEFAULT_NO_OUTPUT_SECONDS)])

    def test_main_routes_to_kitty_when_requested(self) -> None:
        module = load_script_module()
        calls: list[tuple[str, bool, float]] = []

        module.run_terminal = lambda debug=False, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS: calls.append(("run", debug, no_output_timeout)) or 0
        module.launch_kitty = lambda debug=False, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS: calls.append(("kitty", debug, no_output_timeout)) or 0
        original_argv = module.sys.argv
        module.sys.argv = ["lyrics", "--kitty", "--debug"]
        try:
            result = module.main()
        finally:
            module.sys.argv = original_argv

        self.assertEqual(result, 0)
        self.assertEqual(calls, [("kitty", True, module.DEFAULT_NO_OUTPUT_SECONDS)])

    def test_run_terminal_logs_initial_cache_hit_and_cache_hit_result(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Cached Song", album="", duration_ms=180000, track_id="1")
        fake_lib = FakeLib(
            statuses=["playing", "stopped"],
            tracks=[track, track],
            local_paths={"Cached Song": "/tmp/cached-song.lrc"},
        )

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = lambda *_args, **_kwargs: None
        module.spawn_background_fetch = lambda track, debug=False: (_ for _ in ()).throw(AssertionError("spawn_background_fetch should not run on cache hit"))
        module.stream_sptlrx = lambda track, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS: (_ for _ in ()).throw(AssertionError("stream_sptlrx should not run on cache hit"))
        module.exec_lyrics_local = lambda debug=False: 0

        result = module.run_terminal(debug=True)

        self.assertEqual(result, 0)
        cache_hit_initial = event_payloads(fake_lib, "cache_hit_initial")
        track_results = event_payloads(fake_lib, "track_result")
        lyrics_local_started = event_payloads(fake_lib, "lyrics_local_started")
        self.assertEqual(len(cache_hit_initial), 1)
        self.assertEqual(len(lyrics_local_started), 1)
        self.assertTrue(any(payload.get("result") == "cache_hit" for payload in track_results))
        self.assertEqual(cache_hit_initial[0]["track_id"], "1")
        self.assertEqual(cache_hit_initial[0]["artist"], "Artist")
        self.assertEqual(cache_hit_initial[0]["title"], "Cached Song")
        self.assertEqual(cache_hit_initial[0]["album"], "")
        self.assertEqual(cache_hit_initial[0]["duration_s"], 180.0)
        cache_hit_result = next(payload for payload in track_results if payload.get("result") == "cache_hit")
        self.assertEqual(cache_hit_result["track_id"], "1")
        self.assertEqual(cache_hit_result["artist"], "Artist")
        self.assertEqual(cache_hit_result["title"], "Cached Song")
        self.assertEqual(cache_hit_result["album"], "")
        self.assertEqual(cache_hit_result["duration_s"], 180.0)

    def test_run_terminal_logs_success_after_fetch_when_cache_appears_later(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Slow Song", album="", duration_ms=180000, track_id="1")
        other = types.SimpleNamespace(artist="Artist Two", title="Later Song", album="", duration_ms=200000, track_id="2")
        local_paths = {"Slow Song": None, "Later Song": None}
        fake_lib = FakeLib(
            statuses=["playing"] * 20 + ["stopped"],
            tracks=[track] * 20 + [other] * 5,
            local_paths=local_paths,
        )
        sleep_calls = {"count": 0}
        exec_calls = {"count": 0}

        def fake_sleep(_seconds):
            sleep_calls["count"] += 1
            if sleep_calls["count"] == 2:
                local_paths["Slow Song"] = "/tmp/slow-song.lrc"

        def fake_stream_sptlrx(track, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS):
            proc = types.SimpleNamespace(stdout=object(), poll=lambda: 0, pid=99)
            read_calls = iter([None])

            def fake_read_line(_proc, timeout):
                return next(read_calls, None)

            module.read_line = fake_read_line
            return module.stream_sptlrx_lines(proc, track, no_output_timeout=no_output_timeout)

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = fake_sleep
        module.spawn_background_fetch = lambda track, debug=False: 4321
        module.stream_sptlrx = fake_stream_sptlrx
        def fake_exec_lyrics_local(debug=False):
            exec_calls["count"] += 1
            if exec_calls["count"] == 2:
                return module.EXIT_TRACK_CHANGED
            return 0

        module.exec_lyrics_local = fake_exec_lyrics_local

        result = module.run_terminal(debug=True)

        self.assertEqual(result, 0)
        self.assertGreaterEqual(sleep_calls["count"], 1)
        cache_miss_initial = [payload for payload in event_payloads(fake_lib, "cache_miss_initial") if payload.get("track_id") == "1"]
        self.assertEqual(len(cache_miss_initial), 1)
        fetch_spawned = [payload for payload in event_payloads(fake_lib, "fetch_spawned") if payload.get("track_id") == "1"]
        self.assertEqual(len(fetch_spawned), 1)
        self.assertEqual(fetch_spawned[0]["component"], "lyrics-fetch-go")
        self.assertEqual(len([payload for payload in event_payloads(fake_lib, "sptlrx_started") if payload.get("track_id") == "1"]), 1)
        sptlrx_no_output = [payload for payload in event_payloads(fake_lib, "sptlrx_no_output") if payload.get("track_id") == "1"]
        self.assertEqual(len(sptlrx_no_output), 1)
        self.assertEqual(sptlrx_no_output[0]["track"], "Artist - Slow Song")
        self.assertEqual(sptlrx_no_output[0]["track_id"], "1")
        self.assertEqual(sptlrx_no_output[0]["artist"], "Artist")
        self.assertEqual(sptlrx_no_output[0]["title"], "Slow Song")
        self.assertEqual(sptlrx_no_output[0]["album"], "")
        self.assertEqual(sptlrx_no_output[0]["duration_s"], 180.0)
        self.assertGreaterEqual(sptlrx_no_output[0]["elapsed_s"], 0.0)
        self.assertEqual(sptlrx_no_output[0]["reason"], "proc_exited_without_output")
        self.assertEqual(len([payload for payload in event_payloads(fake_lib, "cache_appeared_after_fetch") if payload.get("track_id") == "1"]), 1)
        self.assertGreaterEqual(len([payload for payload in event_payloads(fake_lib, "lyrics_local_started") if payload.get("track_id") == "1"]), 1)
        debug_logs = [value for label, value in fake_lib.calls if label == "debug_log"]
        self.assertIn(("local_lrc", "appeared"), debug_logs)
        track_results = [payload for payload in event_payloads(fake_lib, "track_result") if payload.get("track_id") == "1"]
        self.assertTrue(any(payload.get("result") == "success_after_fetch" for payload in track_results))
        self.assertFalse(any(payload.get("result") == "track_changed_before_result" for payload in track_results))
        self.assertTrue(any(payload.get("track_id") == "1" and payload.get("result") == "success_after_fetch" for payload in track_results))

    def test_stream_sptlrx_lines_reports_real_timeout_without_output(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Silent Song", album="Album", duration_ms=180000, track_id="track-999")
        fake_lib = FakeLib(
            statuses=["playing", "playing", "playing", "playing", "playing"],
            tracks=[track, track, track, track, track],
            local_paths={},
        )
        monotonic = {"value": 0.0}

        def fake_monotonic():
            monotonic["value"] += 0.6
            return monotonic["value"]

        module.lib = fake_lib
        module.time.monotonic = fake_monotonic
        module.read_line = lambda proc, timeout: None
        module.time.sleep = lambda *_args, **_kwargs: None

        proc = types.SimpleNamespace(stdout=object(), poll=lambda: None, pid=99)
        session = module.new_track_session(track)
        module.set_active_track_session(session)
        try:
            status, next_track = module.stream_sptlrx_lines(proc, track, no_output_timeout=1.0)
        finally:
            module.set_active_track_session(None)

        self.assertEqual(status, "no_output")
        self.assertIsNone(next_track)
        debug_logs = [value for label, value in fake_lib.calls if label == "debug_log"]
        self.assertIn(("sptlrx_no_output", "1s_without_output"), debug_logs)
        payloads = event_payloads(fake_lib, "sptlrx_no_output")
        self.assertEqual(len(payloads), 1)
        payload = payloads[0]
        self.assertEqual(payload["track"], "Artist - Silent Song")
        self.assertEqual(payload["track_id"], "track-999")
        self.assertEqual(payload["artist"], "Artist")
        self.assertEqual(payload["title"], "Silent Song")
        self.assertEqual(payload["album"], "Album")
        self.assertEqual(payload["duration_s"], 180.0)
        self.assertGreater(payload["elapsed_s"], 0.0)
        self.assertEqual(payload["timeout_s"], "1")
        self.assertEqual(payload["reason"], "no_output_timeout")

    def test_run_terminal_keeps_cache_hit_result_when_track_changes_later(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Cached Song", album="", duration_ms=180000, track_id="1")
        next_track = types.SimpleNamespace(artist="Artist Two", title="Next Song", album="", duration_ms=200000, track_id="2")
        fake_lib = FakeLib(
            statuses=["playing", "playing", "stopped"],
            tracks=[track, track, next_track],
            local_paths={"Cached Song": "/tmp/cached-song.lrc", "Next Song": None},
        )
        exec_results = iter([0, module.EXIT_TRACK_CHANGED])

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = lambda *_args, **_kwargs: None
        module.spawn_background_fetch = lambda track, debug=False: 4321
        module.stream_sptlrx = lambda track, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS: (_ for _ in ()).throw(AssertionError("stream_sptlrx should not run on cache hit"))
        module.exec_lyrics_local = lambda debug=False: next(exec_results)

        result = module.run_terminal(debug=True)

        self.assertEqual(result, 0)
        track_results = event_payloads(fake_lib, "track_result")
        self.assertTrue(any(payload.get("track_id") == "1" and payload.get("result") == "cache_hit" for payload in track_results))
        self.assertFalse(any(payload.get("result") == "track_changed_before_result" and payload.get("track_id") == "1" for payload in track_results))

    def test_run_terminal_keeps_live_fallback_result_when_track_changes_later(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Live Song", album="", duration_ms=180000, track_id="1")
        next_track = types.SimpleNamespace(artist="Artist Two", title="Next Song", album="", duration_ms=200000, track_id="2")
        fake_lib = FakeLib(
            statuses=["playing", "playing", "stopped"],
            tracks=[track, track, track, next_track, next_track],
            local_paths={"Live Song": None, "Next Song": None},
        )
        rendered: list[str] = []

        def fake_stream_sptlrx(track, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS):
            proc = types.SimpleNamespace(stdout=object(), poll=lambda: None, pid=99)
            read_calls = iter(["real line\n"])

            def fake_read_line(_proc, timeout):
                return next(read_calls, None)

            module.read_line = fake_read_line
            module.lib.render_single_line = lambda text: rendered.append(text)
            return module.stream_sptlrx_lines(proc, track, no_output_timeout=no_output_timeout)

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = lambda *_args, **_kwargs: None
        module.spawn_background_fetch = lambda track, debug=False: 4321
        module.stream_sptlrx = fake_stream_sptlrx
        module.exec_lyrics_local = lambda debug=False: 0

        result = module.run_terminal(debug=True)

        self.assertEqual(result, 0)
        self.assertEqual(rendered, ["real line"])
        track_results = event_payloads(fake_lib, "track_result")
        self.assertTrue(any(payload.get("track_id") == "1" and payload.get("result") == "live_fallback_only" for payload in track_results))
        self.assertFalse(any(payload.get("result") == "track_changed_before_result" and payload.get("track_id") == "1" for payload in track_results))

    def test_run_terminal_logs_no_output_timeout_when_cache_never_appears(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Silent Song", album="", duration_ms=180000, track_id="1")
        fake_lib = FakeLib(
            statuses=["playing", "stopped"],
            tracks=[track, track],
            local_paths={"Silent Song": None},
        )

        def fake_stream_sptlrx(track, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS):
            fake_lib.log_event("sptlrx_no_output", {"reason": "no_output_timeout", "track": track.title})
            return "no_output", None

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = lambda *_args, **_kwargs: None
        module.spawn_background_fetch = lambda track, debug=False: 4321
        module.stream_sptlrx = fake_stream_sptlrx
        module.exec_lyrics_local = lambda debug=False: 0

        result = module.run_terminal(debug=True)

        self.assertEqual(result, 0)
        track_results = event_payloads(fake_lib, "track_result")
        self.assertTrue(any(payload.get("result") == "no_output_timeout" for payload in track_results))
        self.assertEqual(len(event_payloads(fake_lib, "sptlrx_no_output")), 1)

    def test_run_terminal_logs_interrupted_result_and_restores_terminal(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Interrupted Song", album="Album", duration_ms=181000, track_id="track-123")
        fake_lib = FakeLib(
            statuses=["playing", "playing"],
            tracks=[track, track],
            local_paths={"Interrupted Song": "/tmp/interrupted-song.lrc"},
        )

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = lambda *_args, **_kwargs: None
        module.exec_lyrics_local = lambda debug=False: (_ for _ in ()).throw(KeyboardInterrupt())

        result = module.run_terminal(debug=True)

        self.assertEqual(result, 130)
        track_results = event_payloads(fake_lib, "track_result")
        self.assertEqual(len(track_results), 1)
        self.assertEqual(track_results[0]["result"], "interrupted")
        self.assertEqual(track_results[0]["track_id"], "track-123")
        self.assertEqual(track_results[0]["artist"], "Artist")
        self.assertEqual(track_results[0]["title"], "Interrupted Song")
        self.assertEqual(track_results[0]["album"], "Album")
        self.assertEqual(track_results[0]["duration_s"], 181.0)
        self.assertIn(("restore_terminal", None), fake_lib.calls)

    def test_run_terminal_logs_interrupted_result_for_systemexit_zero(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Signal Song", album="Album", duration_ms=181000, track_id="track-456")
        fake_lib = FakeLib(
            statuses=["playing", "playing"],
            tracks=[track, track],
            local_paths={"Signal Song": "/tmp/signal-song.lrc"},
        )

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = lambda *_args, **_kwargs: None
        module.exec_lyrics_local = lambda debug=False: (_ for _ in ()).throw(SystemExit(0))

        result = module.run_terminal(debug=True)

        self.assertEqual(result, 130)
        track_results = event_payloads(fake_lib, "track_result")
        self.assertEqual(len(track_results), 1)
        self.assertEqual(track_results[0]["result"], "interrupted")
        self.assertEqual(track_results[0]["track_id"], "track-456")
        self.assertEqual(track_results[0]["artist"], "Artist")
        self.assertEqual(track_results[0]["title"], "Signal Song")
        self.assertEqual(track_results[0]["album"], "Album")
        self.assertEqual(track_results[0]["duration_s"], 181.0)
        self.assertIn(("restore_terminal", None), fake_lib.calls)

    def test_run_terminal_logs_error_result_for_systemexit_non_zero(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Error Song", album="Album", duration_ms=181000, track_id="track-789")
        fake_lib = FakeLib(
            statuses=["playing", "playing"],
            tracks=[track, track],
            local_paths={"Error Song": "/tmp/error-song.lrc"},
        )

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = lambda *_args, **_kwargs: None
        module.exec_lyrics_local = lambda debug=False: (_ for _ in ()).throw(SystemExit(7))

        result = module.run_terminal(debug=True)

        self.assertEqual(result, 7)
        track_results = event_payloads(fake_lib, "track_result")
        self.assertEqual(len(track_results), 1)
        self.assertEqual(track_results[0]["result"], "error")
        self.assertEqual(track_results[0]["exit_code"], 7)
        self.assertIn(("restore_terminal", None), fake_lib.calls)

    def test_main_version_prints_metadata(self) -> None:
        module = load_script_module()
        fake_lib = FakeLib(statuses=[], tracks=[], local_paths={})
        module.lib = fake_lib
        original_argv = module.sys.argv
        module.sys.argv = ["lyrics", "--version"]
        try:
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                result = module.main()
        finally:
            module.sys.argv = original_argv

        self.assertEqual(result, 0)
        text = stdout.getvalue()
        self.assertIn("version: 1.2.3", text)
        self.assertIn("commit: abcdef0", text)
        self.assertIn("build_date: 2024-01-02T03:04:05Z", text)

    def test_main_health_reports_statuses(self) -> None:
        module = load_script_module()
        fake_lib = FakeLib(statuses=[], tracks=[], local_paths={})
        temp_dir = tempfile.TemporaryDirectory()
        self.addCleanup(temp_dir.cleanup)
        fake_lib.CACHE_DIR = Path(temp_dir.name) / "cache"
        fake_lib.LOCAL_DIR = Path(temp_dir.name) / "local"
        fake_lib.INDEX_PATH = fake_lib.CACHE_DIR / "index.json"
        fake_lib.CACHE_DIR.mkdir(parents=True, exist_ok=True)
        fake_lib.LOCAL_DIR.mkdir(parents=True, exist_ok=True)
        fake_lib.INDEX_PATH.write_text("{}", encoding="utf-8")
        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.subprocess.run = lambda *args, **kwargs: types.SimpleNamespace(stdout="playing\n", stderr="", returncode=0)
        original_argv = module.sys.argv
        module.sys.argv = ["lyrics", "--health"]
        try:
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                result = module.main()
        finally:
            module.sys.argv = original_argv

        self.assertEqual(result, 0)
        text = stdout.getvalue()
        self.assertIn("PASS spotify:", text)
        self.assertIn("PASS playerctl:", text)
        self.assertIn("PASS kitty:", text)
        self.assertIn("PASS sptlrx:", text)
        self.assertIn("PASS lyrics-fetch-go:", text)
        self.assertIn("PASS cache directory:", text)
        self.assertIn("PASS local lyrics directory:", text)
        self.assertIn("PASS index.json:", text)

    def test_main_health_warns_when_kitty_is_missing_but_current_mode_stays_usable(self) -> None:
        module = load_script_module()
        fake_lib = FakeLib(statuses=[], tracks=[], local_paths={})
        temp_dir = tempfile.TemporaryDirectory()
        self.addCleanup(temp_dir.cleanup)
        fake_lib.CACHE_DIR = Path(temp_dir.name) / "cache"
        fake_lib.LOCAL_DIR = Path(temp_dir.name) / "local"
        fake_lib.INDEX_PATH = fake_lib.CACHE_DIR / "index.json"
        fake_lib.CACHE_DIR.mkdir(parents=True, exist_ok=True)
        fake_lib.LOCAL_DIR.mkdir(parents=True, exist_ok=True)
        fake_lib.INDEX_PATH.write_text("{}", encoding="utf-8")
        module.lib = fake_lib
        module.shutil.which = lambda name: None if name == "kitty" else f"/usr/bin/{name}"
        module.subprocess.run = lambda *args, **kwargs: types.SimpleNamespace(stdout="playing\n", stderr="", returncode=0)
        original_argv = module.sys.argv
        module.sys.argv = ["lyrics", "--health"]
        try:
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                result = module.main()
        finally:
            module.sys.argv = original_argv

        self.assertEqual(result, 0)
        text = stdout.getvalue()
        self.assertIn("WARN kitty:", text)
        self.assertIn("required only for default/--kitty mode", text)

    def test_main_health_fails_when_playerctl_is_missing(self) -> None:
        module = load_script_module()
        fake_lib = FakeLib(statuses=[], tracks=[], local_paths={})
        temp_dir = tempfile.TemporaryDirectory()
        self.addCleanup(temp_dir.cleanup)
        fake_lib.CACHE_DIR = Path(temp_dir.name) / "cache"
        fake_lib.LOCAL_DIR = Path(temp_dir.name) / "local"
        fake_lib.INDEX_PATH = fake_lib.CACHE_DIR / "index.json"
        fake_lib.CACHE_DIR.mkdir(parents=True, exist_ok=True)
        fake_lib.LOCAL_DIR.mkdir(parents=True, exist_ok=True)
        fake_lib.INDEX_PATH.write_text("{}", encoding="utf-8")
        module.lib = fake_lib
        module.shutil.which = lambda name: None if name == "playerctl" else f"/usr/bin/{name}"
        module.subprocess.run = lambda *args, **kwargs: (_ for _ in ()).throw(AssertionError("playerctl status should not run when playerctl is missing"))
        original_argv = module.sys.argv
        module.sys.argv = ["lyrics", "--health"]
        try:
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                result = module.main()
        finally:
            module.sys.argv = original_argv

        self.assertEqual(result, 1)
        text = stdout.getvalue()
        self.assertIn("FAIL spotify:", text)
        self.assertIn("FAIL playerctl:", text)

    def test_launch_kitty_uses_direct_exec(self) -> None:
        module = load_script_module()
        popen_calls = []
        which_calls = []

        module.shutil.which = lambda name: which_calls.append(name) or f"/usr/bin/{name}"
        module.subprocess.Popen = lambda cmd, stdout=None, stderr=None: popen_calls.append(cmd) or types.SimpleNamespace()

        result = module.launch_kitty(debug=True, no_output_timeout=15.0)

        self.assertEqual(result, 0)
        self.assertEqual(which_calls, ["kitty", "playerctl", "python3"])
        self.assertEqual(popen_calls[0][:8], [
            "kitty",
            "--detach",
            "--class",
            "lyrics-terminal",
            "--title",
            "Lyrics",
            "--override",
            "font_family=Monocraft",
        ])
        self.assertIn("-e", popen_calls[0])
        self.assertNotIn("bash", popen_calls[0])
        self.assertNotIn("-lc", popen_calls[0])
        self.assertIn("python3", popen_calls[0])
        self.assertIn("--run", popen_calls[0])
        self.assertIn("--debug", popen_calls[0])
        self.assertIn("--no-output-timeout", popen_calls[0])

    def test_stream_no_output_renders_wait_message(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Silent Song", album="", duration_ms=180000, track_id="1")
        fake_lib = FakeLib(
            statuses=["playing"] * 30,
            tracks=[track] * 30,
            local_paths={},
        )
        render_messages: list[str] = []
        monotonic_value = {"t": 0.0}

        module.lib = fake_lib
        module.time.monotonic = lambda: monotonic_value.__setitem__("t", monotonic_value["t"] + 0.5) or monotonic_value["t"]
        module.read_line = lambda proc, timeout: None
        module.time.sleep = lambda *_args, **_kwargs: None
        module.lib.render_message = lambda title, lines: render_messages.append(title)

        proc = types.SimpleNamespace(stdout=object(), poll=lambda: None, pid=99)

        status, next_track = module.stream_sptlrx_lines(proc, track, no_output_timeout=10.0)

        self.assertEqual(status, "no_output")
        self.assertIsNone(next_track)
        self.assertIn("Buscando letra...", render_messages)
        debug_logs = [value for label, value in fake_lib.calls if label == "debug_log"]
        self.assertIn(("sptlrx_no_output", "10s_without_output"), debug_logs)

    def test_wait_helper_handles_pause_before_local_lrc(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Paused Song", album="", duration_ms=180000, track_id="1")
        other = types.SimpleNamespace(artist="Artist", title="Paused Song (live)", album="", duration_ms=180000, track_id="2")
        fake_lib = FakeLib(
            statuses=["paused", "paused"],
            tracks=[track, other],
            local_paths={},
        )
        slept: list[float] = []

        module.lib = fake_lib
        module.time.sleep = lambda seconds: slept.append(seconds)

        status, fresh = module.wait_for_local_lrc_or_track_change(track)

        self.assertEqual(status, "track_changed")
        self.assertIsNotNone(fresh)
        self.assertIn(module.POLL_INTERVAL_SECONDS, slept)
        debug_logs = [value for label, value in fake_lib.calls if label == "debug_log"]
        self.assertIn(("spotify_paused", "lyrics"), debug_logs)
        self.assertIn(("paused_wait", "lyrics"), debug_logs)
        self.assertIn(("track_changed_while_paused", "Artist - Paused Song -> Artist - Paused Song (live)"), debug_logs)

    def test_wait_helper_detects_local_lrc(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Waiting Song", album="", duration_ms=180000, track_id="1")
        local_paths = {"Waiting Song": None}
        fake_lib = FakeLib(
            statuses=["playing", "playing", "playing", "playing"],
            tracks=[track, track, track, track],
            local_paths=local_paths,
        )
        sleep_calls = {"count": 0}

        def fake_sleep(_seconds):
            sleep_calls["count"] += 1
            if sleep_calls["count"] >= 2:
                local_paths["Waiting Song"] = "/tmp/waiting-song.lrc"

        module.lib = fake_lib
        module.time.sleep = fake_sleep

        status, fresh = module.wait_for_local_lrc_or_track_change(track)

        self.assertEqual(status, "local_ready")
        self.assertIsNotNone(fresh)
        debug_logs = [value for label, value in fake_lib.calls if label == "debug_log"]
        self.assertIn(("local_lrc", "appeared"), debug_logs)

    def test_local_stream_returns_track_changed_while_paused(self) -> None:
        module = load_local_script_module()
        track_one = types.SimpleNamespace(artist="Artist One", title="First Song")
        track_two = types.SimpleNamespace(artist="Artist Two", title="Second Song")
        fake_lib = FakeLib(
            statuses=["playing", "paused", "paused"],
            tracks=[track_one, track_one, track_two],
            local_paths={},
        )
        rendered: list[str] = []
        slept: list[float] = []

        module.lib = fake_lib
        module.time.sleep = lambda seconds: slept.append(seconds)
        module.lib.render_single_line = lambda text: rendered.append(text)

        status = module.stream_local(track_one, [(0, "line one")], 0)

        self.assertEqual(status, module.EXIT_TRACK_CHANGED)
        self.assertEqual(rendered, ["line one"])
        self.assertIn(module.PAUSE_POLL_SECONDS, slept)
        debug_logs = [value for label, value in fake_lib.calls if label == "debug_log"]
        self.assertIn(("spotify_paused", "Artist One - First Song"), debug_logs)
        self.assertIn(("paused_wait", "lyrics-local"), debug_logs)
        self.assertIn(("track_changed_while_paused", "Artist One - First Song -> Artist Two - Second Song"), debug_logs)

    def test_sptlrx_stream_pauses_and_resumes(self) -> None:
        module = load_script_module()
        track = types.SimpleNamespace(artist="Artist", title="Paused Song", album="", duration_ms=180000, track_id="1")
        fake_lib = FakeLib(
            statuses=["playing", "paused", "paused", "playing", "stopped"],
            tracks=[track, track, track, track, track],
            local_paths={},
        )
        slept: list[float] = []

        module.lib = fake_lib
        module.time.sleep = lambda seconds: slept.append(seconds)
        module.read_line = lambda proc, timeout: None

        proc = types.SimpleNamespace(stdout=object(), poll=lambda: None, pid=10)
        status, fresh = module.stream_sptlrx_lines(proc, track, no_output_timeout=10.0)

        self.assertEqual(status, "stopped")
        self.assertIsNone(fresh)
        debug_logs = [value for label, value in fake_lib.calls if label == "debug_log"]
        self.assertIn(("spotify_paused", "Artist - Paused Song"), debug_logs)
        self.assertIn(("paused_wait", "sptlrx"), debug_logs)
        self.assertIn(("spotify_resumed", "Artist - Paused Song"), debug_logs)
        self.assertIn(module.POLL_INTERVAL_SECONDS, slept)

    def test_render_message_falls_back_on_tiny_terminal(self) -> None:
        module = load_script_module()
        original_size = module.lib.shutil.get_terminal_size
        original_clear = module.lib.clear_screen
        try:
            module.lib.LAST_RENDER = None
            module.lib.shutil.get_terminal_size = lambda fallback=(80, 24): types.SimpleNamespace(columns=20, lines=4)
            module.lib.clear_screen = lambda: None
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                module.lib.render_message("Too Small", ["Resize the terminal."])
            text = stdout.getvalue()
        finally:
            module.lib.shutil.get_terminal_size = original_size
            module.lib.clear_screen = original_clear
            module.lib.LAST_RENDER = None

        self.assertIn("Too Small", text)
        self.assertIn("Resize the terminal.", text)
        self.assertNotIn("┌", text)

    def test_render_single_line_falls_back_on_tiny_terminal(self) -> None:
        module = load_script_module()
        original_size = module.lib.shutil.get_terminal_size
        original_clear = module.lib.clear_screen
        try:
            module.lib.LAST_RENDER = None
            module.lib.shutil.get_terminal_size = lambda fallback=(80, 24): types.SimpleNamespace(columns=20, lines=4)
            module.lib.clear_screen = lambda: None
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                module.lib.render_single_line("Short lyric line")
            text = stdout.getvalue()
        finally:
            module.lib.shutil.get_terminal_size = original_size
            module.lib.clear_screen = original_clear
            module.lib.LAST_RENDER = None

        self.assertIn("Short lyric line", text)
        self.assertNotIn("┌", text)

    def test_run_terminal_restarts_after_local_track_changed_exit(self) -> None:
        module = load_script_module()
        track_one = types.SimpleNamespace(artist="Artist One", title="First Song", album="", duration_ms=180000, track_id="1")
        track_two = types.SimpleNamespace(artist="Artist Two", title="Second Song", album="", duration_ms=200000, track_id="2")
        fake_lib = FakeLib(
            statuses=["playing", "playing", "stopped"],
            tracks=[track_one, track_two, track_two],
            local_paths={"Second Song": "/tmp/second-song.lrc"},
        )
        exec_results = iter([module.EXIT_TRACK_CHANGED, 0])

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = lambda *_args, **_kwargs: None
        module.spawn_background_fetch = lambda track, debug=False: 4321
        module.exec_lyrics_local = lambda debug=False: next(exec_results)
        sptlrx_calls: list[str] = []
        module.stream_sptlrx = lambda track, no_output_timeout=module.DEFAULT_NO_OUTPUT_SECONDS: sptlrx_calls.append(track.title) or ("track_changed", track_two)

        result = module.run_terminal(debug=True)

        self.assertEqual(result, 0)
        self.assertEqual(sptlrx_calls, ["First Song"])
        debug_logs = [value for label, value in fake_lib.calls if label == "debug_log"]
        self.assertIn(("track_changed", "Artist One - First Song -> Artist Two - Second Song"), debug_logs)
        self.assertIn(("pipeline_restart", "restarting_pipeline"), debug_logs)

    def test_find_local_lrc_rejects_cjk_mismatch(self) -> None:
        track = real_lib.TrackInfo(artist="Aimar", title="LINGERIE", album="", duration_ms=0, track_id="")
        with tempfile.TemporaryDirectory() as tmp:
            original_dir = real_lib.LOCAL_DIR
            original_cache = real_lib.CACHE_DIR
            original_log = real_lib.LOG_PATH
            try:
                real_lib.LOCAL_DIR = Path(tmp)
                real_lib.CACHE_DIR = Path(tmp)
                real_lib.LOG_PATH = real_lib.CACHE_DIR / "lyrics.log"
                bad_path = real_lib.LOCAL_DIR / "Aimar - LINGERIE.lrc"
                bad_path.write_text("[00:00.000]土砂降りの中 take a trip\n", encoding="utf-8")

                found = real_lib.find_local_lrc(track)

                self.assertIsNone(found)
                quarantined = list((real_lib.LOCAL_DIR / "bad").glob("*.bad"))
                self.assertTrue(quarantined)
            finally:
                real_lib.LOCAL_DIR = original_dir
                real_lib.CACHE_DIR = original_cache
                real_lib.LOG_PATH = original_log


if __name__ == "__main__":
    unittest.main()
