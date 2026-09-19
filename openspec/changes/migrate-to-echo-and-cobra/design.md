## Context

Medido no código, em `master`:

- **47 arquivos** importam o Fiber: 24 em `api/`, 4 em `security/`, 1 em `controller/` e 18 de teste. São **118** usos de `*fiber.Ctx`, **28** de `fiber.Handler`, **38** de `c.Params`, **34** de `c.Query*`, **20** de `fiber.Map`, **10** de `c.Locals` e **10** de `c.Next()`.
- **Testes:** 56 requisições passam por `app.Test(request, -1)`, que é do Fiber.
- **Middlewares do Fiber em uso:** `recover`, `compress` (com `Next` pulando o canal de eventos), `cors` (só com `ENVIRONMENT=dev`), `redirect` (`/index.html` → `/`), `filesystem` (os arquivos estáticos embutidos, com `Browse: true`) e `adaptor` (5 usos).
- **Presos ao `fasthttp`:** `SetBodyStreamWriter` (3 usos, `api/live_updates.go`), `c.Context().RemoteIP()` (8), `server.HeaderReceived` com `fasthttp.RequestConfig` (`controller/controller.go`), `fiber.Config{ReadBufferSize, Network, Immutable}` e a opção pública `web.read-buffer-size`.
- **APIs de mesmo nome e significado diferente**, conferidas no código do Echo v5.3.1: `c.Path()` no Fiber é o caminho **da requisição**, e no Echo é o padrão **registrado** (`/status/:slug`) — 3 usos nossos, entre eles `statusPageHTMLAuth` (`api/status_page_auth.go:140`), o skipper da compressão (`api/api.go:70`) e o limite do restore (`api/admin_middleware.go:45`). `c.Get(...)` no Fiber lê um **cabeçalho** (11 usos: `Origin`, `Sec-Fetch-Site`, `Authorization`, `If-Match`, `Last-Event-ID`...), e no Echo lê o **store** da requisição. `c.Body()` (12 usos) não existe no Echo: o corpo é um `io.ReadCloser` que só se lê uma vez, e `adminRequestProtection` o lê **antes** do manipulador.
- `X-Forwarded-For` é lido com `PeekAll` (5 usos), que devolve **todas** as linhas do cabeçalho; `Header.Get` do `net/http` só devolve a primeira.
- O esquema da requisição vem de `c.Context().IsTLS()` com o `Host` cru (2 usos), de propósito: `Hostname()` e `Protocol()` do Fiber confiam em `X-Forwarded-*`. O `c.Scheme()` do Echo confia também.
- O laço de recarga está em `main_reload.go`, ao lado de `main.go`, e cada ciclo chama `loadConfiguration` de novo.
- **O Fiber deixa mudar a resposta depois do manipulador; o `net/http` não.** `statusPageBadgeHandler` (`api/status_page_auth.go:125-131`) chama o gerador do badge e **só depois** grava `Cache-Control: private, no-store` e `Vary: Authorization`; `api/spa.go:24` troca a resposta por um erro depois de começar a escrever o template. No `net/http`, cabeçalho gravado depois do primeiro byte não chega ao cliente.
- `controller.Handle` trata **qualquer** erro do `Listen` como fatal (`controller/controller.go:40,45`). O Fiber volta sem erro num desligamento; `http.Server` volta `http.ErrServerClosed`.
- O login do OIDC (`security/oidc.go:59`) recebe `*fiber.Ctx`, grava dois cookies e redireciona; só o **callback** já é `net/http`.
- As chaves vêm do caminho com `url.QueryUnescape` nos endpoints e com `url.PathUnescape` no push, que tratam `+` de forma diferente.
- `config.LoadConfiguration` cai para os caminhos padrão quando o caminho pedido não existe.
- O CI tem um passo que cita pacotes por caminho: `go test ./storage/... ./statuspage/... ./managedendpoint/... ./config -race` (`.github/workflows/ci.yml:124`).
- As suítes E2E são **seis**: `admin`, `status-pages`, `push`, `certificate`, `login` e `admin-backup`.
- **A ordem de registro é semântica no Fiber.** `api/api.go` diz: *"ORDER IS IMPORTANT: all routes applied AFTER the security middleware will require authn"*. As rotas públicas, o SPA e o `filesystem` de captura são registrados **antes** do middleware de segurança; as protegidas, depois.
- **Rotas de captura com 404 idêntico:** `unprotectedAPIRouter.All("/v1/status-pages/*")` e as do push existem para que um caminho desconhecido **não** chegue ao middleware de segurança (que responderia 401 e abriria a caixa do navegador).
- **`app.Get` do Fiber registra `HEAD` junto.** A spec do login da página exige o desafio no `HEAD` do HTML, e o canal de eventos responde `HEAD` com os cabeçalhos.
- **Limite de corpo:** o Fiber recusa corpos acima de 4 MiB por padrão. `adminRequestProtection` aplica 256 KB, e o restore aceita 3,5 MiB justamente por ficar abaixo dos 4 MiB do Fiber (`AGENTS.fork.md`).
- **Linha de comando:** não existe. `main.go` lê `GATUS_CONFIG_PATH`, `GATUS_CONFIG_FILE` (obsoleta), `GATUS_LOG_LEVEL` e `GATUS_DELAY_START_SECONDS`. O `Dockerfile` usa `ENTRYPOINT ["/gatus"]` numa imagem sem shell.
- **Upstream:** o `AGENTS.fork.md` tem um roteiro de sincronização com `TwiN/gatus`, que segue no Fiber.
- **Padrão de referência do dono:** `jniltinho/go-ispconfig`, que já usa Echo v5.3.1 e Cobra 1.10. Nele o `main.go` fica na raiz e só chama `cmd.Execute(...)`; o pacote `cmd/` tem um arquivo por comando com o teste ao lado (`serve.go`, `serve_test.go`, `version.go`...); `Version`, `BuildDate` e `GitCommit` são variáveis do pacote `cmd` preenchidas por `-ldflags "-X <módulo>/cmd.Version=..."` no `Makefile`; a configuração é carregada num `PersistentPreRunE` que **pula** os comandos que não dependem dela (`version`, `help`, `completion`); o `rootCmd` usa `SilenceErrors: true`; o servidor é um `http.Server` comum com o Echo como `Handler`, `signal.NotifyContext` e desligamento com prazo; e todo o resto do código vive em `internal/`.

