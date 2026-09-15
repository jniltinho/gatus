## Why

Com `security.basic`, o dashboard e a administração dependem da janela nativa de usuário e senha do navegador. A API protegida responde 401 com `WWW-Authenticate: Basic` e o navegador abre o prompt dele. Essa janela:
- não segue o visual do fork;
- não tem botão de sair e fica aberta até fechar o navegador;
- confunde quem usa gerenciadores de senha.

O OIDC já tem uma tela de login; o basic não. Os pacotes de deploy do fork usam `security.basic`, então todo administrador passa por essa janela.

## What Changes

- **Tela de login** `/login` para `security.basic`, sem o cabeçalho do dashboard:
  - cartão com logo, título (`ui.header`), usuário, senha e botão de entrar, centralizado na horizontal e a 15% do topo da tela;
  - mesmo padrão de tema do fork: claro/escuro pelo cookie de tema, com botão de alternar, e visual quadrado;
  - mensagem de erro genérica, sem revelar se o usuário existe.
- **Sessão por cookie**:
  - `POST /api/v1/auth/login` confere usuário e senha (bcrypt de `security.basic`) e cria uma sessão nova com cookie `gatus_session` (`HttpOnly`, `SameSite=Strict`, `Secure` com HTTPS), válida por `security.basic.session-ttl` (padrão 8h);
  - `POST /api/v1/auth/logout` encerra a sessão na hora, em todas as instâncias.
- **Armazenamento das sessões**:
  - tabela `login_sessions` do fork em SQLite, PostgreSQL e MySQL/MariaDB, com hash SHA-256 do token; com `storage.type: memory`, em memória;
  - as sessões sobrevivem a reinícios e recargas e servem para várias instâncias no mesmo banco;
  - cada requisição consulta a sessão no storage, sem cache, e trocar o usuário ou a senha do `security.basic` invalida as sessões existentes na mesma hora.
- **Rotas protegidas** passam a aceitar a sessão ou o header `Authorization: Basic` (compatível com `curl -u`, scripts e integrações):
  - `WWW-Authenticate: Basic` só é enviado quando a requisição não tem `Sec-Fetch-Site`, `Sec-Fetch-Mode` nem `X-Requested-With`;
  - o navegador nunca mais abre a janela nativa.
- **Frontend**:
  - `/api/v1/config` informa `login: "basic"` e se a requisição está autenticada, pela sessão ou pelo header;
  - sem autenticação, o dashboard, os detalhes de endpoints e suites e a administração levam a `/login?redirect=<caminho>`, com validação estrita do `redirect`;
  - o cabeçalho ganha o botão **Logout**.
- **Proteções**:
  - limite de falhas de autenticação por IP (com `status-pages.trusted-proxies`), válido no login e nas falhas de `Authorization: Basic` das rotas protegidas, conferido antes do bcrypt;
  - a mesma regra de origem da administração nas rotas de login e logout;
  - comparação de usuário e senha sem vazamento de tempo;
  - log de login com sucesso, falha e logout, sem senha nem token.
- **Autoria**: a administração continua tratando o usuário basic como administrador, agora também pela sessão, e a auditoria usa o usuário da sessão.
- **Sem mudança**:
  - status pages públicas, badges, `/api/push`, `/api/v1/endpoints/{key}/external` e `/health`;
  - o fluxo OIDC, cujas sessões continuam em memória. Com `security.oidc` configurado, o basic não é usado.

## Capabilities

### New Capabilities
- `basic-login-page`: tela de login, sessões por cookie guardadas no storage, login e logout, compatibilidade com `Authorization: Basic`, supressão da janela nativa no navegador, limite de falhas de autenticação e proteção de origem.

### Modified Capabilities
- `admin-access-control`: a autorização de administradores com `security.basic` passa a aceitar a sessão da tela de login, além das credenciais basic.

## Impact

- **Backend**:
  - `security/`: novo autenticador basic com sessão e limitador de falhas, `IsAuthenticated` e `RequestAuthor` para basic;
  - `config/`: `security.basic.session-ttl`;
  - `api/`: rotas `/api/v1/auth/*`, `/login` na SPA e campos em `/api/v1/config`;
  - `storage/store/sql` e `storage/store/memory`: tabela e store de sessões.
- **Frontend**:
  - nova tela `views/Login.vue` sem cabeçalho;
  - redirecionamento e botão Logout em `App.vue`;
  - tratamento de 401 em `Home.vue`, `EndpointDetails.vue`, `SuiteDetails.vue` e `utils/adminApi.js`.
- **Compatibilidade**:
  - quem usa `curl -u` ou `Authorization: Basic` continua funcionando, dentro do limite de falhas;
  - quem abria o dashboard com a janela do navegador passa a ver a tela de login.
- **Versão anterior do fork**: ignora a tabela nova e o campo `session-ttl`, pois o YAML é lido sem recusar campos desconhecidos, e volta à janela nativa.
- **Documentação**: `docs/admin-endpoints.md` (seção de segurança e login), `docs/README.md` (nota do fork em `security.basic`) e `README.md`.
- **Testes**:
  - sessão nos 4 bancos e na memória;
  - middleware com sessão, com header, sem credencial, com e sem `Sec-Fetch-*`;
  - limite no login e no header, e origem;
  - E2E com agent-browser: login, redirect, logout, tema claro e escuro, sem janela nativa, status page pública sem login.
