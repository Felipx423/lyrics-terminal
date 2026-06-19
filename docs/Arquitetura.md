# Arquitetura

## Objetivo

O projeto separa runtime, busca, cache e renderização para evitar que uma falha de provider derrube o terminal inteiro.

```
Spotify / MPRIS
        ↓
playerctl
        ↓
scripts/lyrics
        ├── lyrics-local
        ├── sptlrx
        └── lyrics-fetch-go
                ↓
          Providers + cache
```

## Componentes

### `scripts/lyrics`

Launcher principal.

Responsabilidades:

- detectar faixa e estado do Spotify;
- lidar com pause, resume e troca de música;
- escolher modo Kitty ou current-terminal;
- procurar cache local válido;
- iniciar `lyrics-fetch-go` em background;
- iniciar `sptlrx` como fallback;
- reiniciar a pipeline quando um `.lrc` aparece;
- registrar logs de debug e métricas por faixa.

### `lyrics-local`

Renderiza um `.lrc` local válido acompanhando a posição atual da faixa.

Ele não busca letras nem escolhe providers.

### `lyrics-fetch-go`

Fetcher responsável por:

- consultar providers;
- avaliar candidatos;
- validar cache;
- quarentenar arquivos suspeitos;
- salvar `.lrc`;
- atualizar índice e diagnósticos;
- expor `--stats`, `--dry-run`, `--analyze-failures` e limpeza de cache.

### `lyricslib.py`

Biblioteca Python compartilhada.

Responsabilidades:

- leitura de metadados por `playerctl`;
- localização de cache local;
- renderização de terminal;
- logs;
- helpers de validação e paths.

### `install.sh`

Responsável por:

- validar dependências;
- compilar `lyrics-fetch-go`;
- instalar scripts e binários em `~/.local/bin`;
- escrever metadados de build.

## Modos de Execução

### Modo padrão

```
lyrics
```

Abre janela dedicada do Kitty.

### Modo explícito Kitty

```
lyrics --kitty
```

Mesmo comportamento do modo padrão.

### Modo terminal atual

```
lyrics --current
```

Executa na janela atual sem depender de Kitty.

## Comunicação Entre Componentes

Os componentes não dependem de uma arquitetura monolítica.

Eles se comunicam por:

- subprocessos;
- arquivos `.lrc`;
- diretórios de cache;
- índice JSON;
- logs;
- arquivos de falha;
- diretório de quarentena.

## Cache

```
~/.local/share/lyrics/              Letras locais
~/.local/share/lyrics/bad/          Quarentena
~/.cache/lyrics-terminal/           Logs, índice e falhas
```

O cache só deve ser reutilizado após validação estrutural.

Uma letra errada semanticamente ainda é um risco conhecido. Ver [[Provider-validation]].

## Documentação Relacionada

- [[Arquitetura-fetchers]]
- [[Provider-validation]]
- [[DEVELOPMENT]]
- [[REAL_WORLD_TESTING]]