## Goals / Non-Goals

**Goals:**

- Trocar o servidor sem que nenhum cliente perceba: mesmas rotas, corpos, cabeçalhos e códigos.
- Tirar o `adaptor` e o `fasthttp`, e simplificar o streaming.
- Dar ao binário uma linha de comando de verdade, sem quebrar `gatus` sem argumentos nem as variáveis de ambiente.
- Organizar o projeto no layout do Cobra — `main.go` na raiz e os comandos em `cmd/` — e levar os pacotes do fork para `internal/`, sem tornar a sincronização com o upstream inviável.

**Non-Goals:**

- Mudar qualquer URL, payload ou opção do `config.yaml`.
- Reescrever `manager-gatus.py` como comando do binário: ele fala com uma instalação remota pela API, e o binário opera a instalação local. Fica para outra change.
- Mover os pacotes que o upstream tem (`api`, `config`, `client`, `storage`, `watchdog`, `alerting`, `security`...) para `internal/`.
- Trocar o `logr` por `slog` no resto do código: o Echo v5 usa `slog`, mas só ele.
- Adotar o Viper: o carregador de YAML do Gatus, com recarga a quente, continua sendo a única fonte da configuração.
- HTTP/2 e HTTP/3 como entrega: passam a ser possíveis, e só.

## Decisions

### D1 — Três marcos, na ordem do risco

| Marco | Entrega | Por que nessa ordem |
|-------|---------|---------------------|
| 1 | Cobra, com `main.go` na raiz e os comandos em `cmd/` | Não toca o servidor. Dá o `gatus config validate` e o `healthcheck`, que ajudam a validar o marco 2. |
| 2 | Echo v5 | O de maior risco, isolado, com a suíte de contrato de D3 escrita **antes** da troca. |
| 3 | Pacotes do fork em `internal/` | Só renomeia caminhos de importação; fica por último para não misturar diff mecânico com diff de comportamento. |

Cada marco é um PR com CI verde e uma release própria. O marco 2 **não** é fatiado por pacote: `fiber.Ctx` e `echo.Context` não convivem no mesmo roteador, então `api/`, `security/` e `controller/` trocam juntos.

### D2 — Linha de comando

```
gatus                       # igual a `gatus serve` — compatibilidade com o ENTRYPOINT e com quem roda sem argumentos
gatus serve   [--config PATH] [--log-level LEVEL]
gatus version               # versão, commit e data, preenchidos por -ldflags
gatus config validate [--config PATH]   # carrega e valida, sem abrir o storage e sem subir o servidor; sai com 1 se inválido
gatus password hash         # lê a senha da entrada padrão e imprime o bcrypt em base64 — substitui docs/generate-admin-password.py
gatus healthcheck [--url URL]           # GET /health e sai com 0 ou 1; serve de HEALTHCHECK na imagem sem shell
```

