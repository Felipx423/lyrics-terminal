# Provider Improvement Plan

## Objetivo

Avaliar fontes de letras para o `lyrics-fetch-go` com foco em:

- maior cobertura de letras sincronizadas;
- menor risco de salvar `.lrc` errado;
- boa viabilidade para CLI;
- boa observabilidade via `--stats` e `--dry-run`.

## Estado atual do fetcher

O fetcher já usa esta ordem:

1. LRCLIB
2. NetEase via mapa
3. NetEase search
4. `syncedlyrics` CLI

O código atual só salva `.lrc` quando há timestamp real. Isso é o ponto certo para manter segurança.

## Critérios de avaliação

- `sync`: entrega letra com timestamps reais.
- `plain`: entrega apenas texto.
- `CLI`: dá para integrar de forma estável em um binário de terminal.
- `risk`: chance de letra errada, scraping frágil ou bloqueio.
- `effort`: esforço esperado de implementação/manutenção.
- `veredito`: usar, não usar, ou usar só como fallback manual.

## Avaliação por fonte

### LRCLIB

- `sync`: sim.
- `CLI`: sim, via API HTTP direta.
- `risk`: baixo a médio.
- `effort`: baixo.
- `limitações`: cobertura depende do catálogo disponível; ainda pode haver candidatos errados se a checagem for frouxa.
- `veredito`: usar.

Observação:
- já é uma das melhores bases para o projeto porque retorna `.lrc` sincronizado e encaixa bem no fluxo atual.

### NetEase

- `sync`: sim.
- `CLI`: sim, via HTTP.
- `risk`: médio.
- `effort`: médio.
- `limitações`: a busca por ID/mapa e por texto pode falhar em artistas não chineses, versões ao vivo e remixes; matching precisa ser conservador.
- `veredito`: usar.

Observação:
- hoje ele funciona como boa segunda camada para casos que LRCLIB não cobre.

### syncedlyrics CLI

- `sync`: depende do provider consultado.
- `plain`: sim, e isso é importante.
- `CLI`: sim, já existe.
- `risk`: médio.
- `effort`: baixo, porque já está pronto.
- `limitações`: como agregador, o comportamento depende dos backends configurados; alguns backends retornam texto sem timestamps e não devem virar `.lrc`.
- `veredito`: usar como fallback, com política estrita para só salvar quando houver timestamps.

Observação:
- o pacote documenta providers como Musixmatch, LRCLIB, NetEase, Megalobiz e Genius, mas também avisa quando algo está quebrado ou plain-only.

### Genius / SwagLyrics

- `sync`: normalmente não; é majoritariamente texto.
- `plain`: sim.
- `CLI`: sim, mas o fluxo é mais próximo de um app de texto/browser do que de um gerador de `.lrc`.
- `risk`: alto.
- `effort`: médio.
- `limitações`: exige atenção ao ToS da Genius; o próprio SwagLyrics avisa para checar os termos de uso. Não é uma base confiável para salvar `.lrc` sincronizado.
- `veredito`: usar só como fallback manual, nunca como fonte confiável de `.lrc`.

### Letras.mus.br

- `sync`: em geral texto, não um contrato público estável de `.lrc`.
- `plain`: sim.
- `CLI`: não vi API pública oficial adequada para integração direta no audit.
- `risk`: alto.
- `effort`: alto.
- `limitações`: tende a depender de scraping ou integração não documentada; isso cria fragilidade técnica e risco operacional.
- `veredito`: não usar por enquanto.

### Outros nomes relevantes

- Musixmatch:
  - `sync`: sim;
  - `CLI`: viável via `syncedlyrics`;
  - `risk`: médio a alto por questões de acesso/licença;
  - `veredito`: investigar como próximo candidato real de cobertura.

- Megalobiz:
  - `sync`: pode existir em alguns casos;
  - `CLI`: viável via `syncedlyrics`;
  - `risk`: médio;
  - `veredito`: fallback secundário, abaixo de LRCLIB/NetEase.

- Deezer:
  - `sync`: o próprio `syncedlyrics` marca como não funcionando no momento;
  - `veredito`: não priorizar.

## Métricas que esta branch cria

Com `lyrics-fetch-go --stats` e `lyrics-fetch-go --dry-run --debug`, já dá para medir:

- quantidade de `.lrc` locais;
- quantidade de arquivos em quarentena;
- quantidade de entradas negativas no índice;
- origem dos últimos resultados;
- taxa aproximada de sucesso;
- vencedor provável sem escrever cache.

Isso cria a base para comparar qualquer provider novo antes de implementá-lo.

## Recomendação

### Próximo provider a implementar

**Musixmatch**, mas somente se houver uma integração compatível com o projeto e sem transformar o fluxo em scraping frágil.

Motivo:

- é um backend com potencial de cobertura alta;
- já aparece no ecossistema de agregadores de letras sincronizadas;
- pode aumentar a taxa de sucesso antes de pensarmos em providers mais frágeis.

### Não implementar agora

- Genius/SwagLyrics como fonte de `.lrc` sincronizado.
- Letras.mus.br via scraping.

Essas fontes podem servir como fallback manual de texto no futuro, mas não são a melhor próxima etapa para `.lrc`.

## Fontes consultadas

- [syncedlyrics no PyPI](https://pypi.org/project/syncedlyrics/)
- [swaglyrics no PyPI](https://pypi.org/project/swaglyrics/)
- [LRCLIB](https://lrclib.net/)
