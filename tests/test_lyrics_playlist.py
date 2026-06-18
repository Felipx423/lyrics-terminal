from __future__ import annotations

import importlib.util
from importlib.machinery import SourceFileLoader
import types
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "scripts" / "lyrics"


def load_script_module():
    loader = SourceFileLoader("lyrics_script", str(SCRIPT_PATH))
    spec = importlib.util.spec_from_loader(loader.name, loader)
    if spec is None or spec.loader is None:
        raise RuntimeError("unable to load scripts/lyrics")
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


if __name__ == "__main__":
    unittest.main()
