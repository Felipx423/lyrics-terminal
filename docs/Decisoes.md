# Decisões Técnicas

# Decisões Técnicas

## Usar Go para o Fetcher

Status: Ativo

### Contexto

A busca em providers, cache, parsing de `.lrc`, timeouts HTTP e diagnósticos exigem uma camada mais previsível que scripts ad hoc.

### Decisão

Usar Go para o fetcher, cache, validação de candidatos e diagnósticos.

### Consequências

- Python continua responsável pelo runtime e renderização.
- Go precisa manter testes isolados de cache e providers.
- Comunicação ocorre por subprocessos e arquivos compartilhados.

---

## Manter Python no Runtime

Status: Ativo

### Contexto

O launcher já controla terminal, `playerctl`, `sptlrx`, Kitty e comportamento visual.

### Decisão

Manter `scripts/lyrics`, `lyrics-local` e `lyricslib.py` em Python.

### Consequências

- mudanças de UI e fluxo permanecem rápidas;
- fetcher e terminal seguem desacoplados;
- o launcher não deve duplicar lógica de provider.

---

## Não Colocar Fallback Pesado Dentro do Launcher

Status: Ativo

### Contexto

O comando principal precisa iniciar rápido e continuar responsivo durante trocas de faixa.

### Decisão

Executar `lyrics-fetch-go` em background e usar `sptlrx` como fallback separado.

### Consequências

- uma busca lenta não deve travar renderização;
- cache pode aparecer depois do timeout visual;
- logs por faixa são necessários para diferenciar lentidão de ausência real de letra.

---

## Validar e Quarentenar Cache Local

Status: Ativo

### Contexto

Um `.lrc` existente pode estar vazio, quebrado ou ser incompatível com a faixa.

### Decisão

Validar cache antes de reutilizá-lo e mover entradas suspeitas para:

```
~/.local/share/lyrics/bad/
```

### Consequências

- cache ruim vira cache miss;
- dados suspeitos permanecem disponíveis para debugging;
- testes de cache são obrigatórios em mudanças nessa área.

---

## Preferir Ausência de Letra a Letra Possivelmente Errada

Status: Ativo

### Contexto

Uma letra errada pode ser salva e reutilizada em futuras execuções.

### Decisão

Ao aumentar rigor de validação, priorizar reduzir falsos positivos, mas medir impacto em cobertura real.

### Consequências

- mudanças de matching precisam de casos concretos;
- provider sem metadados confiáveis deve ser tratado com cuidado;
- cache precisa guardar proveniência quando essa funcionalidade existir.

---

## Registrar Debug Humano e Eventos Estruturados

Status: Ativo

### Contexto

Logs no terminal ajudam investigação imediata, enquanto eventos estruturados permitem análise posterior.

### Decisão

Manter ambos:

- logs humanos em `lyrics --debug --run`;
- eventos estruturados em `lyrics.log`.

### Consequências

- uma métrica nova não deve remover logs úteis;
- eventos por faixa devem carregar identidade suficiente para triagem;
- launcher mede pipeline, não provider individual.

---

## Não Expandir Providers Sem Evidência

Status: Ativo

### Contexto

Adicionar provider aumenta complexidade, risco de falso positivo e manutenção.

### Decisão

Só adicionar ou priorizar provider novo com dados de cobertura, qualidade e latência.

### Consequências

- v0.6.0 prioriza observação real;
- métricas e diagnósticos vêm antes de expansão;
- decisões futuras devem ser registradas neste documento.