## ADDED Requirements

### Requirement: URL de push compatível com o Uptime Kuma
O sistema MUST receber envios em `/api/push/{token}` por qualquer método HTTP, sem autenticação do Gatus, lendo somente os parâmetros de query `status`, `msg` e `ping`, com as mesmas regras do Uptime Kuma:
- `status` igual a `up` MUST registrar sucesso, e qualquer outro valor MUST registrar falha; o padrão é `up`;
- `msg` MUST ser a mensagem do resultado, com padrão `OK`;
- `ping` MUST ser lido como número em milissegundos e usado como duração do resultado; vazio ou não numérico MUST ser ignorado; menor que 0 ou maior que 100000000000 MUST ser rejeitado.

Um envio aceito MUST responder 200 com `{"ok":true}`. Um envio rejeitado MUST responder 404 com `{"ok":false,"msg":"<motivo>"}` e MUST NOT armazenar resultado. Token desconhecido e endpoint desabilitado MUST usar o motivo `Monitor not found or not active.`. Nenhum envio MUST criar endpoint.

#### Scenario: URL copiada do Uptime Kuma
- **WHEN** um script chama `GET /api/push/keSDu7G855jvVat1xWiY2Gk4CkL1End5?status=up&msg=OK&ping=` e existe um endpoint Push com esse token
- **THEN** a API responde 200 com `{"ok":true}`
- **AND** o endpoint registra um resultado de sucesso com a mensagem `OK` e sem duração

#### Scenario: Falha com mensagem e ping
- **WHEN** um script chama `POST /api/push/<token>?status=down&msg=Falha%20no%20backup&ping=120`
- **THEN** o endpoint registra uma falha com a mensagem `Falha no backup` e duração de 120 ms

#### Scenario: Status diferente de up
- **WHEN** um script envia `status=warning`
- **THEN** o endpoint registra uma falha

#### Scenario: Envio sem parâmetros
- **WHEN** um script chama `GET /api/push/<token>` sem query
- **THEN** o endpoint registra um sucesso com a mensagem `OK`

#### Scenario: Token desconhecido
- **WHEN** um script chama `/api/push/tokeninexistente?status=up`
- **THEN** a API responde 404 com `{"ok":false,"msg":"Monitor not found or not active."}`
- **AND** nenhum resultado é armazenado e nenhum endpoint é criado

#### Scenario: Ping fora do intervalo
- **WHEN** um script envia `ping=-5`
- **THEN** a API responde 404 com `ok` falso e nenhum resultado é armazenado

### Requirement: Escopos das chaves de push
Além do token do próprio endpoint, o sistema MUST aceitar chaves de grupo e globais em `/api/push/{token}/{chave-do-endpoint}`, com os mesmos parâmetros, métodos e respostas da URL do Uptime Kuma:
- uma chave global MUST autorizar qualquer endpoint habilitado que receba push;
- uma chave de grupo MUST autorizar somente os endpoints habilitados que recebam push e cujo grupo seja o da chave;
- o token de um endpoint MUST autorizar somente esse endpoint.

Endpoint inexistente, desabilitado ou que não receba push, e chave que não autorize o endpoint, MUST responder 404 com o mesmo corpo de token desconhecido, sem revelar qual das condições falhou. Os endpoints Push MUST ser os external endpoints do arquivo de configuração e os endpoints Push gerenciados pela web, e MUST sempre receber push. Um endpoint ativo, do arquivo de configuração ou gerenciado pela web, MUST receber push somente quando a opção estiver ligada.

#### Scenario: Chave global
- **WHEN** um script usa a chave global em `/api/push/<global>/jobs_backup?status=up` e `jobs_backup` é um endpoint Push
- **THEN** a API responde 200 e `jobs_backup` registra o sucesso

#### Scenario: Chave de grupo fora do grupo
- **WHEN** um script usa a chave do grupo `jobs` em `/api/push/<jobs>/core_cron`
- **THEN** a API responde 404 com `{"ok":false,"msg":"Monitor not found or not active."}`
- **AND** `core_cron` não registra resultado

#### Scenario: Chave global num endpoint ativo
- **WHEN** a Akamai chama `/api/push/<global>/erp_site?status=down&msg=Latencia%20alta` e `erp_site` é um endpoint HTTP com push ligado
- **THEN** a API responde 200 e `erp_site` registra a falha com a mensagem `Latencia alta`, marcada como origem Push

#### Scenario: Endpoint ativo sem push
- **WHEN** um script usa a chave global com a chave de um endpoint HTTP com push desligado
- **THEN** a API responde 404 e nenhum resultado é armazenado