- **Precedência:** flag > variável de ambiente > padrão, decidida por `cmd.Flags().Changed(...)` — um padrão vazio na flag sombrearia o ambiente. `GATUS_CONFIG_PATH`, `GATUS_CONFIG_FILE`, `GATUS_LOG_LEVEL` e `GATUS_DELAY_START_SECONDS` continuam valendo exatamente como hoje. As flags são do `serve` **e** da raiz, para `gatus --config x.yaml` funcionar como `gatus serve --config x.yaml`; `gatus --help` MUST NOT subir o servidor.
- **`password hash` nunca recebe a senha por argumento** (`cobra.NoArgs`), senão ela fica no histórico do shell e na lista de processos. Num terminal, pede a senha duas vezes sem eco, como o script Python; por pipe, lê uma linha e descarta o `\n` ou `\r\n` final — sem isso `printf 'x\n' | gatus password hash` geraria o hash de `x\n`, que não bate com o login.
- **`config validate` não abre o storage** nem toca a rede: só o que `config.LoadConfiguration` já valida. O único acesso a disco além do YAML é a leitura do par TLS de `web.tls`, que já faz parte da validação.
- **`healthcheck` lê o endereço da configuração** (`web.address`, `web.port` e se há `web.tls`), e não um `127.0.0.1:8080` fixo: a imagem é `FROM scratch`, `ENV PORT` não é lido pelo Gatus, e uma porta ou um TLS configurados quebrariam um `HEALTHCHECK` com URL fixa. Com `web.tls`, ele fala HTTPS com o laço local sem verificar o certificado, que pode ser autoassinado; `--url` troca o alvo.
- **`main_reload.go` vai junto** com o laço de vida para `cmd/`, com os dois arquivos de teste de `main`. O caminho resolvido da configuração (flag ou ambiente) é guardado e **reusado em toda recarga**: resolvê-lo só na primeira carga faria a recarga voltar ao ambiente e carregar outro arquivo. A validação antes de parar, `SkipInvalidConfigUpdate`, a serialização do `lifecycle` e a ordem de fechamento não mudam.
- **`--config` explícito que não existe é erro**, no `serve` e no `config validate`. `config.LoadConfiguration` cai para os caminhos padrão quando o pedido não existe, e o `config validate` aprovaria **outro** arquivo. Pelo ambiente, o comportamento de hoje fica.
- **`healthcheck` não carrega mais do que precisa:** com `--url` não lê a configuração; sem ele, lê só `web`. Não aplica `GATUS_DELAY_START_SECONDS`, não abre storage nem OIDC, troca o endereço curinga (`0.0.0.0`, `::`) pelo laço local, não segue redirecionamento, não usa proxy do ambiente e tem prazo total de 5 s.
- **Estrutura: `main.go` fica na raiz do projeto**, como decisão do dono e como no `go-ispconfig`: `main.go` só chama `cmd.Execute()`, e o pacote `cmd/` tem um arquivo por comando, com o teste ao lado (`root.go`, `serve.go`, `version.go`, `config.go`, `password.go`, `healthcheck.go` e os `*_test.go`). O laço de vida de hoje (início, recarga, sinais) sai de `main.go` para `cmd/serve.go` sem mudar de comportamento.
- **Do mesmo padrão:** `cmd.Version`, `cmd.BuildDate` e `cmd.GitCommit` preenchidas por `-ldflags -X gatus/v5/cmd.Version=...`; `SilenceErrors: true` no `rootCmd`, com o erro impresso uma vez em `Execute`; e um `PersistentPreRunE` que só prepara o que o comando precisa — `version`, `password hash`, `help` e `completion` MUST funcionar com um `config.yaml` quebrado ou ausente.
- **O que não vem do padrão:** o Viper. O `go-ispconfig` lê TOML por ele; o Gatus tem o próprio carregador de YAML, com recarga a quente e validação, e a precedência flag > ambiente cabe em poucas linhas. O Viper entra em Non-Goals.
- Com o `main.go` na raiz, `go build .` continua sendo o alvo: `Makefile`, `Dockerfile`, `Dockerfile.release` e o workflow de release **não mudam de caminho**, só ganham os `-ldflags` da versão. É também o que mantém `main.go` no mesmo lugar que o upstream, e o conflito de sincronização nesse arquivo continua sendo de conteúdo, não de caminho.

### D3 — A suíte de contrato vem antes da troca

O risco do marco 2 é mudar um corpo, um cabeçalho ou um código sem perceber. Antes de tocar o Fiber, o marco 2 começa escrevendo um teste de contrato que roda **contra o roteador atual**: uma tabela de requisições (método, caminho, cabeçalhos) com o status, os cabeçalhos relevantes e o corpo esperados, cobrindo cada rota registrada em `api/api.go` e nos `register*Routes`, mais os casos de borda abaixo. A suíte passa no Fiber, o roteador é trocado, e ela tem de passar igual no Echo.

