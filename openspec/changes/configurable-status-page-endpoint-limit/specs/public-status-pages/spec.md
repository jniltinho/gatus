## MODIFIED Requirements

### Requirement: Seção de configuração status-pages
O arquivo de configuração MUST aceitar a seção opcional `status-pages` com:
- `enabled` (booleano, padrão `true`);
- `trusted-proxies` (lista de IPs ou CIDRs, padrão vazia);
- `rate-limit` (inteiro não negativo, padrão `120`, `0` desliga o limite);
- `maximum-endpoints-per-page` (inteiro de `1` a `1000`, padrão `200`): quantos endpoints uma página mostra;
- `pages` (lista de páginas).

Cada página MUST aceitar `slug`, `title`, `description`, `groups`, `endpoints`, `show-certificate-expiration` (booleano, padrão `false`), `show-messages` (booleano, padrão `false`) e `enabled` (padrão `true` no YAML). Com `enabled: false` na seção, nenhuma página MUST ser publicada, e as rotas públicas MUST continuar respondendo como para uma página inexistente, sem `WWW-Authenticate`, com ou sem `security`.

#### Scenario: Página definida no YAML
- **WHEN** a configuração tem `status-pages.pages` com a página `infra`, título `Infraestrutura` e `groups: [core]`
- **THEN** a configuração é válida
- **AND** `GET /api/v1/status-pages/infra` responde 200

#### Scenario: Seção desligada
- **WHEN** a configuração tem `security.basic`, `status-pages.enabled: false` e a página `infra`
- **THEN** `GET /api/v1/status-pages/infra` responde o 404 idêntico sem `WWW-Authenticate`
- **AND** `GET /status/infra` responde 200 com o HTML da SPA

#### Scenario: Sem a seção
- **WHEN** a configuração não tem a seção `status-pages`
- **THEN** a configuração é válida
- **AND** `GET /api/v1/status-pages/qualquer` responde 404

#### Scenario: Página com a expiração do certificado
- **WHEN** a página `infra` do YAML tem `show-certificate-expiration: true`
- **THEN** a configuração é válida
- **AND** os endpoints da página com certificado têm `certificateExpiresInDays` no payload

#### Scenario: Página com mensagens
- **WHEN** a página `jobs` do YAML tem `show-messages: true`
- **THEN** a configuração é válida
- **AND** o payload de detalhes dos endpoints de `jobs` tem `page.showMessages: true`

#### Scenario: Limite de endpoints fora do intervalo
- **WHEN** a configuração tem `status-pages.maximum-endpoints-per-page` igual a `0`, a `-1`, a `1001` ou a um valor que não é inteiro
- **THEN** a configuração é inválida, no início e em `gatus config validate`, com a mensagem citando o intervalo aceito

### Requirement: Validação das páginas
A validação do arquivo de configuração MUST ser estrutural e MUST recusar:
- `slug` fora de `^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`;
- `slug` reservado (`options`, `validate`, `new`, `preview`, `exposure`);
- `slug` repetido;
- `title` vazio ou com mais de 100 runas depois de remover espaços das pontas;
- `description` com mais de 1000 runas;
- página sem nenhum item em `groups`, em `endpoints` e em `featured` (uma página só com destaques é válida, como define `status-page-highlights`);
- mais de 50 grupos, grupo com mais de 200 runas ou mais de 1000 chaves, qualquer que seja `maximum-endpoints-per-page`;
- itens repetidos;
- entradas de `trusted-proxies` que não sejam IP nem CIDR;
- `rate-limit` negativo.

Um grupo ou uma chave sem endpoint correspondente MUST NOT invalidar a página. O aviso correspondente MUST ser registrado na carga das páginas, depois da carga dos endpoints gerenciados, e MUST NOT ser registrado na validação do arquivo.

#### Scenario: Slug inválido
- **WHEN** uma página do YAML tem `slug: Infra_1`
- **THEN** a configuração é inválida com uma mensagem citando o slug

#### Scenario: Slug reservado
- **WHEN** uma página do YAML tem `slug: options`
- **THEN** a configuração é inválida

#### Scenario: Slug repetido
- **WHEN** duas páginas do YAML têm `slug: infra`
- **THEN** a configuração é inválida

