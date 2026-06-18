# Arquitetura

## Componentes

### lyrics
Comando principal.

Responsabilidades:
- detectar música atual;
- procurar `.lrc` local;
- usar `lyrics-local` se houver `.lrc`;
- usar `sptlrx` se não houver;
- disparar `lyrics-fetch-go` em background.

### lyrics-local
Renderiza `.lrc` local acompanhando a posição do Spotify.

### lyrics-fetch-go
Busca letras sincronizadas e salva `.lrc`.

Providers:
1. LRCLIB
2. NetEase via mapa
3. NetEase search
4. syncedlyrics CLI
