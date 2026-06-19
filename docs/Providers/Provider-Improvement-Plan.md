# Provider Improvement Plan

> Status: Documento estratégico  
> Última revisão: 2026-06-19  
> Fonte de verdade: código do repositório, métricas locais e decisões registradas  
> Documento relacionado: [[Provider-validation]]

## Objetivo

Definir como o projeto avalia, prioriza, adiciona ou remove providers de letras sincronizadas.

O objetivo não é ter o maior número possível de providers.

O objetivo é melhorar cobertura sem aumentar de forma irresponsável:

- letras erradas;
    
- cache contaminado;
    
- dependências frágeis;
    
- scraping instável;
    
- manutenção desnecessária;
    
- falhas difíceis de diagnosticar.
    

## Decisão Atual

Nenhum provider novo deve ser implementado durante a fase **v0.6.0 — Real World Testing**.

Antes de considerar Musixmatch, Megalobiz ou qualquer outra fonte, o projeto precisa:

1. investigar casos de letra errada da issue #1;
    
2. registrar proveniência de candidatos aceitos;
    
3. coletar dados reais de falha e sucesso;
    
4. diferenciar falta de cobertura de falha técnica;
    
5. garantir que cache não perpetue letras semanticamente erradas;
    
6. avaliar manutenção, estabilidade e viabilidade da integração.
    

Provider novo é hipótese de pesquisa, não tarefa automática.

## Providers Atuais

A ordem deve ser confirmada em `lyrics_fetch_go/providers.go`.

|Provider|Papel atual|Risco principal|
|---|---|---|
|LRCLIB|Fonte principal|Cobertura limitada ou candidatos próximos|
|NetEase Map|Busca por mapeamento|Dados inconsistentes ou versões alternativas|
|NetEase Search|Busca e ranking por texto|Matching parcial e falsos positivos|
|syncedlyrics CLI|Fallback agregador|Metadados incompletos e comportamento dependente de backend|

## Regras de Aceitação

Um provider só é útil se o projeto puder responder:

- qual faixa foi buscada;
    
- qual candidato foi encontrado;
    
- por que foi aceito;
    
- por que candidatos alternativos foram rejeitados;
    
- qual provider venceu;
    
- o que foi salvo no cache;
    
- como reproduzir ou investigar uma falha.
    

Resultado sincronizado não é sinônimo de resultado correto.

## Critérios para Avaliar Provider

Todo provider novo deve ser avaliado nos critérios abaixo.

### Sincronização

Pergunta:

> O provider fornece timestamps utilizáveis e confiáveis?

Resultados sem timestamps não devem virar `.lrc` automático.

### Identidade Musical

Pergunta:

> O provider fornece metadados suficientes para confirmar artista, título, álbum, versão e duração?

Quanto menos metadados existir, maior deve ser a exigência de validação.

### Cobertura

Pergunta:

> O provider resolve casos reais que os providers atuais não resolvem?

Cobertura precisa ser medida com amostras reais, não estimada por fama do serviço.

### Qualidade

Pergunta:

> O provider reduz falsos negativos sem aumentar falsos positivos?

Uma fonte que encontra mais letras, mas erra artista ou versão, pode piorar o projeto.

### Estabilidade

Pergunta:

> A integração é previsível, documentada e sustentável?

Evitar dependência que só funciona por scraping frágil ou comportamento não documentado.

### Diagnóstico

Pergunta:

> O provider permite registrar source_id, metadados, duração, score e motivo de aceitação?

Provider que não pode ser auditado deve ser tratado como fallback de risco elevado.

### Manutenção

Pergunta:

> A integração adiciona complexidade proporcional ao ganho real?

Não adicionar biblioteca, CLI externa ou camada de scraping sem evidência de benefício.

## Estado Atual de Métricas

O projeto já possui:

- estatísticas de cache e índice;
    
- análise de falhas persistidas;
    
- eventos estruturados por faixa no launcher;
    
- logs humanos no modo `--debug`;
    
- validação e quarentena de cache local;
    
- comandos de diagnóstico como `--stats` e `--analyze-failures`.
    

O projeto ainda não possui métricas completas e confiáveis por provider individual.

Antes de comparar providers com segurança, o fetcher precisa registrar para cada candidato:

```text
provider
source_id
target track metadata
candidate metadata
score
match type
accepted
rejection reasons
duration delta
cache provenance
```

## Avaliação dos Providers Atuais

### LRCLIB

Decisão atual: manter.

Pontos positivos:

- encaixa bem em busca de letras sincronizadas;
    
- pode retornar metadados úteis;
    
- já está integrado ao fluxo principal.
    

Riscos:

- cobertura pode ser limitada;
    
- candidatos próximos ainda precisam passar por validação forte;
    
- versão errada ou faixa parecida não deve ser aceita apenas por título semelhante.
    

Prioridade:

- manter como provider principal;
    
- melhorar diagnóstico de candidatos antes de mudar regra de matching.
    

---

### NetEase Map

Decisão atual: manter e observar.

Pontos positivos:

- pode resolver faixas que não aparecem em outras fontes;
    
- busca por identificador pode reduzir ambiguidade em alguns casos.
    

Riscos:

- mapeamento pode apontar para versão diferente;
    
- dados podem variar entre catálogo local e Spotify;
    