### Requirement: Tokens e chaves de push
O token de um endpoint Push, e o token opcional de um endpoint ativo com push ligado, MUST ter de 8 a 128 caracteres entre letras, dígitos, `-` e `_`. Ele MUST poder ser informado, para reaproveitar o token de um monitor do Uptime Kuma, ou gerado com 32 letras e dígitos aleatórios, com gerador criptográfico. Um token não pode identificar mais de um endpoint: a administração MUST rejeitar com 409 um token já usado por outro endpoint, do arquivo ou da web. No arquivo de configuração, `push.endpoints` MUST ligar o push de endpoints ativos pela chave, com token opcional, e uma chave que não seja de endpoint ativo do arquivo MUST ser rejeitada na carga. Quando o arquivo de configuração tiver o mesmo token em mais de um external endpoint, o sistema MUST registrar um aviso na carga, e esse token MUST responder 404 em `/api/push/{token}`.

As chaves de grupo e globais criadas pela administração MUST ser geradas com 32 letras e dígitos aleatórios, mostradas uma única vez e armazenadas somente como hash SHA-256 com os 4 últimos caracteres como dica. Elas MUST poder ser revogadas, e uma chave revogada MUST parar de autorizar envios imediatamente. As chaves do arquivo de configuração MUST ficar em `push.keys`, com `scope` (`global` ou `group`), `group` quando o escopo for de grupo e `token` com pelo menos 16 caracteres, e MUST ser somente leitura na administração.

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

#### Scenario: Chave de grupo do arquivo
- **WHEN** o arquivo de configuração define `push.keys` com `scope: group`, `group: jobs` e um token de 32 caracteres
- **THEN** a chave autoriza os endpoints Push do grupo `jobs` e aparece na administração sem ações de revogar

### Requirement: Proteção da rota de push
As rotas `/api/push` e `/api/push/*` MUST ser atendidas antes do middleware de segurança, inclusive para caminhos inválidos, e MUST NOT responder 401 nem enviar `WWW-Authenticate`. As respostas MUST ter `Cache-Control: no-store`. Tokens e chaves MUST NOT aparecer nos logs, e a comparação de chaves de grupo e globais MUST ser feita pelo hash.

Envios válidos MUST ser sempre aceitos. Depois de 30 envios rejeitados no mesmo minuto vindos do mesmo IP de cliente, calculado com `status-pages.trusted-proxies`, os envios rejeitados seguintes desse IP MUST responder 429 com `{"ok":false,"msg":"Too many requests"}`.

#### Scenario: Caminho inválido com basic auth
- **WHEN** `security.basic` está configurado e um cliente chama `PUT /api/push/a/b/c`
- **THEN** a resposta não é 401 e não tem `WWW-Authenticate`

#### Scenario: Tentativas de adivinhar tokens
- **WHEN** um IP envia 31 tokens inválidos no mesmo minuto
- **THEN** o 31º envio responde 429
- **AND** um envio com token válido do mesmo IP continua respondendo 200

### Requirement: Heartbeat de endpoints Push
Um endpoint Push gerenciado pela web MUST ter intervalo de heartbeat, com padrão de 60 segundos e mínimo de 10 segundos. Nos external endpoints do arquivo, o heartbeat continua opcional.

Para cada intervalo completo sem envio, o sistema MUST registrar uma falha que informe o intervalo e tratar os alertas, exceto em janela de manutenção. Intervalos consecutivos sem envio MUST gerar uma falha cada um.

O heartbeat MUST ser controlado pelo registro de monitoramento por chave, e MUST parar antes da resposta quando o endpoint for desabilitado, removido ou renomeado. Um envio aceito MUST reiniciar a contagem do intervalo.

#### Scenario: Intervalos sem envio
- **WHEN** um endpoint Push com intervalo de 1 minuto fica 3 minutos sem envio
- **THEN** o endpoint registra 3 falhas informando que nenhum envio foi recebido em 1m

#### Scenario: Envio depois de falhas
- **WHEN** um endpoint Push com alerta disparado por falta de envio recebe `status=up`
- **THEN** o endpoint registra sucesso e o alerta é resolvido conforme o `success-threshold`

#### Scenario: Endpoint desabilitado
- **WHEN** um administrador desabilita um endpoint Push
- **THEN** nenhuma falha de heartbeat é registrada depois da resposta
- **AND** os envios para o token do endpoint respondem 404

### Requirement: Endpoints Push gerenciados pela web
A administração MUST aceitar definições com `type: push`, com `name`, `group`, `token`, `heartbeat.interval`, `alerts`, `maintenance-windows` e `enabled`. Uma definição Push com campos de endpoint ativo (`url`, `conditions`, `headers`, `client`, `method`, `body`, `interval` e demais) MUST ser rejeitada com 400, e uma definição ativa com `token` ou `heartbeat` também. Uma definição ativa MUST aceitar `push`, com `enabled` e `token` opcional, e uma definição Push MUST rejeitar `push`. Sem `token`, a criação de um endpoint Push MUST gerar um. Os tokens MUST ser devolvidos somente pelas rotas da administração.