Além de cada rota, a suíte MUST cobrir o que esta troca tem mais chance de quebrar:

- **corpo:** `POST` válido de 1 KB na administração **depois** do middleware que já leu o corpo; restore de 3 MiB aceito; 413 em cada limite (4 MiB global, 3,5 MiB no restore, 256 KB na administração, 4 KB no login);
- **CSRF e sessão:** `POST` da administração e do login com `Origin` de outro site → 403; `Sec-Fetch-Site: cross-site` → 403; `If-Match` lido; `Set-Cookie` do login com `Path=/`, `HttpOnly`, `SameSite=Strict` e `Secure` só com TLS;
- **login das páginas:** `GET` e `HEAD` de `/status/<slug>` e de `/status/<slug>/endpoints/<chave>` com `auth` e sem credencial → 401 com `WWW-Authenticate`; o 404 de chave vindo **depois** do desafio;
- **eventos:** canal vivo depois de 20 s; resposta **sem** `Content-Encoding` com `Accept-Encoding: gzip`; vaga liberada quando o cliente aborta; `HEAD` sem abrir stream nem ocupar vaga; requisição de `EventSource` sem credencial **sem** `WWW-Authenticate`;
- **proxy:** `X-Forwarded-For` em **várias linhas** do cabeçalho, de origem confiável e não confiável;
- **roteamento:** `/status/infra/` igual a `/status/infra` e sem `Location`; `/index.html` → 301 para `/`; `POST /health` e o 405; caminho desconhecido sob `/api/` com e sem `security`;
- **estáticos:** fonte `woff2` com o tipo certo e sem `Cache-Control`, como hoje; arquivo ausente, `/index.html` com query, `HEAD` e `Range`;
- **métodos:** `GET`, `HEAD`, `POST` e `OPTIONS` em caminho existente e inexistente, com `security` desligado, básico e OIDC, e com as status pages desabilitadas, comparando status, `Allow`, desafio e corpo;
- **parâmetros:** chave com `%2F`, `%252F`, `%25`, `%2B`, `+`, Unicode e escape inválido, nos endpoints e no push; query repetida, vazia e malformada em `status`, `msg`, `ping` e `lastEventId`;
- **JSON e concorrência otimista:** os mesmos bytes no corpo (sem quebra de linha a mais), `ETag` de versão, e `If-Match` ausente, inválido, fraco e desatualizado;
- **cabeçalhos finais:** badge de página com login com `Cache-Control: private, no-store` e `Vary: Authorization` **no que o cliente recebe**; `Vary` junto de `Accept-Encoding`; `Content-Disposition` do backup;
- **OIDC:** o login gravando os dois cookies com os atributos de hoje e o 302; o callback; e o logout do login básico expirando o cookie.

**Dois níveis de teste.** O contrato roda em memória (`httptest.NewRecorder`), mas o `ResponseRecorder` não tem prazos reais nem conexão para cair. Eventos, desligamento e recarga rodam contra um **servidor TCP de verdade** (`httptest.NewServer` ou o `http.Server` do `controller`): canal vivo além de 15 s, primeiro `Flush`, cliente que desconecta, cliente lento, pânico no manipulador, recarga com canal aberto e os limites de 500 e 10. As seis suítes E2E fecham a prova contra o binário.

Os testes existentes trocam `app.Test(request, -1)` por `httptest.NewRecorder()` + `router.ServeHTTP`, atrás de um único auxiliar de teste, para o diff dos 18 arquivos ser mecânico.

### D4 — As diferenças de comportamento que precisam de decisão

