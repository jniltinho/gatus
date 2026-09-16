## Context

- **Onde os resultados são gravados:** `store.Get().InsertEndpointResult`, chamado em três pontos do `watchdog`, sempre com `lockEndpointResults` da chave:
  - verificações ativas (`watchdog/endpoint.go`);
  - endpoints externos, com push, heartbeat e a API upstream (`processExternalEndpointResult`);
  - pushes em endpoints ativos (`SubmitEndpointResult`).

  As suites gravam os resultados dos seus endpoints por outro caminho (`insertSuiteResultWithoutRetry`) e ficam fora desta change.
- **Detalhes no dashboard:** `EndpointDetails.vue` busca `/api/v1/endpoints/{key}/statuses` ao abrir e no intervalo do botão de atualização (`Settings.vue`: padrão de 300 s, mínimo de 10 s). `fetchData` liga `isRefreshing`. Em página diferente da 1, `currentStatus` (barras, painel e gráfico) não é atualizado.
- **Detalhes públicos:** `StatusPageEndpoint.vue` busca `/api/v1/status-pages/{slug}/endpoints/{key}` a cada 60 s, pausa com a aba oculta e reage à troca de rota (`watch([slug, key])`). A resposta fica em cache por 30 s, com a chave `slug|revisão|geração|endpoint|key`, num LRU de 1000 entradas compartilhado com as páginas.
- **Gráfico:** `ResponseTimeChart.vue` só busca `/response-times/{duração}/history` (sem cache no servidor e público) ao montar e ao trocar o período. A linha é a média por hora.
- **fasthttp v1.71.0 e Fiber v2.52.13, conferidos pelo QA:**
  - `Server.HeaderReceived` devolve `RequestConfig{WriteTimeout}`, e esse prazo vale para a resposta inteira, inclusive o corpo em stream. O Fiber não usa `HeaderReceived`, e `controller.Handle` pode defini-lo junto dos timeouts de 15 s.
  - O `SetBodyStreamWriter` roda o writer numa goroutine própria, sem `recover`.
  - Um cliente que foi embora faz o próximo `Flush` falhar.
  - `ReadTimeout` e `IdleTimeout` não cortam a conexão durante o stream.
  - `Shutdown` espera as conexões ativas terminarem.
  - `compress.New()` comprimiria o `text/event-stream`.
- **Deploy do dono:** o Gatus escuta em 127.0.0.1 atrás do nginx do host. Numa recarga da configuração, o nginx responde 502 enquanto o Gatus não sobe de novo.
- **Autenticação:**
  - com a tela de login, o navegador manda o cookie de sessão; com OIDC, o adaptor do g8 é compatível com stream e seu 401 não tem `WWW-Authenticate`;
  - o `EventSource` não aceita cabeçalhos próprios;
  - os navegadores só mandam `Sec-Fetch-*` em origem confiável (HTTPS ou localhost).

## Goals / Non-Goals

**Goals:**
- Ver na página de detalhes (dashboard e pública) um resultado novo, inclusive push e Pending, em até cerca de 1 segundo.
- Nenhum dado novo publicado: o evento é só um aviso.
- Limitar conexões e custo, e não travar recarga nem desligamento.
- Recuperar o tempo real sozinho depois de recarga, 502 ou 429, e seguir funcionando sem SSE, com a atualização periódica.

**Non-Goals:**
- Tempo real na lista do dashboard, nas páginas de status completas e nas suites.
- Propagar avisos entre instâncias.
- WebSocket.
- Dados do resultado no evento.

## Decisions

### D1. Distribuidor de avisos com sequência por endpoint

Novo pacote do fork `liveupdates`.

**`Publish(key)`:**
- incrementa uma sequência global monotônica (`uint64`) e guarda o valor como a sequência da chave;
- avisa os inscritos da chave sem bloquear, por canal de capacidade 1: um aviso pendente é substituído, porque o cliente sempre busca o estado mais recente.

