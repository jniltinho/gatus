## Context

Estado atual da autenticação:

- **`security.basic`:**
  - `security.ApplySecurityMiddleware` usa o `basicauth` do Fiber no roteador protegido da API. A falha responde 401 com `WWW-Authenticate: Basic`, e o navegador abre a janela nativa.
  - As rotas HTML da SPA (`/`, `/endpoints/:key`, `/suites/:key`, `/admin/...`) não são protegidas: a janela aparece quando a SPA chama a API.
  - A senha fica em `password-bcrypt-base64` e é conferida com bcrypt a cada requisição, sem limite de tentativas. O usuário é comparado com `!=`.
  - Não há logout.
- **`security.oidc`:** tem tela de login no `App.vue` e sessões em `gocache` na memória (`security/sessions.go`), com cookie `gatus_session` (`SameSite=Strict`) e TTL de 8h. Com OIDC configurado, o middleware ignora o basic.
- **`/api/v1/config`:** informa `oidc` e `authenticated`. Com basic, `authenticated` é sempre `false`, e `admin.authorized` é `true` mesmo sem credenciais. O `App.vue` mostra o dashboard quando `!config.oidc || config.authenticated`.
- **Administração:**
  - trata o único usuário basic como administrador, e `RequestAuthor` lê `Locals("username")`;
  - as escritas passam por `adminRequestProtection`, que:
    - recusa `Sec-Fetch-Site: cross-site`;
    - compara `Origin`/`Referer` com `admin.allowed-origins` ou com a origem derivada de `Host` e do esquema, ignorando `X-Forwarded-Host`;
    - aceita requisições sem `Origin` nem `Referer`;
    - aceita `http://localhost:8081` com `ENVIRONMENT=dev`;
    - exige JSON/YAML só quando há corpo.
- **Limitador das status pages:** `statuspage.Limiter` só tem `Hit`, que conta toda chamada permitida, com teto de chaves obrigatório. `statuspage.ClientIP` calcula o IP do cliente com `trusted-proxies`.
- **Fiber:** roda com `Immutable: true` (`api/api.go`). Valores de headers, cookies e corpo já são cópias seguras entre requisições.
- **Configuração:** o YAML é lido com `yaml.Unmarshal` sem `KnownFields`, então campos desconhecidos são ignorados.
- **Hot reload:** `main.go` para o controller, fecha o store (`store.Get().Close()`) e sobe tudo de novo. Goroutines sem cancelamento continuariam usando o store fechado.
- **Tema:** `Settings.vue` e `PublicLayout.vue` leem o cookie de tema e alternam a classe `dark`. O visual é quadrado.

## Goals / Non-Goals

**Goals:**
- Substituir a janela nativa por uma tela de login no padrão do fork, a 15% do topo, com tema claro e escuro.
- Sessões por cookie com logout imediato, validade configurável, persistência no storage e invalidação imediata ao trocar a credencial.
- Manter `Authorization: Basic` para `curl`, scripts e integrações, sob o mesmo limite de falhas do login.
- Proteger contra força bruta, CSRF, fixação de sessão e redirecionamento aberto.

**Non-Goals:**
- Vários usuários, cadastro de usuários ou perfis (continua um único usuário basic).
- Mover as sessões do OIDC para o banco ou mudar o fluxo OIDC.
- "Lembrar de mim", troca de senha pela web e 2FA.
- Proteger as rotas HTML da SPA no servidor (os dados continuam protegidos na API).
- Compartilhar o limitador entre instâncias.

## Decisions

### D1. Sessões na tabela `login_sessions`, em memória com storage memory

**Tabela:** `login_sessions`, criada nos três dialetos:

| Coluna | SQLite | PostgreSQL | MySQL/MariaDB |
|--------|--------|------------|---------------|
| `token_hash` (chave primária, SHA-256 em hexadecimal) | `TEXT` | `CHAR(64)` | `CHAR(64)` |
| `username` | `TEXT` | `TEXT` | `VARCHAR(255)` |
| `credential_fingerprint` (SHA-256 de usuário e hash bcrypt configurados) | `TEXT` | `CHAR(64)` | `CHAR(64)` |
| `created_at` e `expires_at`, em milissegundos | `INTEGER` | `BIGINT` | `BIGINT` |

