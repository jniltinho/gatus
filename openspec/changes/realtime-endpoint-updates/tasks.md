## 1. Backend: avisos e canal SSE

- [ ] 1.1 Pacote `liveupdates`:
  - sequência monotônica, `Publish`, `Subscribe` (canal de capacidade 1 e função de cancelar), `Sequence`, `ForgetExcept`, `Close` e `Open`;
  - `Publish` nos três pontos do `watchdog`, depois de gravar e com o lock da chave (`UpdateEndpointStatus` devolvendo o erro);
  - `Forget` ao remover e renomear, e `ForgetExcept` na recarga bem-sucedida;
  - testes com `-race`: junção, cancelamento, `Close`/`Open` e ordem das sequências.
- [ ] 1.2 `controller`:
  - `HeaderReceived` com o caminho exato das duas rotas e `WriteTimeout` de 6 minutos;
  - `ShutdownWithTimeout` de 10 s;
  - `liveupdates.Close` em `stop` antes de `controller.Shutdown`, e `Open` em `start` antes de `controller.Handle`;
  - teste num listener real com prazos curtos, incluindo uma rota comum que continua com o prazo padrão.
- [ ] 1.3 Rotas `GET /api/v1/endpoints/{key}/events` e `GET /api/v1/status-pages/{slug}/endpoints/{key}/events`:
  - existência pelos endpoints em memória, ou `Lookup` e endpoint mostrado pela página, sem ler o storage;
  - rota pública antes do catch-all;
  - cabeçalhos (`no-transform`, sem `Connection`, cabeçalhos das rotas públicas) e `HEAD`;
  - `retry` e `id` iniciais, `Last-Event-ID` (cabeçalho ou `lastEventId`), `: ping` a cada 15 s e máximo de 5 minutos, com durações em variáveis do pacote;
  - `compress` sem as rotas de eventos.
- [ ] 1.4 Limites de 500 conexões no total e 10 por IP (com `trusted-proxies`): 429 e 503 com `no-store` e corpo JSON; cópia dos valores do `fiber.Ctx`; reserva da vaga antes do stream e liberação em qualquer saída, com `recover` no writer.
- [ ] 1.5 `security/basic_auth.go`: `Accept: text/event-stream` também é requisição de navegador (401 sem `WWW-Authenticate`).
- [ ] 1.6 Cache da API pública de detalhes com a sequência na chave.
- [ ] 1.7 Testes de API e integração:
  - aviso de push, heartbeat e verificação, sem dados no evento;
  - endpoint Push sem resultados;
  - `Last-Event-ID` e rotação sem evento anterior;
  - 401 sem `WWW-Authenticate`, com e sem `Sec-Fetch-*`;
  - 404 público; `Accept-Encoding: br` sem compressão;
  - 429 e 503 com os cabeçalhos;
  - vaga liberada quando o cliente sai e em pânico;
  - cache público renovado antes dos 30 s;
  - recarga sem esperar.

## 2. Frontend

- [ ] 2.1 `utils/liveUpdates.js`: `EventSource`, junção de 500 ms, aba oculta, backoff de 30 s até 5 minutos depois de erro fatal e reabertura com `notifyRefreshSucceeded`. Testes unitários das funções puras.
- [ ] 2.2 `EndpointDetails.vue`: `fetchData({ silent })`, `currentStatus` da página 1 atualizado mesmo em outra página, geração para descartar respostas fora de ordem, canal reaberto na troca de `key`, fallback periódico mantido.
- [ ] 2.3 `StatusPageEndpoint.vue`: aviso com atualização silenciosa, canal reaberto no `watch([slug, key])`.
- [ ] 2.4 `ResponseTimeChart.vue`: prop `refreshKey`, linha recarregada sem spinner no máximo a cada 60 s, faixas na hora.
- [ ] 2.5 Lint, `npm run test:unit` e `make frontend-build`.

## 3. Documentação, E2E e entrega

- [ ] 3.1 Documentação:
  - `docs/push-monitoring.md` e `docs/status-pages.md`: tempo real, rotas, limites, `trusted-proxies` obrigatório atrás de proxy, nginx (`proxy_buffering off`, `proxy_read_timeout`, HTTP/2), várias instâncias, abas e cache;
  - `AGENTS.fork.md`.
- [ ] 3.2 E2E em `test/e2e/push.sh` e `test/e2e/status-pages.sh`: push Pending aparecendo nos detalhes sem recarregar a página (dashboard e página pública) e prints.
- [ ] 3.3 `go test ./... -race` com PostgreSQL, MySQL e MariaDB, `make lint` e `openspec validate realtime-endpoint-updates --strict`.
- [ ] 3.4 PR no `jniltinho/gatus` com CI verde e merge, release com imagem no Docker Hub, pacote `mariadb` e arquivamento da change.
