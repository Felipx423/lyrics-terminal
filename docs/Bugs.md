# Bugs e Problemas

## Abertos

### Letras erradas aceitas por provider

Status: Investigando  
Prioridade: Alta  
Issue: #1

Descrição:

O fetcher pode aceitar uma letra sincronizada com metadados parecidos, mas pertencente a outra faixa, versão ou artista.

Caso conhecido:

```
Artista: Ryxn Pablo
Título: Ainda
Álbum: Ainda
Duração: 165 segundos
Spotify track ID: 38YZseF2ALmg58eVQ9r2mZ
Resultado: letra errada exibida
```

Risco:

- a letra errada pode ser salva no cache;
- execuções futuras podem reutilizar esse cache;
- um falso positivo é pior que ausência de letra.

Próximos passos:

- registrar provider e candidato vencedor;
- registrar metadados de candidatos aceitos e rejeitados;
- identificar se o problema veio de matching permissivo, versão alternativa, cache contaminado ou provider sem metadados confiáveis;
- criar teste de regressão antes de alterar matching.

---

### Cobertura e lentidão de providers

Status: Coletando dados  
Prioridade: Média  
Issues: #2, #4, #5

Descrição:

Algumas faixas ficam sem retorno do `sptlrx` após 10 segundos, mas podem receber cache local posteriormente.

Casos observados:

- `success_after_fetch`: cache apareceu depois do timeout visual;
- `no_output_timeout`: nenhum resultado chegou;
- `track_changed_before_result`: a faixa mudou antes de haver resultado;
- `interrupted`: runtime encerrado manualmente.

Próximos passos:

- usar métricas estruturadas do launcher;
- separar ausência real de letra de fetch lento;
- identificar provider, timeout e padrão de falha quando houver dados suficientes.

## Resolvidos

### Lyrics encerrava ao trocar música

Status: Resolvido

Descrição:

Ao trocar de faixa em uma playlist, o runtime agora detecta a mudança, limpa a tela e reinicia a pipeline para a nova música.

---

### Lyrics encerrava ao pausar Spotify

Status: Resolvido

Descrição:

Quando o Spotify pausa, o runtime mantém a última linha, espera a retomada e reinicia a pipeline se a faixa mudar durante a pausa.

---

### Cache local aceitava `.lrc` inválido

Status: Resolvido

Descrição:

Arquivos vazios, sem timestamps parseáveis, sem linhas úteis ou estruturalmente suspeitos são tratados como cache miss e movidos para:

```
~/.local/share/lyrics/bad/
```

---

### Current-terminal dependia visualmente de Kitty

Status: Resolvido

Descrição:

`lyrics --current` agora funciona sem Kitty. Kitty permanece necessário apenas para o modo padrão e `lyrics --kitty`.