**Demais operações:**
- `Subscribe(key)` devolve o canal, a sequência atual da chave e a função de cancelar, e recusa com `ErrClosed` depois de `Close`.
- `Sequence(key)` devolve a sequência atual (0 sem resultado desde o início do processo).
- `Close()` fecha todas as inscrições e recusa as novas, sem zerar os contadores de conexões (os escritores ainda liberam as vagas ao sair) nem a sequência. `Open()` só volta a aceitar inscrições: a sequência global nunca reinicia no processo, para um `Last-Event-ID` antigo continuar menor que a sequência atual.
- `ForgetExcept(keys)` apaga as chaves que não estão na lista.

**Quem chama:**
- O `watchdog` chama `Publish` logo depois de cada `InsertEndpointResult` bem-sucedido, nos três pontos, ainda com o lock da chave. `UpdateEndpointStatus` passa a devolver o erro da gravação para isso.
- `managedendpoint` chama `ForgetExcept`/`Forget` ao remover ou renomear.
- `main.initializeStorage` chama `ForgetExcept` com as chaves conhecidas, só quando a carga dos endpoints gerenciados deu certo.

A sequência, e não o instante do resultado, é o `id` dos eventos e entra na chave do cache (D6). Instantes não crescem sempre: o check marca o fim da avaliação e o push marca o início, antes do lock, e dois resultados podem cair no mesmo milissegundo.

**Alternativa considerada:** usar o instante do resultado como `id`. Rejeitada no QA, porque podia sair fora de ordem ou repetido.

### D2. Protocolo SSE

**Rotas:**
- `GET /api/v1/endpoints/{key}/events` fica no roteador protegido, depois do middleware de segurança. A existência da chave é decidida pelos endpoints conhecidos em memória, sem ler o storage, de modo que um endpoint Push ainda sem resultados já possa ser observado. Chave desconhecida recebe 404. A verificação usa:
  - `config.GetEndpointByKey` e `GetExternalEndpointByKey`;
  - `managedendpoint.Get(key) != nil`, porque `EndpointByKey` é nulo para Push, conflito e inválido.
- `GET /api/v1/status-pages/{slug}/endpoints/{key}/events` fica no bloco público, registrada antes do catch-all das status pages. Passa pelas mesmas verificações da API de detalhes (`Lookup` e endpoint mostrado pela página), sem ler o storage. Senão, responde o 404 idêntico, contado no limitador, pelo próprio `statusPageNotFound`.
- **Ordem das verificações:** existência (404), depois parada (503) e limite de conexões (429), para o 404 continuar idêntico.

**Cabeçalhos:** `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-store, no-transform` e `X-Accel-Buffering: no`. `Connection` não é enviado: o fasthttp cuida dele, e ele é proibido no HTTP/2. A rota pública também manda os cabeçalhos das rotas públicas. O Fiber registra `HEAD` com o mesmo handler do `GET`, e o fasthttp só descarta o corpo depois do handler. Por isso o handler responde `HEAD` só com os cabeçalhos, antes de inscrever, reservar vaga e chamar `SetBodyStreamWriter`: um `HEAD` nunca ocupa vaga.

**Mensagens:**
- Ao abrir: `retry: 3000` e `id: <sequência atual>` numa mensagem sem dados. Pelo WHATWG, o `id` sem `data` define o `lastEventId` do `EventSource`, que o usa na reconexão nativa.
- Se o `Last-Event-ID` recebido (cabeçalho ou parâmetro `lastEventId`) for menor que a sequência atual, logo em seguida um evento. Ausente ou inválido conta como 0: com sequência maior que 0, a conexão começa com um evento, e a página só faz uma atualização silenciosa a mais.
- Cada aviso: `event: result`, `id: <sequência>`, `data: {}`.
- A cada 15 s, o comentário `: ping`.

**Duração:** cada conexão dura no máximo 5 minutos e termina de forma limpa. O `EventSource` reconecta com `Last-Event-ID`.

**Compressão:** o middleware `compress` ignora as duas rotas (`Next`).

**Testes:** as durações (ping, máximo e prazo) ficam em variáveis do pacote, para os testes usarem valores curtos.

### D3. Prazo de escrita das conexões de eventos

