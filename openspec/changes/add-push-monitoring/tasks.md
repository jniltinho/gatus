## 1. Marco 1: rota de push, chaves do YAML, mensagem e heartbeat

- [x] 1.1 `config/push` e `config/config_push.go`:
  - seção `push.keys` com `name` e `token`, e `push.endpoints` com a chave de endpoint ativo e token opcional;
  - validação de nome, charset, tamanho mínimo de 16 e duplicatas de nome e de token;
  - hash SHA-256 calculado na carga;
  - testes.
- [x] 1.2 `endpoint.Result.Message` e `Origin` e a tabela `endpoint_result_messages` (mensagem e origem) nos três dialetos:
  - gravação em `insertEndpointResultWithSuiteID` com limite de 1024 bytes UTF-8;
  - `LEFT JOIN` nas leituras de resultados;
  - store em memória;
  - teste de cascata na limpeza de resultados antigos;
  - `mysql_schema_test.go` e conformidade nos 4 bancos.
- [x] 1.3 Generalizar `watchdog/registry.go` para heartbeats (`StartExternalEndpoint` e parada por chave). `watchdog.Monitor` passa a registrar os external endpoints do YAML. Testes de parada e de não afetar outros endpoints.
- [x] 1.4 `watchdog.ProcessExternalResult` com lock por chave, compartilhado com `executeEndpoint` das verificações ativas:
  - gravação, métricas, janelas de manutenção, alertas e contadores;
  - `watchdog.SubmitResult` para push em endpoint ativo registrado;
  - uso na rota antiga `POST /api/v1/endpoints/{key}/external`, sem mudar suas respostas;
  - testes de concorrência com `-race`.
- [x] 1.5 Heartbeat pelo último envio aceito:
  - uma falha por intervalo sem envio, inclusive consecutivos;
  - contagem iniciada na carga;
  - testes de 3 intervalos seguidos e de envio que reinicia a contagem.
- [x] 1.6 Resolução de push: índice de tokens de endpoint do YAML (tokens repetidos fora do índice, com aviso) e hashes das chaves do YAML; autorização por token de endpoint e chave global, para endpoints Push e ativos com push ligado do YAML, com rejeição uniforme.
- [x] 1.7 `api/push.go`:
  - `All` em `/push/:token` e `/push/:token/:key` e coringas;
  - leitura de `status`, `msg` e `ping` com a semântica do Kuma, incluindo o `parseFloat`;
  - respostas `{"ok":true}` e 404 `{"ok":false,"msg":...}` com os textos do Kuma;
  - `Cache-Control: no-store` e logs sem token;
  - limitador de rejeitados com `trusted-proxies`.
- [x] 1.8 Testes da rota:
  - URL do Kuma com `status=up&msg=OK&ping=`, `status=down`, `status=warning`, sem query, `ping` com prefixo numérico, `ping=0`, `ping` fora do intervalo;
  - métodos `GET`, `POST`, `PUT` e `HEAD`;
  - token desconhecido e endpoint desabilitado;
  - chave global num endpoint ativo do YAML com push ligado em `push.endpoints`: resultado marcado como Push entre as verificações, alerta disparado e sem corrida com a verificação ativa (`-race`);
  - endpoint ativo sem push respondendo 404;
  - chave global, e token de outro endpoint na URL com chave;
  - sem 401 e sem `WWW-Authenticate` com basic auth;
  - 429 depois de 30 rejeitados com envio válido aceito.

  Comparar as respostas com as do Kuma 2.5.4.
- [x] 1.9 PR do marco 1 no `jniltinho/gatus`, com CI verde

## 2. Marco 2: endpoints Push e chaves pela administração

- [x] 2.1 `managedendpoint.Parse` com `type` e `endpoint.ExternalEndpoint`:
  - campos permitidos e rejeição de campos do outro tipo;
  - `State.Push`;
  - token gerado com `crypto/rand` e validação de 8 a 128 caracteres;
  - campo `push` (`enabled` e `token` opcional) em definições ativas, retirado antes da decodificação estrita;
  - unicidade do token entre o YAML e a web (409);
  - testes.
