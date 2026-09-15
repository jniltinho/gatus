# admin-access-control Specification

## Purpose
TBD - created by archiving change add-admin-endpoint-management. Update Purpose after archive.
## Requirements
### Requirement: Seção de configuração admin
O arquivo de configuração MUST aceitar a seção `admin` com `enabled` (booleano, padrão `false`), `allowed-subjects` (lista de subjects OIDC) e `allowed-origins` (lista de origens aceitas em escritas). Sem a seção, ou com `enabled: false`, nenhuma rota de API, tela ou link de administração MUST ficar disponível.

#### Scenario: Admin desabilitado
- **WHEN** a configuração não tem a seção `admin`
- **THEN** `GET /api/v1/admin/endpoints` responde 404
- **AND** o frontend não mostra o link de administração

### Requirement: Pré-requisitos para habilitar a administração
Com `admin.enabled: true`, a validação da configuração MUST falhar quando `security` não tiver `basic` nem `oidc`, quando `storage.type` não for `sqlite` ou `postgres`, ou quando houver `security.oidc` e `admin.allowed-subjects` estiver vazio.

#### Scenario: Admin sem segurança
- **WHEN** a configuração tem `admin.enabled: true` e não tem `security`
- **THEN** a configuração é inválida com uma mensagem informando que `admin` exige `security`

#### Scenario: Admin com storage em memória
- **WHEN** a configuração tem `admin.enabled: true` e `storage.type: memory`
- **THEN** a configuração é inválida com uma mensagem informando que `admin` exige `sqlite` ou `postgres`

#### Scenario: OIDC sem lista de administradores
- **WHEN** a configuração tem `security.oidc`, `admin.enabled: true` e `admin.allowed-subjects` vazio
- **THEN** a configuração é inválida com uma mensagem informando que `admin.allowed-subjects` é obrigatório com OIDC

### Requirement: Autorização de administradores
Toda requisição a `/api/v1/admin/*` MUST exigir autenticação e autorização de administrador. Com `security.oidc`, o subject da sessão (claim `sub`) MUST constar em `admin.allowed-subjects`, comparado sem diferenciar maiúsculas, e credenciais basic MUST ser ignoradas. Apenas com `security.basic`, o usuário do basic auth MUST ser tratado como administrador. O sistema MUST registrar um aviso na inicialização quando um subject de `admin.allowed-subjects` não constar em `security.oidc.allowed-subjects` e essa lista não estiver vazia.

#### Scenario: Sem autenticação
- **WHEN** uma requisição sem credenciais nem sessão chega a `GET /api/v1/admin/endpoints`
- **THEN** a API responde 401

#### Scenario: Subject OIDC fora da lista
- **WHEN** `admin.allowed-subjects` contém `ops@exemplo.com` e um usuário com subject `dev@exemplo.com` acessa a administração
- **THEN** a API responde 403

#### Scenario: Subject OIDC com maiúsculas diferentes
- **WHEN** `admin.allowed-subjects` contém `ops@exemplo.com` e o subject da sessão é `OPS@exemplo.com`
- **THEN** a API responde normalmente

#### Scenario: Usuário basic
- **WHEN** a configuração usa apenas `security.basic` e a requisição traz as credenciais corretas
- **THEN** a API de administração responde normalmente

### Requirement: Proteção contra CSRF
Para requisições `POST`, `PUT` e `DELETE` a `/api/v1/admin/*`, o sistema MUST:
- rejeitar com 403 requisições com `Sec-Fetch-Site: cross-site`;
- rejeitar com 403 requisições cujo `Origin` (ou `Referer`, na falta de `Origin`) não case com `admin.allowed-origins`, quando definido, ou com a origem derivada do header `Host` e do esquema (TLS da conexão ou `X-Forwarded-Proto`), sem considerar `X-Forwarded-Host` nem `X-Forwarded-Port`;
- aceitar requisições sem `Origin` e sem `Referer`;
- rejeitar com 415 requisições com corpo cujo tipo de mídia não seja `application/json`, `application/yaml`, `application/x-yaml` ou `text/yaml`, aceitando parâmetros como `charset`;
- aceitar a origem `http://localhost:8081` quando `ENVIRONMENT=dev`.

#### Scenario: Origem diferente
- **WHEN** um navegador envia `DELETE /api/v1/admin/endpoints/core_api` com `Origin: https://site-malicioso.exemplo`
- **THEN** a API responde 403
- **AND** o endpoint não é removido

#### Scenario: Credenciais basic reenviadas por outro site
- **WHEN** a configuração usa basic auth e chega uma requisição `POST` com credenciais válidas e `Sec-Fetch-Site: cross-site`
- **THEN** a API responde 403

#### Scenario: Formulário HTML
- **WHEN** uma requisição `POST /api/v1/admin/endpoints` chega com `Content-Type: application/x-www-form-urlencoded`
- **THEN** a API responde 415

#### Scenario: JSON com charset
- **WHEN** uma requisição de criação chega com `Content-Type: application/json; charset=utf-8` e origem válida
- **THEN** a requisição é processada

#### Scenario: Atrás de proxy reverso com HTTPS
- **WHEN** o proxy envia `Host: status.exemplo.com` e `X-Forwarded-Proto: https`, e o navegador envia `Origin: https://status.exemplo.com`
- **THEN** a requisição é aceita

#### Scenario: X-Forwarded-Host ignorado
- **WHEN** chega uma requisição com `Host: status.exemplo.com`, `X-Forwarded-Host: site-malicioso.exemplo` e `Origin: https://site-malicioso.exemplo`
- **THEN** a API responde 403

#### Scenario: Porta não padrão com origem configurada
- **WHEN** `admin.allowed-origins` contém `https://status.exemplo.com:8443` e o navegador envia essa origem
- **THEN** a requisição é aceita

#### Scenario: Cliente de linha de comando
- **WHEN** uma requisição autenticada chega sem `Origin`, sem `Referer` e com `Content-Type: application/json`
- **THEN** a requisição é processada

### Requirement: Auditoria de alterações
Toda criação, alteração, remoção, habilitação e desabilitação de endpoint gerenciado MUST ser registrada no log com a operação, a chave do endpoint e o autor (usuário basic ou subject OIDC), e MUST NOT incluir valores de headers, corpo, URL com credenciais, segredos de client ou SSH, ou `provider-override`.

#### Scenario: Remoção auditada
- **WHEN** o administrador `ops@exemplo.com` remove o endpoint `core_api`
- **THEN** o log contém uma linha com a operação de remoção, `core_api` e `ops@exemplo.com`
- **AND** a linha não contém valores de headers do endpoint

### Requirement: Estado de administração exposto ao frontend
`GET /api/v1/config` MUST incluir o objeto `admin` com `enabled` e `authorized`, onde `authorized` indica se a requisição atual pode usar a administração. Com apenas `security.basic`, `authorized` MUST ser `true` sempre que `enabled` for verdadeiro, pois o único usuário basic é administrador; a autenticação fica a cargo da API.

#### Scenario: Usuário OIDC sem permissão
- **WHEN** um usuário com subject fora de `admin.allowed-subjects` consulta `GET /api/v1/config`
- **THEN** a resposta contém `"admin": {"enabled": true, "authorized": false}`

#### Scenario: Apenas basic auth
- **WHEN** a configuração usa apenas `security.basic`, com `admin.enabled: true`, e `GET /api/v1/config` é consultado sem credenciais
- **THEN** a resposta contém `"admin": {"enabled": true, "authorized": true}`

