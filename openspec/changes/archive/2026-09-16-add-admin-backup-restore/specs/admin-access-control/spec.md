## MODIFIED Requirements

### Requirement: Proteção contra CSRF
Para requisições `POST`, `PUT` e `DELETE` a `/api/v1/admin/*`, o sistema MUST:
- rejeitar com 403 requisições com `Sec-Fetch-Site: cross-site`;
- rejeitar com 403 requisições cujo `Origin` (ou `Referer`, na falta de `Origin`) não case com `admin.allowed-origins`, quando definido, ou com a origem derivada do header `Host` e do esquema (TLS da conexão ou `X-Forwarded-Proto`), sem considerar `X-Forwarded-Host` nem `X-Forwarded-Port`;
- aceitar requisições sem `Origin` e sem `Referer`;
- rejeitar com 415 requisições com corpo cujo tipo de mídia não seja `application/json`, `application/yaml`, `application/x-yaml` ou `text/yaml`, aceitando parâmetros como `charset`;
- aceitar a origem `http://localhost:8081` quando `ENVIRONMENT=dev`.
- nas rotas de backup e restore, exigir `Content-Type: application/json`, inclusive com corpo vazio, respondendo 415 aos demais tipos.

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

#### Scenario: Restore em YAML
- **WHEN** uma requisição `POST /api/v1/admin/restore/preview` chega com `Content-Type: application/yaml`
- **THEN** a API responde 415