- [x] 2.2 Ciclo dos endpoints Push gerenciados:
  - criação, alteração, habilitação, remoção e renomeação usando o registro de heartbeat;
  - índice de tokens atualizado no snapshot;
  - restauração de alertas;
  - `Test` com 400;
  - `statuspage.Endpoints()` com os Push gerenciados.
- [x] 2.3 Máscara e restauração de `token` na definição e `pushToken` no `Detail`. `Service.List`/`Get` incluem os external endpoints do YAML somente para leitura, com o token mascarado.
- [x] 2.4 Tabela `push_keys` nos três dialetos e `store.PushKeyStore` (`List`, `Create`, `Delete` com `apply`); testes nos 4 bancos e `mysql_schema_test.go`.
- [x] 2.5 Pacote `pushkey`:
  - serviço de criação (chave gerada e devolvida uma vez) e revogação;
  - snapshot publicado depois do commit e união com as chaves do YAML;
  - rotas `GET`/`POST /api/v1/admin/push-keys` e `DELETE /api/v1/admin/push-keys/{id}`;
  - auditoria sem a chave.
- [x] 2.6 Testes de API:
  - criação de Push sem token e com token do Kuma;
  - token repetido (409);
  - definição Push com `url` (400);
  - renomeação mantendo o token;
  - endpoint ativo gerenciado com push ligado recebendo push entre as verificações, e com push desligado respondendo 404;
  - chave criada, usada e revogada (404 depois);
  - chave do YAML somente leitura;
  - status page com Push gerenciado.
- [x] 2.7 Frontend:
  - `AdminEndpointForm.vue`: tipo de monitor, campos do Push, URL copiável, exemplo de `curl`, gerar e colar token, sem Testar, opção "Accept push" nos ativos, desligada por padrão, aviso de renomeação para URLs com chave;
  - `AdminEndpoints.vue`: tipo `PUSH` e endpoints do YAML;
  - `AdminPushKeys.vue` com exibição única e confirmação de revogação;
  - `adminApi.js`, rotas e abas;
  - variantes `dark:`;
  - lint e `make frontend-build`.
- [x] 2.8 PR do marco 2 no `jniltinho/gatus`, com CI verde

## 3. Marco 3: verificações recentes, documentação e E2E

- [x] 3.1 `EndpointDetails.vue`: tabela "Recent checks" (Status, Data e hora, origem e Mensagem na ordem do spec), com a paginação atual e as variantes `dark:`
- [x] 3.2 Teste de que os payloads públicos das status pages não trazem `message` nem erros de endpoints Push
- [x] 3.3 `docs/push-monitoring.md`:
  - URL e parâmetros compatíveis com o Kuma;
  - escopos das chaves;
  - migração de scripts do Kuma com o mesmo token;
  - exemplos de cron, backup e shell;
  - diferenças (PENDING e Retries);
  - push em endpoints ativos, com o exemplo de notificação da Akamai;
  - proxy e logs;
  - várias instâncias.

  Referências em `docs/README.md`, `docs/admin-endpoints.md` e `README.md`.
- [x] 3.4 `test/e2e/push.sh` com agent-browser:
  - criar endpoint Push pelo formulário e colar um token;
  - enviar `up`, `down` com mensagem e `up` com `curl`;
  - conferir a tabela "Recent checks" e capturar prints em `dist/prints/`;
  - criar e revogar uma chave global;
  - tema escuro.
- [x] 3.5 `AGENTS.fork.md` (rota pública, resolução, heartbeat no registro, tabela de mensagens) e `openspec/config.yaml` (contexto)
- [x] 3.6 `make lint test`, testes de `storage/store/sql` com PostgreSQL, MySQL e MariaDB, e `openspec validate add-push-monitoring --strict`
- [ ] 3.7 PR do marco 3 no `jniltinho/gatus`, com CI verde e merge, e release
