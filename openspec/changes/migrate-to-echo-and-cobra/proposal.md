## Why

O servidor HTTP do Gatus é o Fiber v2, que roda sobre o `fasthttp` e não sobre o `net/http` da biblioteca padrão. Isso cobra um preço em três lugares que o fork já sente:

- **Tudo que é `net/http` precisa de adaptador.** O Prometheus (`promhttp`), o OIDC e o `g8` (o portão de autenticação) são `net/http`, e entram pelo `fiber/middleware/adaptor`, que converte a requisição a cada chamada (`security/config.go`, `security/admin.go`, `api/api.go`).
- **O tempo real está amarrado ao `fasthttp`.** O canal de eventos usa `SetBodyStreamWriter`, com a regra de nunca tocar o `fiber.Ctx` dentro do escritor, e o prazo de escrita por requisição depende de um gancho do `fasthttp` (`server.HeaderReceived`) decidido **antes** do roteamento, pelo caminho cru.
- **O `fasthttp` não fala HTTP/2** e reaproveita buffers entre requisições, o que obriga o `Immutable: true` que o próprio código explica num comentário.

O Echo v5 (estável desde 2026, hoje na v5.3.1) é `net/http` puro: `promhttp`, OIDC e `g8` entram sem adaptador, o streaming vira `http.ResponseController` e o prazo de escrita passa a ser por requisição, dentro do manipulador.

Do lado do binário, o Gatus **não tem linha de comando**: tudo vem de variáveis de ambiente, não há `--help`, `--version` nem como validar um `config.yaml` sem subir o servidor, e o hash da senha da administração depende de um script Python à parte (`docs/generate-admin-password.py`). O `main.go` de 213 linhas mistura leitura de ambiente, ciclo de vida e sinais.

## What Changes

Em três marcos, cada um entregue e publicado sozinho:

1. **Linha de comando com Cobra** — o binário ganha comandos (`serve`, `version`, `config validate`, `password hash`, `healthcheck`), com `gatus` sem argumentos continuando a subir o servidor, como hoje. O `main.go` **continua na raiz do projeto**, fino, chamando `cmd.Execute()`, e os comandos ficam no pacote `cmd/`, um arquivo por comando — o layout que o próprio gerador do Cobra produz.
2. **Echo v5 no lugar do Fiber** — mesmas rotas, mesmos corpos, mesmos cabeçalhos e mesmos códigos de status, sem `adaptor` e sem `fasthttp`.
3. **Pacotes próprios do fork em `internal/`** — só os que o Gatus original não tem (`adminbackup`, `lifecycle`, `liveupdates`, `managedendpoint`, `push`, `pushkey`, `statuspage`), para não transformar cada sincronização com o upstream num conflito de caminho.

Nada muda para quem usa: mesmas URLs, mesmo `config.yaml`, mesmas variáveis de ambiente, mesma imagem Docker com `ENTRYPOINT ["/gatus"]`.

- **BREAKING (comportamento de rota):** o Fiber casa rotas **sem distinguir maiúsculas** e ignora a barra final; o Echo distingue e não ignora. A change mantém a barra final por middleware e passa a distinguir maiúsculas — `/API/v1/...` deixa de responder. Ver D4.

## Impact

- **Specs novas:** `http-server` (o contrato que a troca não pode quebrar) e `command-line-interface`.
- **Código:** `api/` (24 arquivos e 118 manipuladores), `security/` (4), `controller/` (1), 18 arquivos de teste com 56 requisições via `app.Test`, `main.go` e o pacote novo `cmd/`. `Makefile`, `Dockerfile`, `Dockerfile.release` e `.github/workflows` continuam construindo a raiz (`.`); só ganham os `-ldflags` da versão.
- **Dependências:** entram `github.com/labstack/echo/v5` e `github.com/spf13/cobra`; saem `github.com/gofiber/fiber/v2` e `github.com/valyala/fasthttp`.
- **Sincronização com o upstream:** é o custo real desta change. O Gatus original continua no Fiber, então toda mudança dele em `api/`, `security/` e `controller/` passa a ser portada à mão em vez de mesclada. Ver Risks no design.
- **Documentação:** `docs/README.md` (`web.read-buffer-size`), `AGENTS.fork.md` (as quatro notas que citam o Fiber), `README.md` e um `docs/cli.md` novo.