| Tema | Fiber hoje | Echo v5 | Decisão |
|------|-----------|---------|---------|
| Ordem de registro | Middleware vale para o que vem **depois** | Middleware é do grupo, a árvore de rotas não tem ordem | Dois grupos explícitos sob `/api`: público e protegido. O comentário "ORDER IS IMPORTANT" deixa de existir. |
| Rotas de captura | Casam pela ordem de registro | Precedência estático > parâmetro > `*` | As capturas de `/v1/status-pages/*` e do push continuam no grupo público; a suíte de contrato prova que um caminho desconhecido dá o 404 idêntico **sem** `WWW-Authenticate`. |
| `HEAD` | `Get` registra `HEAD` junto | Não registra; `RouterConfig.AutoHandleHEAD` vem desligado | Auxiliar `getAndHead` para as rotas que hoje respondem `HEAD`. `AutoHandleHEAD` MUST NOT ser ligado: ele executaria o manipulador do canal de eventos, que reservaria vaga e abriria stream num `HEAD`. |
| `c.Path()` | Caminho da requisição | Padrão registrado (`/status/:slug`) | `c.Request().URL.Path` nos três usos. Sem isto `statusPageHTMLAuth` procuraria a página `:slug`, não acharia, e **serviria o HTML de uma página com login sem desafio**. |
| `c.Get` / `c.Set` | Cabeçalho da requisição / da resposta | Store da requisição | Cabeçalhos por `c.Request().Header.Get` e `c.Response().Header().Set`; `c.Get`/`c.Set` **só** para o que hoje é `c.Locals`. Trocar às cegas faria `c.Get("Origin")` devolver `nil` e o CSRF da administração deixar passar tudo. |
| Corpo da requisição | `c.Body()` em cache, lido quantas vezes for, e recusado **antes** do manipulador | `io.ReadCloser`, lido uma vez; o `BodyLimit` só percebe o excesso de um corpo sem `Content-Length` **durante a leitura** | Um middleware global lê o corpo adiantado, até 4 MiB mais um byte, responde 413 acima disso **antes de qualquer efeito do manipulador** e recoloca os mesmos bytes (`io.NopCloser(bytes.NewReader(corpo))`). Resolve os dois problemas de uma vez: o push, que nem lê o corpo, deixaria passar um corpo *chunked* gigante, e a administração leria o corpo duas vezes. Os limites menores (256 KB, 3,5 MiB e os 4 KB do login) continuam nas rotas, sobre os bytes já lidos, e o excesso MUST ser 413, nunca um 400 de parsing. |
| Resposta depois do manipulador | Cabeçalho e corpo podem mudar até o fim | Cabeçalho gravado depois do primeiro byte **se perde** | Nenhum código grava cabeçalho depois de escrever. `statusPageBadgeHandler` passa a definir `Cache-Control` e `Vary` **antes** de chamar o gerador, e os geradores de badge deixam de sobrescrever um `Cache-Control` já definido. `api/spa.go` renderiza o template num buffer, para poder trocar a resposta por um erro. O `HTTPErrorHandler` MUST respeitar resposta já iniciada — o canal de eventos sobretudo. |
| JSON | `json.Marshal` | `c.JSON` usa `Encoder.Encode`, que acrescenta `\n` | Serializador próprio sem a quebra de linha (ou `c.JSONBlob` com `json.Marshal`), para o corpo ter os mesmos bytes. Sem `ETag` automático: o de versão da administração é o único. |
| Parâmetros do caminho | Crus; o código decodifica uma vez (`QueryUnescape` nos endpoints, `PathUnescape` no push) | O roteador pode decodificar, conforme a configuração | O roteador fica **sem** decodificar parâmetros (`EnablePathUnescaping` desligado), e cada manipulador mantém a decodificação que já faz. A suíte fixa o número de decodificações. |
| Esquema e host | `c.Context().IsTLS()` + `Host` cru | `c.Scheme()` confia em `X-Forwarded-Proto` | `c.IsTLS()` e `c.Request().Host`. `c.Scheme()` MUST NOT ser usado no CSRF nem no `Secure` do cookie. |
| Cabeçalho em várias linhas | `PeekAll` devolve todas | `Header.Get` devolve a primeira | `Header.Values("X-Forwarded-For")` nos 5 pontos. |
| Método não permitido | 405 só sem `HEAD` implícito | 405 do roteador | Coberto pela suíte; `getAndHead` evita que `HEAD /health` vire 405. |
| Caminho desconhecido sob `/api/` com `security` | Cai no middleware de segurança: 401 sem credencial | 404 do roteador | Captura `/*` no grupo protegido, para continuar 401 sem credencial e 404 com. |
| Maiúsculas no caminho | Não distingue | Distingue | **Passa a distinguir.** É o comportamento do `net/http` e de todo proxy na frente; ninguém documenta `/API/V1`. Entra nas notas como mudança. |
| Barra final | Ignora | Não ignora | Mantida por `e.Pre(middleware.RemoveTrailingSlash())`, sem redirecionamento, para `/status/infra/` continuar abrindo. Tem de ser `Pre`: em `Use` o roteador já escolheu a rota e a barra vira 404. |
| Corpo do erro padrão | Texto puro (`Cannot GET /x`) | JSON (`{"message":"Not Found"}`) | `HTTPErrorHandler` próprio reproduzindo os corpos de hoje. Um arquivo estático ausente **não** cai no HTML do SPA: hoje ele segue pela cadeia e pode chegar ao middleware de segurança, e a suíte fixa o que cada caso responde. |
| Limite de corpo | 4 MiB por padrão | **Nenhum** | `middleware.BodyLimit(4 << 20)` global (na v5 o argumento é `int64`, não a string `"4M"` da v4). Sem isto o restore e o push aceitariam corpo ilimitado — é o item de segurança desta change. |
| IP do cliente | `c.Context().RemoteIP()` | `c.RealIP()` usa `RemoteAddr` **enquanto `IPExtractor` for nulo**, e o cabeçalho quando não for | O IP da conexão vem de `net.SplitHostPort(r.RemoteAddr)` e continua passando por `statuspage.ClientIP` com `trusted-proxies`. `e.IPExtractor` MUST NOT ser configurado e `c.RealIP()` MUST NOT ser usado, para a regra não depender de um padrão da biblioteca. |
| Arquivos estáticos | `filesystem` com `Browse: true` | `http.ServeContent`: `ETag`, `Last-Modified`, `Range` e 304 | A listagem de diretório de `web/static` **deixa de existir** (é herança sem uso, e lista os nomes dos arquivos do build); o restante fica igual, sem `Cache-Control` nas fontes e com `EnablePathUnescaping` desligado. Echo ≥ 5.2.0, por causa da correção de caminho codificado. |
| TLS | HTTP/1.1 | `net/http` negocia HTTP/2 por ALPN | Aceito, e dito nas notas: com `web.tls`, clientes e proxies passam a poder usar `h2`. |
| `web.read-buffer-size` | Buffer de leitura e teto do cabeçalho | Não existe | Continua aceito e vira `http.Server.MaxHeaderBytes`. É uma **aproximação deliberada**, não uma equivalência: o `net/http` conta de outro jeito e tem uma folga própria, então a fronteira exata muda. A suíte cobre cabeçalho bem abaixo e bem acima do limite, e a documentação passa a dizer "limite aproximado". |
| `Immutable: true` | Necessário | Não existe | Sai, com o comentário. |
| Compressão | `compress` com `Next` | `middleware.Gzip` com `Skipper` | Mesmo pulo do canal de eventos. O Fiber também negocia brotli e deflate; passa a ser só gzip, que é o que os navegadores pedem primeiro. |

