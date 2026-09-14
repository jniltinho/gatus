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

## OpenSpec

- Propostas em `openspec/changes/<change>/`; valide com `openspec validate <change> --strict`.
- Ao concluir uma tarefa de `tasks.md`, marque o checkbox.

## Testes ponta a ponta

- Use a skill `agent-browser`. Com o binário avulso, defina `AGENT_BROWSER_SKILLS_DIR` apontando para o `skill-data` da versão instalada antes de `agent-browser skills get core`.
- Capturas de tela vão para `dist/prints/`. `dist/` está no `.gitignore`: **nunca** commite capturas.

## Sincronização com o upstream

```bash
git fetch upstream --tags
git merge upstream/master
```

- Conflito em `web/static/`: aceite qualquer lado e regenere com `make frontend-install && make frontend-build`.
- Workflows removidos pelo fork (`benchmark`, `labeler`, `publish-*`, `regenerate-static-assets`, `test`, `test-ui`): mantenha removidos.
- `AGENTS.md`: aceite a versão do upstream e mantenha a linha que aponta para este arquivo.
- Atualize `UPSTREAM_BASE` no `Makefile` para o commit do upstream incorporado e rode `make lint test`.
- A próxima release passa a usar a nova versão do upstream como base (`vX.Y.Z-fork.1`).

## Commits e PRs

Siga a regra do `AGENTS.md`: commits e PRs feitos por agente informam que foram feitos por agente, com o nome e a versão do modelo. Use prefixos `feat:`, `fix:`, `chore:`, `docs:`, `ci:` (a skill `create-release` usa esses prefixos para as notas).
