## MODIFIED Requirements

### Requirement: Rotas protegidas com sessão ou Authorization Basic
Com `security.basic` sem `security.oidc`, as rotas protegidas da API MUST aceitar uma sessão válida ou o header `Authorization: Basic` com as credenciais corretas. Um cookie de sessão ausente, desconhecido ou expirado MUST NOT impedir a autenticação pelo header. Sem autenticação, MUST responder 401 com `Cache-Control: no-store`. A resposta 401 MUST incluir `WWW-Authenticate: Basic` somente quando a requisição não tiver `Sec-Fetch-Site`, `Sec-Fetch-Mode`, `X-Requested-With` nem `Accept` com `text/event-stream` (canal de eventos, que não aceita cabeçalhos próprios e, em página HTTP fora de localhost, não recebe `Sec-Fetch-*` do navegador). O frontend MUST enviar `X-Requested-With: XMLHttpRequest` nas chamadas à API protegida. A autoria das escritas da administração MUST ser o usuário da sessão ou do header.

#### Scenario: Script com curl
- **WHEN** `curl -u admin:senha` pede `GET /api/v1/endpoints/statuses`
- **THEN** a resposta é 200

#### Scenario: Curl com cookie antigo
- **WHEN** `curl -u admin:senha` pede `GET /api/v1/endpoints/statuses` com um cookie `gatus_session` expirado
- **THEN** a resposta é 200

#### Scenario: Curl sem credenciais
- **WHEN** `curl` pede `GET /api/v1/endpoints/statuses` sem credenciais nem cookie
- **THEN** a resposta é 401 com `WWW-Authenticate: Basic`

#### Scenario: Navegador sem sessão
- **WHEN** o navegador pede `GET /api/v1/endpoints/statuses` com `Sec-Fetch-Site: same-origin` e sem sessão
- **THEN** a resposta é 401 sem `WWW-Authenticate`

#### Scenario: Canal de eventos sem sessão em HTTP
- **WHEN** o navegador, numa página HTTP fora de localhost, pede `GET /api/v1/endpoints/jobs_backup/events` com `Accept: text/event-stream`, sem `Sec-Fetch-*` e sem sessão
- **THEN** a resposta é 401 sem `WWW-Authenticate`
