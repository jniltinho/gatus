## Context

- **Gravação dos resultados:** todos os resultados passam por `store.Get().InsertEndpointResult` em três pontos do `watchdog`:
  - verificações ativas (`watchdog/endpoint.go`);
  - endpoints externos, com push, heartbeat e a API upstream (`processExternalEndpointResult`);
  - pushes em endpoints ativos (`SubmitEndpointResult`).

  Os resultados de uma chave são serializados por `lockEndpointResults`.
- **Detalhes no dashboard:** `EndpointDetails.vue` busca `/api/v1/endpoints/{key}/statuses` ao abrir e no intervalo do botão de atualização (`Settings.vue`, padrão de 300 s, mínimo de 10 s).
- **Detalhes públicos:** `StatusPageEndpoint.vue` busca `/api/v1/status-pages/{slug}/endpoints/{key}` a cada 60 s e pausa com a aba oculta. A resposta fica em cache por 30 s, com a chave `slug|revisão|geração|endpoint|key` (`statuspage/public.go`).
- **Gráfico:** `ResponseTimeChart.vue` só busca `/response-times/{duração}/history` ao montar e ao trocar o período.
- **Servidor:**
  - `controller.Handle` define `ReadTimeout`, `WriteTimeout` e `IdleTimeout` de 15 s;
  - no fasthttp, o prazo de escrita vale para a resposta inteira, inclusive um corpo em stream (`SetBodyStreamWriter`), e pode ser trocado por requisição com `Server.HeaderReceived`, que devolve `RequestConfig`;
  - `app.Use(compress.New())` comprime todas as respostas, o que acumularia os eventos.
- **Recarga:** `stop` chama `watchdog.Shutdown` e depois `controller.Shutdown`. O `Shutdown` do fasthttp espera as conexões ativas terminarem, então um stream aberto seguraria a recarga.
- **Autenticação:** com a tela de login, o navegador manda o cookie de sessão. `EventSource` não aceita cabeçalhos próprios, mas os navegadores mandam `Sec-Fetch-Mode`, então um 401 não abre a janela nativa (regra da `add-basic-login-page`). Com OIDC, o cookie de sessão também vai.

## Goals / Non-Goals

**Goals:**
- Ver na página de detalhes (dashboard e pública) um resultado novo, inclusive push e Pending, em até cerca de 1 segundo.
- Nenhum dado novo publicado: o evento é só um aviso.
- Limitar conexões e não travar recarga nem desligamento.
- Continuar funcionando sem SSE, com a atualização periódica.

**Non-Goals:**
- Tempo real na lista do dashboard, nas páginas de status completas e nas suites.
- Propagar avisos entre instâncias.
- WebSocket.
- Mandar os dados do resultado pelo evento.

## Decisions

### D1. Distribuidor de avisos em memória

Novo pacote do fork `liveupdates`:

- **`Publish(key, timestamp)`:**
  - guarda o instante do último resultado da chave (`lastResults`, um mapa protegido por mutex);
  - avisa os inscritos da chave sem bloquear, com canal de capacidade 1: se já há aviso pendente, o novo é descartado, porque o cliente vai buscar o estado mais recente de qualquer jeito.
- **`Subscribe(key)`:** devolve o canal, o instante do último resultado e a função de cancelar. Recusa com `ErrClosed` depois de `Close`.
- **`Close()`:** fecha todas as inscrições e recusa as novas até `Open()`, chamado por `start` na recarga.
- **`LastResult(key)`:** devolve o último instante conhecido, usado pelo cache público (D6).
- **`Forget(key)`:** apaga a chave ao remover ou renomear o endpoint, junto com `watchdog.ForgetExternalEndpoint`, e na recarga, para as chaves que deixam de existir.

O `watchdog` chama `liveupdates.Publish` logo depois de cada `InsertEndpointResult` bem-sucedido, nos três pontos, ainda com o lock da chave. Assim os avisos seguem a ordem dos resultados.

**Alternativas consideradas:**
- **Consultar o storage periodicamente em cada conexão:** rejeitada, porque o custo cresce com o número de visitantes.
- **Ler o último resultado do storage no cache público:** rejeitada, porque seria uma consulta a mais em toda requisição.

### D2. Protocolo SSE

- **Rotas:**
  - `GET /api/v1/endpoints/{key}/events`, no roteador protegido, depois do middleware de segurança, com 404 para chave inexistente (a mesma verificação de `/statuses`, sem ler resultados);
  - `GET /api/v1/status-pages/{slug}/endpoints/{key}/events`, no bloco público, com as mesmas verificações da API de detalhes antes de qualquer leitura (página publicada e chave mostrada por ela; senão o 404 idêntico, contado no limitador).
- **Cabeçalhos:** `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-store`, `X-Accel-Buffering: no` e `Connection: keep-alive`.
- **Mensagens:**
  - começa com `retry: 3000`;
  - cada aviso é `event: result`, `id: <instante do resultado em ms>` e `data: {"timestamp":"<RFC 3339>"}`;
  - a cada 15 s, um comentário `: ping`.
