## 1. Marco 1 — Linha de comando com Cobra

- [x] 1.1 `github.com/spf13/cobra` no `go.mod`; `main.go` **na raiz**, fino, chamando `cmd.Execute()`; `cmd/root.go` com `gatus` sem argumentos equivalendo a `gatus serve`, e um arquivo por comando em `cmd/`.
- [x] 1.2 `cmd/serve.go` com o laço de vida que hoje está em `main.go` e em `main_reload.go` (início, recarga, sinais, `GATUS_DELAY_START_SECONDS`), com os testes de `main` indo junto, sem mudança de comportamento; flags `--config` e `--log-level` na raiz e no `serve`, com precedência flag > ambiente > padrão decidida por `Flags().Changed`, e `gatus --help` sem subir o servidor. O caminho resolvido da configuração é reusado em toda recarga, e `--config` explícito inexistente é erro.
- [x] 1.3 `gatus version` com `cmd.Version`, `cmd.GitCommit` e `cmd.BuildDate` por `-ldflags -X gatus/v5/cmd.<variável>=...`, preenchidos no `Makefile`, nos dois `Dockerfile` e no workflow de release; `SilenceErrors: true` e `PersistentPreRunE` que não exige configuração em `version`, `password hash`, `help` e `completion`.
- [x] 1.4 `gatus config validate`, sem abrir o storage nem a rede, saindo com 1 e a mensagem do erro quando inválido.
- [x] 1.5 `gatus password hash` com `cobra.NoArgs`: num terminal pede a senha duas vezes sem eco; por pipe lê uma linha e descarta o `\n`/`\r\n` final; produz o mesmo bcrypt em base64 que `docs/generate-admin-password.py`.
- [x] 1.6 `gatus healthcheck` lendo só `web` da configuração (endereço curinga virando laço local, HTTPS local sem verificar o certificado), sem ler a configuração com `--url`, sem atraso de início, storage nem OIDC, sem seguir redirecionamento nem proxy do ambiente, com prazo total de 5 s; `HEALTHCHECK CMD ["/gatus","healthcheck"]` nas duas imagens `FROM scratch`, testado dentro delas com porta diferente e com HTTPS.
- [x] 1.7 `Makefile`, `Dockerfile`, `Dockerfile.release` e `.github/workflows` continuam construindo a raiz (`.`), com os `-ldflags` da versão; `ENTRYPOINT ["/gatus"]` inalterado.
- [x] 1.8 Testes Go dos comandos (precedência das flags, códigos de saída, `password hash` com LF e CRLF conferido com `security.CheckCredentials`, recarga da configuração escolhida por flag, atualização inválida, SIGTERM depois de uma recarga, `config validate` sem rede e sem banco) e as **seis** suítes E2E (`admin`, `status-pages`, `push`, `certificate`, `login`, `admin-backup`) contra o binário novo.
- [x] 1.9 `docs/cli.md`, a referência no `README.md`, e `docs/generate-admin-password.py` apontando para o comando novo.

## 2. Marco 2 — Echo v5 no lugar do Fiber

