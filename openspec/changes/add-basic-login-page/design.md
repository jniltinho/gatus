## Context

Estado atual da autenticação:

- **`security.basic`**: `security.ApplySecurityMiddleware` usa o `basicauth` do Fiber no roteador protegido da API. A falha responde 401 com `WWW-Authenticate: Basic`, e o navegador abre a janela nativa. As rotas HTML da SPA (`/`, `/endpoints/:key`, `/admin/...`) não são protegidas: a janela aparece quando a SPA chama a API. A senha fica em `password-bcrypt-base64` e é conferida com bcrypt a cada requisição. Não há logout.
- **`security.oidc`**: tem tela de login no frontend (`App.vue`, "Login with OIDC") e sessões em `gocache` na memória (`security/sessions.go`), com cookie `gatus_session` (`SameSite=Strict`) e TTL de 8h.
- **`/api/v1/config`**: informa `oidc` e `authenticated`. Com basic, `authenticated` é sempre `false`, porque `IsAuthenticated` só conhece o OIDC. O objeto `admin.authorized` é `true` com basic.
- **Administração**: trata o único usuário basic como administrador. `RequestAuthor` lê o usuário gravado pelo `basicauth` em `Locals("username")`. As escritas passam por `adminRequestProtection` (`Sec-Fetch-Site`, `Origin`/`Referer` contra `admin.allowed-origins` ou a origem derivada de `Host` e do esquema).
- **Proteção reutilizável**: as status pages já têm `statuspage.NewLimiter` (limitador próprio, sem goroutine) e `statuspage.ClientIP` (com `trusted-proxies`).
- **Tema**: `Settings.vue` e `PublicLayout.vue` leem o cookie de tema (`dark`/`light`, com o padrão do sistema) e alternam a classe `dark`. O visual é quadrado (`rounded-*` zerado no Tailwind).
- **Restrições do fork**: ciclo de hot reload, tabelas novas nos três dialetos com placeholders `$N`, rotas públicas antes do middleware de segurança e código novo em arquivos novos.

## Goals / Non-Goals

**Goals:**
- Substituir a janela nativa por uma tela de login no padrão do fork, a 15% do topo, com tema claro e escuro.
- Sessões por cookie com logout, validade configurável, persistência no storage e invalidação ao trocar a credencial.
- Manter `Authorization: Basic` funcionando para `curl`, scripts e integrações.
- Proteger o login contra força bruta e CSRF.

**Non-Goals:**
- Vários usuários, cadastro de usuários ou perfis (continua um único usuário basic).
- Mover as sessões do OIDC para o banco ou mudar o fluxo OIDC.
- "Lembrar de mim", troca de senha pela web e 2FA.
- Proteger as rotas HTML da SPA no servidor (os dados continuam protegidos na API).

## Decisions

### D1. Sessões na tabela `login_sessions`, em memória com storage memory

**Tabela `login_sessions`**, criada nos três dialetos:
- `token_hash`: SHA-256 do token em hexadecimal, chave primária;
- `username`;
- `credential_fingerprint`: SHA-256 de `username` e do hash bcrypt configurado;
- `created_at` e `expires_at`, em milissegundos.

**Interface:** `store.LoginSessionStore`, com `CreateLoginSession`, `GetLoginSession`, `DeleteLoginSession` e `DeleteExpiredLoginSessions`. O store em memória implementa a mesma interface com um mapa protegido por mutex.

**Token:** 32 bytes de `crypto/rand` em base64 URL. Só o hash vai para o banco.

**Validação:** uma sessão vale quando existe, não expirou e tem o fingerprint da credencial atual. Trocar o usuário ou a senha no YAML invalida as sessões antigas na primeira requisição.

**Alternativas consideradas:**
- **`gocache` em memória, como o OIDC:** rejeitada, porque o login cairia a cada deploy, recarga ou reinício e não serviria para várias instâncias.
- **Cookie assinado sem estado (HMAC):** rejeitada, porque exigiria um segredo novo na configuração e não permitiria logout real nem revogação.
- **Guardar o token em texto:** rejeitada, porque um vazamento do banco daria sessões válidas.

### D2. Autenticador basic próprio no lugar do `basicauth`

O novo middleware, em arquivo novo em `security/`, segue esta ordem:
1. **Cookie `gatus_session`:** busca a sessão pelo hash do token. Se for válida, grava o usuário em `Locals("username")` e segue.
2. **`Authorization: Basic`:** confere usuário e senha com bcrypt, como hoje, e grava `Locals("username")`.
3. **Falha:** responde 401 com `{"error":"authentication required"}` e `Cache-Control: no-store`. O header `WWW-Authenticate: Basic` só entra quando a requisição não tem `Sec-Fetch-Site`, `Sec-Fetch-Mode` nem `X-Requested-With`. Assim o navegador não abre a janela, e ferramentas como `curl` continuam recebendo o desafio.

`IsAuthenticated` passa a reconhecer a sessão basic e o header válido. `RequestAuthor` continua lendo `Locals("username")`.

**Alternativas consideradas:**
- **Nunca enviar `WWW-Authenticate`:** rejeitada, porque algumas ferramentas só mandam credenciais depois do desafio.
- **Interceptar o 401 só no frontend:** rejeitada, porque o navegador abre a janela antes do JavaScript receber a resposta.
- **Proteger as rotas HTML no servidor e redirecionar para `/login`:** rejeitada neste momento, porque mudaria as rotas compartilhadas com o upstream e o ciclo de configuração do `App.vue`. A API continua sendo o único ponto de proteção.

### D3. API de login e logout

As rotas são registradas no roteador não protegido:

