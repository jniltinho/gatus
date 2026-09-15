## Why

Com `security.basic`, o dashboard e a administração dependem da janela nativa de usuário e senha do navegador: a API protegida responde 401 com `WWW-Authenticate: Basic` e o navegador abre o prompt dele. Essa janela não segue o visual do fork, não tem botão de sair, fica aberta até fechar o navegador e confunde quem usa gerenciadores de senha. O OIDC já tem uma tela de login, o basic não. Os pacotes de deploy do fork usam `security.basic`, então todo administrador passa por essa janela.

## What Changes

- **Tela de login** `/login` para `security.basic`:
  - cartão com logo, título (`ui.header`), usuário, senha e botão de entrar, centralizado na horizontal e a 15% do topo da tela;
  - mesmo padrão de tema do fork: claro/escuro pelo cookie de tema, com botão de alternar, e visual quadrado;
  - mensagem de erro genérica, sem revelar se o usuário existe.
- **Sessão por cookie**:
  - `POST /api/v1/auth/login` confere usuário e senha (bcrypt de `security.basic`) e cria uma sessão com cookie `gatus_session` (`HttpOnly`, `SameSite=Strict`, `Secure` com HTTPS);
  - validade de `security.basic.session-ttl`, padrão 8h;
  - `POST /api/v1/auth/logout` encerra a sessão.
- **Armazenamento das sessões**: tabela `login_sessions` do fork em SQLite, PostgreSQL e MySQL/MariaDB, com hash SHA-256 do token. Com `storage.type: memory`, as sessões ficam em memória. As sessões sobrevivem a reinícios e recargas e servem para várias instâncias no mesmo banco. Trocar o usuário ou a senha do `security.basic` invalida as sessões existentes.
- **Rotas protegidas** passam a aceitar a sessão ou o header `Authorization: Basic` (compatível com `curl -u`, scripts e integrações):
  - `WWW-Authenticate: Basic` só é enviado a clientes que não são navegadores (sem `Sec-Fetch-*` nem `X-Requested-With`);
  - o navegador nunca mais abre a janela nativa.
- **Frontend**:
  - `/api/v1/config` informa `login: "basic"` e se a requisição está autenticada;
  - sem autenticação, o dashboard e a administração levam a `/login?redirect=<caminho>`, e depois do login voltam ao caminho;
  - o cabeçalho ganha o botão **Logout**.
- **Proteções**:
  - limite de tentativas erradas por IP (com `status-pages.trusted-proxies`);
  - verificação de origem e de `Sec-Fetch-Site` nas rotas de login e logout, como na administração;
  - comparação sem vazamento de tempo;
  - log de login com sucesso, falha e logout, sem senha nem token.
- **Autoria**: a administração continua tratando o usuário basic como administrador, agora também pela sessão, e a auditoria usa o usuário da sessão.
- **Sem mudança**:
  - status pages públicas, badges, `/api/push`, `/api/v1/endpoints/{key}/external` e `/health`;
  - o fluxo OIDC, cujas sessões continuam em memória.

## Capabilities

### New Capabilities
- `basic-login-page`: tela de login, sessões por cookie guardadas no storage, login e logout, compatibilidade com `Authorization: Basic`, supressão da janela nativa no navegador, limite de tentativas e proteção de origem.

### Modified Capabilities
- `admin-access-control`: a autorização de administradores com `security.basic` passa a aceitar a sessão da tela de login, além das credenciais basic.

## Impact

- **Backend**:
  - `security/` (novo autenticador basic com sessão, `IsAuthenticated` e `RequestAuthor` para basic);
  - `config/` (`security.basic.session-ttl`);
  - `api/` (rotas `/api/v1/auth/*`, `/login` na SPA e campos em `/api/v1/config`);
  - `storage/store/sql` e `storage/store/memory` (tabela e store de sessões).
- **Frontend**:
  - nova tela `views/Login.vue` com layout próprio;
  - redirecionamento e botão Logout em `App.vue`;
  - tratamento de 401 nas chamadas da administração e do dashboard.
- **Compatibilidade**:
  - quem usa `curl -u` ou `Authorization: Basic` continua funcionando;
  - quem abria o dashboard com a janela do navegador passa a ver a tela de login.
- **Versão anterior do fork**: ignora a tabela nova e recusa `session-ttl` no YAML por ser campo desconhecido; o campo precisa ser removido antes de voltar.
- **Documentação**: `docs/admin-endpoints.md` (seção de segurança e login), `docs/README.md` (nota do fork em `security.basic`) e pacotes de deploy (nginx: nada muda).
- **Testes**:
  - sessão nos 4 bancos e na memória;
  - middleware com sessão, com header, sem credencial, com e sem `Sec-Fetch-*`;
  - limite de tentativas e origem;
  - E2E com agent-browser (login, redirect, logout, tema claro e escuro, sem janela nativa).
