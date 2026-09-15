## MODIFIED Requirements

### Requirement: Auditoria de alterações
Toda criação, alteração, renomeação, remoção, habilitação e desabilitação de endpoint gerenciado MUST ser registrada no log com a operação, a chave do endpoint e o autor (usuário basic ou subject OIDC); na renomeação, a linha MUST conter a chave antiga e a nova. O log MUST NOT incluir valores de headers, corpo, URL com credenciais, segredos de client ou SSH, ou `provider-override`.

#### Scenario: Remoção auditada
- **WHEN** o administrador `ops@exemplo.com` remove o endpoint `core_api`
- **THEN** o log contém uma linha com a operação de remoção, `core_api` e `ops@exemplo.com`
- **AND** a linha não contém valores de headers do endpoint

#### Scenario: Renomeação auditada
- **WHEN** o administrador `ops@exemplo.com` troca o grupo de `web_site` para `clientes`
- **THEN** o log contém uma linha com a operação de renomeação, `web_site`, `clientes_site` e `ops@exemplo.com`
