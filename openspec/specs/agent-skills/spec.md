# agent-skills Specification

## Purpose
TBD - created by archiving change add-admin-endpoint-management. Update Purpose after archive.
## Requirements
### Requirement: Skills de Go
O repositório MUST incluir em `.claude/skills/` as skills de Go do repositório `jniltinho/llama-model` aplicáveis ao Gatus: `golang-how-to`, `golang-code-style`, `golang-naming`, `golang-error-handling`, `golang-concurrency`, `golang-structs-interfaces`, `golang-database`, `golang-testing`, `golang-security`, `golang-lint`, `golang-modernize`, `golang-dependency-management`, `golang-documentation`, `golang-observability` e `golang-popular-libraries`, com conteúdo idêntico ao de origem, acompanhadas do texto da licença MIT de `samber/cc-skills-golang`.

#### Scenario: Skills presentes com licença
- **WHEN** alguém lista `.claude/skills/`
- **THEN** cada uma das 15 skills tem um `SKILL.md` idêntico ao do `jniltinho/llama-model`
- **AND** o texto da licença MIT de `samber/cc-skills-golang` está presente

### Requirement: Skills fora do escopo do Gatus
Skills voltadas a bibliotecas ou tipos de projeto que o Gatus não usa (`golang-cli`, `golang-spf13-cobra`, `golang-spf13-viper`, `golang-samber-lo`, `golang-samber-slog` e `golang-swagger`) MUST NOT ser incluídas.

#### Scenario: Skill de CLI ausente
- **WHEN** alguém lista `.claude/skills/`
- **THEN** não existe o diretório `golang-spf13-cobra`

### Requirement: Skill agent-browser
O repositório MUST incluir `.claude/skills/agent-browser/SKILL.md` idêntico ao do repositório `jniltinho/go-postfixadmin`, acompanhado do aviso de licença Apache-2.0 de `vercel-labs/agent-browser`.

#### Scenario: Skill de testes pelo navegador
- **WHEN** alguém lista `.claude/skills/agent-browser/`
- **THEN** existem o `SKILL.md` com o mesmo conteúdo do `jniltinho/go-postfixadmin` e o aviso de licença

### Requirement: Skill create-release
O repositório MUST incluir `.claude/skills/create-release/SKILL.md` com a mesma estrutura da skill do `jniltinho/llama-model` (pré-checagens, esquema de versão, revisão e categorização de commits, criação e envio da tag, acompanhamento do workflow, ajuste das notas e verificação), adaptada ao Gatus: branch `master`, tags `v<base>-fork.<N>` com a última tag do fork encontrada ignorando as tags do upstream, pacotes `linux/amd64` e `linux/arm64` pelo workflow, imagem publicada com `make docker-release`, notas a partir da tag anterior do fork e link de changelog para `jniltinho/gatus`.

#### Scenario: Próxima versão ignora tags do upstream
- **WHEN** a última tag do fork é `v5.36.0-fork.1`, o repositório também contém a tag `v5.36.0` do upstream, e um agente segue a skill para uma nova release sem sincronizar o upstream
- **THEN** a versão proposta é `v5.36.0-fork.2`

### Requirement: Regras do fork para agentes
O repositório MUST ter `AGENTS.fork.md` com: as skills de Go obrigatórias; o uso da skill `create-release`; os alvos `build`, `fmt`, `vet`, `lint` e `release-cross`; as regras da administração de endpoints (endpoints gerenciados no storage, registro do watchdog, labels congeladas, validação estrita); a execução dos testes ponta a ponta com `agent-browser` e capturas em `dist/prints/` fora do git; a instrução de usar `go mod tidy` sem gerar `vendor/`; e o procedimento de sincronização com o upstream. O `AGENTS.md` MUST conter uma referência a `AGENTS.fork.md`, com as demais alterações nele evitadas.

#### Scenario: Nova dependência Go
- **WHEN** um agente segue as regras do repositório para adicionar uma dependência Go
- **THEN** a instrução é executar `go mod tidy`, sem gerar `vendor/`

#### Scenario: Conflito em arquivos estáticos ao sincronizar
- **WHEN** a sincronização com o upstream gera conflito em `web/static/`
- **THEN** o procedimento orienta resolver regenerando os arquivos com `make frontend-build`

