#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${HOME}/.local/bin"
DATA_DIR="${HOME}/.local/share/lyrics"
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
mkdir -p "${DATA_DIR}"

VERSION_INFO_FILE="${DATA_DIR}/build-info.json"
VERSION_VALUE="$(git -C "${ROOT_DIR}" describe --tags --always --dirty 2>/dev/null || echo dev)"
COMMIT_VALUE="$(git -C "${ROOT_DIR}" rev-parse HEAD 2>/dev/null || echo unknown)"
BUILD_DATE_VALUE="$(date -Iseconds)"

cat > "${VERSION_INFO_FILE}" <<EOF
{
  "version": "${VERSION_VALUE}",
  "commit": "${COMMIT_VALUE}",
  "build_date": "${BUILD_DATE_VALUE}"
}
EOF

echo "building lyrics-fetch-go"
(cd "${GO_DIR}" && go build -o "${BIN_DIR}/lyrics-fetch-go" .)

cp "${ROOT_DIR}/scripts/lyrics" "${BIN_DIR}/lyrics"
cp "${ROOT_DIR}/scripts/lyrics-local" "${BIN_DIR}/lyrics-local"
cp "${ROOT_DIR}/lyricslib.py" "${BIN_DIR}/lyricslib.py"

chmod +x "${BIN_DIR}/lyrics" "${BIN_DIR}/lyrics-local" "${BIN_DIR}/lyrics-fetch-go"

echo "installed to ${BIN_DIR}"
