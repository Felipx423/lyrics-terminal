# Arquitetura

## Componentes

### lyrics
Comando principal.

Responsabilidades:
- detectar música atual;
- procurar `.lrc` local válido;
- rejeitar e quarentenar `.lrc` suspeito antes de renderizar;
- usar `lyrics-local` se houver `.lrc`;
- usar `sptlrx` se não houver;
- disparar `lyrics-fetch-go` em background.

### lyrics-local
Renderiza `.lrc` local válido acompanhando a posição do Spotify.

### lyrics-fetch-go
Busca letras sincronizadas, valida o cache local antes de reutilizá-lo e salva `.lrc` com escrita atômica.

Providers:
1. LRCLIB
2. NetEase via mapa
3. NetEase search
4. syncedlyrics CLI
