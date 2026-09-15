## MODIFIED Requirements

### Requirement: API pública de detalhes do endpoint
A rota pública `GET /api/v1/status-pages/:slug/endpoints/:key` MUST responder sem autenticação, para um endpoint mostrado pela página publicada, `{"page": {"slug", "title"}, "name", "group", "status", "updatedAt", "uptime", "responseTime", "results", "events"}`, com os resultados e o uptime iguais aos da página e os eventos `START`, `HEALTHY` e `UNHEALTHY` mais recentes (no máximo 50, em ordem cronológica, só com `type` e `timestamp`), sem chave, URL, hostname, erros ou condições. Quando a página tem `show-certificate-expiration: true` e o endpoint tem resultado publicado com certificado, a resposta MUST ter também `certificateExpiresInDays`, calculado como no payload da página. Página não publicada, chave de endpoint fora da página ou caminho desconhecido MUST responder o 404 idêntico das status pages, contando no limitador e sem ler o storage. A resposta MUST ficar em cache por 30 segundos por slug, revisão, geração e chave, com `singleflight`, o mesmo semáforo das montagens e cache negativo de 5 segundos para erro de leitura. Um endpoint da página ainda sem registro no storage MUST sair com estado `unknown` e sem eventos.

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
