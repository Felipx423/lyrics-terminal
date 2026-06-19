## Projeto

**Lyrics Terminal** é um sistema de letras sincronizadas para Spotify no Linux, executado no terminal.

O projeto prioriza:

- letras sincronizadas quando disponíveis;
- cache local reutilizável;
- fallback ao vivo com `sptlrx`;
- diagnóstico de falhas;
- segurança contra cache inválido ou potencialmente errado.

O código é a fonte de verdade. Esta documentação existe para facilitar manutenção, decisões e onboarding.

## Estado Atual

- Fase atual: **v0.6.0 — Real World Testing**
- Release atual: **v0.5.0 — Observability**
- Prioridade técnica atual: investigar letras erradas, cobertura real dos providers e timeouts.

O projeto está funcional, mas ainda não deve ser tratado como estável.

## Fluxo Principal

```
Spotify / playerctl
        ↓
lyrics
        ↓
Procura .lrc local válido
        ↓
Cache encontrado?
 ┌──────┴──────┐
Sim             Não
↓               ↓
lyrics-local    sptlrx + lyrics-fetch-go em background
↓               ↓
Renderiza       Se o fetcher salvar um .lrc válido:
cache           reinicia usando lyrics-local
```

## Comandos Principais

```
lyrics
lyrics --kitty
lyrics --current
lyrics --health
lyrics --version
lyrics-fetch-go --stats
lyrics-fetch-go --analyze-failures
```

## Pastas Importantes

```
~/.local/share/lyrics/              Letras .lrc locais
~/.local/share/lyrics/bad/          Quarentena de cache suspeito
~/.cache/lyrics-terminal/           Logs, índice, falhas e diagnósticos
```

## Documentação

- [[Arquitetura]] — visão dos componentes e comunicação entre eles.
- [[Arquitetura-fetchers]] — fluxo do fetcher, providers, cache e persistência.
- [[Provider-validation]] — regras, riscos e critérios para aceitar candidatos.
- [[Bugs]] — bugs abertos, resolvidos e evidências conhecidas.
- [[Decisoes]] — decisões técnicas e consequências.
- [[Roadmap]] — prioridades por milestone.
- [[REAL_WORLD_TESTING]] — critérios para encerrar a fase v0.6.0.
- [[DEVELOPMENT]] — guia completo para desenvolvimento, testes e debugging.

## Regra de Produto

Entre mostrar uma letra possivelmente errada e não mostrar letra, o projeto deve preferir não mostrar letra quando não houver confiança suficiente na identidade da faixa.