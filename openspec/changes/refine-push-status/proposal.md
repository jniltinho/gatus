## Why

Hoje o push só tem dois status: `status=up` registra sucesso e qualquer outro valor registra falha. Um job não consegue avisar que está em andamento, e um heartbeat atrasado vira queda na hora. Por isso o dono do fork quer um terceiro status, **Pending**, em amarelo. É uma funcionalidade do Gatus: no Uptime Kuma 2.5.4, conferido localmente, `status=pending` vira Down e o painel só tem verde e vermelho. O Pending entra por dois caminhos:
- explícito, com `status=pending`;
- pelas tentativas (Retries), que seguram as quedas de um endpoint Push antes do Down.

As páginas de detalhes do endpoint (dashboard e status page pública) também devem mostrar as informações da página do monitor do Kuma:
- números atuais e de uptime numa linha;
- gráfico com os períodos fora do ar destacados.

Hoje o uptime aparece em imagens de badge, o gráfico marca as quedas com linhas tracejadas, e a página pública de detalhes não tem a tabela de verificações com mensagens que o dashboard tem.

## What Changes

- **Status Pending (funcionalidade do Gatus):**
  - `status=pending` na URL de push registra um resultado Pending com a mensagem do envio. `up` e ausente continuam sucesso, e qualquer outro valor continua falha;
  - Pending não conta falha nem sucesso para os alertas e não cria evento. No uptime e nas métricas conta como indisponível;
  - vale para endpoints Push e para pushes em endpoints ativos;
  - sem `status=pending` e sem tentativas, nada muda.
- **Tentativas (Retries) nos endpoints Push:**
  - nova opção `heartbeat.retries` (0 a 100, padrão 0) nos endpoints Push gerenciados e nos external endpoints do arquivo com heartbeat;
  - com tentativas, um envio down ou um intervalo sem envio registra Pending até esgotar as tentativas seguidas, e só então registra down, com alertas e evento;
  - um envio `up` zera a contagem;
  - o formulário da administração ganha o campo "Retries".
- **Amarelo nas telas:** barras, tooltip, estado atual, badge da tabela de verificações e resumo do dashboard, e barras, ponto, estado e tabela das status pages públicas mostram Pending em amarelo, com o rótulo "Pending". Os contadores de falha tratam Pending como falha.
- **Tabela de verificações do dashboard:** continua como está (todas as verificações, colunas Status, Date and time, Message e Origin, e a mesma regra de mensagem). A única mudança é o badge Pending em amarelo.
- **Mesma tabela na status page pública (opcional por página):**
  - nova opção `show-messages` (desligada por padrão, no YAML e no formulário da administração);
  - com a opção, a tabela da página pública de detalhes do endpoint fica igual à do dashboard: Status (Up, Down ou Pending), Date and time, Message e Origin;
  - sem a opção, a tabela pública continua com Status, Date and time e Response time;
  - mensagens publicáveis: a mensagem do envio, o texto do heartbeat gerado pelo Gatus e o status HTTP das verificações ativas (`HTTP 200`);
  - erros das verificações ativas nunca são publicados.
- **Informações das páginas de detalhes (dashboard e pública), como no Kuma:**
  - um único painel de números no lugar dos cartões e das imagens de uptime: Response (Current), ou Ping em endpoints Push; Avg. Response (24h); Uptime (24h), Uptime (7d) e Uptime (30d), em texto, com "—" sem dados;
  - o gráfico destaca os períodos fora do ar como faixas vermelhas, no lugar das linhas tracejadas, e mantém o seletor 24h/7d/30d;
  - não há Uptime de 1 ano, porque o Gatus guarda 30 dias de uptime;
  - a API protegida de status do endpoint ganha os campos do fork `push`, `uptime` e `responseTime`, calculados como no payload público.
- **Documentação:** `docs/push-monitoring.md` (status aceitos, tentativas, diferença para o Kuma) e `docs/status-pages.md` (estado `pending` e `show-messages`).

## Capabilities

### New Capabilities

- `endpoint-details-summary`: painel de números, faixas fora do ar no gráfico e campos `push`, `uptime` e `responseTime` na API protegida de status do endpoint.

### Modified Capabilities

- `push-monitoring`: `status=pending`, status Pending, tentativas no heartbeat e nos envios, exceção de Pending para os endpoints ativos e badge Pending na tabela de verificações.
- `public-status-pages`: resultados e estado Pending, agregação dos estados e opção `show-messages`.
- `status-page-highlights`: página pública de detalhes com Pending, mensagens opcionais, painel de números e faixas fora do ar.

## Impact

- **Backend:**
  - `api/push.go`, `watchdog/push.go` e `watchdog/external_endpoint.go` (Pending, tentativas, alertas);
  - `config/endpoint/result.go` e `config/endpoint/heartbeat` (`retries`);
  - `managedendpoint` (validação de `retries`);
  - eventos no storage SQL e em memória;
  - coluna `pending` na tabela do fork `endpoint_result_messages`, sem mudar tabelas do upstream;
  - `ResultSummary`;
  - `api/endpoint_status.go` (campos do fork);
  - `config/statuspage` (`show-messages`) e `statuspage/payload.go`.
- **API:**
  - resultados com `pending`;
  - status protegido do endpoint com `push`, `uptime` e `responseTime`;
  - detalhes públicos com `message` e `origin` nos resultados somente com `show-messages`.
- **Frontend:**
  - `EndpointCard.vue`, `Tooltip.vue`, `StatusBadge.vue`, `RecentChecksTable.vue`, `Home.vue`, `EndpointDetails.vue`, `ResponseTimeChart.vue`;
  - `public/EndpointRow.vue`, `views/public/StatusPageEndpoint.vue`, `utils/statusPage.js`;
  - `AdminEndpointForm.vue` (Retries) e `AdminStatusPageForm.vue` (Show messages).
- **Fora do escopo:** `SuiteDetails.vue` e as suites.
- **Compatibilidade:** um binário anterior mostra os resultados Pending como falhas e ignora `heartbeat.retries` e `show-messages` no YAML do arquivo. As definições gerenciadas com esses campos são rejeitadas pela decodificação estrita da versão anterior, então o rollback exige remover os campos antes.
- **Testes:** Go nos 4 bancos e na memória; testes unitários do frontend; E2E de `test/e2e/push.sh` e `test/e2e/status-pages.sh` com prints claro e escuro.
