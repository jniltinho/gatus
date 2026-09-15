## MODIFIED Requirements

### Requirement: Lista de endpoints
A tela `/admin` MUST listar os endpoints ativos e Push, do arquivo de configuração (inclusive external endpoints) e gerenciados pela web. Cada linha MUST mostrar nome, grupo, tipo (`PUSH` para endpoints Push), URL (com credenciais mascaradas e vazia para Push), intervalo (de heartbeat para Push), estado habilitado e origem (Web ou YAML). A tela MUST permitir buscar por nome, grupo ou URL, MUST destacar endpoints em conflito ou com erro de validação e MUST permitir habilitar e desabilitar endpoints de origem Web. Endpoints de origem YAML MUST ser apenas visualizáveis.

#### Scenario: Busca
- **WHEN** o administrador digita `core` na busca
- **THEN** a lista mostra apenas endpoints cujo nome, grupo ou URL contém `core`

#### Scenario: Endpoint do YAML
- **WHEN** o administrador abre um endpoint de origem YAML
- **THEN** a tela mostra a definição em YAML com segredos mascarados, sem ações de salvar, habilitar, desabilitar ou remover

#### Scenario: External endpoint do YAML
- **WHEN** o arquivo de configuração define o external endpoint `jobs_backup`
- **THEN** a lista mostra `jobs_backup` com tipo `PUSH` e origem YAML, somente para visualização

#### Scenario: Endpoint em conflito
- **WHEN** existe um endpoint gerenciado marcado como em conflito
- **THEN** a lista mostra esse endpoint com um aviso de conflito com o YAML

### Requirement: Formulário e editor YAML
As telas de criação e edição MUST oferecer um modo formulário e um modo YAML, preservando o conteúdo ao alternar entre eles.

O formulário MUST começar pelo tipo de monitor:
- **ativos** (HTTP(s), TCP, Ping, DNS e os demais inferidos pela URL): nome, grupo, URL, método, intervalo, condições, headers, alertas entre os tipos configurados, estado habilitado e a opção "Accept push" (receber push), desligada por padrão, que ao ser ligada mostra o token opcional, a URL de push copiável e o exemplo com a chave global;
- **Push (passivo):** nome, grupo, token, intervalo de heartbeat, alertas e estado habilitado.

No tipo Push, a tela MUST mostrar:
- a URL de push copiável no formato do Uptime Kuma (`<endereço do Gatus>/api/push/<token>?status=up&msg=OK&ping=`);
- a explicação de que o envio deve ocorrer a cada intervalo de heartbeat e aceita `status`, `msg` e `ping`;
- um exemplo de `curl`;
- a ação de gerar um token novo;
- um campo para informar um token existente.

O grupo MUST ser escolhido entre os grupos dos endpoints existentes, "sem grupo" ou um grupo novo digitado. Na edição de um endpoint gerenciado, nome e grupo MUST ser editáveis; o tipo MUST ser somente leitura. Quando a chave derivada mudar, a tela MUST avisar antes de salvar:
- que a chave muda;
- que as URLs de badges e da página de detalhes mudam;
- nos endpoints que recebem push, que as URLs com chave global mudam, e que a URL com o token do endpoint não muda;
- quais status pages do arquivo de configuração deixam de mostrar o endpoint.

Depois de salvar, a tela MUST usar a chave nova. Segredos mascarados MUST ser exibidos como `********` e mantidos quando não forem alterados.

#### Scenario: Formulário para YAML
- **WHEN** o administrador preenche o formulário e alterna para o modo YAML
- **THEN** o editor mostra a definição equivalente em YAML

#### Scenario: YAML inválido
- **WHEN** o administrador digita um YAML inválido e tenta alternar para o modo formulário
- **THEN** a tela mostra o erro e permanece no modo YAML com o texto digitado

#### Scenario: Grupo existente ou novo
- **WHEN** o administrador abre o seletor de grupo num formulário de endpoint e existem endpoints no grupo `core`
- **THEN** o seletor oferece `core`, "sem grupo" e a opção de digitar um grupo novo

