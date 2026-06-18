# Visão Geral

Projeto: Lyrics Terminal

Objetivo:
Criar um sistema de letras sincronizadas para Spotify no Linux.

Fluxo principal:
- `lyrics` é o comando usado no dia a dia.
- Se existir `.lrc` local, usa `lyrics-local`.
- Se não existir, usa `sptlrx`.
- Em background, `lyrics-fetch-go` tenta baixar `.lrc`.

Pastas:
- Letras locais: `~/.local/share/lyrics/`
- Cache: `~/.cache/lyrics-terminal/`
