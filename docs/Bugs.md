# Bugs e Problemas

## lyrics encerra ao trocar música

Status:
Resolvido.

Descrição:
Quando a faixa muda em uma playlist, o comando principal reavalia a faixa atual, emite `track_changed`, limpa a tela e reinicia o pipeline para a nova música. Se o `sptlrx` ficar mudo por tempo configurável (`--no-output-timeout`, padrão 10s), o fluxo entra em espera e continua checando se um `.lrc` apareceu no cache local a cada 2s.

Prioridade:
Alta.