- **`Last-Event-ID`:** quando o cabeçalho (ou o parâmetro `lastEventId`) é menor que o último resultado conhecido, a conexão já começa com um aviso. Isso cobre o intervalo de reconexão.
- **Duração:** cada conexão dura no máximo 5 minutos e termina de forma limpa; o `EventSource` reconecta sozinho com `Last-Event-ID`.
- **Compressão:** o middleware `compress` ignora as rotas de eventos (`Next`).

**Alternativa considerada:** mandar o resultado inteiro no evento. Rejeitada, porque exigiria outra sanitização para a página pública e duplicaria a lógica das rotas atuais.

### D3. Prazo de escrita das conexões de eventos

`controller.Handle` define `server.HeaderReceived`. Para os caminhos terminados em `/events` das duas rotas, devolve `RequestConfig` com `WriteTimeout` de 6 minutos, acima dos 5 minutos de D2. As outras requisições mantêm os 15 s. O `ReadTimeout` não muda, porque o corpo da requisição é vazio.

**Alternativa considerada:** encerrar cada conexão antes de 15 s. Rejeitada, porque faria cerca de 4 reconexões por minuto por visitante.

### D4. Limites de conexões

- **Contadores:** um contador atômico do total e um mapa por IP do cliente (calculado com `status-pages.trusted-proxies`, como as rotas públicas), valendo para as duas rotas.
- **Tetos:** 500 conexões no total e 10 por IP. Acima disso, a resposta é 429 com `Retry-After: 30`.
- **Frontend:** o `EventSource` fecha em erro de conexão, e a página segue com a atualização periódica. A próxima tentativa só acontece quando a página é aberta de novo ou a aba volta a ficar visível.

### D5. Ciclo de vida

- `stop` chama `liveupdates.Close()` antes de `controller.Shutdown()`.
- Cada escritor de stream escolhe entre o canal da inscrição, o `ping`, o prazo de 5 minutos e o fechamento; ao fechar, sai logo, e o `Shutdown` do fasthttp deixa de esperar.
- Um erro de escrita, como o cliente que foi embora, também encerra a conexão e libera os contadores.
- `start` chama `liveupdates.Open()`.

### D6. Cache da API pública de detalhes

A chave do cache de detalhes passa a incluir `liveupdates.LastResult(key)`: `slug|revisão|geração|endpoint|key|<último resultado>`. Um resultado novo gera outra chave, e a próxima requisição monta de novo, com a mesma deduplicação e o mesmo semáforo. Sem resultado novo, o cache de 30 s continua valendo. O payload da página inteira não muda.

### D7. Frontend

- **`utils/liveUpdates.js`:** `watchEndpointResults(url, onResult)`:
  - abre um `EventSource` com credenciais do mesmo site;
  - junta avisos próximos com um atraso de 500 ms;
  - fecha a conexão com a aba oculta e, ao voltar, reabre e chama `onResult` uma vez;
  - desiste depois de um erro que deixe o `EventSource` fechado;
  - devolve a função de parar, usada no `onUnmounted`.
- **`EndpointDetails.vue` e `StatusPageEndpoint.vue`:** chamam a atualização que já existe ao receber o aviso, sem o indicador de carregamento. A atualização periódica continua como fallback.
- **`ResponseTimeChart.vue`:** ganha a prop `refreshKey` (instante do último resultado da página). Quando ela muda, a linha é buscada de novo sem o spinner e sem apagar o gráfico. As faixas já seguem as props.

## Risks / Trade-offs

- **[Várias instâncias]** → O aviso só sai da instância que gravou o resultado. Os visitantes de outra instância seguem com a atualização periódica. Documentado.
- **[Proxy que acumula o stream]** → O `X-Accel-Buffering: no` resolve no nginx. Para outros proxies, a documentação pede para desligar o buffer e usar um timeout de leitura acima de 15 s.
- **[Conexões ocupam goroutines e sockets]** → Limites de D4 e duração máxima de 5 minutos.
- **[Rajada de pushes]** → Os avisos se juntam (canal de capacidade 1 e atraso de 500 ms no cliente). O cache público monta no máximo uma vez por resultado novo, com o semáforo atual.
- **[Autenticação que expira com a conexão aberta]** → A conexão dura no máximo 5 minutos, e a reconexão passa de novo pela autenticação.
- **[HTTP/1.1 limita 6 conexões por site no navegador]** → Cada aba de detalhes usa uma conexão. Com muitas abas do mesmo site abertas, novas requisições podem esperar. Documentado, e atrás de HTTP/2 no nginx isso não acontece.

## Migration Plan

- Não há migração de dados nem configuração nova obrigatória.
- **Rollback:** voltar ao binário anterior. As páginas deixam de abrir o `EventSource` e usam só a atualização periódica.

## Open Questions

- Os tetos de 500 conexões e 10 por IP ficam fixos nesta versão e podem virar configuração depois.
