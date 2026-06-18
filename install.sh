#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${HOME}/.local/bin"
GO_DIR="${ROOT_DIR}/lyrics_fetch_go"

require_cmd() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    echo "missing dependency: ${name}" >&2
    exit 1
  fi
}

for dep in go python3 playerctl kitty sptlrx; do
  require_cmd "$dep"
done

mkdir -p "${BIN_DIR}"

echo "building lyrics-fetch-go"
(cd "${GO_DIR}" && go build -o "${BIN_DIR}/lyrics-fetch-go" .)

cp "${ROOT_DIR}/scripts/lyrics" "${BIN_DIR}/lyrics"
cp "${ROOT_DIR}/scripts/lyrics-local" "${BIN_DIR}/lyrics-local"
cp "${ROOT_DIR}/lyricslib.py" "${BIN_DIR}/lyricslib.py"

chmod +x "${BIN_DIR}/lyrics" "${BIN_DIR}/lyrics-local" "${BIN_DIR}/lyrics-fetch-go"

echo "installed to ${BIN_DIR}"
