## ADDED Requirements

### Requirement: Mensagens opcionais nas status pages
Cada status page, do arquivo ou gerenciada, MUST aceitar `show-messages`, booleana e desligada por padrão. O formulário da administração MUST ter a opção "Show messages" junto de "Show certificate expiration". Somente com a opção ligada, o payload de detalhes do endpoint MUST incluir em cada resultado:
- `message`: a mensagem do resultado (envio ou heartbeat); senão o texto de heartbeat de resultados antigos, reconhecido pelo prefixo `heartbeat: no update received within ` nos erros; senão `HTTP <código>` de uma verificação ativa; senão ausente;
- `origin`: `push` nos resultados de envio.

Os erros das verificações ativas MUST NOT ser publicados. O payload da página MUST NOT conter mensagens nem origem, com ou sem a opção. Mudar a opção MUST publicar uma nova revisão da página.

#### Scenario: Página com mensagens
- **WHEN** a página `jobs` tem `show-messages: true` e o endpoint Push `jobs/backup` recebeu `status=down&msg=Disco cheio`
- **THEN** o resultado no payload de detalhes tem `message` `Disco cheio` e `origin` `push`

#### Scenario: Erro de verificação ativa
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/api` falhou com o erro `dial tcp 10.0.0.5:443` sem status HTTP
- **THEN** o payload de detalhes não contém `10.0.0.5` nem `dial tcp`

#### Scenario: Verificação ativa com status HTTP
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/site` recebeu HTTP 200
- **THEN** o resultado no payload de detalhes tem `message` `HTTP 200`

#### Scenario: Falha de heartbeat pública
- **WHEN** a página `jobs` tem `show-messages: true` e o endpoint Push `jobs/backup` ficou um intervalo de 1 minuto sem envio
- **THEN** o resultado no payload de detalhes tem `message` `heartbeat: no update received within 1m0s`

#### Scenario: Página sem a opção
- **WHEN** a página `jobs` não tem `show-messages`
- **THEN** o payload de detalhes não tem `message` nem `origin`

## MODIFIED Requirements

### Requirement: Seção de configuração status-pages
O arquivo de configuração MUST aceitar a seção opcional `status-pages` com:
- `enabled` (booleano, padrão `true`);
- `trusted-proxies` (lista de IPs ou CIDRs, padrão vazia);
- `rate-limit` (inteiro não negativo, padrão `120`, `0` desliga o limite);
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

### Requirement: Payload público sanitizado
A resposta de `GET /api/v1/status-pages/:slug` MUST conter apenas:
- `slug`, `title`, `description`, `status`, `updatedAt`, `truncated`, `featured` e `groups`;
- em cada grupo, `name`, `status` e `endpoints`;
- em cada destaque, os campos de endpoint e `group`;
- em cada endpoint, `name`, `status`, `uptime` (`24h`, `7d`, `30d`), `responseTime` (`24h`, `7d`, `30d`) e `results`, e `certificateExpiresInDays` somente quando a página tem `show-certificate-expiration: true` e o endpoint tem resultado publicado com certificado;
- em cada resultado, `timestamp`, `success` e `durationMs`, e `pending: true` somente quando o resultado é Pending.

MUST NOT conter nenhum outro campo, nem os valores de chave, URL, hostname, IP, porta, código HTTP, código DNS, erros, mensagens, condições, eventos, datas de expiração, alertas, `extra-labels` ou origem do endpoint. `certificateExpiresInDays` MUST ser um número inteiro de dias, sem data. `updatedAt` MUST ser o instante da montagem, no relógio do servidor.

Os resultados MUST ser os últimos `min(50, storage.maximum-number-of-results)`, do mais antigo para o mais recente. O uptime MUST ser `null` num período sem execuções, com qualquer tipo de storage. Uma página com mais de 200 endpoints MUST devolver os 200 primeiros na ordem de exibição, com `truncated: true`. As telas públicas MUST mostrar os resultados Pending em amarelo, com o rótulo "Pending".

#### Scenario: Resultado com dados sensíveis
- **WHEN** o último resultado do endpoint `core/api` tem hostname `10.0.0.5`, código HTTP 500, erro `dial tcp 10.0.0.5:443` e condições resolvidas
- **THEN** o JSON da página é decodificado sem erro por structs que só conhecem os campos permitidos, rejeitando campos desconhecidos
- **AND** não contém os textos `10.0.0.5` nem `dial tcp`

#### Scenario: Resultado Pending com mensagem
- **WHEN** o último resultado do endpoint Push `jobs/backup` é Pending com a mensagem `Aguardando disco`
- **THEN** o resultado no JSON da página tem `success: false` e `pending: true`
- **AND** o JSON da página não contém `Aguardando disco`, mesmo com `show-messages: true`

#### Scenario: Endpoint sem histórico de 30 dias
- **WHEN** o endpoint começou a ser monitorado há 2 horas
- **THEN** `uptime.24h`, `uptime.7d` e `uptime.30d` não são `null` e são calculados sobre as execuções existentes

#### Scenario: Endpoint sem execuções
- **WHEN** um endpoint selecionado ainda não tem nenhum registro no storage, com storage `memory`, `sqlite` ou `postgres`
- **THEN** ele aparece com `status: unknown`, `results` vazio e uptime `null` nos três períodos
- **AND** a resposta é 200

#### Scenario: Página grande
- **WHEN** a página seleciona 250 endpoints
- **THEN** a resposta tem 200 endpoints e `truncated: true`

#### Scenario: Expiração do certificado só com a opção
- **WHEN** o endpoint `core/site` tem resultados HTTPS com certificado e a página não tem `show-certificate-expiration`
- **THEN** o JSON da página não tem `certificateExpiresInDays` e é decodificado pelas structs dos campos permitidos
- **AND** com a opção ligada, o JSON tem `certificateExpiresInDays` inteiro para `site` e continua sem datas de expiração

### Requirement: Estados agregados
O estado de um endpoint MUST ser:
- `up` quando o último resultado teve sucesso;
- `pending` quando o último resultado é Pending;
- `down` quando o último resultado falhou sem ser Pending;
- `unknown` sem resultados.

O estado de um grupo e o da página MUST ser calculados sobre os endpoints conhecidos (ignorando `unknown`):
- `operational` se todos estiverem `up`;
- `down` se todos estiverem `down`;
- `degraded` em qualquer outra combinação, inclusive quando todos estiverem `pending`;
- `unknown` se nenhum for conhecido.

#### Scenario: Todos no ar
- **WHEN** todos os endpoints da página estão `up` e um está `unknown`
- **THEN** a página tem `status: operational`

#### Scenario: Falha parcial
- **WHEN** um endpoint do grupo `core` está `down` e os outros estão `up`
- **THEN** o grupo `core` e a página têm `status: degraded`

#### Scenario: Tudo fora
- **WHEN** todos os endpoints conhecidos da página estão `down`
- **THEN** a página tem `status: down`

#### Scenario: Endpoint Pending
- **WHEN** o grupo `jobs` tem um endpoint `pending` e os outros `up`
- **THEN** o endpoint tem `status: pending` e o grupo tem `status: degraded`

#### Scenario: Só Pending
- **WHEN** o único endpoint conhecido da página está `pending`
- **THEN** a página tem `status: degraded`
