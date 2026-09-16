## MODIFIED Requirements

### Requirement: Tokens e chaves de push
O token de um endpoint Push, e o token opcional de um endpoint ativo com push ligado, MUST ter de 8 a 128 caracteres entre letras, dígitos, `-` e `_`. Ele MUST poder ser informado, para reaproveitar o token de um monitor do Uptime Kuma, ou gerado com 32 letras e dígitos aleatórios, com gerador criptográfico. Um token não pode identificar mais de um endpoint: a administração MUST rejeitar com 409 um token já usado por outro endpoint, do arquivo ou da web. No arquivo de configuração, `push.endpoints` MUST ligar o push de endpoints ativos pela chave, com token opcional, e uma chave que não seja de endpoint ativo do arquivo MUST ser rejeitada na carga. Quando o arquivo de configuração tiver o mesmo token em mais de um external endpoint, o sistema MUST registrar um aviso na carga, e esse token MUST responder 404 em `/api/push/{token}`.

As chaves globais criadas pela administração MUST ter um nome único de 1 a 64 caracteres e MUST ser geradas com 32 letras e dígitos aleatórios, mostradas uma única vez e armazenadas somente como hash SHA-256 com os 4 últimos caracteres como dica. Uma chave restaurada de um backup MUST manter o hash e a dica do arquivo, sem token conhecido pelo sistema, e MUST NOT ter o hash de uma chave existente nem do token de qualquer endpoint. Elas MUST poder ser revogadas, e uma chave revogada MUST parar de autorizar envios imediatamente. As chaves globais do arquivo de configuração MUST ficar em `push.keys`, com `name` e `token` com pelo menos 16 caracteres, e MUST ser somente leitura na administração.

#### Scenario: Token do Uptime Kuma reaproveitado
- **WHEN** um administrador cria um endpoint Push informando o token `keSDu7G855jvVat1xWiY2Gk4CkL1End5`
- **THEN** o endpoint é criado e `/api/push/keSDu7G855jvVat1xWiY2Gk4CkL1End5` passa a registrar os envios nesse endpoint

#### Scenario: Token repetido
- **WHEN** um administrador cria um endpoint Push com o token de um external endpoint do arquivo de configuração
- **THEN** a API responde 409 e o endpoint não é criado

#### Scenario: Chave global criada e revogada
- **WHEN** um administrador cria uma chave global
- **THEN** a resposta mostra a chave completa uma única vez e a lista mostra somente a dica
- **AND** depois que a chave é revogada, os envios com ela respondem 404

#### Scenario: Chave global do arquivo
- **WHEN** o arquivo de configuração define `push.keys` com `name: akamai` e um token de 32 caracteres
- **THEN** a chave autoriza os endpoints que recebem push e aparece na administração sem ações de revogar

#### Scenario: Chave global restaurada
- **WHEN** um backup com a chave `akamai` é restaurado em outra instalação
- **THEN** os envios com o token original de `akamai` são aceitos, e a lista mostra somente a dica