`controller.Handle` define `server.HeaderReceived`. O `HeaderReceived` roda antes do roteamento, então usa uma função compartilhada com os testes, `isEventsPath`. Ela extrai o caminho do `RequestURI` (forma absoluta ou não, sem query) e divide em segmentos. Casam só estes padrões, com um segmento não vazio no lugar de cada variável:
- exatamente `api`, `v1`, `endpoints`, `{key}`, `events`;
- exatamente `api`, `v1`, `status-pages`, `{slug}`, `endpoints`, `{key}`, `events`.

Um caminho como `/api/v1/endpoints/statuses/events-export` não casa. Nesses caminhos devolve `RequestConfig{WriteTimeout: 6 * time.Minute}`, e nas outras requisições uma configuração vazia, que mantém os 15 s. `controller.Shutdown` passa a usar `ShutdownWithTimeout(10 * time.Second)`.

**Teste:** um listener real com `WriteTimeout` de 1 s no servidor e 3 s nas rotas de eventos mostra a conexão aberta além de 1 s, sem deixar o `go test` lento.

**Alternativa considerada:** encerrar cada conexão antes de 15 s. Rejeitada, porque faria cerca de 4 reconexões por minuto por visitante.

### D4. Limites e robustez das conexões

- **Contadores:**
  - um contador atômico do total e um mapa de contadores por endereço IP do cliente, calculado com `statuspage.ClientIP` e `status-pages.trusted-proxies`, valendo para as duas rotas;
  - a entrada do mapa é apagada quando chega a 0, e o mapa nunca passa de 500 entradas, o teto total;
  - tetos de 500 no total e 10 por IP;
  - o limite é por endereço, não por /64, então rotação de IPv6 temporário fura o teto por IP, mas não o total. Isso fica documentado.
  - Acima deles, a resposta é 429 com `Retry-After: 30`, `Cache-Control: no-store` e `{"error":"too many requests"}`.
  - Durante a parada, a resposta é 503 com os mesmos cabeçalhos.
- **Liberação da vaga:**
  - o handler copia chave, IP, `Last-Event-ID` e a inscrição **antes** de `SetBodyStreamWriter`, porque a goroutine do writer começa nessa chamada, o `fiber.Ctx` volta ao pool quando o handler retorna e o fasthttp proíbe usar o `RequestCtx` no writer;
  - a vaga é reservada imediatamente antes de `SetBodyStreamWriter` e liberada pelo handler se algo falhar antes disso;
  - o writer faz `defer` do cancelamento da inscrição, da liberação da vaga e de um `recover` com log, porque a goroutine do fasthttp não tem `recover`.
- **Cliente que foi embora:** é detectado no próximo `Flush`, no máximo 15 s depois, no `ping`. Isso fica documentado.
- **Atrás do nginx:** `status-pages.trusted-proxies` é obrigatório, senão todos os visitantes contam como 127.0.0.1 e só 10 recebem o tempo real. A documentação diz isso, e o aviso de `ObserveConnection` passa a citar também o limite dos canais de tempo real.
- **`curl`:** com `Accept: text/event-stream` numa rota protegida, o 401 sai sem `WWW-Authenticate`, como para o `EventSource`. Isso fica documentado.

### D5. Ciclo de vida

- `stop` chama `liveupdates.Close()` antes de `controller.Shutdown()`.
- O writer escolhe entre o aviso, o `ping`, o prazo máximo e o fechamento. Ao fechar, sai logo.
- `ShutdownWithTimeout` cobre um writer preso num `Flush` de cliente lento.
- `start` chama `liveupdates.Open()` antes de `go controller.Handle(cfg)`, para o servidor novo aceitar conexões assim que escutar.

### D6. Cache da API pública de detalhes

A chave do cache de detalhes passa a incluir `liveupdates.Sequence(key)`: `slug|revisão|geração|endpoint|key|<sequência>`. Um resultado novo leva a próxima requisição a montar de novo, com a mesma deduplicação e o mesmo semáforo. Sem resultado novo, vale o cache de 30 s. Pushes muito frequentes podem criar muitas chaves no LRU compartilhado e tirar páginas do cache antes dos 30 s, o que fica documentado. O payload da página inteira não muda.

