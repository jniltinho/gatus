## 1. Marco 1: rota de push, chaves do YAML, mensagem e heartbeat

- [ ] 1.1 `config/push` e `config/config_push.go`:
  - seção `push.keys` com `scope`, `group` e `token`;
  - validação de escopo, grupo, charset, tamanho mínimo de 16 e duplicatas;
  - hash SHA-256 calculado na carga;
  - testes.
- [ ] 1.2 `endpoint.Result.Message` e a tabela `endpoint_result_messages` nos três dialetos:
  - gravação em `insertEndpointResultWithSuiteID` com limite de 1024 bytes UTF-8;
  - `LEFT JOIN` nas leituras de resultados;
  - store em memória;
  - teste de cascata na limpeza de resultados antigos;
  - `mysql_schema_test.go` e conformidade nos 4 bancos.
- [ ] 1.3 Generalizar `watchdog/registry.go` para heartbeats (`StartExternalEndpoint` e parada por chave). `watchdog.Monitor` passa a registrar os external endpoints do YAML. Testes de parada e de não afetar outros endpoints.
- [ ] 1.4 `watchdog.ProcessExternalResult` com lock por chave:
  - gravação, métricas, janelas de manutenção, alertas e contadores;
  - uso na rota antiga `POST /api/v1/endpoints/{key}/external`, sem mudar suas respostas;
  - testes de concorrência com `-race`.
- [ ] 1.5 Heartbeat pelo último envio aceito:
  - uma falha por intervalo sem envio, inclusive consecutivos;
  - contagem iniciada na carga;
  - testes de 3 intervalos seguidos e de envio que reinicia a contagem.
- [ ] 1.6 Resolução de push: índice de tokens de endpoint do YAML (tokens repetidos fora do índice, com aviso) e hashes das chaves do YAML; autorização por endpoint, grupo e global com rejeição uniforme.
- [ ] 1.7 `api/push.go`:
  - `All` em `/push/:token` e `/push/:token/:key` e coringas;
  - leitura de `status`, `msg` e `ping` com a semântica do Kuma, incluindo o `parseFloat`;
  - respostas `{"ok":true}` e 404 `{"ok":false,"msg":...}` com os textos do Kuma;
  - `Cache-Control: no-store` e logs sem token;
  - limitador de rejeitados com `trusted-proxies`.
- [ ] 1.8 Testes da rota:
  - URL do Kuma com `status=up&msg=OK&ping=`, `status=down`, `status=warning`, sem query, `ping` com prefixo numérico, `ping=0`, `ping` fora do intervalo;
  - métodos `GET`, `POST`, `PUT` e `HEAD`;
  - token desconhecido, endpoint desabilitado e endpoint ativo;
  - chave de grupo dentro e fora do grupo, chave global;
  - sem 401 e sem `WWW-Authenticate` com basic auth;
  - 429 depois de 30 rejeitados com envio válido aceito.

  Comparar as respostas com as do Kuma 2.5.4.
- [ ] 1.9 PR do marco 1 no `jniltinho/gatus`, com CI verde

## 2. Marco 2: endpoints Push e chaves pela administração

- [ ] 2.1 `managedendpoint.Parse` com `type` e `endpoint.ExternalEndpoint`:
  - campos permitidos e rejeição de campos do outro tipo;
  - `State.Push`;
  - token gerado com `crypto/rand` e validação de 8 a 128 caracteres;
  - unicidade do token entre o YAML e a web (409);
  - testes.
- [ ] 2.2 Ciclo dos endpoints Push gerenciados:
  - criação, alteração, habilitação, remoção e renomeação usando o registro de heartbeat;
  - índice de tokens atualizado no snapshot;
  - restauração de alertas;
  - `Test` com 400;
  - `statuspage.Endpoints()` com os Push gerenciados.
- [ ] 2.3 Máscara e restauração de `token` na definição e `pushToken` no `Detail`. `Service.List`/`Get` incluem os external endpoints do YAML somente para leitura, com o token mascarado.
- [ ] 2.4 Tabela `push_keys` nos três dialetos e `store.PushKeyStore` (`List`, `Create`, `Delete` com `apply`); testes nos 4 bancos e `mysql_schema_test.go`.
- [ ] 2.5 Pacote `pushkey`:
  - serviço de criação (chave gerada e devolvida uma vez) e revogação;
  - snapshot publicado depois do commit e união com as chaves do YAML;
  - rotas `GET`/`POST /api/v1/admin/push-keys` e `DELETE /api/v1/admin/push-keys/{id}`;
  - auditoria sem a chave.
- [ ] 2.6 Testes de API:
  - criação de Push sem token e com token do Kuma;
  - token repetido (409);
  - definição Push com `url` (400);
  - renomeação mantendo o token;
  - chave criada, usada e revogada (404 depois);
  - chave do YAML somente leitura;
  - status page com Push gerenciado.
- [ ] 2.7 Frontend:
  - `AdminEndpointForm.vue`: tipo de monitor, campos do Push, URL copiável, exemplo de `curl`, gerar e colar token, sem Testar, aviso de renomeação para URLs com chave;
  - `AdminEndpoints.vue`: tipo `PUSH` e endpoints do YAML;
  - `AdminPushKeys.vue` com exibição única e confirmação de revogação;
  - `adminApi.js`, rotas e abas;
  - variantes `dark:`;
  - lint e `make frontend-build`.
- [ ] 2.8 PR do marco 2 no `jniltinho/gatus`, com CI verde

## 3. Marco 3: verificações recentes, documentação e E2E

- [ ] 3.1 `EndpointDetails.vue`: tabela "Recent checks" (Status, Data e hora, Mensagem na ordem do spec), com a paginação atual e as variantes `dark:`
- [ ] 3.2 Teste de que os payloads públicos das status pages não trazem `message` nem erros de endpoints Push
- [ ] 3.3 `docs/push-monitoring.md`:
  - URL e parâmetros compatíveis com o Kuma;
  - escopos das chaves;
  - migração de scripts do Kuma com o mesmo token;
  - exemplos de cron, backup e shell;
  - diferenças (PENDING e Retries);
  - proxy e logs;
  - várias instâncias.

  Referências em `docs/README.md`, `docs/admin-endpoints.md` e `README.md`.
- [ ] 3.4 `test/e2e/push.sh` com agent-browser:
  - criar endpoint Push pelo formulário e colar um token;
  - enviar `up`, `down` com mensagem e `up` com `curl`;
  - conferir a tabela "Recent checks" e capturar prints em `dist/prints/`;
  - criar e revogar uma chave de grupo;
  - tema escuro.
- [ ] 3.5 `AGENTS.fork.md` (rota pública, resolução, heartbeat no registro, tabela de mensagens) e `openspec/config.yaml` (contexto)
- [ ] 3.6 `make lint test`, testes de `storage/store/sql` com PostgreSQL, MySQL e MariaDB, e `openspec validate add-push-monitoring --strict`
- [ ] 3.7 PR do marco 3 no `jniltinho/gatus`, com CI verde e merge, e release
