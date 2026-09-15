# admin-web-ui Specification

## Purpose
TBD - created by archiving change add-admin-endpoint-management. Update Purpose after archive.
## Requirements
### Requirement: Acesso às telas de administração
Com a administração habilitada e o usuário autorizado, o frontend MUST mostrar no cabeçalho um link "Admin" para `/admin`. As rotas `/admin`, `/admin/endpoints/new` e `/admin/endpoints/{key}/edit` MUST abrir diretamente pela URL, inclusive ao recarregar a página, com a chave codificada na URL.

#### Scenario: Link visível para administrador
- **WHEN** `GET /api/v1/config` indica `admin.enabled` e `admin.authorized` verdadeiros
- **THEN** o cabeçalho mostra o link "Admin"

#### Scenario: Link oculto sem permissão
- **WHEN** a configuração usa OIDC e `GET /api/v1/config` indica `admin.authorized` falso
- **THEN** o cabeçalho não mostra o link "Admin"

#### Scenario: Abertura direta da edição
- **WHEN** o navegador abre `/admin/endpoints/core_api/edit` diretamente
- **THEN** o frontend carrega a tela de edição de `core_api`

### Requirement: Lista de endpoints
A tela `/admin` MUST listar os endpoints com nome, grupo, tipo, URL (com credenciais mascaradas), intervalo, estado habilitado e origem (Web ou YAML), MUST permitir buscar por nome, grupo ou URL, MUST destacar endpoints em conflito ou com erro de validação e MUST permitir habilitar e desabilitar endpoints de origem Web. Endpoints de origem YAML MUST ser apenas visualizáveis.

#### Scenario: Busca
- **WHEN** o administrador digita `core` na busca
- **THEN** a lista mostra apenas endpoints cujo nome, grupo ou URL contém `core`

#### Scenario: Endpoint do YAML
- **WHEN** o administrador abre um endpoint de origem YAML
- **THEN** a tela mostra a definição em YAML com segredos mascarados, sem ações de salvar, habilitar, desabilitar ou remover

#### Scenario: Endpoint em conflito
- **WHEN** existe um endpoint gerenciado marcado como em conflito
- **THEN** a lista mostra esse endpoint com um aviso de conflito com o YAML

### Requirement: Formulário e editor YAML
As telas de criação e edição MUST oferecer um modo formulário (nome, grupo, URL, método, intervalo, condições, headers, alertas entre os tipos configurados e estado habilitado) e um modo YAML, preservando o conteúdo ao alternar entre eles. Na edição, nome e grupo MUST ser somente leitura, e segredos mascarados MUST ser exibidos como `********` e mantidos quando não forem alterados.

#### Scenario: Formulário para YAML
- **WHEN** o administrador preenche o formulário e alterna para o modo YAML
- **THEN** o editor mostra a definição equivalente em YAML

#### Scenario: YAML inválido
- **WHEN** o administrador digita um YAML inválido e tenta alternar para o modo formulário
- **THEN** a tela mostra o erro e permanece no modo YAML com o texto digitado

#### Scenario: Nome bloqueado na edição
- **WHEN** o administrador abre a edição de um endpoint gerenciado
- **THEN** os campos nome e grupo aparecem desabilitados

#### Scenario: Edição concorrente
- **WHEN** o administrador salva e a API responde 412
- **THEN** a tela informa que o endpoint foi alterado por outra pessoa e oferece recarregar a versão atual, sem descartar o conteúdo digitado

### Requirement: Validar e testar antes de salvar
As telas de criação e edição MUST ter as ações Validar, Testar e Salvar. Testar MUST exibir o resultado de cada condição e a duração. Erros devolvidos pela API MUST ser exibidos junto ao formulário sem perder o conteúdo digitado.

#### Scenario: Erro ao salvar
- **WHEN** o administrador salva e a API responde 400
- **THEN** a mensagem de erro aparece na tela
- **AND** o conteúdo digitado continua no formulário

#### Scenario: Resultado do teste
- **WHEN** o administrador clica em Testar
- **THEN** a tela lista cada condição com indicação de atendida ou não e mostra a duração

### Requirement: Confirmação de remoção
A remoção de um endpoint gerenciado MUST exigir confirmação explícita que informe que o histórico será apagado e, quando houver alertas disparados, que os provedores de alerta não serão notificados.

#### Scenario: Remoção cancelada
- **WHEN** o administrador clica em remover e cancela a confirmação
- **THEN** nenhuma requisição de remoção é enviada

### Requirement: Padrões do frontend
As telas de administração MUST seguir as convenções do projeto: Vue 3 com `<script setup>`, Tailwind com variantes `dark:` em todos os componentes novos, dados passados por props (sem provide/inject) e build incluído em `web/static/`.

#### Scenario: Tema escuro
- **WHEN** o usuário usa o tema escuro
- **THEN** as telas de administração são exibidas com as cores do tema escuro

### Requirement: Testes ponta a ponta com agent-browser
O repositório MUST ter um roteiro de testes ponta a ponta em `test/e2e/` que suba o Gatus local com SQLite temporário, `security.basic` e `admin.enabled`, e use o `agent-browser` para percorrer lista, criação, validação, teste, salvamento, edição, desabilitação, remoção e acesso sem credenciais (401), nos temas claro e escuro, salvando capturas de tela em `dist/prints/`. O diretório `dist/` MUST ser ignorado pelo git.

#### Scenario: Execução do roteiro
- **WHEN** alguém executa o roteiro de testes ponta a ponta
- **THEN** o roteiro termina com sucesso e grava as capturas de tela em `dist/prints/`
- **AND** `git status` não mostra arquivos novos em `dist/`