A tabela tem um índice em `expires_at`. `mysql_schema_test.go` passa a esperar a tabela.

**Interface:** `store.LoginSessionStore`, com `CreateLoginSession`, `GetLoginSession`, `DeleteLoginSession` e `DeleteExpiredLoginSessions`. O store em memória implementa a mesma interface com um mapa protegido por mutex.

**Token:** 32 bytes de `crypto/rand` em base64 URL. Só o hash vai para o storage.

**Validação:** uma sessão vale quando existe, `expires_at` é futuro e o fingerprint é o da credencial atual. A consulta pela chave primária acontece em toda requisição protegida, sem cache. Assim logout e troca de credencial valem na hora, em todas as instâncias.

**Limpeza:** acontece quando é preciso, sem goroutine:
- o login remove as sessões expiradas;
- uma consulta que encontra sessão expirada ou com fingerprint antigo a remove.

Assim nada depende do ciclo de recarga, que fecha e reabre o store.

**Alternativas consideradas:**
- **`gocache` em memória, como o OIDC:** rejeitada, porque o login cairia a cada deploy, recarga ou reinício.
- **Cache de 30 segundos por hash:** rejeitada, porque atrasaria o logout e a troca de senha em outras instâncias.
- **Cookie assinado sem estado (HMAC):** rejeitada, porque exigiria um segredo novo e não permitiria logout real.
- **Goroutine de limpeza a cada hora:** rejeitada, porque precisaria de cancelamento no ciclo de recarga e usaria o store fechado.
- **Guardar o token em texto:** rejeitada, porque um vazamento do banco daria sessões válidas.

### D2. Autenticador basic próprio no lugar do `basicauth`

O novo middleware fica em arquivo novo em `security/` e é usado só com `security.basic` sem OIDC. Para cada requisição:

1. Se o cookie `gatus_session` traz uma sessão válida (D1), grava o usuário em `Locals("username")` e segue. Um cookie ausente, desconhecido, expirado ou antigo não encerra a verificação e não bloqueia o header.
2. Se há `Authorization: Basic`:
   - consulta o limitador (D4) e, com o IP bloqueado, responde 429 com `Retry-After`, sem rodar o bcrypt;
   - senão confere usuário e senha (D4, tempo constante);
   - com credenciais corretas, grava `Locals("username")` e segue;
   - com credenciais erradas, registra a falha no limitador.
3. Sem autenticação, responde 401 com `{"error":"authentication required"}` e `Cache-Control: no-store`. O header `WWW-Authenticate: Basic` só é enviado quando a requisição não tem `Sec-Fetch-Site`, `Sec-Fetch-Mode` nem `X-Requested-With`. Esse é o único critério, também usado nos testes, que precisam enviar esses headers para simular o navegador.

`IsAuthenticated` passa a reconhecer a sessão basic válida e o header correto, sem registrar falha no limitador. `RequestAuthor` continua lendo `Locals("username")`. O token do OIDC nunca é validado pelo autenticador basic, porque as sessões ficam em stores diferentes.

**Alternativas consideradas:**
- **Nunca enviar `WWW-Authenticate`:** rejeitada, porque algumas ferramentas só mandam credenciais depois do desafio.
- **Interceptar o 401 só no frontend:** rejeitada, porque o navegador abre a janela antes do JavaScript receber a resposta.
- **Deixar o header fora do limitador:** rejeitada, porque ele viraria o caminho óbvio de força bruta e de sobrecarga por bcrypt.

### D3. API de login e logout

As rotas ficam no roteador não protegido e só existem com `security.basic` sem OIDC. Nos outros casos, respondem 404.

