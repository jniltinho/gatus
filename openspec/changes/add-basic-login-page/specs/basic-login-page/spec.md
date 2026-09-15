## ADDED Requirements

### Requirement: Tela de login do security.basic
Com `security.basic` configurado e sem `security.oidc`, o frontend MUST oferecer a rota `/login`. A rota MUST ser atendida pela SPA e MUST mostrar:
- um cartão com o logo, o título de `ui.header`, os campos de usuário e de senha e o botão de entrar;
- o cartão centralizado na horizontal, com a borda superior a 15% da altura da janela;
- o tema claro ou escuro lido do mesmo cookie de tema das outras telas, com um botão para alternar;
- o visual quadrado do fork e variantes `dark:`.

Com credenciais erradas, a tela MUST mostrar uma mensagem genérica que não indique se o usuário existe. Com o limite de tentativas estourado, a tela MUST pedir para tentar mais tarde. Depois do login, a tela MUST levar ao caminho de `redirect` quando ele for um caminho interno (começa com `/` e não com `//`), e ao dashboard nos outros casos.

#### Scenario: Visitante sem sessão abre a administração
- **WHEN** um navegador sem sessão abre `/admin`
- **THEN** a SPA mostra `/login?redirect=/admin` sem abrir a janela nativa de usuário e senha do navegador
- **AND** o cartão de login fica a 15% do topo da janela, no tema do cookie de tema

#### Scenario: Senha errada
- **WHEN** o visitante envia a senha errada
- **THEN** a tela mostra "Invalid username or password" e continua em `/login`

#### Scenario: Login e redirecionamento
- **WHEN** o visitante envia as credenciais corretas em `/login?redirect=/admin/status-pages`
- **THEN** a SPA abre `/admin/status-pages`

#### Scenario: Redirecionamento externo ignorado
- **WHEN** o visitante faz login em `/login?redirect=//site-malicioso.exemplo`
- **THEN** a SPA abre o dashboard `/`

### Requirement: Sessões de login
Um login com sucesso MUST criar uma sessão identificada por um token aleatório de 32 bytes e enviar o cookie `gatus_session` com `HttpOnly`, `Path=/`, `SameSite=Strict` e `Max-Age` igual à validade da sessão. O cookie MUST ter `Secure` quando a conexão for TLS ou `X-Forwarded-Proto` for `https`.

O storage MUST guardar só o hash SHA-256 do token, com o usuário, o fingerprint da credencial (usuário e hash bcrypt configurados) e as datas de criação e de expiração:
- com `sqlite`, `postgres` e `mysql`, na tabela `login_sessions`;
- com `memory`, em memória.

A validade MUST ser `security.basic.session-ttl`, com padrão de 8 horas, e a configuração MUST ser recusada fora da faixa de 5 minutos a 30 dias. Uma sessão expirada, removida ou com fingerprint diferente da credencial atual MUST ser recusada. Sessões expiradas MUST ser removidas periodicamente.

#### Scenario: Sessão sobrevive a um reinício
- **WHEN** o administrador faz login com storage `sqlite` e o Gatus reinicia antes da expiração
- **THEN** a mesma sessão continua autenticando as requisições

#### Scenario: Troca de senha
- **WHEN** o administrador troca `password-bcrypt-base64` no YAML e a configuração é recarregada
- **THEN** as sessões criadas com a senha anterior são recusadas

#### Scenario: Token no banco
- **WHEN** uma sessão é criada com storage `postgres`
- **THEN** a tabela `login_sessions` não contém o token em texto, só o hash SHA-256

#### Scenario: Validade inválida
- **WHEN** o YAML tem `security.basic.session-ttl: 1m`
- **THEN** a configuração é inválida

### Requirement: API de login e logout
`POST /api/v1/auth/login` MUST aceitar JSON com `username` e `password` e MUST responder:
- 204 com o cookie da sessão para credenciais corretas;
- 401 com a mensagem genérica "Invalid username or password" para credenciais erradas;
- 429 quando o limite de tentativas estiver estourado;
- 404 quando a configuração não tiver `security.basic` ou tiver `security.oidc`.

`POST /api/v1/auth/logout` MUST remover a sessão do cookie, se existir, expirar o cookie e responder 204.

