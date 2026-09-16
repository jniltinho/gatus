## 1. Backend: avisos e canal SSE

- [x] 1.1 Pacote `liveupdates` (a sequência nunca reinicia; `Close` não zera contadores nem sequência):
  - sequência monotônica, `Publish`, `Subscribe` (canal de capacidade 1 e função de cancelar), `Sequence`, `ForgetExcept`, `Close` e `Open`;
  - `Publish` nos três pontos do `watchdog`, depois de gravar e com o lock da chave (`UpdateEndpointStatus` devolvendo o erro);
  - `Forget` ao remover e renomear, e `ForgetExcept` na recarga bem-sucedida;
  - testes com `-race`: junção, cancelamento, `Close`/`Open` e ordem das sequências.
- [x] 1.2 `controller`:
  - função `isEventsPath` por segmentos (forma absoluta, sem query) no `HeaderReceived`, com `WriteTimeout` de 6 minutos, e testes com `/api/v1/endpoints/jobs_backup/events`, `?lastEventId=`, forma absoluta e `/api/v1/endpoints/statuses/events-export`;
  - `ShutdownWithTimeout` de 10 s;
  - `liveupdates.Close` em `stop` antes de `controller.Shutdown`, e `Open` em `start` antes de `controller.Handle`;
  - teste num listener real com prazos curtos, incluindo uma rota comum que continua com o prazo padrão.
- [x] 1.3 Rotas `GET /api/v1/endpoints/{key}/events` e `GET /api/v1/status-pages/{slug}/endpoints/{key}/events`:
  - existência sem ler o storage: `config.GetEndpointByKey`/`GetExternalEndpointByKey` e `managedendpoint.Get(key) != nil`; na pública, `Lookup` e endpoint mostrado, com 404 por `statusPageNotFound`, antes do 503 e do 429;
  - `HEAD` só com cabeçalhos, antes da inscrição e da vaga;
  - `Last-Event-ID` ausente ou inválido vale 0;
  - rota pública antes do catch-all;
  - cabeçalhos (`no-transform`, sem `Connection`, cabeçalhos das rotas públicas) e `HEAD`;
  - `retry` e `id` iniciais, `Last-Event-ID` (cabeçalho ou `lastEventId`), `: ping` a cada 15 s e máximo de 5 minutos, com durações em variáveis do pacote;
  - `compress` sem as rotas de eventos.
- [x] 1.4 Limites de 500 conexões no total e 10 por endereço IP (com `trusted-proxies`):
  - 429 e 503 com `no-store` e corpo JSON;
  - cópia dos valores do `fiber.Ctx` antes de `SetBodyStreamWriter`;
  - reserva da vaga antes do stream e liberação em qualquer saída, com `recover` no writer;
  - entrada do mapa apagada em 0;
  - aviso de `ObserveConnection` citando os canais de tempo real.
- [x] 1.5 `security/basic_auth.go`: `Accept: text/event-stream` também é requisição de navegador (401 sem `WWW-Authenticate`).
- [x] 1.6 Cache da API pública de detalhes com a sequência na chave.
- [x] 1.7 Testes de API e integração:
  - aviso de push, heartbeat e verificação, sem dados no evento;
  - endpoint Push sem resultados;
  - `Last-Event-ID` e rotação sem evento anterior;
  - 401 sem `WWW-Authenticate`, com e sem `Sec-Fetch-*`;
  - 404 público; `Accept-Encoding: br` sem compressão;
  - 429 e 503 com os cabeçalhos;
  - vaga liberada quando o cliente sai e em pânico; `HEAD` sem vaga;
  - cache público renovado antes dos 30 s;
  - recarga sem esperar.

## 2. Frontend

- [x] 2.1 `utils/liveUpdates.js`:
  - `EventSource`, junção de 500 ms e aba oculta;
  - backoff só com `readyState === CLOSED`, de 30 s até 5 minutos, reabrindo com `?lastEventId=` guardado;
  - nada a fazer com `CONNECTING`;
  - reabertura com `notifyRefreshSucceeded`.
  - Testes unitários das funções puras.
- [x] 2.2 `EndpointDetails.vue`:
  - `fetchData({ silent })`;
  - barras, painel e gráfico lendo `currentStatus` (página 1) e a tabela lendo `endpointStatus`;
  - `currentStatus` atualizado mesmo em outra página;
  - geração para descartar respostas fora de ordem;
  - canal reaberto na troca de `key`;
  - `notifyRefreshSucceeded` nas atualizações periódicas, com o fallback mantido.
- [x] 2.3 `StatusPageEndpoint.vue`: aviso com atualização silenciosa, canal reaberto no `watch([slug, key])` e `notifyRefreshSucceeded` nas atualizações periódicas.
- [x] 2.4 `ResponseTimeChart.vue`: prop `refreshKey`, linha recarregada sem spinner (spinner só na primeira carga e na troca de período ou endpoint) no máximo a cada 60 s, contador reiniciado na troca de `endpointKey`, faixas na hora.
- [x] 2.5 Lint, `npm run test:unit` e `make frontend-build`.

## 3. Documentação, E2E e entrega

- [x] 3.1 Documentação:
  - `docs/push-monitoring.md` e `docs/status-pages.md`: tempo real, rotas, limites (por endereço, IPv6 temporário), `trusted-proxies` obrigatório atrás de proxy, nginx (`proxy_buffering off`, `proxy_read_timeout`, HTTP/2), várias instâncias, abas, cache e `curl` com `Accept: text/event-stream` sem desafio;
  - `AGENTS.fork.md`.
- [x] 3.2 E2E em `test/e2e/push.sh` e `test/e2e/status-pages.sh`: push Pending aparecendo nos detalhes sem recarregar a página (dashboard e página pública) e prints.
- [x] 3.3 `go test ./... -race` com PostgreSQL, MySQL e MariaDB, `make lint` e `openspec validate realtime-endpoint-updates --strict`.
- [x] 3.4 PR no `jniltinho/gatus` com CI verde e merge, release com imagem no Docker Hub, pacote `mariadb` e arquivamento da change.
