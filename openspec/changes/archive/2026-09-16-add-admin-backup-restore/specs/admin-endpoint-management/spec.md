## MODIFIED Requirements

### Requirement: API de administração de endpoints
O sistema MUST expor as operações abaixo, respondendo em JSON com erros no formato `{"error": "<mensagem>"}` (exceto o 401 do middleware de autenticação, que mantém o formato atual):
- `GET /api/v1/admin/endpoints`: lista endpoints gerenciados e do YAML, com origem (`admin` ou `config`), chave, nome, grupo, tipo, URL com credenciais mascaradas, intervalo, estado habilitado, conflito, erro de validação, versão e metadados de alteração;
- `GET /api/v1/admin/endpoints/{key}`: definição armazenada e efetiva, em YAML e JSON, com `ETag` da versão;
- `POST /api/v1/admin/endpoints`: cria;
- `PUT /api/v1/admin/endpoints/{key}`: altera;
- `POST /api/v1/admin/endpoints/{key}/enable` e `POST /api/v1/admin/endpoints/{key}/disable`: habilitam e desabilitam;
- `DELETE /api/v1/admin/endpoints/{key}`: remove;
- `POST /api/v1/admin/endpoints/parse`: decodifica a definição e a devolve como documento, sem validar, sem ler dados armazenados e sem mascarar;
- `POST /api/v1/admin/endpoints/validate`: valida sem persistir;
- `POST /api/v1/admin/endpoints/test`: valida e executa uma verificação única;
- `GET /api/v1/admin/metadata`: tipos de alerta configurados, túneis disponíveis e labels Prometheus permitidas.

Operações que alteram um endpoint existente MUST exigir `If-Match` com a versão atual, respondendo 428 sem o header e 412 com versão diferente. Corpos acima de 256 KB MUST ser rejeitados com 413, exceto nas rotas de restore da capability `admin-backup-restore`, que aceitam até 3,5 MiB. Enquanto um ciclo de partida ou recarga estiver em andamento (incluindo a partida inicial dos endpoints), as escritas MUST responder 503 sem validar nem gravar nada.

#### Scenario: Listagem indica a origem
- **WHEN** o YAML define 2 endpoints e existe 1 endpoint gerenciado
- **THEN** a listagem devolve 3 itens, 2 com origem `config` e 1 com origem `admin`

#### Scenario: Chave inexistente
- **WHEN** um administrador consulta `GET /api/v1/admin/endpoints/nao_existe`
- **THEN** a API responde 404

#### Scenario: Validação não persiste
- **WHEN** um administrador envia uma definição válida para `validate`
- **THEN** a API responde 200 com as definições armazenável e efetiva
- **AND** o endpoint não é persistido nem monitorado

#### Scenario: Edição concorrente
- **WHEN** dois administradores leem `core_api` na versão 2 e o primeiro salva uma alteração
- **THEN** o `PUT` do segundo, com `If-Match` da versão 2, recebe 412
- **AND** a alteração do primeiro é mantida

#### Scenario: Escrita durante recarga
- **WHEN** um administrador envia uma criação enquanto o hot-reload está entre parar e iniciar o Gatus
- **THEN** a API responde 503 e nada é persistido
