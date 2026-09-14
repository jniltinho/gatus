## 1. Marco 1 — CI, release e skills

- [x] 1.1 `Makefile`: `UPSTREAM_BASE` com o commit do upstream e checagem de que ele existe; alvos `build` (`CGO_ENABLED=0` só na receita, saída `dist/gatus`), `fmt` e `lint` (gofmt nos arquivos Go adicionados ou alterados desde `UPSTREAM_BASE` + `go vet ./...`), `vet`, `release-cross VERSION=...` (tarballs `linux_amd64` e `linux_arm64` com `gatus`, `config.yaml`, `LICENSE` e `README.md`, mais `dist/pkg/linux_<arch>/gatus`) e `docker-release VERSION=...`; `docker-build` e `docker-run` usando `jniltinho/gatus`
- [x] 1.2 Criar `Dockerfile.release` (certificados e tzdata de um estágio na plataforma de build, imagem final `scratch` com o binário de `dist/pkg/linux_${TARGETARCH}`) e `Dockerfile.release.dockerignore`; criar `web/app/.nvmrc`
- [x] 1.3 Criar `.github/workflows/ci.yml`: checkout com histórico completo antes de `setup-go`; `make lint`; `make build`; testes com `sudo ... -race` preservando `GOCACHE` e `GOMODCACHE`
- [x] 1.4 Criar `.github/workflows/release.yml`: tags `v*-fork.*`; testes; `make release-cross`; `gh release create` com `--notes-start-tag` da tag anterior do fork, sem pré-release, marcada como latest, com os dois tarballs
- [x] 1.5 Remover `benchmark.yml`, `labeler.yml`, `publish-custom.yml`, `publish-experimental.yml`, `publish-latest.yml`, `publish-release.yml`, `regenerate-static-assets.yml`, `test.yml` e `test-ui.yml`; restringir `.github/dependabot.yml` a `github-actions`
- [x] 1.6 Conferir as 15 skills `golang-*` e a `agent-browser` copiadas; adicionar os arquivos de licença de `samber/cc-skills-golang` (MIT) e `vercel-labs/agent-browser` (Apache-2.0)
- [x] 1.7 Criar `.claude/skills/create-release/SKILL.md` adaptada ao Gatus (branch `master`, última tag com `git describe --tags --abbrev=0 --match 'v*-fork.*'`, próxima `-fork.N`, pacotes amd64/arm64 pelo workflow, imagem com `make docker-release`, notas com `--notes-start-tag`, link de changelog para `jniltinho/gatus`)
- [x] 1.8 Criar `AGENTS.fork.md` (skills obrigatórias, release, alvos do Makefile, testes E2E com `agent-browser` e capturas em `dist/prints/`, não usar `go mod vendor`, sincronização com o upstream) e adicionar ao `AGENTS.md` uma linha apontando para ele
- [ ] 1.9 Abrir o pull request no fork, deixar o CI verde e fazer merge
- [ ] 1.10 Publicar `v5.36.0-fork.1` pela skill `create-release`: tag e release dos binários pelo workflow, imagem `jniltinho/gatus:v5.36.0-fork.1` com `make docker-release`; conferir release, tarballs e imagem

## 2. Marco 2 — Visual quadrado

- [ ] 2.1 `--radius: 0` em `web/app/src/index.css` e `theme.borderRadius` substituído no `tailwind.config.js` (todas as variantes 0, `full` mantido)
- [ ] 2.2 Trocar `rounded-full` por `rounded-none` em `Badge.vue`, no contador de falhas de `Home.vue`, na barra e nos botões de `Settings.vue` e nas barras de progresso de `SequentialFlowDiagram.vue`
- [ ] 2.3 Definir `cornerRadius: 0` no tooltip de `ResponseTimeChart.vue`
- [ ] 2.4 Remover `rx` dos `rect` dos badges SVG em `api/badge.go` e ajustar os testes
- [ ] 2.5 `make frontend-build` e commit de `web/static/`
- [ ] 2.6 Capturas com `agent-browser` do dashboard, detalhes de endpoint e suite, anúncios e configurações nos temas claro e escuro, em `dist/prints/`
- [ ] 2.7 Pull request, CI verde e merge

## 3. Marco 3 — Fundação: hot-reload, watchdog e métricas

