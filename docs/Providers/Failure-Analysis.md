# Failure Analysis

> Status: Documento de referência  
> Última revisão: 2026-06-19  
> Fonte de verdade: código do repositório, logs locais e dados persistidos  
> Documento relacionado: [[Provider-validation]]

## Objetivo

Este documento explica como investigar falhas registradas pelo `lyrics-fetch-go` e pelo runtime.

O objetivo não é apenas descobrir que uma música falhou. É classificar a falha corretamente:

- ausência real de cobertura;
    
- provider lento ou indisponível;
    
- metadados incompatíveis;
    
- candidato rejeitado;
    
- cache inválido;
    
- letra errada aceita;
    
- comportamento interrompido por troca de faixa ou encerramento manual.
    

## Fontes de Evidência

A análise deve começar pelos dados persistidos, não por suposições.

Arquivos e diretórios relevantes:

```text
~/.cache/lyrics-terminal/lyrics.log
~/.cache/lyrics-terminal/candidate_evaluations.jsonl
~/.cache/lyrics-terminal/index.json
~/.cache/lyrics-terminal/failures.jsonl
~/.cache/lyrics-terminal/negative/
~/.local/share/lyrics/
~/.local/share/lyrics/bad/
```

Função de cada fonte:

|Fonte|Uso principal|
|---|---|
|`lyrics.log`|Linha do tempo do runtime, métricas do launcher e eventos por faixa|
|`candidate_evaluations.jsonl`|Diagnóstico estruturado de candidatos avaliados pelos providers|
|`index.json`|Estado recente de cache e resultados persistidos|
|`failures.jsonl`|Histórico acumulado de falhas|
|`negative/`|Resultados negativos e tentativas sem sucesso|
|`lyrics/`|Arquivos `.lrc` reutilizáveis|
|`lyrics/bad/`|Arquivos rejeitados e quarentenados|

## Comandos de Diagnóstico

Use estes comandos antes de apagar cache, trocar provider ou alterar matching:

```bash
lyrics --health
lyrics-fetch-go --stats
lyrics-fetch-go --analyze-failures
```

Durante reprodução manual, use:

```bash
lyrics --debug --run
```

Para filtrar eventos estruturados:

```bash
grep -aE 'cache_hit_initial|cache_miss_initial|fetch_spawned|sptlrx_started|sptlrx_no_output|cache_appeared_after_fetch|track_result' \
  ~/.cache/lyrics-terminal/lyrics.log | tail -n 80
```

Para investigar por que um candidato foi aceito ou rejeitado, consulte também:

```bash
grep -aE 'candidate_evaluated|cache_provenance_missing' \
  ~/.cache/lyrics-terminal/candidate_evaluations.jsonl | tail -n 80
```

## Categorias de Falha

### `timeout`

O provider ou fallback não retornou resultado dentro do tempo esperado.

Investigar:

- duração total da tentativa;
    
- provider ou componente envolvido;
    
- se houve `sptlrx_no_output`;
    
- se o cache apareceu depois;
    
- se a faixa mudou antes de terminar.
    

Possíveis causas:

- provider lento;
    
- falha de rede;
    
- timeout agressivo;
    
- provider indisponível;
    
- faixa sem cobertura.
    

---

### `no_result`

Nenhum provider retornou letra sincronizada válida.

Investigar:

- metadados enviados;
    
- providers tentados;
    
- existência da letra em fontes conhecidas;
    
- cache negativo;
    
- possíveis diferenças de título, artista, remix ou versão.
    

Não tratar automaticamente como bug. Pode ser ausência legítima de cobertura.

---

### `artist_mismatch`

O candidato tinha título ou duração parecidos, mas artista incompatível.

Investigar:

- artista principal da faixa alvo;
    
- participações e nomes alternativos;
    
- se o artista apareceu apenas em álbum, título ou campo secundário;
    
- normalização aplicada;
    
- candidato e provider responsável.
    

Esse tipo de falha é relevante para reduzir letras erradas.

---

### `title_mismatch`

O candidato tinha artista ou duração parecidos, mas título incompatível.

Investigar:

- uso de prefixo, `contains` ou normalização excessiva;
    
- títulos curtos ou comuns;
    
- versão live, remix, acústica, edit ou remaster;
    
- diferença entre título usado para busca e título usado para aceitação.
    

---

### `duration_mismatch`

O candidato tinha duração distante da faixa alvo.

Investigar:

- duração do Spotify;
    
- duração retornada pelo provider;
    
- delta de duração;
    
- tolerância aplicada;
    
- possibilidade de versão ao vivo, remix ou faixa diferente.
    

Duração deve ajudar a validar, mas não deve ser a única prova de identidade.

---

### `unsynced_result`

O provider retornou texto sem timestamps ou timestamps inválidos.

Investigar:

- se o resultado tinha formato `.lrc`;
    
