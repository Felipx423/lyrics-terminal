# Validação de Providers e Candidatos

## Objetivo

Este documento registra como o projeto decide se uma letra retornada por um provider é aceitável.

O maior risco atual é aceitar uma letra errada, salvá-la no cache e reutilizá-la como se fosse confiável.

## Fluxo de decisão

```text
Faixa do Spotify
        ↓
Busca em provider
        ↓
Candidatos retornados
        ↓
Normalização e comparação
        ↓
Validação de título, artista, duração e timestamps
        ↓
Candidato aceito ou rejeitado
        ↓
Se aceito: salva .lrc e atualiza cache
```

## Dados da faixa alvo

Sempre que possível, a validação deve usar:

- artista;
    
- título;
    
- álbum;
    
- duração;
    
- `track_id`;
    
- versão da faixa, como live, remaster, acústica ou edit.
    

Esses dados também devem aparecer em logs e diagnósticos para permitir investigação posterior.

## Estado atual e riscos conhecidos

O código atual possui heurísticas permissivas em alguns caminhos de validação.

Riscos observados:

- título pode ser aceito por igualdade, prefixo ou `contains`;
    
- artista pode ser considerado compatível por aparecer em título, artista ou álbum;
    
- duração pode aceitar diferença de até aproximadamente 15 segundos;
    
- tags como `live`, `remaster`, `edit`, `version` e `acoustic` podem ser removidas durante normalização;
    
- `syncedlyrics` pode fornecer letra com poucos metadados verificáveis;
    
- um candidato aceito pode ser salvo imediatamente no cache;
    
- cache antigo pode conter letras estruturalmente válidas, mas semanticamente erradas.
    

Essas regras aumentam cobertura, mas também aumentam risco de falso positivo.

## Caso conhecido

Caso reportado:

```text
Artista: Ryxn Pablo
Título: Ainda
Álbum: Ainda
Duração: 165 segundos
Spotify track ID: 38YZseF2ALmg58eVQ9r2mZ
Problema: letra errada foi exibida apesar de o fetch ter sido considerado bem-sucedido.
```

Hipóteses principais:

1. título curto e comum aceito por comparação parcial;
    
2. artista incompatível aceito por aparecer em metadados secundários;
    
3. versão alternativa aceita após remoção excessiva de tags;
    
4. resultado de NetEase Search com score parcial;
    
5. resultado de syncedlyrics sem identidade forte;
    
6. cache antigo ou contaminado reutilizado.
    

A causa não deve ser considerada confirmada sem diagnóstico do provider vencedor e do candidato aceito.

## Separação obrigatória de normalização

Existem dois usos diferentes para normalização:

### Normalização para busca

Pode ser mais tolerante.

Exemplos:

- remover pontuação;
    
- ignorar caixa alta/baixa;
    
- tratar acentos;
    
- remover tags de versão para ampliar busca.
    

### Normalização para aceitação

Deve ser mais rígida.

Princípios:

- título precisa ser compatível de forma forte;
    
- artista principal precisa ser compatível diretamente;
    
- álbum não deve provar artista sozinho;
    
- duração deve ser sinal complementar, não justificativa principal;
    
- versões live, remaster ou acústicas não devem ser aceitas automaticamente como equivalentes;
    
- ausência de metadados deve reduzir confiança.
    

## Regras desejadas para evolução futura

Antes de persistir uma letra no cache, o projeto deve conseguir explicar:

- qual provider retornou o candidato;
    
- qual identificador de origem foi usado;
    
- quais metadados do candidato foram comparados;
    
- qual regra aceitou ou rejeitou o candidato;
    
- qual score foi atribuído;
    
- por que o cache foi salvo.
    

A decisão precisa ser auditável sem registrar a letra completa.

## Diagnóstico recomendado

Para cada candidato avaliado, registrar apenas metadados:

```text
provider
source_id
target_track_id
target_artist
target_title
target_album
target_duration_ms
candidate_artist
candidate_title
candidate_album
candidate_duration_ms
title_match_type
artist_match_type
duration_delta_ms
score
accepted
rejection_reasons
```

Para cache salvo, preservar proveniência mínima:

```text
provider
source_id
candidate_artist
candidate_title
candidate_album
candidate_duration_ms
score
validation_version
accepted_at
```

## Testes obrigatórios

Toda mudança de validação deve cobrir pelo menos:

1. mesmo título, artista diferente, duração próxima → rejeitar;
    
2. mesma música, versão live/remaster diferente → não aceitar automaticamente;
    
3. score alto sem validação final forte → rejeitar;
    
4. resultado sincronizado sem identidade suficiente → não salvar;
    
5. candidato errado não deve ser persistido no cache;
    
6. cache legado sem proveniência continua funcionando sem ser tratado como confirmação semântica.
    

## Regra de produto

Entre:

```text
mostrar uma letra possivelmente errada
```

e:

```text
não mostrar letra
```

o projeto deve preferir não mostrar letra quando a identidade da faixa não puder ser comprovada com confiança suficiente.