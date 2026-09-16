## MODIFIED Requirements

### Requirement: Destaques e página de detalhes nas telas
A página pública MUST mostrar os destaques no topo, em cartões com nome, grupo, estado, uptime e tempo médio de resposta de 24h, 7d e 30d, último tempo de resposta, barras e um link "View details", e MUST NOT mostrar gráfico embutido. O nome de cada endpoint MUST levar à página pública `/status/<slug>/endpoints/<chave>`, que MUST seguir o layout da página de detalhes do endpoint do dashboard (`/endpoints/<chave>`), nesta ordem:
- barras;
- painel de números;
- gráfico Response Time Trend com o mesmo componente do dashboard, no formato do Uptime Kuma, com o seletor Recent/3h/6h/24h/1w e a API pública do gráfico;
- tabela de verificações;
- badges de tempo de resposta, saúde e eventos.

A página MUST ter link de volta à status page, sem chamar `/api/v1/config` e sem 401. O estado Pending MUST aparecer em amarelo. A tabela de verificações MUST ser igual à do dashboard (Status, Date and time, Message e Origin) quando `page.showMessages` é verdadeiro, mesmo que nenhum resultado tenha mensagem, e MUST mostrar Status, Date and time e Response time sem a opção. Na página pública, a tabela MUST mostrar somente `message` e `origin` do payload, sem montar mensagem a partir de outros campos. O formulário da administração MUST permitir marcar "em destaque", respeitando o limite de 10, e o cabeçalho da edição MUST ter o endereço público como link que abre em nova aba.

#### Scenario: Visitante abre os detalhes de um endpoint
- **WHEN** o visitante clica no nome de `panel` na status page `services`
- **THEN** a página `/status/services/endpoints/_panel` mostra o painel de números, o gráfico de tempo de resposta e os eventos
- **AND** ao escolher `24h` no seletor, o gráfico busca `/api/v1/status-pages/services/endpoints/_panel/response-time-chart?period=24h`, sem chamar `/api/v1/config` e sem 401

#### Scenario: Tabela pública com mensagens
- **WHEN** a página `jobs` tem `show-messages: true` e o visitante abre os detalhes de `backup`
- **THEN** a tabela de verificações tem as colunas Status, Date and time, Message e Origin, como no dashboard

#### Scenario: Mensagens ligadas sem mensagens
- **WHEN** a página `infra` tem `show-messages: true` e o endpoint TCP `core/db` só tem resultados sem mensagem
- **THEN** a tabela de verificações tem as colunas Status, Date and time, Message e Origin, com a mensagem vazia

#### Scenario: Endpoint Pending na página pública
- **WHEN** o último resultado de `backup` é Pending
- **THEN** o estado, a barra e o badge da tabela aparecem em amarelo com o rótulo Pending, e não "No data"

#### Scenario: Administrador marca destaque
- **WHEN** o administrador marca `api` como destaque e salva
- **THEN** a definição salva tem `featured: [core_api]` e não tem `charts`
- **AND** a pré-visualização mostra `api` em destaque