- se havia linhas sincronizadas utilizáveis;
    
- se o provider retornou apenas letra textual;
    
- se a fonte deve continuar sendo usada como fallback.
    

Resultado sem sincronização não deve ser salvo como `.lrc`.

---

### `invalid_cache`

Um `.lrc` local foi rejeitado antes de renderização.

Investigar:

- arquivo original;
    
- motivo da rejeição;
    
- localização na quarentena;
    
- se havia timestamps;
    
- se havia linhas úteis;
    
- se o idioma ou estrutura eram suspeitos.
    

Arquivos inválidos devem ficar em:

```text
~/.local/share/lyrics/bad/
```

---

### `wrong_lyrics`

Uma letra foi aceita, exibida ou reutilizada, mas pertence a outra faixa, versão ou artista.

Investigar:

- provider vencedor;
    
- cache ou fetch novo;
    
- metadados alvo;
    
- metadados do candidato;
    
- score ou regra de matching;
    
- duração;
    
- versão da faixa;
    
- proveniência salva no cache;
    
- se o problema é reproduzível.
    

Este é um problema de alta prioridade porque uma letra errada pode contaminar cache e reaparecer em execuções futuras.

---

### `provider_unavailable`

O provider falhou por indisponibilidade, bloqueio, erro HTTP ou problema externo.

Investigar:

- código de erro;
    
- tempo da tentativa;
    
- recorrência;
    
- se os outros providers funcionaram;
    
- se o problema ocorre apenas naquele provider.
    

Não corrigir isso alterando matching. Indisponibilidade e matching são problemas diferentes.

---

### `interrupted`

A execução foi encerrada manualmente ou pela troca de faixa antes de existir resultado final.

Investigar:

- `track_changed`;
    
- `track_result`;
    
- presença de cache;
    
- presença de output do `sptlrx`;
    
- encerramento manual do processo.
    

Esse resultado não deve ser classificado como falha de provider sem evidência adicional.

## Como Investigar uma Letra Errada

Quando uma letra errada aparecer, registrar:

```text
artista
título
álbum
duração
track_id
data e hora aproximada
veio de cache ou fetch novo
provider, quando disponível
comportamento observado
resultado esperado
logs relevantes
arquivo .lrc associado, se existir
```

Não registrar letra completa nos logs ou documentos de diagnóstico.

Fluxo recomendado:

1. Confirmar que o Spotify detectou a faixa correta.
    
2. Verificar se houve `cache_hit_initial`.
    
3. Verificar se o `.lrc` já existia antes da execução.
    
4. Consultar `index.json` e `failures.jsonl`.
    
5. Identificar provider e candidato aceito, quando disponível.
    
6. Comparar artista, título, álbum e duração.
    
7. Preservar o arquivo suspeito antes de apagar ou quarentenar.
    
8. Criar caso de teste antes de alterar validação.
    

## Como Interpretar Eventos do Runtime

|Sequência observada|Interpretação|
|---|---|
|`cache_hit_initial` → `track_result=cache_hit`|Cache local válido encontrado|
|`cache_miss_initial` → `cache_appeared_after_fetch` → `success_after_fetch`|Fetch lento, mas bem-sucedido|
|`sptlrx_no_output` → `no_output_timeout`|Fallback não retornou e cache não apareceu|
|`track_changed_before_result`|Faixa mudou antes de haver resultado final|
|`interrupted`|Processo foi encerrado manualmente|
|`live_fallback_only`|`sptlrx` exibiu saída, mas não houve cache local posterior|

## Limitações Conhecidas

- `index.json` representa estado recente por faixa, não histórico completo.
    
- Falhas antigas podem ser sobrescritas por resultados posteriores.
    
- `failures.jsonl` é mais útil para histórico.
    
- Eventos do launcher medem pipeline, não métricas completas por provider.
    
- Nem todo provider fornece metadados suficientes para confirmar identidade musical.
    
- Cache estruturalmente válido ainda pode estar semanticamente errado.
    

## Exemplo Histórico

Em uma execução anterior, o analisador encontrou uma entrada como:

```text
Faixa: Aimar - LINGERIE
Origem registrada: local-cache
Categoria: invalid_cache
Ação: arquivo local movido para quarentena
```

Esse registro serve apenas como exemplo de diagnóstico.

O estado atual deve sempre ser obtido executando:

```bash
lyrics-fetch-go --analyze-failures
```

## Regras de Manutenção

Ao alterar análise de falhas:

1. não apagar evidências silenciosamente;
    
2. preservar compatibilidade com registros antigos;
    
3. adicionar testes para novas categorias;
    
4. não confundir erro de provider com ausência de cobertura;
    
5. atualizar [[Bugs]] quando houver um caso real relevante;
    
6. atualizar [[Provider-validation]] quando a mudança afetar aceitação de candidatos.
