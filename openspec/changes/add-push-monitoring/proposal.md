## Why

O Gatus só aceita monitoramento passivo pelos external endpoints do arquivo de configuração. Eles têm um token por endpoint e o resultado é enviado por `POST /api/v1/endpoints/{key}/external` com `Authorization: Bearer`. Não dá para criá-los pela administração, cada script precisa conhecer a chave e o token do endpoint, e a mensagem enviada só aparece quando há falha.

Quem vem do Uptime Kuma usa monitores Push com uma URL simples, sem header, que qualquer cron, backup ou job consegue chamar:

```
https://kuma.exemplo.com/api/push/<token>?status=up&msg=OK&ping=
```

Na tela do monitor, cada envio aparece com status, data e hora e mensagem. Queremos o mesmo no fork: juntar o monitoramento passivo ao ativo, com criação pela web e chaves que sirvam para vários endpoints (grupo ou global), sem criar um token por script.

## What Changes

- **Push com a mesma entrada do Uptime Kuma**, para migrar os scripts trocando só o endereço do servidor:
  - `/api/push/<token>`, sem autenticação do Gatus e com qualquer método HTTP, como o `router.all` do Kuma;
  - os mesmos parâmetros de query, com os mesmos padrões:
    - `status`: `up` é sucesso e qualquer outro valor é falha; padrão `up`;
    - `msg`: padrão `OK`;
    - `ping`: número em milissegundos; vazio ou inválido é ignorado, e fora de 0 a 100000000000 é erro;
  - as mesmas respostas: 200 com `{"ok":true}`, e 404 com `{"ok":false,"msg":"..."}` para token desconhecido, monitor inativo ou `ping` fora do intervalo;
  - o token do endpoint Push pode ser digitado ao criar ou editar, para colar o token que já existe no Kuma, ou gerado com 32 caracteres, como o botão "Reset Token" do Kuma;
  - extensão do fork, que não conflita com o formato do Kuma: `/api/push/<token>/<chave-do-endpoint>`, com a chave global, a de grupo ou o token do endpoint.

  O `POST /api/v1/endpoints/{key}/external` existente continua igual.
- **Três escopos de chave:**
  - **endpoint:** o token do endpoint Push, ou o token de um endpoint ativo que recebe push, que identifica o endpoint sozinho;
  - **grupo:** autoriza os endpoints daquele grupo que recebem push;
  - **global:** autoriza todos os endpoints que recebem push.

  Chaves de grupo e global são geradas com 32 caracteres aleatórios, guardadas só como hash e mostradas uma única vez. Podem ser revogadas.
- **Sem autocriação:** um envio para token desconhecido ou para chave de endpoint inexistente ou desabilitado responde 404 com a mensagem do Kuma ("Monitor not found or not active."). A resposta é igual em todos esses casos, para não revelar quais chaves existem.
- **Push também em endpoints ativos**, para serviços verificados pelo Gatus que também recebem notificações externas. Exemplo: alertas de métricas da Akamai chamando a URL com `?status=down&msg=...`.
  - Receber push é uma opção de cada endpoint ativo, desligada por padrão: no formulário, a opção "Accept push" (receber push), com token próprio opcional; no YAML, a lista `push.endpoints`, com a chave do endpoint e o token opcional.
  - Com a opção ligada, o endpoint ativo aceita a chave global e a de grupo em `/api/push/<chave>/<chave-do-endpoint>`, e o próprio token na URL do Kuma `/api/push/<token>`. Com a opção desligada, o envio responde 404.
  - Endpoints do tipo Push sempre recebem push.
  - Cada push vira um resultado no mesmo histórico, marcado como origem Push na tabela "Recent checks". Ele conta para o uptime e para os alertas junto com as verificações do Gatus, então o status pode alternar entre um push `down` e a verificação seguinte `up`.
  - O heartbeat continua só nos endpoints do tipo Push.
- **Endpoints Push pela administração:**
  - o formulário de endpoint ganha o tipo de monitor: ativos (HTTP(s), TCP, Ping, DNS e os demais inferidos pela URL) ou Push (passivo);
  - no Push, o formulário mostra a URL copiável, o botão para gerar um token novo, o intervalo de heartbeat, os alertas e um exemplo de `curl`;
  - a definição fica na mesma tabela `managed_endpoints`, com `type: push`, e ganha renomeação, remoção, histórico, status pages e alertas como os endpoints ativos.
- **Chaves pela administração e pelo YAML:**
  - tela nova "Push keys" para criar e revogar chaves de grupo e globais;
  - seção nova `push.keys` no arquivo de configuração;
  - os external endpoints do YAML passam a aceitar a URL nova, com o próprio token ou com chaves de grupo e globais.
- **Heartbeat:**
  - um endpoint Push sem envio dentro do intervalo registra falha ("nenhum envio recebido em 1m") e dispara alertas, como no Kuma;
  - pela administração o intervalo é obrigatório (padrão 60s, mínimo 10s);
  - o heartbeat passa a ser controlado pelo registro de monitoramento por chave, que também passa a cobrir os external endpoints do YAML;
  - fica corrigida uma falha que registra o heartbeat perdido só a cada dois intervalos.