- [ ] 2.1 **Antes da troca:** suíte de contrato conforme D3, rodando contra o roteador do Fiber — cada rota de `api/api.go` e dos `register*Routes`, o caso sem credencial de cada rota protegida, os 404 idênticos das capturas sem `WWW-Authenticate`, e **toda** a lista de D3: corpo lido depois do middleware e os quatro limites, CSRF e cookie de sessão, login das páginas em `GET` e `HEAD`, eventos (vivo aos 20 s, sem gzip, vaga liberada, `HEAD` sem stream), `X-Forwarded-For` em várias linhas, barra final, `/index.html`, 405, caminho desconhecido sob `/api/` e as fontes.
- [ ] 2.2 Auxiliar de teste único no lugar de `app.Test`, e os 18 arquivos de teste passando a usá-lo, ainda no Fiber.
- [ ] 2.3 `github.com/labstack/echo/v5` fixado em `v5.3.1` no `go.mod`; `api/api.go` com os grupos público e protegido explícitos conforme D4 e a captura `/*` no protegido, `HTTPErrorHandler(c, err)` reproduzindo os corpos de hoje, `BodyLimit(4 << 20)`, `e.Pre(RemoveTrailingSlash())`, `Gzip` com skipper pelo caminho da requisição, `Recover`, o `CORS` de desenvolvimento com `ExposeHeaders: Content-Disposition`, e **sem** `AutoHandleHEAD` e **sem** `IPExtractor`.
- [ ] 2.4 Os 118 manipuladores de `api/` de `*fiber.Ctx` para `*echo.Context`, seguindo a tabela de D4 e **não** uma troca de nomes: `c.Locals` → `c.Set`/`c.Get`; `c.Get(cabeçalho)` → `c.Request().Header.Get`; `c.Set(cabeçalho)` → `c.Response().Header().Set`; `c.Params` → `c.Param`; `c.Query` → `c.QueryParam`; `c.Path()` → `c.Request().URL.Path`; `c.Body()` → o auxiliar de corpo; `PeekAll` → `Header.Values`; `fiber.Map` → `map[string]any`. `api/spa.go` escreve o template num buffer, porque `*echo.Context` não é `io.Writer`.
- [ ] 2.4a Middleware global de corpo conforme D4 (lê adiantado até 4 MiB + 1, 413 antes de qualquer efeito, recoloca os bytes), com os limites de 256 KB, 3,5 MiB e 4 KB nas rotas sobre os bytes já lidos; testes com tamanho conhecido, *chunked*, no limite exato e um byte acima.
- [ ] 2.4c Nenhum cabeçalho depois de escrever, conforme D4: `statusPageBadgeHandler` define `Cache-Control` e `Vary` antes do gerador e os geradores não sobrescrevem; `api/spa.go` em buffer; `HTTPErrorHandler` respeitando resposta iniciada.
- [ ] 2.4d Serializador JSON sem quebra de linha final e sem `ETag` automático; roteador sem decodificar parâmetros, cada manipulador com a decodificação de hoje.
- [ ] 2.4b CSRF e cookie de sessão com `c.IsTLS()` e `c.Request().Host`, sem `c.Scheme()`; `gatus_session` com os mesmos atributos de `setLoginSessionCookie`.
- [ ] 2.5 IP do cliente por `r.RemoteAddr` em todos os 8 pontos que usam `c.Context().RemoteIP()`, sem `c.RealIP()` em lugar nenhum, com teste provando que `X-Forwarded-For` de origem não confiável é ignorado.
- [ ] 2.6 `api/live_updates.go` conforme D5: escrita direta com `http.ResponseController`, `SetWriteDeadline` de `liveupdates.StreamWriteTimeout` antes do primeiro byte, `r.Context().Done()` no `select`, vaga liberada num `defer` com `recover`, `HEAD` sem stream, e a remoção de `controller.eventStreamRequestConfig`.
- [ ] 2.7 `security/` sem `adaptor` conforme D6, o `g8` por `echo.WrapMiddleware`, e o `loginHandler` do OIDC convertido preservando os dois cookies, o `Location` e o 302, com teste de login, callback e autorização da administração por *subject*.
- [ ] 2.8 `controller/controller.go` com `http.Server` (tempos de 15 s, `MaxHeaderBytes` de `web.read-buffer-size`), TLS e `Shutdown` de 10 s, mantendo a ordem `liveupdates.Close()` → `controller.Shutdown()` no sinal e na recarga; `http.ErrServerClosed` como encerramento normal, um `http.Server` novo por ciclo, `Close` depois do prazo do `Shutdown`, e teste de recarga com listener real e canal de eventos aberto.
- [ ] 2.9 Arquivos estáticos embutidos e o 301 de `/index.html`, sem listagem de diretório, sem `Cache-Control` nas fontes e com `EnablePathUnescaping` desligado.
- [ ] 2.10 `go mod tidy` sem `gofiber/fiber` nem `valyala/fasthttp`; suíte de contrato, testes de transporte com servidor TCP real (D3), testes Go com `-race` e as seis suítes E2E verdes.
- [ ] 2.11 Medição de latência das rotas públicas antes e depois, registrada no PR.
- [ ] 2.12 `AGENTS.fork.md` (as quatro notas que citam o Fiber e o roteiro de sincronização com o upstream, agora com a portagem manual), `docs/README.md` (`web.read-buffer-size`) e as notas de release com a mudança de maiúsculas.

## 3. Marco 3 — Pacotes do fork em `internal/`

- [ ] 3.1 `adminbackup`, `lifecycle`, `liveupdates`, `managedendpoint`, `push`, `pushkey` e `statuspage` para `internal/`, com os caminhos de importação atualizados, e a lista dos arquivos do upstream cujo `import` muda (D7) no roteiro de sincronização do `AGENTS.fork.md`.
- [ ] 3.1a O passo do CI que cita `./statuspage/...` e `./managedendpoint/...` (`.github/workflows/ci.yml`), os scripts e os demais seletores de pacote apontando para `internal/`.
- [ ] 3.2 `go build ./...`, testes com `-race` e as seis suítes E2E verdes; `AGENTS.fork.md` e os caminhos citados em `docs/` atualizados.

## 4. Entrega

- [ ] 4.1 Um PR e uma release por marco, com CI verde, notas em inglês, imagem no Docker Hub, pacote `mariadb` e versões dos exemplos.
- [ ] 4.2 `openspec validate migrate-to-echo-and-cobra --strict` e arquivamento depois do marco 3.
