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

## Validar e quarentenar cache local inválido

Motivo:
- `.lrc` existente não pode ser aceito só por existir;
- cache ruim precisa voltar a ser cache miss para permitir novo fetch;
- o arquivo suspeito deve ser preservado em quarentena para debug.

Status:
Ativo.

## Registrar diagnósticos de fetch no índice

Motivo:
- precisamos medir taxa de sucesso, origem e histórico recente sem depender de logs;
- `lyrics-fetch-go --stats` deve ler um índice já enriquecido com `status`, `provider`, `source_id`, `created_at` e `duration`;
- `--dry-run` não deve alterar cache nem quarentena.

Status:
Ativo.
