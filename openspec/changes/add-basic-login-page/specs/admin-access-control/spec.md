## MODIFIED Requirements

### Requirement: Autorização de administradores
Toda requisição a `/api/v1/admin/*` MUST exigir autenticação e autorização de administrador. Com `security.oidc`, o subject da sessão (claim `sub`) MUST constar em `admin.allowed-subjects`, comparado sem diferenciar maiúsculas, e credenciais basic MUST ser ignoradas. Apenas com `security.basic`, o usuário do basic auth MUST ser tratado como administrador, autenticado pela sessão da tela de login ou pelo header `Authorization: Basic`, e a autoria das escritas MUST ser esse usuário. O sistema MUST registrar um aviso na inicialização quando um subject de `admin.allowed-subjects` não constar em `security.oidc.allowed-subjects` e essa lista não estiver vazia.

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

#### Scenario: Usuário basic com sessão da tela de login
- **WHEN** a configuração usa apenas `security.basic` e a requisição traz o cookie de uma sessão válida, sem `Authorization`
- **THEN** a API de administração responde normalmente
- **AND** a auditoria registra o usuário da sessão como autor

### Requirement: Estado de administração exposto ao frontend
`GET /api/v1/config` MUST incluir o objeto `admin` com `enabled` e `authorized`, onde `authorized` indica se a requisição atual pode usar a administração. Com apenas `security.basic`, `authorized` MUST ser `true` somente quando `enabled` for verdadeiro e a requisição estiver autenticada pela sessão da tela de login ou por `Authorization: Basic` correto, pois o único usuário basic é administrador.

#### Scenario: Usuário OIDC sem permissão
- **WHEN** um usuário com subject fora de `admin.allowed-subjects` consulta `GET /api/v1/config`
- **THEN** a resposta contém `"admin": {"enabled": true, "authorized": false}`

#### Scenario: Apenas basic auth sem autenticação
- **WHEN** a configuração usa apenas `security.basic`, com `admin.enabled: true`, e `GET /api/v1/config` é consultado sem credenciais nem sessão
- **THEN** a resposta contém `"admin": {"enabled": true, "authorized": false}`

#### Scenario: Apenas basic auth com sessão
- **WHEN** a configuração usa apenas `security.basic`, com `admin.enabled: true`, e `GET /api/v1/config` é consultado com o cookie de uma sessão válida
- **THEN** a resposta contém `"admin": {"enabled": true, "authorized": true}`
