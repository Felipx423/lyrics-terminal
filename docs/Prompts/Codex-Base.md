# Prompt Base para Codex

Antes de alterar qualquer arquivo, entenda o contexto do projeto.

## Leitura obrigatória

Leia sempre:

- `docs/Visao-Geral.md`
    
- `docs/Arquitetura.md`
    
- `docs/Decisoes.md`
    
- `docs/Bugs.md`
    
- `docs/Roadmap.md`
    

Leia também quando a tarefa envolver fetch, providers, cache, matching ou letras erradas:

- `docs/Arquitetura-fetchers.md`
    
- `docs/Provider-validation.md`
    

Leia também quando a tarefa envolver testes reais, logs, métricas, timeouts, estabilidade ou preparação de release:

- `docs/REAL_WORLD_TESTING.md`
    
- `docs/DEVELOPMENT.md`
    

Se houver conflito entre documentação e código, trate o código como fonte de verdade e registre a inconsistência na documentação relevante.

## Tooling

Prefer using available local tools when relevant:
- `rg`, `fd`, `jq`
- `shellcheck`, `shfmt`
- `go test ./...`, `goimports`
- `ruff`, `pytest`

Do not fail the task only because an optional tool is missing. Mention missing tools in the final summary.
## Regras Gerais

- Não faça commit, push, merge, release ou alteração de issue sem instrução explícita.
    
- Não altere arquitetura, provider order, cache behavior, matching rules ou formato de diagnóstico sem explicar o impacto antes.
    
- Não reescreva arquivos inteiros quando uma alteração pequena resolver o problema.
    
- Preserve comportamentos já existentes, a menos que a tarefa peça mudança explícita.
    
- Não adicione provider, dependência ou framework por impulso.
    
- Não trate ausência de letra como bug automaticamente.
    
- Prefira não mostrar letra a mostrar letra potencialmente errada.
    
- Não registre letras completas em logs, diagnósticos ou testes.
    
- Não invente resultados de execução, testes, providers ou logs.
    

## Regras de Documentação

Atualize documentação quando a mudança alterar comportamento real.

- Decisão arquitetural ou trade-off relevante:
    
    - atualizar `docs/Decisoes.md`
        
- Bug corrigido, novo bug conhecido ou regressão:
    
    - atualizar `docs/Bugs.md`
        
- Mudança de milestone, prioridade ou feature concluída:
    
    - atualizar `docs/Roadmap.md`
        
- Mudança em runtime, cache, launcher ou fluxo geral:
    
    - atualizar `docs/Arquitetura.md`
        
- Mudança em fetcher, provider, matching, cache ou proveniência:
    
    - atualizar `docs/Arquitetura-fetchers.md`
        
    - atualizar `docs/Provider-validation.md` quando houver impacto em aceitação de candidatos
        
- Mudança em logs, métricas, diagnóstico ou critérios de teste real:
    
    - atualizar `docs/DEVELOPMENT.md`
        
    - atualizar `docs/REAL_WORLD_TESTING.md` quando necessário
        

Não atualize documentação só para parecer ocupado. Atualize quando ela deixar de representar o sistema.

## Regras de Código

- Use nomes claros.
    
- Preserve estilo e convenções existentes.
    
- Evite duplicação.
    
- Trate erros e casos de borda.
    
- Não misture mudança de comportamento com refatoração grande sem necessidade.
    
- Adicione testes de regressão para bugs corrigidos.
    
- Não use rede, Spotify, `playerctl`, HOME real ou cache real em testes automatizados.
    
- Use diretórios temporários e mocks para testes que mexem em cache, processo ou ambiente.
    

## Regras Específicas do Projeto

- O Python controla runtime, playerctl, terminal, Kitty, sptlrx e renderização.
    
- O Go controla providers, validação de candidatos, cache, quarentena e diagnósticos do fetcher.
    
- O launcher não deve duplicar lógica de provider.
    
- Eventos estruturados são complemento dos logs humanos de `--debug`, nunca substituição.
    
- Métricas de pipeline não devem ser apresentadas como métricas individuais de provider.
    
- Cache local deve ser validado antes de reutilização.
    
- Resultado de provider não deve ser salvo no cache sem identidade musical suficientemente confiável.
    

## Fluxo de Trabalho Esperado

1. Leia a documentação e os arquivos diretamente relacionados à tarefa.
    
2. Inspecione o código atual antes de propor mudança.
    
3. Explique resumidamente o problema encontrado.
    
4. Faça a menor alteração que resolva o problema.
    
5. Adicione ou atualize testes.
    
6. Rode validações relevantes.
    
7. Rode `git diff --check`.
    
8. Mostre no relatório:
    
    - arquivos alterados;
        
    - motivo de cada alteração;
        
    - testes executados;
        
    - limitações restantes;
        
    - impacto em documentação;
        
    - diff resumido.
        

## Commits e Branches

- Não crie branch automaticamente.
    
- Não faça commit automaticamente.
    
- Não faça push automaticamente.
    
- Quando solicitado a criar commit, use uma mensagem curta e descritiva.
    
- Quando solicitado a trabalhar em branch, use branch separada e não faça merge na `main` sem autorização explícita.
    

## Regra Final

Quando houver dúvida entre uma mudança rápida e uma mudança segura, prefira a mudança segura, pequena, testável e documentada.