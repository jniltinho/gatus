## MODIFIED Requirements

### Requirement: Payload público sanitizado
A resposta de `GET /api/v1/status-pages/:slug` MUST conter apenas:
- `slug`, `title`, `description`, `status`, `updatedAt`, `truncated`, `groupsCollapsed`, `summary`, `featured` e `groups`;
- em `summary`, a contagem dos endpoints da página por estado: `total`, `up`, `down`, `pending` e `unknown`;
- em cada grupo, `name`, `status`, `summary` e `endpoints`, com `summary` nos mesmos cinco campos do `summary` da página;
- em cada destaque, os campos de endpoint e `group`;
- em cada endpoint, `name`, `status`, `uptime` (`24h`, `7d`, `30d`), `responseTime` (`24h`, `7d`, `30d`) e `results`, e `certificateExpiresInDays` somente quando a página tem `show-certificate-expiration: true` e o endpoint tem resultado publicado com certificado;
- em cada resultado, `timestamp`, `success` e `durationMs`, e `pending: true` somente quando o resultado é Pending.

MUST NOT conter nenhum outro campo, nem os valores de chave, URL, hostname, IP, porta, código HTTP, código DNS, erros, mensagens, condições, eventos, datas de expiração, alertas, `extra-labels` ou origem do endpoint. `certificateExpiresInDays` MUST ser um número inteiro de dias, sem data. `updatedAt` MUST ser o instante da montagem, no relógio do servidor. `groupsCollapsed` MUST ser o valor de `groups-collapsed` da definição da página, `false` quando ausente.

A contagem de `summary` MUST considerar todos os endpoints do payload, inclusive os destaques, e `total` MUST ser a soma dos quatro estados. Numa página truncada ela MUST contar os endpoints publicados, que são os mesmos que a página mostra junto do aviso dos 200 primeiros: carregar o resumo dos demais anularia o corte. O `summary` de um grupo MUST contar, com as mesmas regras, só os endpoints listados naquele grupo: um endpoint em destaque não é listado em grupo nenhum e MUST NOT entrar na contagem de nenhum. O payload de detalhes de um endpoint MUST NOT conter `summary`.

Os resultados MUST ser os últimos `min(50, storage.maximum-number-of-results)`, do mais antigo para o mais recente. O uptime MUST ser `null` num período sem execuções, com qualquer tipo de storage. Uma página com mais de 200 endpoints MUST devolver os 200 primeiros na ordem de exibição, com `truncated: true`. As telas públicas MUST mostrar os resultados Pending em amarelo, com o rótulo "Pending".

#### Scenario: Resultado com dados sensíveis
- **WHEN** o último resultado do endpoint `core/api` tem hostname `10.0.0.5`, código HTTP 500, erro `dial tcp 10.0.0.5:443` e condições resolvidas
- **THEN** o JSON da página é decodificado sem erro por structs que só conhecem os campos permitidos, rejeitando campos desconhecidos
- **AND** não contém os textos `10.0.0.5` nem `dial tcp`

#### Scenario: Contagem dos endpoints
- **WHEN** a página `infra` publica 12 endpoints no ar, 2 fora, 1 pendente e 1 sem dados
- **THEN** `summary` é `{"total":16,"up":12,"down":2,"pending":1,"unknown":1}`

#### Scenario: Contagem numa página truncada
- **WHEN** a página `infra` seleciona 250 endpoints e os 200 publicados estão no ar
- **THEN** o payload tem `truncated: true` com 200 endpoints
- **AND** `summary.total` é 200 e `summary.up` é 200

#### Scenario: Contagem de um grupo
- **WHEN** o grupo `apis` lista quatro endpoints: dois no ar, um fora e um sem dados
- **THEN** o grupo tem `summary` igual a `{"total":4,"up":2,"down":1,"pending":0,"unknown":1}`

#### Scenario: Destaque fora da contagem do grupo
- **WHEN** `sites_website` está em destaque e fora do ar, e o grupo `sites` lista outros dois endpoints, no ar
- **THEN** o `summary` de `sites` é `{"total":2,"up":2,"down":0,"pending":0,"unknown":0}`
- **AND** o `summary` da página conta os três

#### Scenario: Estado inicial dos grupos no payload
- **WHEN** uma página tem `groups-collapsed: true` e outra não define o campo
- **THEN** o payload da primeira tem `groupsCollapsed: true` e o da segunda `groupsCollapsed: false`

## ADDED Requirements

### Requirement: Campo groups-collapsed da página
A definição de uma página MUST aceitar o campo opcional `groups-collapsed`, booleano com padrão `false`, no arquivo de configuração e nas páginas gerenciadas, e MUST preservá-lo na normalização, na pré-visualização e no backup da administração. Um valor que não seja booleano MUST ser recusado como os demais campos de tipo errado.

#### Scenario: Página gerenciada
- **WHEN** a administração grava uma página com `groups-collapsed: true` e a lê de volta
- **THEN** a definição devolvida tem `groups-collapsed: true`