**`POST /api/v1/auth/login`:**
1. Recebe JSON `{"username","password"}`: exige `Content-Type` JSON, com corpo de até 4 KB.
2. Consulta o limitador antes do bcrypt e, com o IP bloqueado, responde 429 com `Retry-After`.
3. Confere as credenciais em tempo constante (D4).
4. Com sucesso:
   - remove as sessões expiradas;
   - cria uma sessão nova, ignorando qualquer `gatus_session` recebido, para não fixar sessão;
   - responde 204 com o cookie `gatus_session`: `HttpOnly`, `Path=/`, `SameSite=Strict`, `Max-Age` igual ao TTL e `Secure` quando a conexão é TLS ou `X-Forwarded-Proto` é `https`.
5. Com falha, registra no limitador e responde 401 com "Invalid username or password".

**`POST /api/v1/auth/logout`:** aceita corpo vazio sem `Content-Type`. Remove a sessão do cookie, se existir, e expira o cookie com os mesmos `Path`, `SameSite`, `HttpOnly` e `Secure`. Responde 204.

**Nas duas rotas:**
- `Cache-Control: no-store`;
- a mesma regra de origem da administração, com as funções de `api/admin_middleware.go`:
  - recusa `Sec-Fetch-Site: cross-site`;
  - compara `Origin`/`Referer`, quando presente, com `admin.allowed-origins` ou com a origem derivada de `Host` e do esquema, ignorando `X-Forwarded-Host`;
  - aceita requisições sem `Origin` e sem `Referer`;
  - aceita `http://localhost:8081` com `ENVIRONMENT=dev`.
- logs com a operação, o resultado e o IP, sem senha nem token.

**Alternativas consideradas:**
- **Formulário HTML com `application/x-www-form-urlencoded`:** rejeitada, porque abre CSRF clássico.
- **Reusar `/oidc/login`:** rejeitada, porque são fluxos e configurações distintos.

### D4. Limitador de falhas e comparação em tempo constante

**Limitador:**
- **Onde fica:** próprio do pacote `security`, no modelo do `statuspage.Limiter` e sem goroutine. Janela de 1 minuto, teto fixo de 10.000 chaves e IPv6 agrupado por /64.
- **Métodos:**
  - `Blocked(ip, now)` só consulta e devolve o `Retry-After`;
  - `Failure(ip, now)` conta uma falha.
- **Uso:** o IP vem de `statuspage.ClientIP` com `status-pages.trusted-proxies`. Depois de 10 falhas no mesmo minuto, o IP fica bloqueado até a janela reiniciar. O bloqueio vale para o login e para o header, inclusive com a senha certa.
- **Alcance:** o limitador é por processo. Instâncias diferentes não compartilham a contagem, e uma recarga a zera. Isso fica documentado.

**Comparação:** usuário e senha são sempre conferidos juntos:
- `subtle.ConstantTimeCompare` nos hashes SHA-256 do usuário recebido e do configurado;
- `bcrypt.CompareHashAndPassword` sempre com o hash configurado.

O resultado só é decidido depois das duas comparações, sem retornar cedo por usuário errado. Os testes verificam isso chamando o comparador, sem medir tempo.

**Alternativas consideradas:**
- **Reusar `statuspage.Limiter` com `Hit`:** rejeitada, porque ele conta toda chamada permitida, e o login com sucesso não deve contar.
- **Bloqueio por usuário:** rejeitada, porque há um único usuário e um atacante poderia bloquear o administrador de qualquer IP.
- **Captcha:** fora do escopo e com dependência externa.

### D5. Tela de login e integração no frontend

**Tela:** `views/Login.vue`, na rota `/login` com `meta.login`. A rota HTML `/login` só é registrada com `security.basic` sem OIDC.
- **Layout:** o `App.vue` trata `meta.login` como `meta.public`, sem o cabeçalho do dashboard e sem o link Admin. Um botão de tema no canto lê e grava o mesmo cookie de tema de `PublicLayout.vue`.
- **Cartão:** quadrado `max-w-sm`, centralizado na horizontal com a margem superior de `15vh`, com logo, `ui.header`, usuário, senha, botão "Sign in" e erro genérico ou mensagem de 429. Tem variantes `dark:` e foco no campo de usuário.

