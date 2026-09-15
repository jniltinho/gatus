## 1. Backend: sessões, autenticador e API

- [ ] 1.1 `security.basic.session-ttl` (duração, padrão 8h, mínimo 5 minutos e máximo 30 dias) na validação da configuração, com testes de faixa.
- [ ] 1.2 `store.LoginSessionStore`:
  - tabela `login_sessions` (hash SHA-256 do token, usuário, fingerprint da credencial, criação e expiração) nos três dialetos, com `mysql_schema_test.go`;
  - store em memória;
  - operações de criar, buscar, remover e remover expiradas;
  - testes nos 4 bancos e na memória.
- [ ] 1.3 Autenticador basic em arquivo novo de `security/`, no lugar do `basicauth`:
  - sessão por cookie `gatus_session` e fallback para `Authorization: Basic`;
  - `Locals("username")`;
  - 401 com `Cache-Control: no-store`, e `WWW-Authenticate: Basic` só sem `Sec-Fetch-Site`, `Sec-Fetch-Mode` e `X-Requested-With`;
  - cache de 30 segundos por hash;
  - `IsAuthenticated` reconhecendo basic;
  - testes com sessão, header, sem credencial, com e sem `Sec-Fetch-*`, sessão expirada e credencial trocada.
- [ ] 1.4 Rotas `POST /api/v1/auth/login` e `POST /api/v1/auth/logout` no roteador não protegido:
  - token de 32 bytes, cookie `HttpOnly`/`SameSite=Strict`/`Secure` com HTTPS e `Cache-Control: no-store`;
  - proteção de origem da administração, corpo JSON de até 4 KB e 404 sem basic ou com OIDC;
  - limpeza das sessões expiradas no login e a cada hora, ligada ao ciclo de vida;
  - logs sem senha nem token;
  - testes de API.
- [ ] 1.5 Limite de 10 falhas por minuto por IP com `statuspage.NewLimiter` e `ClientIP` (429 inclusive com a senha certa durante o bloqueio) e bcrypt contra hash fixo para usuário inexistente, com testes.
- [ ] 1.6 `/api/v1/config` com `login` (`basic`, `oidc` ou vazio) e `authenticated` para basic; rota HTML `/login` na SPA. Testes de API, incluindo a auditoria da administração com o usuário da sessão.

## 2. Frontend: tela de login, redirecionamento e logout

- [ ] 2.1 `views/Login.vue` com layout próprio:
  - cartão quadrado `max-w-sm` centralizado na horizontal a `15vh` do topo, com logo, `ui.header`, usuário, senha e "Sign in";
  - erro genérico e mensagem para 429;
  - botão de tema com o mesmo cookie de `PublicLayout.vue`, e variantes `dark:`;
  - rota `/login` com `meta.login`.
- [ ] 2.2 `App.vue`:
  - com `login === "basic"` e sem autenticação, levar as rotas não públicas a `/login?redirect=<caminho>`;
  - na rota `/login` já autenticado, voltar ao `redirect` interno (começa com `/` e não com `//`);
  - botão "Logout" no cabeçalho;
  - OIDC sem mudança.
- [ ] 2.3 `X-Requested-With: XMLHttpRequest` e tratamento de 401 com ida a `/login` nas chamadas do dashboard (`fetch` com `credentials`) e em `utils/adminApi.js`.
- [ ] 2.4 Lint e `make frontend-build`.

## 3. Documentação, E2E e entrega

- [ ] 3.1 Documentação:
  - `docs/admin-endpoints.md`: tela de login, sessões, `session-ttl`, logout, `curl -u` e várias instâncias;
  - `docs/README.md`: nota do fork em `security.basic`;
  - `README.md` e `AGENTS.fork.md`: autenticador, tabela e rotas.
- [ ] 3.2 E2E com agent-browser (`test/e2e/login.sh`, e ajuste de `admin.sh`, `push.sh`, `status-pages.sh` e `certificate.sh` para logar pela tela no lugar de `set credentials`):
  - sem janela nativa;
  - redirect para `/login` e de volta;
  - senha errada, logout e status page pública sem login;
  - prints claro e escuro em `dist/prints/`.
- [ ] 3.3 `go test ./... -race` com PostgreSQL, MySQL e MariaDB, `make lint` e `openspec validate add-basic-login-page --strict`.
- [ ] 3.4 PR no `jniltinho/gatus` com CI verde e merge, release com imagem no Docker Hub, pacote `mariadb` e arquivamento da change.
