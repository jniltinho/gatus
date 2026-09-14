# AGENTS.fork.md

Regras para agentes no fork **jniltinho/gatus**. Complementam o [AGENTS.md](AGENTS.md) do upstream; em caso de conflito, estas regras prevalecem.

## Skills obrigatórias

Em qualquer tarefa com Go, carregue `golang-how-to`, que seleciona as demais skills `golang-*` de `.claude/skills/` (code style, naming, error handling, concurrency, database, testing, security, lint, entre outras). Para releases, use `create-release`. Para testar a interface pelo navegador, use `agent-browser`.

## Comandos do fork

- `make lint` — `go vet ./...` e `gofmt` apenas nos arquivos Go adicionados ou alterados desde `UPSTREAM_BASE` (não reformate código do upstream)
- `make fmt` — aplica `gofmt` nesses mesmos arquivos
- `make build` — binário estático em `dist/gatus`
- `make release-cross VERSION=5.36.0-fork.1` — tarballs `linux/amd64` e `linux/arm64` em `dist/`
- `make docker-release VERSION=5.36.0-fork.1` — publica `jniltinho/gatus:v5.36.0-fork.1` (amd64 e arm64) no Docker Hub a partir da máquina local; nunca publica `latest`
- `make test` e os demais alvos do `AGENTS.md` continuam valendo

`UPSTREAM_BASE` (no `Makefile`) é o commit do upstream em que o fork está baseado.

## Módulo Go

O caminho do módulo do fork é `gatus/v5` (no upstream, `github.com/TwiN/gatus/v5`). Imports internos usam sempre `gatus/v5/...`; nunca reintroduza `github.com/TwiN/gatus/v5`. O fork não é instalável com `go install`/`go get`: gere o binário com `make build` ou use os tarballs e a imagem das releases.

## Dependências

- Depois de adicionar ou atualizar uma dependência Go, execute `go mod tidy`.
- **Não** execute `go mod vendor`: `vendor/` não é versionado neste repositório (a instrução do `AGENTS.md` não vale aqui).

## CI e releases

- `.github/workflows/ci.yml`: `make lint`, `make build` e `go test ./... -race` (com `sudo`, por causa do teste de ICMP).
- `.github/workflows/release.yml`: disparado por tags `v*-fork.*`; gera os tarballs e a GitHub Release. Não publica imagens.
- Tags do fork: `v<versão-upstream>-fork.<N>`. Nunca crie tags `vX.Y.Z` sem o sufixo.

## Administração de endpoints

Mudança em andamento: `openspec/changes/add-admin-endpoint-management/` (leia `design.md` antes de mexer nessas áreas).

- Endpoints cadastrados pela web ficam na tabela `managed_endpoints`; `cfg.Endpoints` contém só o YAML e não é alterado depois do load.
- O watchdog controla cada endpoint por um registro (contexto e `done` por endpoint). Nunca altere um `*endpoint.Endpoint` que esteja em execução: crie um objeto novo e reinicie pelo registro.
- As labels Prometheus são as registradas no ciclo atual; não recalcule a lista fora de `InitializePrometheusMetrics`.
- Endpoints gerenciados passam por validação estrita: sem expansão de variáveis de ambiente e sem campos que usem credenciais ou arquivos do servidor.
- Escritas da administração são serializadas entre si e com a partida e o hot-reload.
- Mantenha código novo em arquivos novos sempre que possível, para reduzir conflitos com o upstream.

## Status pages públicas

Mudança: `openspec/changes/add-public-status-pages/` (leia `design.md` antes de mexer nessas áreas; documentação em `docs/status-pages.md`).

- O payload público usa só os tipos de `statuspage/payload.go`: nunca serialize `endpoint.Status` ou `endpoint.Result` numa rota pública (hostname, erros e condições vazariam). O teste de sanitização decodifica o JSON com `DisallowUnknownFields`.
- As rotas públicas (`/api/v1/status-pages/*` e `/status/*`) ficam no bloco livre de `api/api.go`, antes dos arquivos estáticos e do middleware de segurança. O catch-all de `/api/v1/status-pages` é sempre registrado: um caminho que caia no middleware responderia 401 e abriria o login do navegador.
- Escritas da administração gravam no banco e **só depois do commit** publicam o snapshot com revisão nova; cache e `singleflight` usam `slug|revisão|geração`, nunca a versão do banco.
- A montagem é síncrona na goroutine da requisição e resolve o leitor do store a cada montagem (a recarga fecha e troca o store).
- O limitador é próprio e sem goroutine (o `limiter` do Fiber vaza uma goroutine por recarga) e conta só as respostas 404.
- No frontend, as rotas com `meta.public` não mostram a tela de login nem buscam `/api/v1/config`. O Tailwind do projeto (3.1.8) não tem o tom 950: use `dark:bg-*-900/30`.

## OpenSpec

- Propostas em `openspec/changes/<change>/`; valide com `openspec validate <change> --strict`.
- Ao concluir uma tarefa de `tasks.md`, marque o checkbox.

## Testes ponta a ponta

- Use a skill `agent-browser`. Com o binário avulso, defina `AGENT_BROWSER_SKILLS_DIR` apontando para o `skill-data` da versão instalada antes de `agent-browser skills get core`.
- Capturas de tela vão para `dist/prints/`. `dist/` está no `.gitignore`: **nunca** commite capturas.
- Roteiros: `test/e2e/admin.sh` e `test/e2e/status-pages.sh`. Espere por um seletor (`wait "[data-testid=...]"`) em vez de texto quando a tela anterior tiver o mesmo texto (por exemplo, o botão "Nova status page" e o título do formulário).

## Sincronização com o upstream

```bash
git fetch upstream --tags
git merge upstream/master
# Código novo do upstream chega com o caminho de módulo antigo
grep -rl --include='*.go' 'github.com/TwiN/gatus/v5' . | xargs -r sed -i 's#github.com/TwiN/gatus/v5#gatus/v5#g'
gofmt -w $(git diff --name-only -- '*.go')
```

- Conflitos em imports (quase todo arquivo Go difere do upstream só pelo caminho do módulo): resolva mantendo o conteúdo do upstream e aplique a troca acima; confira com `grep -rn 'github.com/TwiN/gatus/v5' --include='*.go' .` (sem resultados) e `go build ./...`.
- `go.mod`: mantenha `module gatus/v5`.

- Conflito em `web/static/`: aceite qualquer lado e regenere com `make frontend-install && make frontend-build`.
- Workflows removidos pelo fork (`benchmark`, `labeler`, `publish-*`, `regenerate-static-assets`, `test`, `test-ui`): mantenha removidos.
- `AGENTS.md`: aceite a versão do upstream e mantenha a linha que aponta para este arquivo.
- Atualize `UPSTREAM_BASE` no `Makefile` para o commit do upstream incorporado e rode `make lint test`.
- A próxima release passa a usar a nova versão do upstream como base (`vX.Y.Z-fork.1`).

## Commits e PRs

Siga a regra do `AGENTS.md`: commits e PRs feitos por agente informam que foram feitos por agente, com o nome e a versão do modelo. Use prefixos `feat:`, `fix:`, `chore:`, `docs:`, `ci:` (a skill `create-release` usa esses prefixos para as notas).
