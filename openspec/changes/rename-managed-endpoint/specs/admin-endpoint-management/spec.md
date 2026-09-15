## ADDED Requirements

### Requirement: Renomeação de endpoint gerenciado
A alteração de um endpoint gerenciado MUST aceitar `name` e `group` diferentes dos armazenados.

Quando a chave derivada (`grupo_nome`) mudar, o sistema MUST, numa única transação, gravar a definição sob a chave nova e mover para ela o status, os resultados, os eventos, o uptime e os alertas disparados da chave antiga. Se a transação falhar, nada MUST mudar e o monitoramento anterior MUST continuar. Quando só o texto de `name` ou `group` mudar sem mudar a chave, o nome e o grupo exibidos MUST ser atualizados, mantendo o histórico.

A chave nova MUST ser rejeitada com 409, sem alterar nada, quando for usada por endpoint, endpoint externo, suite ou endpoint de suite do arquivo de configuração, por outro endpoint gerenciado, ou quando já tiver status armazenado. Com `storage.type: mysql`, uma chave nova acima de 768 caracteres MUST ser rejeitada com 400.

Depois da resposta, o sistema MUST NOT registrar resultados, eventos, métricas ou alertas sob a chave antiga; MUST apagar as séries Prometheus e invalidar os caches de status da chave antiga; e MUST preservar no endpoint renomeado o estado dos alertas disparados (disparo, chave de resolução e contadores) cuja configuração não mudou, limpando o dos alertas cuja configuração mudou.

Um endpoint gerenciado em conflito com o arquivo de configuração MUST poder ser renomeado movendo apenas a definição: o histórico da chave antiga MUST continuar com o endpoint do arquivo, e a chave nova MUST começar sem histórico.

#### Scenario: Troca de grupo mantém o histórico
- **WHEN** um administrador envia `PUT /api/v1/admin/endpoints/web_site` com `group: clientes` para um endpoint com 100 resultados armazenados
- **THEN** a API responde 200 com a chave `clientes_site`
- **AND** `GET /api/v1/endpoints/clientes_site/statuses` mostra os 100 resultados
- **AND** `web_site` não aparece em `GET /api/v1/endpoints/statuses`

#### Scenario: Histórico preservado depois de reiniciar
- **WHEN** o Gatus reinicia depois da renomeação de `web_site` para `clientes_site`
- **THEN** `clientes_site` continua com o histórico anterior à renomeação
- **AND** nenhum dado é registrado sob `web_site`

#### Scenario: Troca de nome sem mudar a chave
- **WHEN** um administrador altera o nome de `My API` para `my-api`, que geram a mesma chave
- **THEN** a API responde 200 com a mesma chave
- **AND** o dashboard mostra o nome `my-api` com o histórico anterior

#### Scenario: Chave nova usada pelo arquivo de configuração
- **WHEN** um administrador renomeia `web_site` para `core_health`, definido no arquivo de configuração
- **THEN** a API responde 409
- **AND** `web_site` continua monitorado, com a definição e o histórico inalterados

#### Scenario: Chave nova com status armazenado
- **WHEN** um administrador renomeia `web_site` para `web_antigo`, que ainda tem resultados armazenados sem estar em uso
- **THEN** a API responde 409
- **AND** nenhum resultado de `web_site` ou de `web_antigo` é movido ou apagado

#### Scenario: Versão desatualizada na renomeação
- **WHEN** um administrador renomeia `web_site` enviando uma versão desatualizada em `If-Match`
- **THEN** a API responde 412
- **AND** `web_site` continua monitorado com a mesma chave e o mesmo histórico

#### Scenario: Renomeação durante verificação lenta
- **WHEN** um administrador renomeia um endpoint durante uma verificação em andamento
- **THEN** o resultado dessa verificação é descartado
- **AND** nenhum resultado, evento, métrica ou alerta é registrado sob a chave antiga depois da resposta

#### Scenario: Alerta disparado na renomeação
- **WHEN** um endpoint gerenciado tem um alerta disparado e um administrador troca apenas o grupo
- **THEN** nenhum novo disparo é enviado para o mesmo incidente
- **AND** quando o endpoint volta a ter sucesso, o resolve é enviado com a chave de resolução original

#### Scenario: Renomeação de gerenciado em conflito
- **WHEN** um administrador renomeia para `web_novo` um endpoint gerenciado marcado como em conflito com `web_site` do arquivo de configuração
- **THEN** `web_novo` passa a ser monitorado sem histórico
- **AND** `web_site` do arquivo continua monitorado com todo o histórico

### Requirement: Status pages na renomeação
Ao renomear a chave de um endpoint gerenciado, o sistema MUST trocar a chave antiga pela nova em `endpoints` e `featured` das status pages gerenciadas pela administração, na mesma transação da renomeação, incrementando a versão de cada página alterada e registrando o autor. O sistema MUST NOT alterar status pages do arquivo de configuração, e a resposta MUST listar as status pages do arquivo que selecionam a chave antiga por `endpoints` ou `featured`. Status pages que selecionam por grupo MUST seguir a regra de grupo com o grupo novo.

#### Scenario: Status page gerenciada selecionando pela chave
- **WHEN** a status page gerenciada `team` tem `web_site` em `featured` e um administrador renomeia `web_site` para `clientes_site`
- **THEN** `team` passa a ter `clientes_site` em `featured`, com a versão incrementada
- **AND** a página pública `team` continua mostrando o endpoint em destaque

#### Scenario: Status page do arquivo selecionando pela chave
- **WHEN** a status page `services` do arquivo de configuração tem `web_site` em `endpoints` e um administrador renomeia `web_site` para `clientes_site`
- **THEN** a resposta lista `services` entre as páginas do arquivo afetadas
- **AND** o arquivo de configuração não é alterado

## REMOVED Requirements

### Requirement: Chave imutável
**Reason**: Substituída por "Renomeação de endpoint gerenciado": nome e grupo podem mudar na edição, mantendo o histórico.
**Migration**: `PUT /api/v1/admin/endpoints/{chave}` com `name` ou `group` diferente passa a renomear em vez de responder 400. Integrações que dependiam do 400 devem conferir o nome e o grupo antes de enviar; a chave nova vem na resposta.
