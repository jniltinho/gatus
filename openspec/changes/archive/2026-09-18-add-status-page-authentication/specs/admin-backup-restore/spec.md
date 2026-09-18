## MODIFIED Requirements

### Requirement: Prévia do restore
`POST /api/v1/admin/restore/preview` MUST aceitar só `application/json` com corpo de até 3,5 MiB, no formato `{"file", "password", "overwrite", "disableEndpoints"}`. O limite de 256 KB das outras rotas da administração MUST NOT valer para as rotas de restore.

**Resposta e efeitos:**
- MUST devolver o plano (resumo, avisos, fingerprint e itens com ação, motivo e avisos);
- MUST NOT alterar o store, o monitoramento nem as publicações;
- MUST responder 503 quando um dos registros estiver indisponível.

**Simulação:** o plano MUST prever a aplicação simulando o estado acumulado, na ordem chaves de push, endpoints e status pages.

**Regras dos endpoints e das status pages:**
- `create` para item inexistente na web;
- item existente com definição normalizada igual é `unchanged`; diferente é `update` com `overwrite` e `skip` ("already exists") sem ele;
- `skip` com motivo para:
  - chave ou slug usado pelo arquivo de configuração;
  - definição recusada pelas validações da web, considerando os tokens e as chaves planejados;
  - `key` diferente da chave calculada;
  - troca de tipo de endpoint;
  - segredo mascarado em qualquer lugar mascarado pela API (headers, senha ou query da URL, segredos de client ou SSH, tokens e folhas de `provider-override`), **exceto** o hash da credencial de uma status page que já existe no destino: nesse caso o hash do destino MUST ser mantido e o item MUST seguir como `update` ou `unchanged` pelo restante da definição. A mesclagem MUST acontecer **antes** da validação da definição, senão o valor mascarado é recusado como hash inválido. Uma status page nova com o hash mascarado MUST ser `skip`, porque não há credencial para manter;
  - endpoint Push sem token;
- nome ou grupo diferentes com a mesma chave MUST ser `update` com o motivo "name or group changes";
- com `disableEndpoints`, os endpoints criados ou atualizados MUST ficar desabilitados.

**Regras das chaves de push:**
- mesmo nome e mesmo hash de uma chave existente é `unchanged`;
- MUST ser `skip`:
  - formato inválido;
  - nome em uso por uma chave do YAML ou por outra chave;
  - hash em uso por outra chave;
  - hash igual ao token de qualquer endpoint: do arquivo de configuração, de qualquer endpoint gerenciado (inclusive em conflito ou inválido) ou de qualquer endpoint do backup.

**Avisos:**
- a prévia MUST informar quantos endpoints habilitados começarão a ser monitorados e quantos têm alertas;
- as status pages MUST trazer os avisos de seleção calculados com os endpoints existentes e os planejados.

#### Scenario: Prévia sem efeitos
- **WHEN** o administrador pede a prévia de um arquivo com um endpoint novo
- **THEN** a resposta mostra o endpoint como `create` e ele não é criado nem monitorado

#### Scenario: Existente sem sobrescrever
- **WHEN** o arquivo tem `web_site` com outro intervalo e `overwrite` é falso
- **THEN** o item aparece como `skip` com "already exists"

#### Scenario: Página em conflito com o YAML
- **WHEN** a página gerenciada `services` existe e o arquivo de configuração passou a definir `services`
- **THEN** o item do arquivo aparece como `skip` com o motivo do conflito

#### Scenario: Token repetido dentro do arquivo
- **WHEN** o arquivo tem dois endpoints Push com o mesmo token
- **THEN** o primeiro aparece como `create` e o segundo como `skip` com o motivo do token repetido

#### Scenario: Chave de push igual a token de endpoint
- **WHEN** o hash de uma chave do arquivo é o SHA-256 do token do endpoint `jobs_backup`
- **THEN** a chave aparece como `skip` com "token hash in use"

#### Scenario: Segredo mascarado
- **WHEN** um endpoint do arquivo tem `headers.Authorization: "********"`, a URL `https://user:********@api.example.org` ou `provider-override.webhook-url: "********"`
- **THEN** o item aparece como `skip` com "masked secret"

#### Scenario: Chave com token de endpoint em conflito
- **WHEN** o hash de uma chave do arquivo é o SHA-256 do token de um endpoint gerenciado em conflito com o arquivo de configuração
- **THEN** a chave aparece como `skip` com "token hash in use"

#### Scenario: Restaurar desabilitado
- **WHEN** a prévia e a aplicação usam `disableEndpoints` com um endpoint novo
- **THEN** a prévia mostra a definição com `enabled: false` e o endpoint é criado sem começar a ser monitorado

#### Scenario: Troca de tipo
- **WHEN** o arquivo tem `jobs_backup` como endpoint ativo e ele existe como Push, com `overwrite`
- **THEN** o item aparece como `skip` com "type cannot change"

#### Scenario: Chave de push repetida
- **WHEN** o arquivo tem a chave `akamai` com o mesmo hash da chave `akamai` existente
- **THEN** o item aparece como `unchanged`

#### Scenario: Credencial mascarada de uma página existente
- **WHEN** o arquivo restaurado traz a página `clientes`, que já existe no destino exigindo login, com o hash mascarado
- **THEN** a prévia mostra `update` ou `unchanged` pelo restante da definição
- **AND** a credencial do destino é mantida

#### Scenario: Credencial mascarada de uma página nova
- **WHEN** o mesmo arquivo traz a página `parceiros`, que não existe no destino, com o hash mascarado
- **THEN** a prévia mostra `skip` com o motivo do segredo mascarado

