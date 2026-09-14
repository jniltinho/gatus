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

### Requirement: Gráficos de tempo de resposta escolhidos por endpoint
Cada página MUST aceitar `charts`, uma lista de até 10 chaves de endpoints. Só os endpoints presentes na página (em destaque, por grupo ou por chave) e listados em `charts` MUST ter `chart: true` no payload; uma chave de `charts` fora da página MUST gerar aviso e ser ignorada.

A rota pública `GET /api/v1/status-pages/:slug/response-times/:duration`, com `duration` em `24h`, `7d` ou `30d`, MUST responder sem autenticação `{"duration": ..., "endpoints": [{"name", "group", "points": [{"timestamp", "ms"}]}]}` com as médias horárias de tempo de resposta dos endpoints com gráfico, na ordem da página, sem chave, URL, hostname ou erros. Página não publicada, duração inválida ou caminho desconhecido MUST responder o 404 idêntico das status pages, contando no limitador. A resposta MUST ficar em cache por 5 minutos por slug, revisão, geração e duração, com `singleflight`, o mesmo semáforo das montagens e cache negativo de 5 segundos para erro de leitura.

#### Scenario: Série de um endpoint com gráfico
- **WHEN** a página publicada `infra` tem `charts: [core_api]` e `api` teve execuções na última hora
- **THEN** `GET /api/v1/status-pages/infra/response-times/24h` responde 200 com `api` e ao menos um ponto
- **AND** a resposta não contém a chave `core_api`

#### Scenario: Duração inválida
- **WHEN** um visitante pede `GET /api/v1/status-pages/infra/response-times/1y`
- **THEN** a resposta é o 404 idêntico, sem `WWW-Authenticate`

#### Scenario: Página sem gráficos
- **WHEN** a página publicada não tem `charts`
- **THEN** a rota responde 200 com `endpoints` vazio, sem consultar as médias horárias

### Requirement: Destaques e gráficos nas telas
A página pública MUST mostrar os destaques no topo, em cartões com nome, grupo, estado, uptime e tempo médio de resposta de 24h, 7d e 30d, último tempo de resposta e barras, e MUST mostrar o gráfico de tempo de resposta dos endpoints com `chart: true` no mesmo formato do gráfico da página de detalhes do endpoint do dashboard (`/endpoints/<chave>`), com um seletor de período (24h, 7d, 30d) em cada gráfico; nas linhas das seções, o gráfico MUST abrir por um botão acessível por teclado, e nos cartões em destaque MUST aparecer aberto. O formulário da administração MUST permitir marcar, para os endpoints resolvidos pela seleção atual, "em destaque" e "gráfico", respeitando os limites de 10.

#### Scenario: Visitante troca o período de um gráfico
- **WHEN** o visitante abre o gráfico de `api` e escolhe `7d` no seletor dele
- **THEN** a página busca `/api/v1/status-pages/<slug>/response-times/7d` e redesenha o gráfico, sem chamar `/api/v1/config` e sem 401

#### Scenario: Administrador marca destaque e gráfico
- **WHEN** o administrador marca `api` como destaque e com gráfico e salva
- **THEN** a definição salva tem `featured: [core_api]` e `charts: [core_api]`
- **AND** a pré-visualização mostra `api` em destaque
