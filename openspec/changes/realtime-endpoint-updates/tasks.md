## 1. Backend: avisos e canal SSE

- [ ] 1.1 Pacote `liveupdates`:
  - `Publish`, `Subscribe` (canal de capacidade 1, com a função de cancelar), `LastResult`, `Forget`, `Close` e `Open`;
  - chamadas de `Publish` nos três pontos do `watchdog`, depois de gravar;
  - `Forget` junto de `watchdog.ForgetExternalEndpoint` (renomear, remover e recarga).
  - Testes com `-race`: junção de avisos, cancelamento, `Close`/`Open` e ordem.
- [ ] 1.2 Rotas `GET /api/v1/endpoints/{key}/events` (protegida, 404 sem registro) e `GET /api/v1/status-pages/{slug}/endpoints/{key}/events` (pública, 404 idêntico antes de ler o storage):
  - cabeçalhos, `retry`, eventos `result` só com o instante, `: ping` a cada 15 s;
  - `Last-Event-ID` (cabeçalho ou `lastEventId`) e duração máxima de 5 minutos;
  - `compress` sem as rotas de eventos.
- [ ] 1.3 Limites de 500 conexões no total e 10 por IP (com `trusted-proxies`), com 429 e `Retry-After: 30`, liberando a vaga ao encerrar. Encerramento em `stop` antes de `controller.Shutdown`, 503 durante a parada e `Open` em `start`.
- [ ] 1.4 `controller`: `HeaderReceived` com `WriteTimeout` de 6 minutos só para as rotas de eventos. Teste de conexão aberta além de 15 s num servidor real.
- [ ] 1.5 Cache da API pública de detalhes com o instante do último resultado na chave. Teste de resultado novo antes dos 30 s.
- [ ] 1.6 Testes de API: aviso de push, heartbeat e verificação; evento sem mensagem; `Last-Event-ID`; 401 sem `WWW-Authenticate`; 404 público; limites; recarga sem esperar.

## 2. Frontend

- [ ] 2.1 `utils/liveUpdates.js` (`EventSource`, junção de 500 ms, aba oculta, desistência depois de erro fatal), com testes unitários da lógica pura.
- [ ] 2.2 `EndpointDetails.vue` e `StatusPageEndpoint.vue`: atualização sem indicador de carregamento a cada aviso, fallback periódico mantido.
- [ ] 2.3 `ResponseTimeChart.vue`: prop `refreshKey`, com a linha recarregada sem spinner a cada atualização da página.
- [ ] 2.4 Lint, `npm run test:unit` e `make frontend-build`.

## 3. Documentação, E2E e entrega

- [ ] 3.1 `docs/push-monitoring.md` e `docs/status-pages.md`: tempo real, rotas, limites, nginx (`proxy_buffering off`, `proxy_read_timeout`, HTTP/2), várias instâncias. `AGENTS.fork.md`.
- [ ] 3.2 E2E em `test/e2e/push.sh` e `test/e2e/status-pages.sh`: push Pending aparecendo nos detalhes sem recarregar (dashboard e página pública) e prints.
- [ ] 3.3 `go test ./... -race` com PostgreSQL, MySQL e MariaDB, `make lint` e `openspec validate realtime-endpoint-updates --strict`.
- [ ] 3.4 PR no `jniltinho/gatus` com CI verde e merge, release com imagem no Docker Hub, pacote `mariadb` e arquivamento da change.