#### Scenario: Página sem seleção
- **WHEN** uma página do YAML não tem `groups`, nem `endpoints`, nem `featured`
- **THEN** a configuração é inválida

#### Scenario: Título com acentos
- **WHEN** o título tem 100 caracteres acentuados (mais de 100 bytes)
- **THEN** a configuração é válida

#### Scenario: Proxy confiável inválido
- **WHEN** `status-pages.trusted-proxies` contém `nginx.local`
- **THEN** a configuração é inválida

#### Scenario: Grupo ainda inexistente
- **WHEN** uma página tem `groups: [pagamentos]` e nenhum endpoint do YAML nem gerenciado usa esse grupo
- **THEN** a configuração é válida
- **AND** a carga das páginas registra no log um aviso citando a página e o grupo

#### Scenario: Grupo existente só em gerenciados
- **WHEN** uma página do YAML tem `groups: [clientes]` e só endpoints gerenciados usam esse grupo
- **THEN** nenhum aviso de grupo sem correspondência é registrado

**Credencial da página:** com `auth` na definição, o usuário MUST ser não vazio e a senha MUST ser um hash bcrypt válido, codificado em base64 com o alfabeto URL, como em `security.basic`. Uma definição com `auth` incompleto ou com hash inválido MUST ser recusada, com a mesma severidade das demais validações de página. Sem `auth`, a página continua pública. Uma página com `auth` numa instalação **sem** `security` MUST registrar um aviso na carga, porque as rotas por chave do dashboard continuam abertas e publicam mais do que a página protegida.

#### Scenario: Credencial incompleta
- **WHEN** uma página traz `auth` com o usuário vazio, ou com um hash que não é bcrypt
- **THEN** a definição é recusada com erro de validação

#### Scenario: Página com login sem security na instalação
- **WHEN** o arquivo de configuração define uma página com `auth` e não define `security`
- **THEN** a carga registra um aviso de que as rotas por chave continuam públicas

#### Scenario: Mais chaves do que o limite de exibição
- **WHEN** uma definição lista 300 chaves de endpoint e `maximum-endpoints-per-page` é `200`
- **THEN** a definição é válida, no arquivo de configuração, na administração e num restore

#### Scenario: Acima do teto de chaves
- **WHEN** uma definição lista 1001 chaves de endpoint
- **THEN** a definição é recusada, mesmo com `maximum-endpoints-per-page: 1000`

### Requirement: Payload público sanitizado
A resposta de `GET /api/v1/status-pages/:slug` MUST conter apenas:
- `slug`, `title`, `description`, `status`, `updatedAt`, `truncated`, `groupsCollapsed`, `summary`, `featured` e `groups`;
- em `summary`, a contagem dos endpoints da página por estado: `total`, `up`, `down`, `pending` e `unknown`;
- em cada grupo, `name`, `status`, `summary` e `endpoints`, com `summary` nos mesmos cinco campos do `summary` da página;
- em cada destaque, os campos de endpoint e `group`;
- em cada endpoint, `name`, `status`, `uptime` (`24h`, `7d`, `30d`), `responseTime` (`24h`, `7d`, `30d`) e `results`, e `certificateExpiresInDays` e `certificateExpiresAt` somente quando a página tem `show-certificate-expiration: true` e o endpoint tem resultado publicado com certificado;
- em cada resultado, `timestamp`, `success` e `durationMs`, e `pending: true` somente quando o resultado é Pending.

MUST NOT conter nenhum outro campo, nem os valores de chave, URL, hostname, IP, porta, código HTTP, código DNS, erros, mensagens, condições, eventos, outras datas de expiração, alertas, `extra-labels` ou origem do endpoint. `certificateExpiresInDays` MUST ser um número inteiro de dias, e `certificateExpiresAt` MUST ser o instante do vencimento do mesmo resultado, como define a capacidade `certificate-expiration`: é a única data de expiração que o payload publica. `groupsCollapsed` MUST ser o valor de `groups-collapsed` da definição da página, `false` quando ausente. `updatedAt` MUST ser o instante da montagem, no relógio do servidor.

