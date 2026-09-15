# ci-release-pipeline Specification

## Purpose
TBD - created by archiving change add-admin-endpoint-management. Update Purpose after archive.
## Requirements
### Requirement: Workflow de CI
O repositório MUST ter `.github/workflows/ci.yml`, executado em push para `master` e em pull requests para `master`, que faça checkout com histórico completo antes de configurar o Go pela versão de `go.mod`, execute `make lint` e `make build`, e execute os testes Go com detector de race, com os privilégios exigidos pelo teste de ICMP e preservando os caches do Go. Quando existirem testes do store em PostgreSQL, o workflow MUST disponibilizar um PostgreSQL para eles. O workflow MUST falhar se qualquer etapa falhar.

#### Scenario: Código do fork sem formatação
- **WHEN** um pull request adiciona um arquivo Go que não passa no `gofmt`
- **THEN** o CI falha na etapa de lint

#### Scenario: Pull request correto
- **WHEN** um pull request passa em lint, build e testes
- **THEN** o CI conclui com sucesso

### Requirement: Alvos do Makefile
O `Makefile` MUST definir `UPSTREAM_BASE` com o commit do upstream em que o fork se baseia e oferecer:
- `build`: binário estático em `dist/gatus`, com `CGO_ENABLED=0` apenas nessa receita;
- `fmt`: `gofmt` nos arquivos Go adicionados ou alterados desde `UPSTREAM_BASE`;
- `vet`: `go vet ./...`;
- `lint`: falha se algum arquivo Go adicionado ou alterado desde `UPSTREAM_BASE` precisar de `gofmt`, se `go vet` falhar ou se o commit `UPSTREAM_BASE` não existir no clone;
- `release-cross VERSION=<versão>`: gera `dist/gatus_<versão>_linux_amd64.tar.gz` e `dist/gatus_<versão>_linux_arm64.tar.gz`, cada um com `gatus`, `config.yaml`, `LICENSE` e `README.md`, e os binários em `dist/pkg/linux_<arch>/gatus`;
- `docker-release VERSION=<versão>`: executa `release-cross` e publica a imagem `jniltinho/gatus:v<versão>` para `linux/amd64` e `linux/arm64`.

Os alvos existentes (`install`, `run`, `test`, `frontend-install`, `frontend-build`, `frontend-dev`) MUST continuar funcionando.

#### Scenario: Lint ignora arquivos intocados do upstream
- **WHEN** um arquivo do upstream que não foi alterado pelo fork não passa no `gofmt`
- **THEN** `make lint` não falha por causa dele

#### Scenario: Pacotes de release
- **WHEN** alguém executa `make release-cross VERSION=5.36.0-fork.1`
- **THEN** são gerados os dois pacotes com `gatus`, `config.yaml`, `LICENSE` e `README.md`

### Requirement: Imagem de release
O repositório MUST ter `Dockerfile.release`, que monta a imagem a partir dos binários de `dist/pkg/linux_<arch>/`, com certificados de CA e dados de fuso horário obtidos num estágio executado na plataforma de build, sem executar comandos na plataforma de destino. O `Dockerfile` do upstream MUST permanecer inalterado. `make docker-release` MUST publicar apenas a tag da versão, MUST NOT publicar `latest` e MUST recusar execução sem `VERSION` explícita.

#### Scenario: Build multi-arquitetura sem emulação
- **WHEN** a imagem de release é construída para `linux/amd64` e `linux/arm64` numa máquina sem QEMU
- **THEN** o build conclui com sucesso para as duas arquiteturas

#### Scenario: Publicação local da imagem
- **WHEN** alguém com `docker login` no Docker Hub executa `make docker-release VERSION=5.36.0-fork.1`
- **THEN** a imagem `jniltinho/gatus:v5.36.0-fork.1` fica disponível para `linux/amd64` e `linux/arm64`
- **AND** a tag `latest` não é criada nem alterada

#### Scenario: Sem versão explícita
- **WHEN** alguém executa `make docker-release` sem `VERSION`
- **THEN** o alvo falha sem publicar nada

### Requirement: Versionamento do fork
As releases do fork MUST usar tags no formato `v<versão-upstream-base>-fork.<N>`, com `N` começando em 1 e incrementado a cada release sobre a mesma base. Tags no formato `vX.Y.Z` sem o sufixo MUST NOT ser criadas pelo fork. A documentação MUST explicar que, pela precedência do SemVer, essas tags ordenam abaixo da versão do upstream de mesma base.

#### Scenario: Nova release sobre a mesma base
- **WHEN** a última release do fork é `v5.36.0-fork.2` e o fork continua baseado na v5.36.0
- **THEN** a próxima release é `v5.36.0-fork.3`

#### Scenario: Nova base do upstream
- **WHEN** o fork é sincronizado com a v5.37.0 do upstream
- **THEN** a próxima release é `v5.37.0-fork.1`

### Requirement: Workflow de release
O repositório MUST ter `.github/workflows/release.yml`, executado no push de tags `v*-fork.*`, que, nesta ordem, execute os testes Go, execute `make release-cross` com a versão da tag sem o prefixo `v` e crie a GitHub Release com notas geradas a partir da tag anterior do fork (ou, na primeira release sobre uma base, a partir da tag do upstream dessa base), sem marcar como pré-release, marcada como a mais recente, com os dois pacotes anexados. O workflow MUST NOT publicar imagens de container.

#### Scenario: Release publicada
- **WHEN** a tag `v5.36.0-fork.1` é enviada ao GitHub
- **THEN** existe a GitHub Release `v5.36.0-fork.1` com os pacotes `linux_amd64` e `linux_arm64`

#### Scenario: Testes falham
- **WHEN** os testes falham no workflow de release
- **THEN** nenhuma GitHub Release é criada

#### Scenario: Notas da segunda release
- **WHEN** a tag `v5.36.0-fork.2` é publicada
- **THEN** as notas da release cobrem apenas os commits desde `v5.36.0-fork.1`

### Requirement: Limpeza dos workflows herdados
O fork MUST NOT manter workflows que dependam de segredos ou processos do projeto original: `benchmark.yml`, `labeler.yml`, `publish-custom.yml`, `publish-experimental.yml`, `publish-latest.yml`, `publish-release.yml`, `regenerate-static-assets.yml`, `test.yml` e `test-ui.yml` MUST ser removidos, com os testes Go cobertos por `ci.yml`. O Dependabot MUST atualizar apenas `github-actions`.

#### Scenario: Push para master
- **WHEN** um commit é enviado para `master`
- **THEN** apenas o workflow de CI é executado
- **AND** nenhum login em registry de container é tentado