- [ ] 3.1 Hot-reload: carregar e validar o YAML novo antes de `stop()`; com `skip-invalid-config-update: true`, manter tudo rodando se for inválido; testes
- [ ] 3.2 Registro por endpoint no `watchdog` (origem, contexto, cancelamento, `done`, estado `closed`) com `StartEndpoint`, `StopEndpoint` e `RestartEndpoint`; `Monitor` e `Shutdown` sobre o registro sem mudar o comportamento para o YAML
- [ ] 3.3 `executeEndpoint` com o contexto do endpoint no semáforo e descarte de resultados após cancelamento (sem métricas, store ou alertas)
- [ ] 3.4 `metrics`: expor a lista de labels registrada no ciclo e usá-la no watchdog e em `api/external_endpoint.go`; função para apagar as séries de uma chave com `DeletePartialMatch`
- [ ] 3.5 Lock de ciclo de vida (exclusivo na partida e no hot-reload; tentativa compartilhada para escritas administrativas) com indicação de ciclo em andamento
- [ ] 3.6 Extrair de `initializeStorage` a restauração de alertas disparados para uma função reutilizável
- [ ] 3.7 Testes: parada durante verificação lenta (`httptest`) não grava nada depois; reinício sem resultados ou alertas duplicados; parar um endpoint não reinicia outro; `go test -race` limpo
- [ ] 3.8 Pull request, CI verde e merge

## 4. Marco 4 — Backend de administração

- [ ] 4.1 Pacote `config/admin` (`enabled`, `allowed-subjects`, `allowed-origins`) e validação dos pré-requisitos antes da exigência de endpoints; configuração sem endpoints com `admin.enabled`; testes
- [ ] 4.2 Tipo `ManagedEndpoint` (com `version`) e interface `ManagedEndpointStore`; tabela `managed_endpoints` em SQLite e PostgreSQL com erro no `CREATE`; implementação com transações e conflito de unicidade como erro próprio; testes em SQLite e PostgreSQL (`GATUS_TEST_POSTGRES_URL`) e serviço PostgreSQL no `ci.yml`
- [ ] 4.3 Remoção de todos os dados de uma chave nos stores SQL e memory, invalidando o `writeThroughCache` da chave e o cache de status da API; testes
- [ ] 4.4 Validação dos gerenciados: decodificação estrita sem env, campos bloqueados, `client.tunnel`, validação estrita de alertas por endpoint, chave única por `Key()` com origem na mensagem, `extra-labels` contidos na lista registrada; padrões só em memória; serialização `definition` e `effective`; testes
- [ ] 4.5 Mascaramento de segredos e preservação do valor armazenado quando a máscara é reenviada; testes
- [ ] 4.6 Registro dos gerenciados (copy-on-write) e carga em `initializeStorage` independente de `admin.enabled`: preservação de chaves, conflito com o YAML, gerenciados inválidos, restauração de alertas; busca por chave em `api/badge.go` também nos gerenciados; aviso de várias instâncias com PostgreSQL; testes de reinício, recarga, conflito e admin desligado
- [ ] 4.7 Autor da requisição (usuário basic ou subject OIDC), middleware de autorização de administrador, middleware de CSRF, limite de corpo; `admin` em `GET /api/v1/config`; helper de teste para sessão OIDC injetada; testes de 401, 403 (subject, `Origin`, `Sec-Fetch-Site`), 413, 415, proxy HTTPS, porta não padrão e ambiente dev
- [ ] 4.8 Serviço de administração com mutex e rotas: listar, obter com `ETag`, criar, alterar e remover com `If-Match`, habilitar, desabilitar, validar, testar (limites) e metadados; 503 durante ciclo; auditoria; remoção de séries Prometheus; testes para os cenários da spec
- [ ] 4.9 Pull request, CI verde e merge

## 5. Marco 5 — Frontend, E2E, documentação e release

- [ ] 5.1 Rotas SPA `/admin`, `/admin/endpoints/new` e `/admin/endpoints/:key/edit` em `api/api.go` e no `vue-router`
- [ ] 5.2 Link "Admin" no cabeçalho conforme `config.admin`
- [ ] 5.3 View de lista: busca, origem, conflito e erro, habilitar/desabilitar com `If-Match`, remoção com confirmação e aviso de alertas disparados
- [ ] 5.4 View de formulário: modo formulário e modo YAML, conversão via `POST /validate`, nome e grupo somente leitura, segredos mascarados, tratamento de 412
- [ ] 5.5 Ações Validar, Testar (resultado por condição e duração) e Salvar, com erros exibidos sem perder o conteúdo
- [ ] 5.6 Conferir variantes `dark:` e as convenções do `AGENTS.md` nos componentes novos; `make frontend-build` e commit de `web/static/`
- [ ] 5.7 Roteiro `test/e2e/admin.sh` com `agent-browser` (lista, criar, validar, testar, salvar, editar, desabilitar, remover, sem credenciais) nos temas claro e escuro, com capturas em `dist/prints/`
- [ ] 5.8 `README.md`: seção `admin` com exemplo, pré-requisitos, API, segredos mascarados, várias instâncias, ordenação das tags do fork e rollback
- [ ] 5.9 `openspec validate add-admin-endpoint-management --strict`, `make lint`, `make test`, roteiro E2E e CI verdes
- [ ] 5.10 Pull request e merge; publicar a próxima release pela skill `create-release` e conferir a imagem no Docker Hub
