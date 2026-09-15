## ADDED Requirements

### Requirement: Endpoints em destaque
Cada página MUST aceitar `featured`, uma lista de até 10 chaves de endpoints, no YAML e na administração. Os endpoints em destaque MUST contar como seleção da página (uma página só com `featured` é válida), MUST aparecer no payload em `featured`, na ordem da lista, com o nome real do grupo, e MUST NOT aparecer de novo nas seções de grupo. Uma chave sem endpoint publicável MUST ser ignorada na página e registrada como aviso na carga e na validação da administração. Os destaques MUST contar no limite de 200 endpoints, antes das seções.

#### Scenario: Endpoint em destaque fora da seção do grupo
- **WHEN** a página tem `groups: [core]` e `featured: [core_api]`, e o grupo `core` tem `api` e `web`
- **THEN** `featured` do payload tem só `api`, com `group` igual a `core`
- **AND** a seção `core` tem só `web`

#### Scenario: Página só com destaques
- **WHEN** a página não tem `groups` nem `endpoints` e tem `featured: [core_api]`
- **THEN** a página é válida e mostra `api` em destaque

#### Scenario: Limite de destaques
- **WHEN** a página tem 11 chaves em `featured`
- **THEN** a validação falha com erro de definição inválida (400 na administração)

### Requirement: Tempo de resposta médio no payload
Cada endpoint do payload público MUST ter `responseTime` com as médias de tempo de resposta em milissegundos de 24h, 7d e 30d (`null` sem execução no período), calculadas a partir das mesmas somas horárias do uptime, sem consulta adicional ao storage.

#### Scenario: Médias por período
- **WHEN** um endpoint teve execuções de 100 ms e 300 ms nas últimas 24 horas
- **THEN** `responseTime["24h"]` é 200

### Requirement: API pública de detalhes do endpoint
A rota pública `GET /api/v1/status-pages/:slug/endpoints/:key` MUST responder sem autenticação, para um endpoint mostrado pela página publicada, `{"page": {"slug", "title"}, "name", "group", "status", "updatedAt", "uptime", "responseTime", "results", "events"}`, com os resultados e o uptime iguais aos da página e os eventos `START`, `HEALTHY` e `UNHEALTHY` mais recentes (no máximo 50, em ordem cronológica, só com `type` e `timestamp`), sem chave, URL, hostname, erros ou condições. Página não publicada, chave de endpoint fora da página ou caminho desconhecido MUST responder o 404 idêntico das status pages, contando no limitador e sem ler o storage. A resposta MUST ficar em cache por 30 segundos por slug, revisão, geração e chave, com `singleflight`, o mesmo semáforo das montagens e cache negativo de 5 segundos para erro de leitura. Um endpoint da página ainda sem registro no storage MUST sair com estado `unknown` e sem eventos.

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

### Requirement: Campo charts obsoleto
O campo `charts` MUST continuar aceito no YAML e nas definições gerenciadas gravadas por versões anteriores, com os mesmos limites, mas MUST ser ignorado: o payload não tem `chart`, a carga MUST registrar aviso de campo obsoleto e a validação da administração MUST devolver aviso `type: "charts"`. O formulário da administração MUST NOT enviar `charts`, de modo que salvar a página remove o campo.

#### Scenario: Página gravada com charts
- **WHEN** uma página gerenciada gravada pelo `v5.36.0-fork.2` tem `charts: [core_api]`
- **THEN** a página continua publicada, sem gráfico embutido, e a validação devolve o aviso `charts`

### Requirement: Destaques e página de detalhes nas telas
A página pública MUST mostrar os destaques no topo, em cartões com nome, grupo, estado, uptime e tempo médio de resposta de 24h, 7d e 30d, último tempo de resposta, barras e um link "View details", e MUST NOT mostrar gráfico embutido. O nome de cada endpoint MUST levar à página pública `/status/<slug>/endpoints/<chave>`, que MUST seguir o layout da página de detalhes do endpoint do dashboard (`/endpoints/<chave>`): cartões de estado, tempo médio, faixa de tempo de resposta e última verificação, barras, gráfico Response Time Trend com o mesmo componente do dashboard e seletor 24h/7d/30d, badges de tempo de resposta, uptime e saúde, e eventos, com link de volta à status page, sem chamar `/api/v1/config` e sem 401. O formulário da administração MUST permitir marcar "em destaque", respeitando o limite de 10, e o cabeçalho da edição MUST ter o endereço público como link que abre em nova aba.

#### Scenario: Visitante abre os detalhes de um endpoint
- **WHEN** o visitante clica no nome de `panel` na status page `services`
- **THEN** a página `/status/services/endpoints/_panel` mostra o gráfico de tempo de resposta e os eventos
- **AND** ao escolher `7d` no seletor, o gráfico busca `/api/v1/endpoints/_panel/response-times/7d/history`, sem chamar `/api/v1/config` e sem 401

#### Scenario: Administrador marca destaque
- **WHEN** o administrador marca `api` como destaque e salva
- **THEN** a definição salva tem `featured: [core_api]` e não tem `charts`
- **AND** a pré-visualização mostra `api` em destaque
