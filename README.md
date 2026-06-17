# Lyrics Terminal

Sistema de letras sincronizadas para Spotify no Linux.

## Comandos

- `lyrics`: comando principal.
- `lyrics-local`: usa `.lrc` local sincronizado.
- `lyrics-fetch-go`: busca `.lrc` em providers externos e salva localmente.

## Fluxo

1. `lyrics` verifica se já existe `.lrc`.
2. Se existir, usa `lyrics-local`.
3. Se não existir, usa `sptlrx pipe`.
4. Em background, chama `lyrics-fetch-go` para tentar baixar a letra.

## Pastas

- Letras locais: `~/.local/share/lyrics/`
- Cache: `~/.cache/lyrics-terminal/`

## Dependências

- `kitty`
- `playerctl`
- `sptlrx`
- `go`, apenas para build