Os endpoints Push gerenciados MUST ter o mesmo ciclo dos endpoints gerenciados ativos: versão e `If-Match`, conflito de chave com o arquivo, renomeação com o histórico, remoção, restauração dos alertas disparados e aplicação sem reinício. A ação `test` MUST responder 400 para uma definição Push. Os endpoints Push gerenciados MUST poder ser selecionados pelas status pages por grupo, por chave e em destaque, como os external endpoints do arquivo.

#### Scenario: Criação sem token
- **WHEN** um administrador cria um endpoint Push `backup` no grupo `jobs` sem token
- **THEN** a resposta traz um token de 32 caracteres e a chave `jobs_backup`

#### Scenario: Campo de endpoint ativo
- **WHEN** um administrador envia uma definição com `type: push` e `url: https://exemplo.com`
- **THEN** a API responde 400

#### Scenario: Renomeação mantém o token
- **WHEN** um administrador troca o grupo de `jobs_backup` para `infra`
- **THEN** `/api/push/<token-do-endpoint>` continua registrando os envios em `infra_backup`, com o histórico anterior

#### Scenario: Status page com endpoint Push
- **WHEN** uma status page seleciona o grupo `jobs` e `jobs_backup` é um endpoint Push gerenciado
- **THEN** a página pública mostra `jobs_backup` com seus resultados

### Requirement: Push em endpoints ativos
Receber push MUST ser uma opção de cada endpoint ativo, desligada por padrão. Um push aceito para um endpoint ativo com a opção ligada MUST registrar um resultado no mesmo histórico das verificações do Gatus, com status, mensagem, duração e origem Push. Esse resultado MUST contar para o uptime, os eventos, as métricas e os alertas como as verificações. O processamento das verificações ativas e dos pushes de um mesmo endpoint MUST ser serializado, sem perder os contadores dos alertas. O heartbeat MUST NOT ser aplicado a endpoints ativos. Um push para um endpoint ativo com a opção desligada, parado, desabilitado ou removido MUST responder 404.

#### Scenario: Notificação da Akamai num serviço ativo
- **WHEN** `erp_site` tem push ligado, verifica a URL a cada minuto com sucesso e a Akamai envia `status=down&msg=Latencia alta` às 10:00:20
- **THEN** o histórico de `erp_site` mostra o push down às 10:00:20, com origem Push, entre as verificações up de 10:00:00 e 10:01:00
- **AND** um alerta com `failure-threshold: 1` dispara com a mensagem `Latencia alta`

#### Scenario: Push desligado
- **WHEN** um administrador desliga o push de `erp_site`
- **THEN** os envios para `erp_site` com a chave global ou com o token do endpoint respondem 404
- **AND** as verificações do Gatus continuam registrando resultados

#### Scenario: Push durante a verificação ativa
- **WHEN** um push chega enquanto a verificação ativa do mesmo endpoint grava seu resultado
- **THEN** os dois resultados são armazenados, e os contadores de falhas e sucessos seguidos refletem a ordem em que foram processados

### Requirement: Mensagem dos resultados e verificações recentes
O sistema MUST armazenar a mensagem e a origem Push de cada envio com o resultado, em qualquer status, com a mensagem limitada a 1024 bytes sem cortar caracteres UTF-8. Numa falha, a mensagem MUST também entrar nos erros do resultado, usados pelos alertas.

A API protegida de status MUST devolver `message` em cada resultado. A página de detalhes do endpoint no dashboard MUST mostrar a tabela "Recent checks", do mais recente para o mais antigo, com status, data e hora, origem (Push ou verificação) e mensagem. A mensagem da tabela MUST ser, nesta ordem: a mensagem do envio; os erros; o status HTTP da verificação ativa.

Os payloads das status pages públicas MUST NOT conter mensagens nem erros.

#### Scenario: Envios do Uptime Kuma no dashboard
- **WHEN** um endpoint Push recebe `status=up&msg=Backup OK`, depois `status=down&msg=Falha no backup: disco cheio` e depois `status=up&msg=Backup recuperado`
- **THEN** a tabela "Recent checks" mostra, nesta ordem, Up com `Backup recuperado`, Down com `Falha no backup: disco cheio` e Up com `Backup OK`, cada um com data e hora

#### Scenario: Mensagem fora da página pública
- **WHEN** o endpoint Push com a falha `Falha no backup: disco cheio` aparece numa status page publicada
- **THEN** a resposta pública não contém `Falha no backup`