As duas rotas MUST:
- responder com `Cache-Control: no-store`;
- recusar com 403 `Sec-Fetch-Site: cross-site` e `Origin` (ou `Referer`) fora de `admin.allowed-origins` ou da origem derivada de `Host` e do esquema;
- recusar com 415 corpos que não sejam JSON e com 413 corpos acima de 4 KB.

Cada login MUST criar uma sessão nova. A senha e o token MUST NOT aparecer em logs nem em respostas.

#### Scenario: Login pela API
- **WHEN** um cliente envia `POST /api/v1/auth/login` com as credenciais corretas e `Content-Type: application/json`
- **THEN** a resposta é 204 com `Set-Cookie: gatus_session=...; HttpOnly; SameSite=Strict`

#### Scenario: Login de outro site
- **WHEN** chega `POST /api/v1/auth/login` com `Origin: https://site-malicioso.exemplo`
- **THEN** a resposta é 403 e nenhuma sessão é criada

#### Scenario: Logout
- **WHEN** o administrador com sessão envia `POST /api/v1/auth/logout`
- **THEN** a resposta é 204 com o cookie expirado
- **AND** requisições seguintes com o token antigo recebem 401

### Requirement: Rotas protegidas com sessão ou Authorization Basic
Com `security.basic`, as rotas protegidas da API MUST aceitar uma sessão válida ou o header `Authorization: Basic` com as credenciais corretas. Sem nenhum dos dois, MUST responder 401 com `Cache-Control: no-store`. A resposta 401 MUST incluir `WWW-Authenticate: Basic` somente quando a requisição não tiver `Sec-Fetch-Site`, `Sec-Fetch-Mode` nem `X-Requested-With`. O frontend MUST enviar `X-Requested-With: XMLHttpRequest` nas chamadas à API protegida. A autoria das escritas da administração MUST ser o usuário da sessão ou do header.

#### Scenario: Script com curl
- **WHEN** `curl -u admin:senha` pede `GET /api/v1/endpoints/statuses`
- **THEN** a resposta é 200

#### Scenario: Curl sem credenciais
- **WHEN** `curl` pede `GET /api/v1/endpoints/statuses` sem credenciais nem cookie
- **THEN** a resposta é 401 com `WWW-Authenticate: Basic`

#### Scenario: Navegador sem sessão
- **WHEN** o navegador pede `GET /api/v1/endpoints/statuses` com `Sec-Fetch-Site: same-origin` e sem sessão
- **THEN** a resposta é 401 sem `WWW-Authenticate`

### Requirement: Limite de tentativas de login
O sistema MUST contar as tentativas de login com falha por IP de cliente, calculado com `status-pages.trusted-proxies` e com IPv6 agrupado por /64. Depois de 10 falhas no mesmo minuto, toda tentativa desse IP MUST responder 429 até a janela reiniciar, inclusive com as credenciais corretas. A verificação MUST levar tempo equivalente para usuário existente e inexistente.

#### Scenario: Força bruta
- **WHEN** um IP envia 11 senhas erradas no mesmo minuto
- **THEN** a 11ª tentativa responde 429
- **AND** uma tentativa com a senha certa do mesmo IP no mesmo minuto também responde 429

### Requirement: Estado de login exposto ao frontend
`GET /api/v1/config` MUST incluir `login`: `"basic"` com `security.basic` sem OIDC, `"oidc"` com `security.oidc`, e vazio sem `security`. `authenticated` MUST ser verdadeiro com uma sessão basic válida ou `Authorization: Basic` correto. Com `login: "basic"` e sem autenticação, o frontend MUST levar as telas protegidas (dashboard, detalhes e administração) para `/login` com o `redirect` do caminho atual. Um 401 recebido pelas chamadas do dashboard ou da administração MUST levar à mesma tela. Com `login: "basic"` e autenticação, o cabeçalho MUST mostrar o botão "Logout", que chama `POST /api/v1/auth/logout` e leva a `/login`. As status pages públicas MUST continuar abrindo sem login.

#### Scenario: Estado sem sessão
- **WHEN** a configuração usa `security.basic` e um navegador sem sessão consulta `GET /api/v1/config`
- **THEN** a resposta contém `"login": "basic"` e `"authenticated": false`

#### Scenario: Logout pelo cabeçalho
- **WHEN** o administrador logado clica em "Logout" no dashboard
- **THEN** a sessão é encerrada e a SPA mostra `/login`

#### Scenario: Status page pública
- **WHEN** um visitante sem sessão abre `/status/services`
- **THEN** a página pública abre sem passar pela tela de login