- **`POST /api/v1/auth/login`:**
  - recebe JSON `{"username","password"}`;
  - em caso de sucesso, responde 204 com o cookie `gatus_session` (`HttpOnly`, `Path=/`, `SameSite=Strict`, `Max-Age` igual ao TTL). O cookie é `Secure` quando a conexão é TLS ou quando `X-Forwarded-Proto` é `https`;
  - com credenciais erradas, responde 401 com a mensagem genérica "Invalid username or password";
  - responde 429 quando o limite estoura;
  - responde 404 quando não há `security.basic` ou quando há OIDC.
- **`POST /api/v1/auth/logout`:** remove a sessão do cookie, se existir, expira o cookie e responde 204.
- **Nas duas rotas:**
  - `Cache-Control: no-store`;
  - a mesma proteção de origem da administração (rejeita `Sec-Fetch-Site: cross-site` e `Origin`/`Referer` fora de `admin.allowed-origins` ou da origem derivada), com o corpo limitado a 4 KB e em JSON;
  - cada login cria uma sessão nova, sem reaproveitar a anterior;
  - a limpeza das sessões expiradas roda no login e a cada hora, em goroutine ligada ao ciclo de vida da configuração.

**Alternativas consideradas:**
- **Formulário HTML com `application/x-www-form-urlencoded`:** rejeitada, porque abre CSRF clássico e foge do padrão JSON da API do fork.
- **Reusar `/oidc/login`:** rejeitada, porque são fluxos e configurações distintos.

### D4. Limite de tentativas por IP

- **Limitador:** instância própria de `statuspage.NewLimiter`, com IP do cliente de `statuspage.ClientIP` e `status-pages.trusted-proxies`.
- **Contagem:** 10 falhas por minuto por IP (IPv6 por /64). Acima disso, o IP recebe 429 em qualquer tentativa até a janela reiniciar, inclusive com a senha certa, para não deixar adivinhar durante o bloqueio.
- **Tempo:** o bcrypt roda também para usuário inexistente, contra um hash fixo, para não vazar a existência do usuário pelo tempo de resposta.

**Alternativas consideradas:**
- **Bloqueio por usuário:** rejeitada, porque há um único usuário e um atacante poderia bloquear o administrador.
- **Captcha:** fora do escopo e com dependência externa.

### D5. Tela de login e integração no frontend

- **`views/Login.vue`**, rota `/login` com `meta.login` e rota HTML `/login` na SPA:
  - layout próprio sem o cabeçalho do dashboard;
  - botão de tema no canto, lendo e gravando o mesmo cookie de tema de `PublicLayout.vue`;
  - cartão quadrado `max-w-sm`, centralizado na horizontal com a margem superior de `15vh`, com logo, `ui.header`, usuário, senha, botão "Sign in" e erro genérico;
  - variantes `dark:` e foco no campo de usuário.
- **`/api/v1/config`** ganha `login` (`"basic"`, `"oidc"` ou vazio). `authenticated` passa a valer também para basic.
- **`App.vue`:** com `login === "basic"` e sem autenticação, uma rota não pública leva a `/login?redirect=<caminho atual>`. Com autenticação, a rota `/login` volta ao `redirect`, aceito só quando é um caminho interno (começa com `/` e não com `//`). Um 401 recebido pelas chamadas do dashboard ou da administração leva à mesma tela.
- **Cabeçalho:** ganha o botão "Logout" com `login === "basic"`.

**Alternativas consideradas:**
- **Mostrar o formulário no lugar do conteúdo, sem rota:** rejeitada, porque perde o `redirect` e o histórico do navegador.
- **Centralizar o cartão na vertical:** rejeitada por pedido do dono, que quer o cartão a 15% do topo.

### D6. Configuração

`security.basic.session-ttl` é uma duração opcional, com padrão de 8h, mínimo de 5 minutos e máximo de 30 dias. Um valor fora da faixa invalida a configuração.

**Alternativa considerada:** usar o `security.oidc.session-ttl`. Rejeitada, porque as duas autenticações são independentes.

## Risks / Trade-offs

- **[Custo do bcrypt nas requisições com header]** → É o mesmo de hoje. O navegador passa a usar a sessão, que custa uma leitura por hash no banco ou na memória.
- **[Leitura do banco a cada requisição autenticada]** → Consulta pela chave primária. Um cache em memória de 30 segundos por hash reduz as idas ao banco; o logout e a troca de credencial limpam o cache da instância.
- **[Várias instâncias]** → As sessões ficam no banco compartilhado. O cache de 30 segundos de outra instância pode aceitar uma sessão por até 30 segundos depois do logout. Isso fica documentado.
- **[Storage memory]** → As sessões se perdem ao reiniciar e é preciso logar de novo. Isso fica documentado.
- **[`X-Forwarded-Proto` usado para o atributo `Secure`]** → Só afeta o atributo do cookie, não a autorização. Atrás de proxy HTTPS, o proxy precisa enviar o header, como já pede a documentação da administração.
- **[Detecção de navegador por `Sec-Fetch-*`]** → Navegadores antigos sem esses headers ainda veriam a janela nativa. O frontend também envia `X-Requested-With: XMLHttpRequest` nas chamadas à API protegida.
- **[Rollback]** → Uma versão anterior ignora a tabela `login_sessions` e volta à janela nativa. `session-ttl` precisa ser removido do YAML, porque a versão anterior o recusa como campo desconhecido.

## Migration Plan

- A tabela `login_sessions` é criada automaticamente e de forma idempotente. Não há migração de dados.
- Sem `session-ttl`, o padrão é 8h. Nada muda para quem usa `Authorization: Basic`.
- **Rollback:** remover `security.basic.session-ttl` do YAML, se tiver sido usado, antes de voltar à versão anterior.

## Open Questions

- Nenhuma que bloqueie. O limite de 10 falhas por minuto pode virar configuração se o uso pedir.