### D5 — Streaming sem `fasthttp`

`api/live_updates.go` deixa de usar `SetBodyStreamWriter`: o manipulador escreve direto em `c.Response()` e descarrega com `http.NewResponseController(...).Flush()`. Consequências:

- a regra "nunca use o `fiber.Ctx` dentro do escritor" some, porque não há mais goroutine de escrita separada — o manipulador **é** o laço;
- o cliente que foi embora é percebido por `r.Context().Done()`, que entra no `select` ao lado das notificações, do ping e da duração máxima;
- o prazo de escrita maior do canal de eventos deixa de depender do gancho `HeaderReceived`: **antes do primeiro byte**, o manipulador chama `SetWriteDeadline(time.Now().Add(liveupdates.StreamWriteTimeout))` — os mesmos 6 minutos de hoje, acima dos 5 da duração máxima — e `controller.eventStreamRequestConfig` some. Sem esse valor, o `WriteTimeout` de 15 s do servidor cortaria o canal aos 15 s, e o corte pareceria "o cliente foi embora", porque também dispara `r.Context().Done()`;
- o skipper do `Gzip` decide pelo caminho **da requisição**: se o canal fosse envolvido, o `Flush` do escritor de gzip forçaria `Content-Encoding: gzip` e o `EventSource` quebraria. O `Recover` não envolve o escritor e não atrapalha o `Flush`;
- `c.Response()` no Echo v5 já é um `http.ResponseWriter`, então é ele que vai para o `ResponseController`;
- a vaga do limitador de streams continua reservada antes do primeiro byte e liberada num `defer`, com o `recover` de hoje.

O servidor passa a ser um `http.Server` comum com o Echo como `Handler`, como no `go-ispconfig`, e não o `StartConfig` do Echo: é o `http.Server` que dá `MaxHeaderBytes` (D4) e os quatro tempos — `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout` e `IdleTimeout` de 15 s, como hoje. O desligamento usa `server.Shutdown` com prazo de 10 s no lugar de `ShutdownWithTimeout`, e continua sendo chamado tanto no sinal quanto na recarga da configuração, que é o que `controller.Shutdown` faz hoje. `http.ErrServerClosed` MUST ser tratado como encerramento normal: hoje qualquer erro do `Listen` é `Fatalf`, e a porta literal mataria o processo **em toda recarga**. Cada ciclo cria um `http.Server` novo, e o ciclo seguinte só começa depois de o anterior ter terminado de fato. `Shutdown` não derruba conexão ativa ao estourar o prazo: passados os 10 s, o `controller` chama `Close`. **A ordem de hoje fica:** `liveupdates.Close()` primeiro, `controller.Shutdown()` depois. O contexto do sinal MUST NOT ser o que desliga o servidor: ele começaria o `Shutdown` com os canais ainda abertos e esperaria os 10 s em cada um.

