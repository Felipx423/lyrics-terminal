# Lyrics Terminal

<p align="center">
  <img src="assets/demo.gif" alt="Demonstração do Lyrics Terminal">
</p>

Lyrics Terminal é uma aplicação para Linux que exibe letras sincronizadas do Spotify diretamente no terminal, com detecção automática de música, cache local, fallback entre providers, diagnósticos de saúde e ferramentas de observabilidade.

Foi criado para quem quer uma experiência leve, bonita e focada no terminal, sem depender de overlays no navegador ou aplicações pesadas.

## Funcionalidades

* Letras sincronizadas para músicas tocando no Spotify
* Modo com janela dedicada no Kitty
* Modo no terminal atual
* Suporte a playlists contínuas
* Detecção automática de troca de música
* Suporte a pause e resume
* Cache local em arquivos `.lrc`
* Validação automática do cache
* Quarentena automática de letras inválidas
* Sistema de fallback entre providers
* Diagnóstico com `lyrics --health`
* Análise de falhas dos providers
* Logs rotativos
* Observabilidade em tempo de execução
* Versionamento com commit e build date

## Screenshot

<p align="center">
  <img src="assets/screenshot.png" alt="Screenshot do Lyrics Terminal">
</p>

## Arquitetura

O projeto é dividido em alguns componentes principais.

### lyrics

Comando principal usado pelo usuário.

Responsável por:

* monitorar o Spotify
* controlar o ciclo de vida da música atual
* gerenciar troca de faixas
* lidar com pause e resume
* controlar a renderização da letra
* integrar com o Kitty

### lyrics-fetch-go

Motor de busca e seleção de letras.

Responsável por:

* consultar providers
* escolher a melhor letra disponível
* validar metadados
* gerenciar cache
* registrar falhas
* gerar estatísticas

### lyricslib

Biblioteca Python compartilhada usada pelo runtime.

## Providers suportados

Ordem atual dos providers:

1. LRCLIB
2. NetEase Map
3. NetEase Search
4. syncedlyrics

Nos testes reais, o LRCLIB tem sido o provider mais consistente e responsável pela maior parte dos sucessos.

## Instalação

Clone o repositório:

```bash
git clone https://github.com/Felipx423/lyrics-terminal.git
cd lyrics-terminal
```

Execute o instalador:

```bash
chmod +x install.sh
./install.sh
```

## Uso

Abrir no modo padrão, com janela dedicada no Kitty:

```bash
lyrics
```

Rodar no terminal atual:

```bash
lyrics --current
```

Abrir explicitamente no modo Kitty:

```bash
lyrics --kitty
```

Mostrar versão:

```bash
lyrics --version
```

Rodar diagnóstico:

```bash
lyrics --health
```

## Diagnóstico

O comando de diagnóstico verifica:

* Spotify
* playerctl
* Kitty
* sptlrx
* lyrics-fetch-go
* diretórios de cache
* configuração básica do ambiente

Exemplo:

```bash
lyrics --health
```

## Observabilidade

O projeto registra eventos importantes de execução para facilitar debug e análise de problemas reais.

Arquivo de log:

```text
~/.cache/lyrics-terminal/lyrics.log
```

Eventos registrados incluem:

* startup
* track_detected
* track_changed
* provider_selected
* provider_rejected
* fetch_success
* fetch_failure
* cache_hit
* cache_miss
* cache_invalid
* quarantine
* spotify_paused
* spotify_resumed

Rotação de logs:

* 5 arquivos mantidos
* 5 MB por arquivo

## Análise de falhas

O motor de busca registra falhas dos providers para análise posterior.

Rodar análise:

```bash
lyrics-fetch-go --analyze-failures
```

Arquivo de falhas:

```text
~/.cache/lyrics-terminal/failures.jsonl
```

Categorias analisadas:

* timeout
* letra não encontrada
* diferença de duração
* diferença de artista
* diferença de título
* letra não sincronizada
* cache inválido
* provider indisponível

## Estatísticas

Ver estatísticas do sistema:

```bash
lyrics-fetch-go --stats
```

As estatísticas incluem:

* quantidade de letras no cache local
* quantidade de arquivos em quarentena
* uso por provider
* taxa de sucesso
* últimos resultados

## Sistema de cache

As letras são salvas localmente em arquivos `.lrc`.

Entradas válidas do cache são reutilizadas automaticamente.

Arquivos inválidos são movidos para:

```text
~/.local/share/lyrics/bad/
```

Isso evita que letras corrompidas ou incorretas continuem afetando execuções futuras.

## Solução de problemas

### Nenhuma letra aparece

Verifique se o Spotify está tocando:

```bash
playerctl metadata
```

Depois rode:

```bash
lyrics --health
```

### Letra errada aparece

O projeto valida o cache automaticamente e move arquivos inválidos para quarentena.

Também é possível analisar falhas com:

```bash
lyrics-fetch-go --analyze-failures
```

### A janela do Kitty não abre

Verifique se o Kitty está instalado e disponível no PATH:

```bash
kitty --version
```

## Status do projeto

Versão atual:

**v0.5.0-observability**

O projeto está em fase de teste real, com foco em:

* letras incorretas
* músicas sem letra
* confiabilidade dos providers
* problemas de UX
* estabilidade em uso contínuo

Novas features grandes estão sendo deixadas para depois até o comportamento real do sistema ficar mais sólido.

## Licença

Este projeto está licenciado sob a licença MIT.

Consulte o arquivo LICENSE para mais detalhes.
