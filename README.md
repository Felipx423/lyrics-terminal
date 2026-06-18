# Lyrics Terminal

Sistema de letras sincronizadas para Spotify no Linux.

## Comandos

- `lyrics`: comando principal.
- `lyrics-local`: usa `.lrc` local sincronizado.
- `lyrics-fetch-go`: busca `.lrc` em providers externos e salva localmente.

## Fluxo

1. `lyrics` verifica se já existe `.lrc` local válido.
2. Se existir e passar na validação, usa `lyrics-local`.
3. Se não existir, usa `sptlrx pipe`.
4. Em background, chama `lyrics-fetch-go` para tentar baixar a letra.

Arquivos `.lrc` suspeitos são movidos para `~/.local/share/lyrics/bad/` e tratados como cache miss.

## Pastas

- Letras locais: `~/.local/share/lyrics/`
- Cache: `~/.cache/lyrics-terminal/`

## Dependências

- `kitty`
- `playerctl`
- `sptlrx`
- `go`, apenas para build
