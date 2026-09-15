## MODIFIED Requirements

### Requirement: Formulário e editor YAML
As telas de criação e edição MUST oferecer um modo formulário (nome, grupo, URL, método, intervalo, condições, headers, alertas entre os tipos configurados e estado habilitado) e um modo YAML, preservando o conteúdo ao alternar entre eles. O grupo MUST ser escolhido entre os grupos dos endpoints existentes, "sem grupo" ou um grupo novo digitado. Na edição de um endpoint gerenciado, nome e grupo MUST ser editáveis; quando a chave derivada mudar, a tela MUST avisar antes de salvar que a chave muda, que as URLs de badges e da página de detalhes mudam e quais status pages do arquivo de configuração deixam de mostrar o endpoint, e depois de salvar MUST usar a chave nova. Segredos mascarados MUST ser exibidos como `********` e mantidos quando não forem alterados.

#### Scenario: Formulário para YAML
- **WHEN** o administrador preenche o formulário e alterna para o modo YAML
- **THEN** o editor mostra a definição equivalente em YAML

#### Scenario: YAML inválido
- **WHEN** o administrador digita um YAML inválido e tenta alternar para o modo formulário
- **THEN** a tela mostra o erro e permanece no modo YAML com o texto digitado

#### Scenario: Grupo existente ou novo
- **WHEN** o administrador abre o seletor de grupo num formulário de endpoint e existem endpoints no grupo `core`
- **THEN** o seletor oferece `core`, "sem grupo" e a opção de digitar um grupo novo

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
