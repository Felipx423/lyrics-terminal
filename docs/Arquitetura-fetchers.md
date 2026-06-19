# Arquitetura do Fetcher

> Escopo: `lyrics_fetch_go/`, providers, validação de candidatos, cache e diagnósticos.  
> Fonte de verdade: código do repositório.  
> Última revisão: 2026-06-19.

## Objetivo

O fetcher procura letras sincronizadas, avalia candidatos, salva resultados aceitos e preserva evidências suficientes para investigar falhas.

Ele não renderiza letras nem controla janelas de terminal.

## Pipeline

```
Metadados da faixa
        ↓
Busca de cache local
        ↓
Cache válido?
 ┌──────┴──────┐
Sim             Não
↓               ↓
Retorna cache   Consulta providers
                ↓
          Avalia candidatos
                ↓
         Candidato aceito?
          ┌─────┴─────┐
          Sim         Não
          ↓           ↓
    Salva .lrc      Registra falha
    Atualiza índice
```

## Ordem de Providers

A ordem deve ser confirmada em `lyrics_fetch_go/providers.go`.

Ordem atual esperada:

1. LRCLIB
2. NetEase Map
3. NetEase Search
4. syncedlyrics CLI

A ordem é importante porque um provider aceito encerra a busca atual.

## Responsabilidades

O fetcher deve:

- buscar resultados de providers;
- validar metadados e timestamps;
- evitar persistir resultados estruturalmente inválidos;
- salvar `.lrc` com escrita segura;
- atualizar índice de cache;
- registrar diagnósticos e falhas;
- permitir análise posterior via comandos próprios.

O fetcher não deve:

- assumir que qualquer resultado sincronizado está correto;
- aceitar cache apenas porque o arquivo existe;
- esconder rejeições importantes;
- depender de stdout como única fonte de diagnóstico.

## Cache e Quarentena

Locais principais:

```
~/.local/share/lyrics/
~/.local/share/lyrics/bad/
~/.cache/lyrics-terminal/
```

Um `.lrc` é rejeitado quando for, por exemplo:

- vazio;
- sem timestamps parseáveis;
- sem linhas úteis;
- incompatível com a faixa em critérios estruturais;
- suspeito de idioma incompatível quando houver sinal forte.

Arquivos rejeitados devem ser movidos para quarentena, não apagados silenciosamente.

## Risco Conhecido

O maior risco atual não é apenas “não encontrar letra”.

É aceitar uma letra errada com metadados parecidos e salvá-la no cache.

Títulos curtos, versões ao vivo, remasters, remixes e artistas com participações aumentam esse risco.

As regras e hipóteses de correção ficam em [[Provider-validation]].

## Observabilidade Desejada

Cada aceitação relevante deve ser explicável por metadados, sem registrar letras completas.

Informações desejadas:

```
provider
source_id
target track metadata
candidate metadata
score
match type
accepted or rejected
rejection reasons
cache provenance
```

Métricas individuais de provider ainda não são consideradas completas enquanto o projeto não puder atribuir sucesso, rejeição, timeout ou falha a LRCLIB, NetEase Map, NetEase Search e syncedlyrics.

## Regras de Mudança

Ao mudar o fetcher:

1. criar caso de teste de regressão;
2. não relaxar matching sem evidência;
3. não endurecer matching sem medir impacto em cobertura;
4. preservar compatibilidade com cache legado;
5. registrar mudanças de comportamento em [[Decisoes]].