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
    def __init__(self) -> None:
        self.DEBUG = False
        self.statuses = iter(["playing", "playing", "stopped"])
        self.tracks = iter(
            [
                types.SimpleNamespace(artist="Artist One", title="First Song", album="", duration_ms=180000, track_id="1"),
                types.SimpleNamespace(artist="Artist Two", title="Second Song", album="", duration_ms=200000, track_id="2"),
            ]
        )
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
        if track.title == "Second Song":
            return "/tmp/second-song.lrc"
        return None

    def render_message(self, title: str, lines: list[str]) -> None:
        self.calls.append(("render_message", title))

    def debug_log(self, label: str, value) -> None:
        self.calls.append(("debug_log", label))


class LyricsPlaylistTest(unittest.TestCase):
    def test_main_command_continues_across_tracks(self) -> None:
        module = load_script_module()
        fake_lib = FakeLib()
        stream_calls: list[str] = []
        local_calls: list[str] = []
        fetch_calls: list[str] = []

        module.lib = fake_lib
        module.shutil.which = lambda name: f"/usr/bin/{name}"
        module.time.sleep = lambda *_args, **_kwargs: None
        module.spawn_background_fetch = lambda track, debug=False: fetch_calls.append(track.title)
        module.stream_sptlrx = lambda track: stream_calls.append(track.title) or 0
        module.exec_lyrics_local = lambda debug=False: local_calls.append("local") or 0

        result = module.run_terminal(debug=False)

        self.assertEqual(result, 0)
        self.assertEqual(fetch_calls, ["First Song"])
        self.assertEqual(stream_calls, ["First Song"])
        self.assertEqual(local_calls, ["local"])


if __name__ == "__main__":
    unittest.main()