A contagem de `summary` MUST considerar todos os endpoints do payload, inclusive os destaques, e `total` MUST ser a soma dos quatro estados. Numa página truncada ela MUST contar os endpoints publicados, que são os mesmos que a página mostra junto do aviso de página truncada: carregar o resumo dos demais anularia o corte. O `summary` de um grupo MUST contar, com as mesmas regras, só os endpoints listados naquele grupo: um endpoint em destaque não é listado em grupo nenhum e MUST NOT entrar na contagem de nenhum. O payload de detalhes de um endpoint MUST NOT conter `summary`.

Os resultados MUST ser os últimos `min(50, storage.maximum-number-of-results)`, do mais antigo para o mais recente. O uptime MUST ser `null` num período sem execuções, com qualquer tipo de storage. Uma página que seleciona mais endpoints do que `maximum-endpoints-per-page` MUST devolver os primeiros, até esse limite, na ordem de exibição — os destaques antes das seções —, com `truncated: true`. O limite MUST ser o mesmo para o payload, para as rotas por endpoint da página e para as contagens da administração, inclusive durante uma recarga da configuração. As telas públicas MUST mostrar os resultados Pending em amarelo, com o rótulo "Pending".

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

#### Scenario: Sem a opção
- **WHEN** a seção `status-pages` não define `maximum-endpoints-per-page`, ou a configuração não tem a seção, e uma página seleciona 250 endpoints
- **THEN** o payload tem 200 endpoints e `truncated: true`

#### Scenario: Limite maior
- **WHEN** `maximum-endpoints-per-page` é `500` e a página `infra` seleciona 250 endpoints
- **THEN** o payload traz os 250, com `truncated: false`

#### Scenario: Limite menor depois de a página existir
- **WHEN** uma página gerenciada seleciona 300 endpoints, e o limite passa de `500` a `200` numa recarga da configuração
- **THEN** a página continua publicada, com 200 endpoints e `truncated: true`
- **AND** o payload guardado antes da recarga não é mais servido

#### Scenario: Limite 1 com vários destaques
- **WHEN** `maximum-endpoints-per-page` é `1` e a página tem três destaques e um grupo
- **THEN** o payload traz só o primeiro destaque, com `truncated: true` e `summary.total` igual a 1

## ADDED Requirements

### Requirement: O limite de exibição também limita o acesso
Um endpoint que a página seleciona mas que fica além de `maximum-endpoints-per-page` MUST NOT ser acessível pelas rotas por endpoint dessa página: a API de detalhes, o gráfico de tempo de resposta, o stream de eventos e os badges MUST responder 404, como respondem para um endpoint que a página não seleciona, depois das respostas que vêm antes dessa verificação — 401 numa página com login próprio sem a credencial, e 429 do limitador. A rota HTML de detalhes MUST continuar respondendo 200 com a SPA, sem depender da chave. Um payload de detalhes ou de gráfico guardado em cache MUST NOT ser servido para um endpoint que saiu do corte. A página, a credencial e o limite usados para autorizar um pedido MUST ser os mesmos usados para montar sua resposta, sem nova resolução da página no meio. Subir o limite MUST tornar acessíveis os endpoints que passam a ser exibidos, e baixá-lo MUST torná-los inacessíveis, a partir da recarga da configuração.

#### Scenario: Endpoint fora do corte
- **WHEN** a página `infra` seleciona 250 endpoints com o limite em `200`, e o visitante pede a API de detalhes, o gráfico, o stream de eventos e o badge do 201º na ordem de exibição
- **THEN** as quatro rotas respondem 404, como para um endpoint que a página não seleciona
- **AND** `GET /status/infra/endpoints/<chave>` responde 200 com o HTML da SPA

#### Scenario: Página com login
- **WHEN** a mesma página exige login e o pedido do 201º endpoint vem sem a credencial
- **THEN** a resposta é 401, e só com a credencial passa a 404

#### Scenario: Detalhes guardados antes de baixar o limite
- **WHEN** os detalhes do 201º endpoint foram servidos com o limite em `500`, o limite passa a `200` numa recarga, e nenhum resultado novo chegou para esse endpoint
- **THEN** o pedido seguinte responde 404, e não o payload guardado

#### Scenario: Limite elevado
- **WHEN** o limite passa a `500` numa recarga da configuração
- **THEN** as mesmas quatro rotas passam a responder para o 201º endpoint
