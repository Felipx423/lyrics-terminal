# Decisões Técnicas

## Usar Go para o fetcher

Motivo:
- binário único;
- HTTP com timeout melhor;
- cache e parsing mais organizados.

Status:
Ativo.

## Não colocar fallback pesado dentro do lyrics principal

Motivo:
- o comando principal precisa abrir rápido;
- fallback externo não pode quebrar o sptlrx.

Status:
Ativo.
