# Roadmap

## v0.6.0 — Real World Testing

Objetivo: validar o projeto em uso real antes de expandir funcionalidades.

### Concluído

- Suporte a playlists e reinício de pipeline em troca de faixa.
    
- Pausa e retomada sem encerrar o runtime.
    
- Validação e quarentena de cache `.lrc` inválido.
    
- Health check e comandos de diagnóstico.
    
- Estatísticas básicas do fetcher.
    
- Logs estruturados por faixa no launcher.
    
- Suporte a modo current-terminal sem Kitty.
    
- Issue templates e documentação de contribuição.
    

### Em andamento

- #1 Investigar letras erradas aceitas por providers.
    
- #2 Coletar métricas reais de cobertura, sucesso e lentidão.
    
- #4 Identificar padrões recorrentes de falha.
    
- #5 Investigar edge cases de timeout dos providers.
    
- Registrar pelo menos 14 dias de uso real.
    
- Registrar pelo menos 100 faixas observadas.
    
- Revisar logs, cache, quarentena e falhas antes de fechar a milestone.
    

## v0.7.0 — Reliability and UX

Objetivo: melhorar confiabilidade com base nos dados coletados na v0.6.0.

Possíveis entregas:

- endurecer validação de título, artista e versão da faixa;
- registrar proveniência de provider no cache;
- melhorar mensagens de estado no terminal;
- classificar falhas de provider com mais precisão;
- revisar timeout e fallback com base em casos reais;
- reduzir falsos positivos sem destruir cobertura;
- melhorar comandos de análise de falhas.

## v0.8.0 — Provider Quality

Objetivo: melhorar cobertura somente onde os dados mostrarem necessidade.

Possíveis entregas:

- ajustar ranking de LRCLIB, NetEase e syncedlyrics;
- adicionar provider novo apenas se houver ganho comprovado;
- introduzir regras específicas para títulos curtos, remixes e versões ao vivo;
- melhorar detecção de artista principal e participantes;
- permitir configuração segura de providers e timeout.

## v1.0.0 — Stable Release

Critérios desejados:

- uso real consistente;
- nenhuma falha crítica reproduzível;
- casos de letra errada investigados;
- cache e quarentena confiáveis;
- documentação pública completa;
- instalação simples;
- release notes e changelog claros.

## Ideias Futuras

Estas ideias não são prioridade enquanto v0.6.0 estiver aberta:

- pacote AUR;
- instalador simplificado;
- configuração por arquivo;
- expansão para outros players;
- interface gráfica;
- Tauri ou app desktop;
- novos providers sem evidência de cobertura.