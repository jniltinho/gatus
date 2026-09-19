## MODIFIED Requirements

### Requirement: Endpoints em destaque
Cada página MUST aceitar `featured`, uma lista de até 10 chaves de endpoints, no YAML e na administração. Os endpoints em destaque MUST contar como seleção da página (uma página só com `featured` é válida), MUST aparecer no payload em `featured`, na ordem da lista, com o nome real do grupo, e MUST NOT aparecer de novo nas seções de grupo. Uma chave sem endpoint publicável MUST ser ignorada na página e registrada como aviso na carga e na validação da administração. Os destaques MUST contar no limite de endpoints da página (`status-pages.maximum-endpoints-per-page`, `200` por padrão), antes das seções.

#### Scenario: Endpoint em destaque fora da seção do grupo
- **WHEN** a página tem `groups: [core]` e `featured: [core_api]`, e o grupo `core` tem `api` e `web`
- **THEN** `featured` do payload tem só `api`, com `group` igual a `core`
- **AND** a seção `core` tem só `web`

#### Scenario: Página só com destaques
- **WHEN** a página não tem `groups` nem `endpoints` e tem `featured: [core_api]`
- **THEN** a página é válida e mostra `api` em destaque

#### Scenario: Limite de destaques
- **WHEN** a página tem 11 chaves em `featured`
- **THEN** a validação falha com erro de definição inválida (400 na administração)

#### Scenario: Destaques além do limite
- **WHEN** `maximum-endpoints-per-page` é `2` e a página tem três destaques distintos e publicáveis
- **THEN** o payload traz os dois primeiros destaques, na ordem da lista, nenhuma seção, e `truncated: true`