- **Mensagem de cada resultado:** o `msg` do push é guardado com o resultado em qualquer status, limitado a 1024 bytes.
  - A página de detalhes do endpoint no dashboard ganha a tabela "Recent checks" (Status, Data e hora, Mensagem), com a mensagem do push, os erros ou o status HTTP das verificações ativas.
  - A mensagem nunca aparece nas status pages públicas.
- **Proteção da rota pública:**
  - a rota é registrada no bloco sem autenticação, com rotas coringa, para nunca pedir login ao navegador;
  - comparação por hash;
  - limite de tentativas inválidas por IP (429), usando `status-pages.trusted-proxies`;
  - `Cache-Control: no-store`;
  - o token nunca aparece nos logs.
- **Documentação:** `docs/push-monitoring.md`, com migração do Uptime Kuma e exemplos de cron, backup e script. Também `docs/README.md`, `docs/admin-endpoints.md` e `AGENTS.fork.md`.
- **Fora do escopo:**
  - criar endpoints automaticamente no primeiro envio;
  - importar monitores do banco do Uptime Kuma: cada monitor Push é criado no Gatus com o mesmo token;
  - webhooks com corpo JSON próprio, como o formato nativo de webhook da Akamai: a entrada é a query do Kuma;
  - o estado PENDING das tentativas (`Retries`) do Kuma: no Gatus, o equivalente é o `failure-threshold` dos alertas;
  - modo "upside down" e "manual" do Kuma;
  - mudar o label `type` das métricas Prometheus dos external endpoints do YAML, o que quebraria séries existentes.
- **Entrega:** em 3 marcos, cada um num pull request no `jniltinho/gatus`, com release ao final:
  1. rota de push, chaves do YAML, mensagem e heartbeat;
  2. endpoints Push e chaves pela administração;
  3. tabela "Recent checks", documentação e E2E.

Nenhuma mudança é **BREAKING**: a rota `POST /api/v1/endpoints/{key}/external`, os external endpoints do YAML e os endpoints ativos continuam iguais.

## Capabilities

### New Capabilities

- `push-monitoring`: monitoramento passivo por push, que cobre:
  - a URL compatível com o Uptime Kuma e seus parâmetros;
  - os escopos e o ciclo de vida das chaves (endpoint, grupo e global);
  - a proteção da rota pública;
  - o heartbeat;
  - os endpoints Push gerenciados pela web;
  - o push em endpoints ativos, no mesmo histórico;
  - a mensagem dos resultados e a tabela "Recent checks";
  - a presença dos endpoints Push nas status pages.

### Modified Capabilities

- `admin-web-ui`:
  - o formulário de endpoint ganha o tipo de monitor e os campos do Push;
  - a lista mostra o tipo PUSH e os external endpoints do YAML somente para leitura;
  - aba nova "Push keys".

## Impact

- **Go:**
  - `api/push.go` (novo): rota, coringas e limitador;
  - `config/push/` e `config/config_push.go`: seção `push.keys` e validação de tokens únicos;
  - `managedendpoint/`: `Parse` com `type`, `State` com o external endpoint, máscara e restauração de `token`, `Test` indisponível para Push;
  - `watchdog/registry.go`, `watchdog/endpoint.go` e `watchdog/external_endpoint.go`: heartbeat no registro por chave, lock por chave compartilhado entre verificações ativas e pushes, contadores sem corrida e correção do intervalo;
  - `statuspage/registry.go`: endpoints Push gerenciados em `Endpoints()`;
  - `config/endpoint/result.go`: campos `Message` e `Origin`.
- **Banco:**
  - tabela nova `push_keys` (escopo, grupo, hash, dica, autor e datas);
  - tabela nova `endpoint_result_messages`, ligada ao resultado com `ON DELETE CASCADE`;
  - nos três bancos, sem alterar tabelas do upstream.
- **Frontend:**
  - `AdminEndpointForm.vue`: tipo de monitor e campos do Push;
  - `AdminEndpoints.vue`: tipo e endpoints do YAML;
  - tela nova `AdminPushKeys.vue`;
  - `EndpointDetails.vue`: tabela "Recent checks";
  - `adminApi.js` e `web/static/`.
- **API:**
  - `GET|POST /api/push/:token` e `/api/push/:token/:key`;
  - `GET|POST|DELETE /api/v1/admin/push-keys`;
  - `message` nos resultados de `GET /api/v1/endpoints/{key}/statuses`, rota protegida.
- **Testes:**
  - rota de push, com escopos, 404 uniforme, 429, sem 401 e parâmetros do Kuma;
  - heartbeat;
  - store nos 4 bancos;
  - admin API;
  - E2E com agent-browser.
- **Segurança e operação:**
  - o token vai na URL, como no Kuma, e pode aparecer em logs de proxy; a documentação orienta HTTPS e rotação das chaves;
  - com várias instâncias no mesmo banco, cada uma registra o próprio heartbeat, limitação que já existe nos external endpoints.