### D6 — Sem `adaptor`

- `promhttp`: `e.GET("/metrics", echo.WrapHandler(metricsHandler))`.
- OIDC: o `callbackHandler` já é `net/http`. O `loginHandler` **não**: recebe `*fiber.Ctx`, grava dois cookies e redireciona. Ele é convertido para `*echo.Context`, preservando os dois `Set-Cookie` com os atributos de hoje, o `Location` e o 302.
- `g8`: `gate.Protect` é um middleware `net/http`; entra por `echo.WrapMiddleware`.
- `security/admin.go` e `security/config.go` deixam de converter a requisição (`adaptor.ConvertRequest`): usam `c.Request()` direto.

### D7 — Só os pacotes do fork vão para `internal/`

`internal/` é o que a documentação do Go recomenda para o código que ninguém de fora deve importar. Mover **tudo** para lá mudaria o caminho de importação de todos os arquivos que o upstream também tem, e cada sincronização viraria conflito em massa. Então o marco 3 move só o que é do fork — `adminbackup`, `lifecycle`, `liveupdates`, `managedendpoint`, `push`, `pushkey`, `statuspage`. Os pacotes compartilhados com o upstream ficam onde estão, o `main.go` fica na raiz e o `cmd/` do marco 1 fica onde o Cobra o põe.

Isso **reduz** o conflito de sincronização, não o zera: arquivos que o upstream também tem importam pacotes do fork — `watchdog/endpoint.go`, `watchdog/push.go`, `controller/controller.go`, `api/api.go`, `api/badge.go`, `api/endpoint_status_summary.go` e `main.go` — e a linha de `import` deles muda de `gatus/v5/liveupdates` para `gatus/v5/internal/liveupdates`. É conflito de uma linha em arquivos que a sincronização já toca, e o roteiro do `AGENTS.fork.md` passa a listá-los. Não são só `import`: o passo do CI que cita `./statuspage/...` e `./managedendpoint/...` por caminho, os scripts e qualquer seletor de pacote mudam junto. `config/admin`, `config/push` e `config/statuspage` **não** se movem: vivem sob `config/`, que é do upstream.

### D8 — As diferenças aceitas, medidas pelo contrato

O contrato tem **188 respostas gravadas contra o Fiber** — as primeiras 169 antes da troca, e as demais num worktree do commit anterior a ela, cada vez que uma revisão mostrou um caso que faltava. No Echo, 175 são idênticas; estas mudam, e são aceitas:

| Diferença | Por quê |
|-----------|---------|
| `Vary: Accept-Encoding` em toda resposta (nas 159 também) | O `Gzip` do Echo declara sempre; é o correto para um cache, que senão entregaria uma resposta comprimida a quem não pediu. |
| Atributos do `Set-Cookie` noutra ordem e caixa (`Path=/; Max-Age=...` no lugar de `max-age=...; path=/`) | Mesmo cookie; é só a serialização do `net/http`. |
| Corpo acima de 4 MiB: **413** no lugar de conexão derrubada | O fasthttp fechava a conexão no meio do envio, e o cliente via erro de rede. Agora vê o 413. |
| `text/css; charset=utf-8` nos estáticos | Detecção de tipo do `net/http`. |
| `Allow: OPTIONS, GET, HEAD` no 405 | O roteador do Echo responde `OPTIONS`. |
| Corpo do 400 de um escape inválido (`%zz`): `400 Bad Request` | É o `net/http` que recusa a linha da requisição, antes do manipulador. |
| Caminho com maiúsculas (`/API/v1/config`, `/HEALTH`): **404** | Decidido em D4. **Entra nas notas como mudança.** |
| Diretório dos estáticos (`/js/`): **404** no lugar da listagem | Decidido em D4. |
| Caminho com `..` nos estáticos (`/css/../index.html`, `/css/%2e%2e/index.html`): **404** no lugar do 301 | O fasthttp resolvia o `..` antes de rotear; o `net/http` não. Recusar é mais seguro do que resolver depois que as rotas já foram escolhidas, e o template do SPA nunca é servido como arquivo. |
| `Range` nos estáticos: **206** com o trecho, no lugar do 200 com o arquivo inteiro | O Fiber ignorava o `Range`; o `net/http` atende. |

