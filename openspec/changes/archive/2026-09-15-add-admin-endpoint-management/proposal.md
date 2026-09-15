## Why

No Gatus, os endpoints monitorados só podem ser cadastrados editando o YAML de configuração. Para uma equipe que mantém instâncias em vários servidores, isso exige acesso ao servidor para cada host novo, e um YAML inválido salvo no lugar para todo o monitoramento até ser corrigido (comportamento confirmado na v5.36.0). O mantenedor recusou uma interface de configuração pela web ([TwiN/gatus#1345](https://github.com/TwiN/gatus/issues/1345), fechada como *not planned*), então a funcionalidade será mantida neste fork (`jniltinho/gatus`).

O fork também precisa de automação própria: os workflows herdados publicam imagens com segredos e processos do projeto original, e as tags do upstream foram copiadas para o fork, o que impede usar a mesma numeração. Aproveitando a mudança de interface, o visual passa a ser quadrado, sem cantos arredondados.

## What Changes

- Nova seção `admin` (`enabled`, `allowed-subjects`, `allowed-origins`) que habilita a administração de endpoints pela web; exige `security` e storage `sqlite` ou `postgres` e, com OIDC, `allowed-subjects` preenchido.
- Endpoints cadastrados pela web ficam na tabela `managed_endpoints` (com versão para concorrência otimista) e convivem com os endpoints do YAML, que continuam somente leitura. Com storage SQL eles são sempre carregados, mesmo com `admin` desligado, para não perder monitoramento nem histórico.
- API `/api/v1/admin/*` para listar, obter (com segredos mascarados), criar, alterar, habilitar, desabilitar, remover, validar e testar endpoints, com o mesmo formato do YAML e validação mais estrita.
- Watchdog com ciclo de vida por endpoint: alterar ou remover um endpoint espera a verificação em andamento, descarta resultados de execuções canceladas e não reinicia os demais; escritas da administração são serializadas com a partida e o hot-reload.
- Hot-reload valida o YAML novo antes de parar o Gatus: com `skip-invalid-config-update: true`, um YAML inválido não derruba mais HTTP e monitoramento.
- Endpoints gerenciados não expandem variáveis de ambiente e não podem usar credenciais ou arquivos do servidor (`identity-aware-proxy`, arquivos TLS); `extra-labels` ficam limitados às labels Prometheus já registradas.
- Segurança: autorização de administrador, proteção CSRF compatível com proxy reverso, limites de corpo e de testes, auditoria em log.
- Telas de administração (`/admin`): lista, formulário, editor YAML, validar e testar antes de salvar, confirmação de remoção.
- Visual quadrado: dashboard, telas de admin e badges SVG sem cantos arredondados (indicadores circulares continuam redondos).
- CI e release no padrão de `jniltinho/llama-model`, por enquanto só para o código Go e o binário: `ci.yml` com lint do código do fork, build e testes com race (SQLite e PostgreSQL), e `release.yml` que gera os pacotes `linux/amd64` e `linux/arm64` e cria a GitHub Release.
- Imagem multi-arch (`Dockerfile.release`) publicada no Docker Hub `jniltinho/gatus` com `make docker-release`, a partir de uma máquina com login, sempre com tag de versão e nunca `latest`; publicar a imagem pelo workflow fica para uma mudança futura.
- Remoção dos workflows herdados; Dependabot restrito a GitHub Actions.
- Versionamento do fork: `v<versão-upstream>-fork.<N>` (ex.: `v5.36.0-fork.1`).
- Skills de agente (`golang-*`, `create-release`, `agent-browser`) com as licenças de origem; regras do fork em `AGENTS.fork.md`, incluindo o procedimento de sincronização com o upstream.
- Testes ponta a ponta com `agent-browser`, capturas de tela em `dist/prints/` (fora do git).
- Entrega em 5 marcos, cada um num pull request próprio.

Sem a seção `admin`, as únicas mudanças de comportamento são o visual quadrado e o hot-reload que não para mais com YAML inválido quando `skip-invalid-config-update` está ativo. Nenhuma mudança é **BREAKING**.

## Capabilities

### New Capabilities
- `admin-endpoint-management`: persistência, validação, API, ciclo de vida em tempo de execução e convivência com o YAML dos endpoints cadastrados pela web.
- `admin-access-control`: seção `admin`, pré-requisitos, autorização, CSRF, auditoria e estado de admin exposto ao frontend.
- `admin-web-ui`: telas de administração e testes ponta a ponta com `agent-browser`.
- `config-hot-reload`: validação do YAML novo antes de parar o Gatus.
- `ui-square-style`: remoção dos cantos arredondados da interface e dos badges SVG.
- `ci-release-pipeline`: workflows de CI e release, alvos do Makefile, imagem de release, versionamento e limpeza dos workflows herdados.
- `agent-skills`: skills de agente, licenças e `AGENTS.fork.md`.

### Modified Capabilities
Nenhuma (ainda não existem specs em `openspec/specs/`).

## Impact

- **Go:** `main.go` (hot-reload, carga dos gerenciados, restauração de alertas), `watchdog/` (registro por endpoint), `metrics/` (labels registradas, remoção de séries), `config/config.go` e novo `config/admin`, novo pacote de endpoints gerenciados, `storage/store/` e `storage/store/sql/` (tabela, interface, remoção por chave, caches), `security/` (autor da requisição), `api/` (rotas e middlewares admin, `/api/v1/config`, rotas SPA, `badge.go`).
- **Frontend:** `web/app/src` (rotas, views e componentes de admin, tema Tailwind, componentes com `rounded-*`, tooltip do gráfico), `web/app/.nvmrc` e `web/static/` regenerado.
- **Banco:** tabela `managed_endpoints` em SQLite e PostgreSQL, criada automaticamente.
- **API:** rotas `/api/v1/admin/*`; `GET /api/v1/config` ganha o campo `admin`.
- **CI/Release:** `.github/workflows/` (novos `ci.yml` e `release.yml`; remoção de `benchmark`, `labeler`, `publish-*`, `regenerate-static-assets`, `test` e `test-ui`), `.github/dependabot.yml`, `Makefile` (incluindo `docker-release`), `Dockerfile.release`; nenhum secret novo no GitHub.
- **Dependências:** nenhuma dependência Go ou npm nova.
- **Documentação e testes:** `README.md`, `AGENTS.fork.md` (e uma linha no `AGENTS.md`), `.claude/skills/`, roteiro E2E em `test/e2e/`.
- **Upstream:** o fork diverge mais do TwiN/gatus; o código novo fica em arquivos novos sempre que possível e o procedimento de sincronização fica documentado.