#### Scenario: Tipo Push
- **WHEN** o administrador escolhe o tipo Push num endpoint novo
- **THEN** o formulário esconde URL, método, condições e headers e mostra a URL de push copiável com um token gerado, o intervalo de heartbeat de 60 segundos e um exemplo de `curl`

#### Scenario: Push num endpoint ativo
- **WHEN** o administrador liga "Accept push" num endpoint HTTP
- **THEN** o formulário mostra um token gerado, a URL de push copiável e o exemplo com a chave global, sem esconder URL, condições e headers

#### Scenario: Push desligado no endpoint ativo
- **WHEN** o administrador desliga "Accept push" num endpoint HTTP e salva
- **THEN** a definição fica sem `push` e o endpoint deixa de aceitar envios

#### Scenario: Token do Uptime Kuma
- **WHEN** o administrador cola o token de um monitor Push do Uptime Kuma no campo de token
- **THEN** a URL de push mostrada passa a usar esse token

#### Scenario: Troca de grupo na edição
- **WHEN** o administrador troca o grupo de `web_site` para `clientes` na edição
- **THEN** antes de salvar, a tela avisa que a chave muda de `web_site` para `clientes_site` e que as URLs de badges e da página de detalhes mudam
- **AND** depois de salvar, a lista mostra `clientes_site` com o histórico anterior

#### Scenario: Status page do arquivo afetada
- **WHEN** o administrador troca o nome de um endpoint selecionado pela chave na status page `services` do arquivo de configuração
- **THEN** antes de salvar, a tela avisa que `services` deixa de mostrar o endpoint até o arquivo ser corrigido

#### Scenario: Edição concorrente
- **WHEN** o administrador salva e a API responde 412
- **THEN** a tela informa que o endpoint foi alterado por outra pessoa e oferece recarregar a versão atual, sem descartar o conteúdo digitado

### Requirement: Validar e testar antes de salvar
As telas de criação e edição MUST ter as ações Validar e Salvar e, nos tipos ativos, a ação Testar. Testar MUST exibir o resultado de cada condição e a duração. No tipo Push, a ação Testar MUST ser substituída pela URL de push e pelo exemplo de `curl`. Erros devolvidos pela API MUST ser exibidos junto ao formulário sem perder o conteúdo digitado.

#### Scenario: Erro ao salvar
- **WHEN** o administrador salva e a API responde 400
- **THEN** a mensagem de erro aparece na tela
- **AND** o conteúdo digitado continua no formulário

#### Scenario: Resultado do teste
- **WHEN** o administrador clica em Testar
- **THEN** a tela lista cada condição com indicação de atendida ou não e mostra a duração

#### Scenario: Endpoint Push sem Testar
- **WHEN** o administrador edita um endpoint Push
- **THEN** a tela não mostra a ação Testar e mostra a URL de push com o exemplo de `curl`

## ADDED Requirements

### Requirement: Tela de chaves de push
A administração MUST ter a aba "Push keys" em `/admin/push-keys`. A aba MUST listar as chaves globais com nome, dica (4 últimos caracteres), origem (Web ou YAML), autor e data de criação.

Ela MUST permitir criar uma chave global informando o nome. Depois de criada, a aba MUST mostrar a chave completa uma única vez, com um exemplo de URL `/api/push/<chave>/<chave-do-endpoint>?status=up&msg=OK&ping=` e o aviso de que ela não será mostrada de novo. Revogar uma chave de origem Web MUST exigir confirmação. As chaves de origem YAML MUST ser apenas visualizáveis.

#### Scenario: Chave criada
- **WHEN** o administrador cria a chave global `akamai`
- **THEN** a tela mostra a chave completa com a URL de exemplo e o aviso de exibição única
- **AND** ao recarregar a aba, a lista mostra somente a dica da chave

#### Scenario: Revogação cancelada
- **WHEN** o administrador clica em revogar uma chave e cancela a confirmação
- **THEN** nenhuma requisição de revogação é enviada