O que **não** mudou, e o contrato prova: todos os 401 e seus `WWW-Authenticate`, os 404 idênticos das capturas, os 403 do CSRF com `Origin` e `X-Forwarded-Host` forjados, os quatro limites de corpo, `If-Match` em suas quatro formas, as chaves com `%2F`, `%252F` e `+`, o login das páginas em `GET` e `HEAD` e o 429 depois de dez falhas.

### D9 — Desempenho medido

Mesma máquina, mesma configuração, 6.000 requisições com 32 conexões, binários de `master` (Fiber) e desta branch (Echo):

| Rota | Fiber | Echo v5 |
|------|-------|---------|
| `/health` | 85 mil req/s, p99 1,7 ms | 86 mil req/s, p99 1,7 ms |
| `/api/v1/status-pages/<slug>` (com cache) | 77 mil req/s, p99 2,1 ms | 81 mil req/s, p99 2,1 ms |
| badge de saúde (lê o banco) | 8,1 mil req/s, p99 17 ms | 7,3 mil req/s, p99 19 ms |
| `/` (SPA) | 22,7 mil req/s, p99 6,7 ms | **43,9 mil req/s, p99 2,7 ms** |

Empate onde o servidor é o que se mede, cerca de 10% atrás onde o gargalo é o banco, e o dobro no SPA — não por causa do Echo, mas porque a porta mostrou que o template era parseado de novo a cada requisição, e agora é uma vez só. O binário ficou 2,3 MB menor, mesmo com o Cobra.

## Risks / Trade-offs

- **Divergência do upstream — o custo que fica.** Hoje uma correção do `TwiN/gatus` em `api/` é mesclada; depois desta change ela é **portada à mão** de Fiber para Echo, para sempre. O fork já diverge bastante nesses arquivos, mas isto torna a divergência estrutural. É a decisão que vale pesar antes de aprovar o marco 2; os marcos 1 e 3 não têm esse custo.
- **Desempenho:** o `fasthttp` é mais rápido que o `net/http` em benchmark sintético. O Gatus serve um painel e uma API de leitura com cache, e o gargalo é o storage; o marco 2 mede a latência das rotas públicas antes e depois e registra os números no PR.
- **Regressão silenciosa de contrato:** mitigada por D3, e pelas quatro suítes E2E, que exercitam navegador, SSE e `curl` contra o binário real.
- **Roteador sem ordem:** uma rota protegida registrada no grupo errado ficaria pública. A suíte de contrato inclui, para **cada** rota protegida, o caso sem credencial esperando 401.
- **O corpo é lido antes da autenticação**, como no fasthttp: um anônimo consegue fazer o servidor guardar até 4 MiB por requisição, em qualquer rota. Não é regressão — o Fiber lia o corpo inteiro antes dos manipuladores —, e é o preço de recusar com 413 antes de qualquer efeito. Um `Content-Length` acima do limite é recusado sem ler nada.
- **Echo v5 é recente** (linha 5.x de 2026, mantida em paralelo à 4.x): a versão é fixada no `go.mod` e o Dependabot cuida das atualizações.
- **Cobra aumenta o binário** em algumas centenas de KB e muda o texto de erro de argumento inválido; `gatus` sem argumentos não muda.

## Migration Plan

1. Marco 1 publicado: o binário aceita os comandos novos e continua subindo igual. Reversão: reverter o PR.
2. Marco 2: suíte de contrato no Fiber → troca → suíte no Echo → E2E → medição → release. Reversão: reverter o PR; nenhuma mudança de dados ou de configuração acompanha.
3. Marco 3: renomeação mecânica, com `go build ./...` e a suíte inteira como prova.

Nenhum marco muda o banco, o `config.yaml` ou as variáveis de ambiente.

## Open Questions

- ~~O custo de sincronização com o upstream é aceitável?~~ **Decidido pelo dono em 2026-09-19: sim, a migração para o Echo v5 segue**, ciente de que as correções do upstream em `api/`, `security/` e `controller/` passam a ser portadas à mão. Os três marcos ficam na change.
- **Distinguir maiúsculas nos caminhos** é aceitável, ou é preciso um middleware que normalize?
- **`internal/` para tudo, como no `go-ispconfig`, ou só para os pacotes do fork (D7)?** O padrão do dono põe todo o código em `internal/`. Aqui isso mudaria o caminho de todos os arquivos que o upstream também tem, e a sincronização deixaria de ser possível por `git merge`. D7 recomenda só os pacotes do fork; se a sincronização com o upstream deixar de ser um objetivo — e o marco 2 já a enfraquece —, o marco 3 pode ser o `internal/` completo.