#### Scenario: Tipo errado
- **WHEN** a definição enviada tem `groups-collapsed: "sim"`
- **THEN** a validação recusa a página

#### Scenario: Backup e restore
- **WHEN** uma página com `groups-collapsed: true` é salva num backup e restaurada noutra instalação da mesma versão
- **THEN** a página restaurada tem `groups-collapsed: true`

### Requirement: Grupos recolhíveis na página pública
A página pública MUST permitir recolher e expandir cada grupo por um elemento `button` no cabeçalho do grupo, com `aria-expanded` e `aria-controls`, acionável por mouse, Enter e Espaço, e com foco visível nos temas claro e escuro. Um grupo recolhido MUST NOT renderizar as linhas dos seus endpoints. Recolhido ou expandido, o cabeçalho MUST mostrar o nome do grupo, o estado agregado e a contagem do `summary` do grupo, omitindo os estados com zero. A seção de destaques MUST NOT ser recolhível. O aviso de página truncada MUST continuar visível com grupos recolhidos.

#### Scenario: Recolher um grupo
- **WHEN** o visitante aciona o cabeçalho do grupo `sites`, expandido e operacional
- **THEN** as linhas de `sites` deixam de existir no documento, o botão passa a `aria-expanded="false"` e o cabeçalho mostra o estado e a contagem do grupo

#### Scenario: Teclado
- **WHEN** o visitante foca o cabeçalho de um grupo e pressiona Enter ou Espaço
- **THEN** o grupo alterna entre recolhido e expandido

#### Scenario: Destaques
- **WHEN** a página tem endpoints em destaque
- **THEN** a seção `Featured` não tem botão de recolher

### Requirement: Precedência do estado de um grupo
A cada payload recebido, inclusive os do recarregamento periódico, o estado de cada grupo MUST ser decidido nesta ordem: um grupo cujo estado agregado não é `operational` MUST estar expandido; senão, vale a escolha lembrada do visitante para aquele grupo; senão, o grupo está recolhido quando `groupsCollapsed` é `true` e expandido quando é `false`. O visitante MAY recolher um grupo não operacional, e essa ação MUST valer só até o próximo payload e MUST NOT ser lembrada. Expandir à força MUST NOT apagar nem alterar a escolha lembrada.

#### Scenario: Página com grupos recolhidos e um problema
- **WHEN** a página tem `groupsCollapsed: true`, `sites` está `operational` e `apis` está `degraded`, sem escolhas lembradas
- **THEN** `sites` aparece recolhido e `apis` expandido

#### Scenario: Grupo recolhido pelo visitante passa a falhar
- **WHEN** o visitante recolheu `sites`, e um payload seguinte traz `sites` como `degraded`
- **THEN** `sites` aparece expandido
- **AND** quando um payload posterior traz `sites` como `operational`, ele volta a aparecer recolhido

#### Scenario: Recolher durante um incidente
- **WHEN** `apis` está `degraded` e o visitante o recolhe
- **THEN** `apis` fica recolhido até o próximo payload, que o expande de novo se o estado continuar não operacional
- **AND** nada é gravado no navegador

#### Scenario: Padrão da página sem escolha lembrada
- **WHEN** a página tem `groupsCollapsed: false` e o visitante nunca mexeu em `sites`
- **THEN** `sites` aparece expandido

### Requirement: Escolha do visitante lembrada sem nomes de grupos
A escolha de recolher ou expandir um grupo operacional MUST ser lembrada no navegador, por página e por grupo, e MUST NOT gravar o nome de nenhum grupo nem o slug em texto legível: a chave de cada escolha MUST ser derivada por SHA-256 do slug e do nome bruto do grupo do payload, em que o grupo sem nome é a string vazia. Quando o navegador não oferece `crypto.subtle` ou armazenamento, a página MUST funcionar sem lembrar. Dados guardados inválidos MUST ser ignorados. A pré-visualização da administração MUST NOT ler nem gravar essas escolhas.

#### Scenario: Lembrar entre visitas
- **WHEN** o visitante recolhe `sites` na página `services` e recarrega
- **THEN** `sites` aparece recolhido

#### Scenario: Páginas diferentes
- **WHEN** o visitante recolhe `sites` na página `services`
- **THEN** o grupo `sites` da página `internal` não é afetado

#### Scenario: Grupo sem nome e grupo chamado "Other services"
- **WHEN** a página tem um grupo sem nome, exibido como "Other services", e um grupo cujo nome é `Other services`
- **THEN** recolher um não recolhe o outro

#### Scenario: Página com login
- **WHEN** o visitante recolhe um grupo numa página com login próprio
- **THEN** o que fica no navegador não contém o nome do grupo nem o slug

#### Scenario: Armazenamento indisponível
- **WHEN** o navegador bloqueia o armazenamento ou a página é servida sem contexto seguro
- **THEN** recolher e expandir funcionam, e a escolha não é lembrada
