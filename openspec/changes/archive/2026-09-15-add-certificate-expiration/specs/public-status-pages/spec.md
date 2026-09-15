## MODIFIED Requirements

### Requirement: Seção de configuração status-pages
O arquivo de configuração MUST aceitar a seção opcional `status-pages` com:
- `enabled` (booleano, padrão `true`);
- `trusted-proxies` (lista de IPs ou CIDRs, padrão vazia);
- `rate-limit` (inteiro não negativo, padrão `120`, `0` desliga o limite);
- `pages` (lista de páginas).

Cada página MUST aceitar `slug`, `title`, `description`, `groups`, `endpoints`, `show-certificate-expiration` (booleano, padrão `false`) e `enabled` (padrão `true` no YAML). Com `enabled: false` na seção, nenhuma página MUST ser publicada, e as rotas públicas MUST continuar respondendo como para uma página inexistente, sem `WWW-Authenticate`, com ou sem `security`.

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

### Requirement: Payload público sanitizado
A resposta de `GET /api/v1/status-pages/:slug` MUST conter apenas:
- `slug`, `title`, `description`, `status`, `updatedAt`, `truncated` e `groups`;
- em cada grupo, `name`, `status` e `endpoints`;
- em cada endpoint, `name`, `status`, `uptime` (`24h`, `7d`, `30d`) e `results`, e `certificateExpiresInDays` somente quando a página tem `show-certificate-expiration: true` e o endpoint tem resultado publicado com certificado;
- em cada resultado, `timestamp`, `success` e `durationMs`.

MUST NOT conter nenhum outro campo, nem os valores de chave, URL, hostname, IP, porta, código HTTP, código DNS, erros, condições, eventos, datas de expiração, alertas, `extra-labels` ou origem do endpoint. `certificateExpiresInDays` MUST ser um número inteiro de dias, sem data. `updatedAt` MUST ser o instante da montagem, no relógio do servidor.

Os resultados MUST ser os últimos `min(50, storage.maximum-number-of-results)`, do mais antigo para o mais recente. O uptime MUST ser `null` num período sem execuções, com qualquer tipo de storage. Uma página com mais de 200 endpoints MUST devolver os 200 primeiros na ordem de exibição, com `truncated: true`.

#### Scenario: Resultado com dados sensíveis
- **WHEN** o último resultado do endpoint `core/api` tem hostname `10.0.0.5`, código HTTP 500, erro `dial tcp 10.0.0.5:443` e condições resolvidas
- **THEN** o JSON da página é decodificado sem erro por structs que só conhecem os campos permitidos, rejeitando campos desconhecidos
- **AND** não contém os textos `10.0.0.5` nem `dial tcp`

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