- precisa de evidência de correspondência antes de salvar cache.
    

Prioridade:

- coletar métricas de sucesso e rejeição;
    
- investigar casos de mismatch.
    

---

### NetEase Search

Decisão atual: manter com cautela.

Pontos positivos:

- aumenta chance de encontrar cobertura;
    
- permite ranking de múltiplos candidatos.
    

Riscos:

- título curto e artista parcial aumentam falsos positivos;
    
- score alto não deve substituir validação final;
    
- versões live, remix e edit exigem atenção.
    

Prioridade:

- registrar candidatos e scores;
    
- endurecer critérios somente após evidência real.
    

---

### syncedlyrics CLI

Decisão atual: manter apenas como fallback.

Pontos positivos:

- pode ampliar cobertura por agregar fontes;
    
- já existe no ambiente do projeto;
    
- funciona como alternativa quando providers diretos falham.
    

Riscos:

- comportamento depende de backend;
    
- pode retornar letra sem metadados suficientes;
    
- pode retornar texto sem timestamps;
    
- não deve salvar resultado apenas porque contém timestamps.
    

Prioridade:

- manter política estrita de validação;
    
- registrar origem quando possível;
    
- não confiar cegamente em saída agregada.
    

## Candidatos para Pesquisa Futura

### Musixmatch

Status: candidato de pesquisa.

Possível valor:

- pode aumentar cobertura de letras sincronizadas.
    

Riscos:

- integração, acesso e manutenção precisam ser avaliados;
    
- cobertura maior não compensa se aumentar falsos positivos;
    
- não deve entrar sem evidência de ganho em casos reais.
    

Condição para avançar:

- comparar amostra real de faixas sem cobertura;
    
- validar metadados e sincronização;
    
- definir forma sustentável e auditável de integração.
    

---

### Megalobiz

Status: fallback secundário de pesquisa.

Possível valor:

- pode oferecer letras sincronizadas em alguns casos.
    

Riscos:

- cobertura e estabilidade precisam ser verificadas;
    
- pode exigir validação mais rígida;
    
- não deve ser usado como solução automática para qualquer ausência de letra.
    

---

### Genius, SwagLyrics e Fontes de Texto

Status: não usar como fonte automática de `.lrc`.

Motivo:

- foco principal em letra textual;
    
- ausência comum de timestamps;
    
- risco de depender de scraping ou comportamento instável;
    
- não resolvem, por si só, o problema de sincronização.
    

Podem ser considerados apenas em funcionalidades futuras de consulta manual de texto, nunca como fonte automática para cache sincronizado.

---

### Letras.mus.br e Fontes Sem API Estável

Status: não priorizar.

Motivo:

- integração pode depender de scraping;
    
- manutenção e risco operacional são altos;
    
- não há evidência de que resolvam o gargalo atual de letras sincronizadas.
    

## Processo para Adicionar um Provider

Nenhum provider novo entra sem cumprir as etapas abaixo.

### 1. Identificar Lacuna Real

Registrar faixas que falharam nos providers atuais.

A amostra precisa incluir:

- artista;
    
- título;
    
- duração;
    
- gênero ou idioma;
    
- resultado atual;
    
- provider já tentado;
    
- existência confirmada de letra sincronizada em outra fonte, quando conhecida.
    

### 2. Fazer Prova de Cobertura

Testar o provider candidato apenas contra a amostra de falhas reais.

Avaliar:

- quantas faixas ele encontra;
    
- quantas possuem timestamps;
    
- quantas possuem metadados verificáveis;
    
- quantas seriam aceitas pelas regras atuais;
    
- quantas parecem ambiguidades ou falsos positivos.
    

### 3. Avaliar Integração

Confirmar:

- forma de acesso;
    
- dependências;
    
- timeout;
    
- tratamento de erro;
    
- necessidade de autenticação;
    
- confiabilidade;
    
- manutenção esperada.
    

### 4. Projetar Diagnóstico

Antes de integrar, definir:

- campos de log;
    
- source_id;
    
- metadados de candidato;
    
- motivos de rejeição;
    
- formato de proveniência no cache;
    
- testes de regressão.
    

### 5. Implementar com Feature Isolada

A implementação deve:

- ficar isolada;
    
- ter timeout;
    
- não alterar comportamento dos providers existentes sem motivo;
    
- não salvar resultado sem validação;
    
- ter testes sem rede;
    
- permitir desativação simples se necessário.
    

### 6. Revisar com Dados Reais

Depois de uso real, comparar:

- ganho de cobertura;
    
- taxa de timeout;
    
- taxa de rejeição;
    
- falsos positivos;
    
- impacto em manutenção;
    
- impacto no cache.
    

## Critérios para Remover ou Rebaixar Provider

Um provider deve ser rebaixado ou removido quando:

- gera falsos positivos recorrentes;
    
- não oferece metadados suficientes;
    
- falha com frequência alta;
    
- aumenta tempo de resposta sem ganho real;
    
- depende de integração frágil;
    
- não contribui para cobertura mensurável;
    
- torna investigações mais difíceis.
    

## Regra Final

Não adicionar provider para compensar matching fraco.

Primeiro garantir que o sistema sabe rejeitar uma letra errada.

Depois medir cobertura.

Só então expandir fontes.