# Lyrics Terminal

<p align="center">
  <img src="assets/demo.gif" alt="Demonstração do Lyrics Terminal">
</p>

[English](README.md) | [Português](README.pt-BR.md)

Lyrics Terminal é uma aplicação para Linux que mostra letras sincronizadas do Spotify no terminal. Ela detecta a faixa atual, reaproveita entradas válidas do cache local, faz fallback entre providers quando necessário e expõe ferramentas de saúde, estatísticas e análise de falhas para uso real.

O projeto foi pensado para quem quer uma experiência leve, focada no terminal, sem overlays no navegador ou aplicativos desktop pesados.

## Status Atual

Status do projeto: Real World Testing

Release atual: `v0.5.0 Observability`

Milestone atual: `v0.6.0 Real World Testing`

O projeto é funcional, mas ainda não está sendo apresentado como totalmente estável. Os dados de uso real continuam sendo coletados antes de ampliar o escopo com features maiores.

Feedback é bem-vindo, especialmente sobre:

- letras erradas
- músicas sem letra
- qualidade dos providers
- validação e quarentena de cache
- fluxo de instalação

## Demonstração

<p align="center">
  <img src="assets/screenshot.png" alt="Captura de tela do Lyrics Terminal">
</p>

## O Que Ele Faz

- Mostra letras sincronizadas da faixa atual no Spotify
- Detecta automaticamente as trocas de música
- Trata pause e resume
- Roda em uma janela dedicada do Kitty ou no terminal atual
- Reaproveita arquivos locais `.lrc` válidos
- Valida entradas do cache local antes de reutilizá-las
- Move arquivos `.lrc` inválidos ou suspeitos para quarentena
- Faz fallback entre vários providers
- Expõe comandos de saúde, estatísticas e análise de falhas
- Registra logs de execução para diagnóstico

## Requisitos

- Linux
- Spotify visível via MPRIS
- `python3`
- `playerctl`
- `kitty` para o modo padrão com janela dedicada e para `lyrics --kitty`
- Kitty não é necessário para `lyrics --current`
- `sptlrx` para o fallback de renderização em tempo real
- Toolchain do Go para compilar o `lyrics-fetch-go` durante a instalação

## Instalação

Clone o repositório:

```bash
git clone https://github.com/Felipx423/lyrics-terminal.git
cd lyrics-terminal
```

Instale o runtime:

```bash
chmod +x install.sh
./install.sh
```

O instalador compila `lyrics-fetch-go` e instala as peças do runtime em `~/.local/bin`.

## Uso Básico

Abra a visualização padrão das letras:

```bash
lyrics
```

Rode no terminal atual:

```bash
lyrics --current
```

Abra explicitamente o modo Kitty:

```bash
lyrics --kitty
```

Mostre as informações de versão:

```bash
lyrics --version
```

Rode uma verificação do ambiente:

```bash
lyrics --health
```

## Modos de Execução

### `lyrics`

Launcher padrão.

Comportamento:

- abre a janela dedicada do Kitty por padrão
- consulta a reprodução do Spotify via `playerctl`
- tenta primeiro o cache local
- inicia busca em segundo plano e o fallback de renderização quando necessário

### `lyrics --kitty`

Alias explícito do comportamento padrão baseado em Kitty.

Use este modo quando quiser deixar a intenção de janela dedicada clara em scripts ou anotações.

### `lyrics --current`

Executa o runtime no terminal atual, sem abrir o Kitty.

Use este modo quando quiser manter a saída na sessão ativa do terminal.
Ele não depende de Kitty.

### `lyrics --health`

Verifica se o ambiente está pronto.

Ele confere:

- `playerctl`
- `kitty`
- `sptlrx`
- `lyrics-fetch-go`
- diretórios de cache
- `index.json`
- o status do Spotify via `playerctl`

Se o Kitty estiver ausente, `lyrics --health` o mostra como `WARN`, porque `lyrics --current` ainda pode funcionar.

### `lyrics --version`

Mostra os metadados de build expostos pelo runtime Python.

## Diagnóstico

Use os diagnósticos do fetcher em Go quando precisar enxergar cache e providers.

```bash
lyrics-fetch-go --stats
lyrics-fetch-go --analyze-failures
```

`lyrics-fetch-go --stats` resume:

- arquivos locais `.lrc`
- arquivos em quarentena
- entradas de cache negativo
- contagens de providers e status
- buscas recentes

`lyrics-fetch-go --analyze-failures` inspeciona:

- eventos persistidos de falha
- o índice de busca
- o conteúdo da quarentena
- o conteúdo do cache negativo

Limpeza destrutiva do cache:

```bash
lyrics-fetch-go --clear-cache
```

Esse comando remove o diretório de cache do fetcher em `~/.cache/lyrics-terminal/`. Use apenas quando você realmente quiser apagar os dados de cache persistidos.

## Provedores

Ordem atual dos providers:

1. LRCLIB
2. NetEase Map
3. NetEase Search
4. syncedlyrics

Os providers são validados antes que um resultado seja aceito. Uma resposta tecnicamente válida ainda pode ser rejeitada se os metadados não baterem com a faixa atual ou se as letras sincronizadas parecerem suspeitas.

## Cache, Validação e Quarentena

Arquivos locais `.lrc` válidos são reutilizados depois da validação.

Caminhos atuais de cache e quarentena:

- cache e dados de diagnóstico: `~/.cache/lyrics-terminal/`
- cache local de letras: `~/.local/share/lyrics/`
- arquivos locais inválidos: `~/.local/share/lyrics/bad/`

Se um arquivo local `.lrc` estiver vazio, sem timestamps, sem linhas úteis de letra ou de outra forma suspeito, ele é movido para quarentena em vez de ser reutilizado.

Isso evita que letras quebradas ou erradas sejam reaproveitadas em execuções futuras.

## Limitações Conhecidas

- O projeto é focado em Linux
- O acesso ao Spotify depende de MPRIS e `playerctl`
- A cobertura dos providers é limitada pelas fontes atuais
- O modo Kitty exige que o Kitty esteja instalado
- O modo no terminal atual não oferece a mesma experiência de janela do Kitty
- O projeto ainda está em testes reais

## Relate Problemas e Contribua

Se você encontrar letras erradas, músicas sem letra, problemas de cache, falhas de instalação ou regressões de providers, abra uma issue com detalhes suficientes para reproduzir o caso.

Evidências úteis normalmente incluem:

- artista
- título
- provider usado
- se o resultado veio do cache ou de uma busca ao vivo
- logs relevantes de `~/.cache/lyrics-terminal/lyrics.log`
- saída da análise de falhas

Antes de mudanças maiores, leia a documentação de manutenção:

- [Guia de Desenvolvimento](docs/DEVELOPMENT.md)
- [Critérios de Saída do Real World Testing](docs/REAL_WORLD_TESTING.md)

[Abra uma issue](https://github.com/Felipx423/lyrics-terminal/issues)
