## 1. Backend: status Pending e tentativas

- [ ] 1.1 `endpoint.Result.Pending` (JSON `pending,omitempty`) e `status=pending` em `api/push.go`: resultado sem sucesso, mensagem fora dos erros, `lastPush` atualizado. Testes de API com `up`, `pending`, `warning` e sem parâmetros.
- [ ] 1.2 Coluna `pending` em `endpoint_result_messages`:
  - `CREATE TABLE` com a coluna;
  - `ALTER` idempotente nos três dialetos, tolerando o erro 1060 no MySQL/MariaDB;
  - gravação com mensagem, origem ou Pending;
  - leitura em `loadEndpointResultMessages`;
  - `ResultSummary` com `Pending`, `Message`, `Origin`, `HTTPStatus` e `Errors` (interno) em `GetEndpointSummaries` (SQL com `LEFT JOIN` e memória);
  - índice `(endpoint_id, endpoint_event_id)` em `endpoint_events` no SQLite e no PostgreSQL;
  - testes nos 4 bancos, incluindo tabela antiga sem a coluna (`mysql_schema_test.go` e SQLite/PostgreSQL).
- [ ] 1.3 Alertas e eventos:
  - Pending sem `HandleAlerting` nem cópia de contadores em `processExternalEndpointResult` e em `SubmitEndpointResult`;
  - eventos pelo tipo do último HEALTHY/UNHEALTHY no SQL e na memória (ignorando START), com a ausência tratada como vazio, sem erro.
  - Testes nos 4 storages: UP→PENDING→DOWN, UP→PENDING→UP, primeiro resultado Pending, Pending além de `maximum-number-of-results`, sequências sem Pending iguais às anteriores, contadores de alerta, uptime de 75%.
- [ ] 1.4 `heartbeat.retries` (0 a 100):
  - validação no arquivo (inválido sem `heartbeat.interval`) e nos endpoints gerenciados (400 fora da faixa e em endpoint ativo);
  - contador por chave junto de `lastPush`, zerado por `up`, intocado por `pending` e pela manutenção;
  - descarte do contador e do `lastPush` nos fluxos de renomear e remover de `managedendpoint` e num gancho novo da recarga, depois de `managedendpoint.Load`, para as chaves que deixam de existir;
  - conversão em Pending sem erros (erros viram mensagem), também na API upstream `/api/v1/endpoints/:key/external`;
  - texto do heartbeat em `Message` em todo resultado de heartbeat.
  - Testes do heartbeat, dos envios down, da API upstream, da manutenção, da remoção e da recarga.
- [ ] 1.5 API protegida de status com `push` (gerenciados em qualquer estado e `cfg.ExternalEndpoints` habilitados ou não), `uptime`, `responseTime` e `currentResponseTime` (nulos sem execução, independentes da paginação), usando funções exportadas do `statuspage`, com testes em memória e SQLite.
- [ ] 1.6 Status pages:
  - `show-messages` no YAML e nas definições gerenciadas (validação, versão, revisão);
  - `pending` nos resultados e estado `pending`;
  - `aggregateStatus` com `degraded`;
  - `page.showMessages` no payload de detalhes;
  - `message` (mensagem do resultado, prefixo de heartbeat nos erros antigos ou `HTTP <código>`, sem outros erros) e `origin` só no payload de detalhes com a opção.
  - Testes de sanitização com `DisallowUnknownFields` com e sem a opção, incluindo erro de verificação ativa fora do JSON.

## 2. Frontend: Pending, tabela pública e formulários

- [ ] 2.1 Amarelo e rótulo Pending:
  - `EndpointCard.vue`, `Tooltip.vue`, `StatusBadge.vue` (validator e estado atual), `RecentChecksTable.vue`;
  - resumo de `Home.vue` com Pending fora de down, e contadores de falha dos grupos;
  - `public/EndpointRow.vue`, `views/public/StatusPageEndpoint.vue` (sem "No data" para `pending`), `utils/statusPage.js`.
- [ ] 2.2 Tabela pública igual à do dashboard quando `page.showMessages` (só `message` e `origin` do payload, sem montar com erros), e com Response time sem a opção.
- [ ] 2.3 Formulários: "Retries" nos endpoints Push de `AdminEndpointForm.vue` e "Show messages" em `AdminStatusPageForm.vue`, com carga, gravação e pré-visualização.

## 3. Frontend: painel de números e gráfico

- [ ] 3.1 Painel de números em `EndpointDetails.vue` e `StatusPageEndpoint.vue`, no lugar dos quatro cartões e das imagens de uptime: rótulos Ping para `push: true` no dashboard, "—", duas colunas no celular. Formatação reutilizando `formatUptime` de `utils/statusPage.js`, e o restante em `utils/detailsSummary.js` com testes unitários.
- [ ] 3.2 Faixas fora do ar com `utils/downtime.js` (intervalos, START→HEALTHY sem faixa, HEALTHY inicial só com lista truncada, recorte por sobreposição) e anotações `box` em `ResponseTimeChart.vue` com eixo fixo no período e tooltip de início e duração, no lugar das linhas tracejadas. Gráfico visível com pelo menos um resultado nas duas telas, inclusive com durações zero. Testes unitários dos intervalos.
- [ ] 3.3 Lint, `npm run test:unit` e `make frontend-build`.

## 4. Documentação, E2E e entrega

- [ ] 4.1 `docs/push-monitoring.md` (status aceitos, tentativas, contador em memória, diferença para o Kuma, Pending no uptime e nas métricas, média com envios sem `ping`, rollback removendo os campos das definições gerenciadas), `docs/status-pages.md` (`pending`, `show-messages` e cuidado com mensagens de envio), `README.md` e `AGENTS.fork.md`.
- [ ] 4.2 E2E:
  - `test/e2e/push.sh`: envio Pending em amarelo, tentativas com heartbeat curto, painel com Ping e faixa de queda com envios com `ping`;
  - `test/e2e/status-pages.sh`: página com `show-messages` (tabela igual à do dashboard, erro de verificação ativa fora) e sem a opção, e estado Pending;
  - prints claro e escuro em `dist/prints/`.
- [ ] 4.3 `go test ./... -race` com PostgreSQL, MySQL e MariaDB, `make lint` e `openspec validate refine-push-status --strict`.
- [ ] 4.4 PR no `jniltinho/gatus` com CI verde e merge, release com imagem no Docker Hub, pacote `mariadb` e arquivamento da change.
