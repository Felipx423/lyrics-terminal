# Failure Analysis

Data atualizada a partir do comando:

- `lyrics-fetch-go --analyze-failures`

## O que o analisador considera

- índice persistido em `~/.cache/lyrics-terminal/index.json`
- eventos persistidos em `~/.cache/lyrics-terminal/failures.jsonl`
- quarentena em `~/.local/share/lyrics/bad/`
- cache negativo em `~/.cache/lyrics-terminal/negative/`

## Estado atual observado

No estado local atual, o analisador encontrou uma falha persistida:

- `Aimar - LINGERIE`
- provider: `local-cache`
- categoria: `cache inválido`
- motivo: arquivo local quarentenado

Isso confirma o bug real de cache local errado que já havia aparecido no histórico do projeto.

## Categorias de falha

- `timeout`
- `letra inexistente`
- `mismatch de duração`
- `mismatch de artista`
- `mismatch de título`
- `resultado não sincronizado`
- `cache inválido`
- `provider indisponível`
- `outros`

## Leitura técnica

### Musixmatch

Provavelmente ajuda em:

- `letra inexistente`
- `resultado não sincronizado`

Motivo:

- é o tipo de fonte que costuma aumentar cobertura de letras quando LRCLIB, NetEase ou `syncedlyrics` não entregam uma versão sincronizada útil;
- ainda assim, não resolve cache local ruim nem falha de rede.

### Melhoria de ranking

Provavelmente ajuda em:

- `mismatch de duração`
- `mismatch de artista`
- `mismatch de título`
- parte dos casos de `cache inválido`

Motivo:

- os falsos positivos surgem quando um candidato passa por heurísticas frouxas ou quando um `.lrc` antigo/ruim é aceito sem validação suficiente;
- isso é problema de matching, não de cobertura.

### Problemas de rede

São os casos típicos de:

- `timeout`
- `provider indisponível`

Motivo:

- o provedor pode estar fora do ar, lento, bloqueado ou inacessível no ambiente;
- isso não se corrige com outro ranking.

### Sem solução prática

Casos em que:

- a letra realmente não existe em nenhum provider acessível;
- o provider só entrega texto sem timestamps e não há como converter isso em `.lrc` com segurança;
- a faixa está tão mal identificada que não há como afirmar a correspondência correta.

## Resumo de impacto

- Se o problema é cobertura: Musixmatch é o próximo candidato mais útil.
- Se o problema é matching: ajustar ranking e validação antes de adicionar mais providers.
- Se o problema é rede: tratar retry, timeout e indisponibilidade.
- Se o problema é dado errado em cache: invalidar, quarentenar e re-fetch.

## Observação importante

O índice atual guarda o estado mais recente por faixa. Isso significa que falhas antigas podem ser substituídas por resultados posteriores. Para análise histórica mais rica, o arquivo `failures.jsonl` é o ponto certo para acumular eventos futuros.

## Fontes consultadas

- https://pypi.org/project/syncedlyrics/
- https://pypi.org/project/swaglyrics/
- https://lrclib.net/
- https://genius.com/developers

