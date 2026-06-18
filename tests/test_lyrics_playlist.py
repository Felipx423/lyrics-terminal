from __future__ import annotations

import importlib.util
from importlib.machinery import SourceFileLoader
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

    def test_runtime_flags_propagate_no_output_timeout(self) -> None:
        module = load_script_module()

        flags = module.runtime_flags(True, 15.0)

        self.assertEqual(flags, ["--debug", "--run", "--no-output-timeout", "15"])

    def test_main_defaults_to_current_terminal(self) -> None:
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
            try:
                real_lib.LOCAL_DIR = Path(tmp)
                bad_path = real_lib.LOCAL_DIR / "Aimar - LINGERIE.lrc"
                bad_path.write_text("[00:00.000]土砂降りの中 take a trip\n", encoding="utf-8")

                found = real_lib.find_local_lrc(track)

                self.assertIsNone(found)
                quarantined = list((real_lib.LOCAL_DIR / "bad").glob("*.bad"))
                self.assertTrue(quarantined)
            finally:
                real_lib.LOCAL_DIR = original_dir


if __name__ == "__main__":
    unittest.main()