### D7. Frontend

**`utils/liveUpdates.js`**, `watchEndpointResults(url, onResult)`:
- abre o `EventSource` e junta avisos próximos com um atraso de 500 ms;
- fecha com a aba oculta e, ao voltar, reabre e chama `onResult` uma vez;
- no evento `error`, só age quando `readyState === EventSource.CLOSED`, o que acontece com 502 do proxy na recarga, 503, 429, 404 ou 401. Com `CONNECTING` (queda depois de um 200, como o fim dos 5 minutos), deixa a reconexão nativa agir, para não abrir duas conexões;
- com o canal fechado, guarda `lastEventId` e reabre com backoff de 30 s, 60 s, 120 s e depois a cada 5 minutos, num `EventSource` novo com `?lastEventId=<guardado>`, porque uma instância nova não herda o id;
- quando a página diz que uma atualização periódica deu certo (`notifyRefreshSucceeded()`), reabre na hora se estiver fechado;
- devolve a função de parar.

A lógica de backoff e junção fica em funções puras, com testes unitários.

**`EndpointDetails.vue`:**
- `fetchData({ silent })`: o aviso chama com `silent: true`, sem ligar `isRefreshing`.
- Barras (`EndpointCard`), painel (`DetailsSummary`) e gráfico (eventos e resultados) passam a ler `currentStatus`, sempre a página 1. `endpointStatus`, que é paginado, fica só com a tabela.
- Em página diferente da 1, o aviso também atualiza `currentStatus` com uma busca da página 1.
- Toda atualização periódica bem-sucedida chama `notifyRefreshSucceeded()`.
- Um contador de geração descarta respostas que chegam fora de ordem.
- O canal é reaberto quando a `key` da rota muda.

**`StatusPageEndpoint.vue`:**
- o aviso chama a atualização silenciosa já existente (`load`, com geração);
- a troca de rota reabre o canal junto com o `watch([slug, key])`;
- toda atualização periódica bem-sucedida chama `notifyRefreshSucceeded()`.

**`ResponseTimeChart.vue`:**
- prop `refreshKey`. Quando ela muda, a linha é buscada de novo sem spinner e sem apagar o gráfico, no máximo uma vez a cada 60 s por componente, porque a linha é média por hora e a rota de histórico não tem cache;
- o spinner só aparece na primeira carga e na troca de período ou de endpoint;
- o contador dos 60 s reinicia na troca de `endpointKey`;
- as faixas vermelhas e amarelas seguem as props na hora.

## Risks / Trade-offs

- **[Várias instâncias]** → O aviso só sai da instância que gravou. As outras seguem com a atualização periódica. Documentado.
- **[Proxy que acumula o stream]** → `X-Accel-Buffering: no` e `no-transform`. A documentação pede `proxy_buffering off`, `proxy_read_timeout` acima de 15 s e HTTP/2.
- **[Rajada de pushes]** → Avisos juntados no servidor (capacidade 1) e no cliente (500 ms). O gráfico recarrega no máximo a cada 60 s. O cache público monta de novo no máximo uma vez por aviso, com semáforo.
- **[Conexões presas]** → Tetos, duração máxima, `ping` de 15 s e `ShutdownWithTimeout`.
- **[HTTP/1.1 com 6 conexões por site]** → O canal fecha com a aba oculta. Várias janelas visíveis ainda podem esperar. Documentado.
- **[Autenticação expirada]** → A reconexão a cada 5 minutos passa de novo pela autenticação. Um 401 leva ao backoff e à reabertura depois de uma atualização bem-sucedida.

## Migration Plan

- Não há migração de dados nem configuração nova obrigatória. Atrás de proxy, `trusted-proxies` e a configuração do nginx ficam documentadas.
- **Rollback:** voltar ao binário anterior. As páginas deixam de abrir o `EventSource` e usam só a atualização periódica.

## Open Questions

- Os tetos de 500 conexões e 10 por IP ficam fixos nesta versão.