**`/api/v1/config`:**
- ganha `login`: `"basic"`, `"oidc"` ou vazio;
- `authenticated` passa a valer também para basic, pela sessão ou pelo header;
- o campo `oidc` continua existindo.

**`App.vue`:**
- **Redirecionamento:** com `login === "basic"` e sem autenticação, uma rota que não é pública nem de login leva a `/login?redirect=<caminho atual>`.
- **Rota `/login`:** já autenticado, volta ao `redirect` validado; com `login !== "basic"`, volta a `/`.
- **Cabeçalho:** com `login === "basic"` e autenticação, ganha o botão "Logout" e mostra o link Admin como hoje.

**Validação do `redirect`:** decodifica o valor e o aceita somente quando:
- começa com um único `/`;
- não contém `//`, `\`, `:` antes do primeiro `/`, nem caracteres de controle;
- não aponta para `/login`.

Nos outros casos, o destino é `/`.

**Chamadas protegidas:** `Home.vue`, `EndpointDetails.vue`, `SuiteDetails.vue` e `utils/adminApi.js` enviam `X-Requested-With: XMLHttpRequest` e levam a `/login` ao receber 401.

**Alternativas consideradas:**
- **Mostrar o formulário no lugar do conteúdo, sem rota:** rejeitada, porque perde o `redirect` e o histórico do navegador.
- **Centralizar o cartão na vertical:** rejeitada por pedido do dono, que quer o cartão a 15% do topo.
- **Validar o `redirect` só pelo prefixo `/`:** rejeitada, porque `/%2F%2Fhost` e `/\host` passariam.

### D6. Configuração

`security.basic.session-ttl` é uma duração opcional, com padrão de 8h, mínimo de 5 minutos e máximo de 30 dias. Um valor fora da faixa invalida a configuração.

**Alternativa considerada:** usar o `security.oidc.session-ttl`. Rejeitada, porque as duas autenticações são independentes.

## Risks / Trade-offs

- **[Leitura do storage a cada requisição autenticada]** → É uma consulta pela chave primária, mais barata que o bcrypt de hoje. É o preço do logout e da troca de senha imediatos em todas as instâncias.
- **[Custo do bcrypt nas requisições com header]** → É o mesmo de hoje, agora com limite de falhas por IP. Scripts com credenciais corretas não são afetados, a menos que dividam o IP com um atacante bloqueado.
- **[Limitador por processo]** → Com várias instâncias, cada uma conta as falhas separadamente, e uma recarga zera a contagem. Isso fica documentado. Um proxy à frente pode limitar globalmente.
- **[Storage memory]** → As sessões se perdem ao reiniciar e é preciso logar de novo. Isso fica documentado.
- **[`X-Forwarded-Proto` usado para o atributo `Secure`]** → Só afeta o atributo do cookie. Atrás de proxy HTTPS sem esse header, o cookie sai sem `Secure`. A documentação pede o header, como já pede para a administração.
- **[Detecção de navegador por `Sec-Fetch-*`]** → Navegadores antigos sem esses headers ainda veriam a janela nativa. O frontend também envia `X-Requested-With`.
- **[Mesmo nome de cookie do OIDC]** → Os dois fluxos nunca convivem, porque com OIDC o basic não é usado. Os tokens ficam em stores diferentes.
- **[Rollback]** → Uma versão anterior ignora a tabela `login_sessions` e o campo `session-ttl` (YAML sem `KnownFields`) e volta à janela nativa, sem erro de configuração.

## Migration Plan

- A tabela `login_sessions` é criada automaticamente e de forma idempotente. Não há migração de dados.
- Sem `session-ttl`, o padrão é 8h. Nada muda para quem usa `Authorization: Basic` corretamente.
- **Rollback:** voltar ao binário anterior. `session-ttl` pode ficar no YAML, e a tabela fica sem uso.

## Open Questions

- Nenhuma que bloqueie. O limite de 10 falhas por minuto pode virar configuração se o uso pedir.
