## MODIFIED Requirements

### Requirement: API pública de detalhes do endpoint
A rota pública `GET /api/v1/status-pages/:slug/endpoints/:key` MUST responder sem autenticação, para um endpoint mostrado pela página publicada, `{"page": {"slug", "title"}, "name", "group", "status", "updatedAt", "uptime", "responseTime", "results", "events"}`, com os resultados e o uptime iguais aos da página e os eventos `START`, `HEALTHY` e `UNHEALTHY` mais recentes (no máximo 50, em ordem cronológica, só com `type` e `timestamp`), sem chave, URL, hostname, erros ou condições. Os resultados MUST ter `pending: true` quando forem Pending. Quando a página tem `show-certificate-expiration: true` e o endpoint tem resultado publicado com certificado, a resposta MUST ter também `certificateExpiresInDays`, calculado como no payload da página. Quando a página tem `show-messages: true`, os resultados MUST ter `message` e `origin` conforme as mensagens opcionais das status pages. Página não publicada, chave de endpoint fora da página ou caminho desconhecido MUST responder o 404 idêntico das status pages, contando no limitador e sem ler o storage. A resposta MUST ficar em cache por 30 segundos por slug, revisão, geração e chave, com `singleflight`, o mesmo semáforo das montagens e cache negativo de 5 segundos para erro de leitura. Um endpoint da página ainda sem registro no storage MUST sair com estado `unknown` e sem eventos.

#### Scenario: Detalhes de um endpoint da página
- **WHEN** a página publicada `infra` mostra `api` do grupo `core`, que teve uma execução com sucesso e depois uma falha
- **THEN** `GET /api/v1/status-pages/infra/endpoints/core_api` responde 200 com `name` `api`, `status` `down` e eventos de `START` a `UNHEALTHY`
- **AND** a resposta não contém a chave `core_api`, o hostname nem os erros

#### Scenario: Endpoint fora da página
- **WHEN** um visitante pede `GET /api/v1/status-pages/infra/endpoints/database_pg` e `database_pg` não está na página `infra`
- **THEN** a resposta é o 404 idêntico, sem `WWW-Authenticate` e sem leitura do storage

#### Scenario: Rota antiga de gráficos
- **WHEN** um visitante pede `GET /api/v1/status-pages/infra/response-times/24h`
- **THEN** a resposta é o 404 idêntico

#### Scenario: Detalhes com a expiração do certificado
- **WHEN** a página `infra` tem `show-certificate-expiration: true` e `api` tem certificado que vence em 73 dias
- **THEN** `GET /api/v1/status-pages/infra/endpoints/core_api` tem `certificateExpiresInDays` igual a 73
- **AND** sem a opção, a resposta não tem o campo

#### Scenario: Detalhes com mensagens
- **WHEN** a página `jobs` tem `show-messages: true` e o endpoint Push `backup` recebeu `status=pending&msg=Aguardando`
- **THEN** o último resultado de `GET /api/v1/status-pages/jobs/endpoints/jobs_backup` tem `pending: true`, `message` `Aguardando` e `origin` `push`

### Requirement: Destaques e página de detalhes nas telas
A página pública MUST mostrar os destaques no topo, em cartões com nome, grupo, estado, uptime e tempo médio de resposta de 24h, 7d e 30d, último tempo de resposta, barras e um link "View details", e MUST NOT mostrar gráfico embutido. O nome de cada endpoint MUST levar à página pública `/status/<slug>/endpoints/<chave>`, que MUST seguir o layout da página de detalhes do endpoint do dashboard (`/endpoints/<chave>`), nesta ordem:
- barras;
- painel de números;
- gráfico Response Time Trend com o mesmo componente do dashboard, seletor 24h/7d/30d e faixas fora do ar;
- tabela de verificações;
- badges de tempo de resposta, saúde e eventos.

A página MUST ter link de volta à status page, sem chamar `/api/v1/config` e sem 401. O estado Pending MUST aparecer em amarelo. A tabela de verificações MUST ser igual à do dashboard (Status, Date and time, Message e Origin) quando a página tem `show-messages: true`, e MUST mostrar Status, Date and time e Response time sem a opção. O formulário da administração MUST permitir marcar "em destaque", respeitando o limite de 10, e o cabeçalho da edição MUST ter o endereço público como link que abre em nova aba.

#### Scenario: Visitante abre os detalhes de um endpoint
- **WHEN** o visitante clica no nome de `panel` na status page `services`
- **THEN** a página `/status/services/endpoints/_panel` mostra o painel de números, o gráfico de tempo de resposta e os eventos
- **AND** ao escolher `7d` no seletor, o gráfico busca `/api/v1/endpoints/_panel/response-times/7d/history`, sem chamar `/api/v1/config` e sem 401

#### Scenario: Tabela pública com mensagens
- **WHEN** a página `jobs` tem `show-messages: true` e o visitante abre os detalhes de `backup`
- **THEN** a tabela de verificações tem as colunas Status, Date and time, Message e Origin, como no dashboard

#### Scenario: Endpoint Pending na página pública
- **WHEN** o último resultado de `backup` é Pending
- **THEN** o estado, a barra e o badge da tabela aparecem em amarelo com o rótulo Pending, e não "No data"

#### Scenario: Administrador marca destaque
- **WHEN** o administrador marca `api` como destaque e salva
- **THEN** a definição salva tem `featured: [core_api]` e não tem `charts`
- **AND** a pré-visualização mostra `api` em destaque
