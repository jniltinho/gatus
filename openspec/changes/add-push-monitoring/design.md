## Context

Hoje o monitoramento passivo do Gatus são os external endpoints do arquivo de configuração:

- **Envio:** `POST /api/v1/endpoints/{key}/external?success=&error=&duration=` com `Authorization: Bearer <token>` (`api/external_endpoint.go`). O endpoint é procurado antes do token, o que revela quais chaves existem. O token é comparado com `!=`.
- **Heartbeat:** opcional (`watchdog/external_endpoint.go`). Roda em goroutines iniciadas por `watchdog.Monitor`, fora do registro por chave do fork (`watchdog/registry.go`). Falha a cada intervalo sem resultado novo, mas a própria falha conta como resultado novo no intervalo seguinte, então intervalos consecutivos sem envio registram falha só a cada dois.
- **Contadores:** os de falhas e sucessos seguidos ficam no objeto de configuração e são escritos sem lock pelo envio e pelo heartbeat.
- **Endpoints gerenciados pela web:** só existem ativos (`managedendpoint.Parse` decodifica `endpoint.Endpoint` com `KnownFields`). A administração não lista os external endpoints do YAML.
- **Resultado:** não tem mensagem de sucesso, só `Errors` (coluna `errors` em `endpoint_results`). O dashboard mostra barras com tooltip e a lista de eventos, sem tabela de verificações.

O Uptime Kuma 2.5.4 foi avaliado localmente:

- **Rota:** `router.all("/api/push/:pushToken")` lê só a query.
  - `status`: `"up"` é UP e qualquer outro valor é DOWN.
  - `msg`: padrão `"OK"`.
  - `ping`: `parseFloat(ping) || null`, com erro fora de 0 a 100000000000.
  - Respostas: `{"ok":true}`, ou 404 com `{"ok":false,"msg":...}` e `Monitor not found or not active.`.
- **Token:** `genSecret(32)` com letras e dígitos.
- **Dashboard do monitor:** mostra Status, DateTime e Message de cada envio.
- **Heartbeat:** marca DOWN com `No heartbeat in the time window` quando nada chega no intervalo.

O usuário vai migrar scripts do Kuma trocando só o endereço do servidor.

Restrições do fork:

- toda inicialização suporta o ciclo de hot reload;
- rotas públicas ficam antes do middleware de segurança, com rotas coringa;
- tabelas novas em SQLite, PostgreSQL e MySQL/MariaDB, com placeholders `$N`;
- consultas ao store nunca dentro do `apply`;
- frontend com variantes `dark:`;
- código novo em arquivos novos sempre que possível.

## Goals / Non-Goals

**Goals:**

- Receber o push com a mesma entrada e as mesmas respostas do Uptime Kuma, inclusive reaproveitando tokens existentes.
- Chaves por endpoint, grupo e global, sem criar endpoints no envio.
- Endpoints Push pela administração com o mesmo ciclo dos endpoints gerenciados, e chaves pela administração e pelo YAML.
- Heartbeat confiável por endpoint, controlado pelo registro por chave.
- Mensagem de cada resultado no dashboard, nunca nas páginas públicas.

**Non-Goals:**

- Importar monitores do banco do Kuma.
- O estado PENDING e as tentativas do Kuma, o modo "upside down" e os monitores manuais.
- Mudar o label `type` das métricas dos external endpoints do YAML.
- Criar endpoints no primeiro envio.
- Sincronizar heartbeats entre instâncias que compartilham o banco.

## Decisions

### D1. Rota `/api/push` idêntica à do Kuma

Arquivo novo `api/push.go`, registrado no bloco sem autenticação de `api/api.go`, logo depois da rota de external endpoints:

- `All("/push/:token")` e `All("/push/:token/:key")`;
- coringas `All("/push")` e `All("/push/*")`, que respondem 404 em JSON, no mesmo padrão das status pages, para nenhum caminho chegar ao basic auth ou ao OIDC.

**Leitura da query:**
- **`status`:** primeiro valor; `up` é sucesso e qualquer outro valor é falha; ausente vale `up`.
- **`msg`:** ausente vale `OK`.
- **`ping`:** o prefixo numérico é lido como no `parseFloat` do JavaScript (`12abc` vale 12). Vazio, não numérico ou `0` é ignorado, como `|| null`. Um número fora de 0 a 100000000000 é rejeitado.

**Resultado:** `Timestamp` agora, `Success` pelo status, `Duration` pelo `ping` em milissegundos e `Message` pelo `msg`. Numa falha, o `msg` também vai para `Errors`.

**Respostas:**
- 200 com `{"ok":true}` quando o envio é aceito;
- 404 com `{"ok":false,"msg":...}` quando é rejeitado, com os textos do Kuma: `Monitor not found or not active.` e `Invalid ping value. Must be between 0 and 100000000000 ms.`;
- sempre com `Cache-Control: no-store`.

**Alternativas consideradas:**
- **Aceitar só `GET`:** rejeitada, porque o Kuma aceita qualquer método e há scripts com `POST`.
- **Parâmetros do Gatus (`success`, `error`, `duration`):** rejeitados, porque confundem a compatibilidade; a rota antiga continua para eles.
- **Validação estrita do `ping`:** rejeitada, porque um script que funciona no Kuma não pode falhar no Gatus.

### D2. Resolução de tokens e chaves

Um componente único resolve cada envio, lendo snapshots atômicos atualizados na carga, na recarga e em cada alteração da administração:

- **token de endpoint → endpoint Push:** índice dos external endpoints do YAML (montado a partir do `cfg` do router) e dos endpoints Push gerenciados (índice novo no snapshot de `managedendpoint`, atualizado em `putState`/`replaceState`/`removeState`);
- **hash da chave → escopo:** chaves de grupo e globais do YAML, com hash calculado na carga, e da tabela `push_keys`, publicadas por um pacote novo `pushkey` depois do commit, como as status pages.

Com `/api/push/{token}`, só tokens de endpoint valem. Com `/api/push/{token}/{key}`, o endpoint é procurado pela chave; o envio é autorizado quando o token é o dele, quando o hash é de uma chave global ou quando o hash é de uma chave do grupo atual do endpoint. Todas as rejeições respondem da mesma forma.

Um token presente em mais de um endpoint Push fica fora do índice de `/api/push/{token}` e gera um aviso na carga. A administração impede duplicatas com 409.

- **Alternativa:** consultar o banco a cada envio. Rejeitada, porque um cron de minuto em centenas de endpoints não pode depender do banco para autorizar.
- **Alternativa:** chave de grupo com o grupo na URL. Rejeitada, porque o grupo sai da chave do endpoint e a URL ficaria redundante.

### D3. Endpoints Push na tabela `managed_endpoints`

A definição ganha `type: push`:

- `managedendpoint.Parse` lê primeiro o documento para descobrir o tipo e depois decodifica com `KnownFields`: em `endpoint.Endpoint` para os ativos, ou em `endpoint.ExternalEndpoint` para o Push.
- Campos aceitos no Push: `name`, `group`, `token`, `heartbeat`, `alerts`, `maintenance-windows` e `enabled`.
- `State` ganha `Push *endpoint.ExternalEndpoint` ao lado de `Endpoint`.
- Criação, alteração, renomeação (`RenameManagedEndpoint` e participantes), remoção, conflitos de chave, histórico e restauração de alertas continuam trabalhando por chave e servem aos dois tipos.
- `Test` responde 400 para Push.
- `statuspage.Endpoints()` passa a incluir os Push gerenciados habilitados.

- **Alternativa:** tabela separada `managed_push_endpoints`. Rejeitada, porque duplicaria renomeação, remoção, histórico e verificação de chave entre tabelas, e a unicidade da chave passaria a depender de duas tabelas.

### D4. Token do endpoint visível só na administração

O token fica em texto na definição, como o `token` dos external endpoints do YAML e os demais segredos da administração, porque a tela precisa montar a URL a qualquer momento.

- Nas respostas, a definição mascara `token` (`MaskSecrets`/`RestoreMaskedSecrets` ganham o campo), e o `Detail` traz `pushToken` separado.
- O gerador usa `crypto/rand` com 32 letras e dígitos.
- Tokens informados são validados: 8 a 128 caracteres entre `[A-Za-z0-9_-]`.

- **Alternativa:** guardar só o hash e mostrar uma vez. Rejeitada para o token de endpoint, porque o Kuma mostra a URL sempre e a migração cola tokens existentes.
- **Alternativa:** botão "revelar" com rota própria. Rejeitada por complexidade sem ganho, já que o administrador vê e troca o token de qualquer forma.

### D5. Chaves de grupo e globais com hash

**Tabela `push_keys`, criada nos três dialetos:**
- `push_key_id`;
- `scope` (`global` ou `group`);
- `group_name` (`VARCHAR(768)` no MySQL);
- `token_hash` (SHA-256 em hexadecimal, único);
- `hint` (4 últimos caracteres);
- `created_at` em milissegundos e `created_by`.

A interface é `store.PushKeyStore`, com `List`, `Create` e `Delete` com `apply` e o mesmo contrato das tabelas gerenciadas.

**No YAML**, as chaves ficam na seção nova `push.keys` (pacote `config/push` e `config/config_push.go`, no padrão de `admin` e `status-pages`). Cada uma tem `scope`, `group` e `token` com pelo menos 16 caracteres, e é validada na carga: escopo, grupo obrigatório no escopo de grupo, charset e duplicatas.

**Na administração:**
- rotas `GET` e `POST /api/v1/admin/push-keys` e `DELETE /api/v1/admin/push-keys/{id}`;
- o `POST` gera a chave e a devolve uma única vez;
- a listagem mistura as origens Web e YAML.

A autorização por grupo usa o grupo atual do endpoint, então trocar o grupo muda quais chaves valem.

- **Alternativa:** guardar a chave em texto. Rejeitada, porque uma chave global autoriza todos os endpoints.
- **Alternativa:** bcrypt. Rejeitada, porque o custo por envio é alto e não permite busca indexada.

### D6. Heartbeat no registro por chave, a partir do último envio

`watchdog/registry.go` generaliza a entrada: o registro guarda uma função de execução e o tempo de espera na parada, em vez de chamar sempre `monitorEndpoint`.

- `StartExternalEndpoint(ee, source)` registra o heartbeat.
- `watchdog.Monitor` passa a registrar os external endpoints do YAML (`SourceConfig`).
- A administração registra os Push gerenciados (`SourceAdmin`).
- Parar, renomear e remover usam as mesmas funções por chave.

**Contagem:**
- o último envio aceito fica em memória por chave;
- a cada tique do intervalo, se o último envio for mais antigo que o intervalo, é registrada uma falha;
- sem envio depois da carga, a contagem começa na carga, com uma janela de tolerância, como o Kuma depois de reiniciar;
- isso corrige a falha só a cada dois intervalos.

**Processamento único:** `watchdog.ProcessExternalResult(ee, result, cfg)` concentra gravação, métricas, janelas de manutenção, alertas e contadores, sob um lock por chave. É usado pela rota antiga, pela rota nova e pelo heartbeat, o que elimina a corrida dos contadores.

- **Alternativa:** manter goroutines fora do registro. Rejeitada, porque a administração não conseguiria parar o heartbeat de um endpoint desabilitado ou renomeado.
- **Alternativa:** continuar usando `HasEndpointStatusNewerThan`. Rejeitada, pela falha descrita no contexto.

### D7. Mensagem em tabela própria do fork

`endpoint.Result` ganha `Message` (`json:"message,omitempty"`). A tabela nova `endpoint_result_messages` tem `endpoint_result_id` como chave primária, com chave estrangeira para `endpoint_results` e `ON DELETE CASCADE`, e `message`.

- A gravação acontece em `insertEndpointResultWithSuiteID`, só quando a mensagem não é vazia.
- As leituras de resultados fazem `LEFT JOIN`.
- A limpeza de resultados antigos remove as mensagens pela cascata.
- O limite é de 1024 bytes, cortado em fronteira de caractere UTF-8.
- O store em memória guarda o campo direto.
- O payload público continua com a lista de campos permitidos e um teste impede `message`.

- **Alternativa:** coluna `message` em `endpoint_results`. Rejeitada, porque altera tabela do upstream, exige `ALTER TABLE` com detecção por dialeto (MySQL 8.4 não tem `ADD COLUMN IF NOT EXISTS`) e aumenta conflitos nas sincronizações.
- **Alternativa:** usar `Errors` também no sucesso. Rejeitada, porque mudaria a semântica de sucesso nos alertas e na UI.

### D8. Proteção da rota

- **Limitador:** `statuspage.NewLimiter` com instância própria. Conta só envios rejeitados, até 30 por minuto por IP (IPv6 por /64), com o IP do cliente de `statuspage.ClientIP` e `status-pages.trusted-proxies`.
- **Ordem:** a resolução acontece primeiro. Um envio válido é aceito mesmo acima do limite; um rejeitado acima do limite responde 429, com o corpo no formato do Kuma.
- **Logs:** mostram só a chave do endpoint.

- **Alternativa:** limitar todos os envios por IP. Rejeitada, porque bloquearia scripts legítimos atrás do mesmo NAT.

### D9. Administração e dashboard

- **Formulário:**
  - seletor de tipo de monitor: ativos, com a URL sugerida pelo tipo (`tcp://`, `icmp://` etc.), ou Push;
  - campos do Push: token com gerar e colar, URL copiável no formato do Kuma, heartbeat e alertas;
  - aviso de renomeação para as URLs com chave;
  - sem Testar no Push.
- **Lista:** tipo `PUSH`, e os external endpoints do YAML somente para leitura (`Service.List`/`Get` passam a incluí-los, com o token mascarado).
- **Chaves:** aba nova "Push keys" (`AdminPushKeys.vue`).
- **Dashboard:** `EndpointDetails.vue` ganha a tabela "Recent checks" com os resultados já carregados pela paginação atual.

### D10. Entrega em 3 marcos

1. Rota `/api/push`, resolução, chaves do YAML, mensagem com a tabela nova, heartbeat no registro com o processamento único, testes da rota e do store nos 4 bancos.
2. Endpoints Push e chaves pela administração: API, formulário, lista, aba "Push keys" e testes de API.
3. Tabela "Recent checks", `docs/push-monitoring.md` (migração do Kuma, exemplos de cron, backup e script, proxy), E2E com agent-browser e release.

## Risks / Trade-offs

- **[Token na URL aparece em logs de proxy e histórico]** → Mesmo modelo do Kuma. A documentação orienta HTTPS, logs sem query e path e rotação de chaves. O Gatus não loga o token.
- **[Diferença de semântica com PENDING e Retries do Kuma]** → Documentar que `failure-threshold` dos alertas substitui as tentativas e que o status aparece como falha imediatamente.
- **[Janela sem falha depois de reiniciar ou recarregar]** → A contagem começa na carga, como no Kuma. Documentado.
- **[Heartbeat duplicado com várias instâncias no mesmo banco]** → Limitação já existente dos external endpoints, citada na documentação de várias instâncias.
- **[Limitador por IP atrás de proxy não confiável]** → O mesmo aviso de `trusted-proxies` das status pages vale para o push.
- **[Token repetido no YAML]** → Aviso na carga e 404 para a URL sem chave. A URL com chave continua funcionando.
- **[Crescimento da tabela de mensagens]** → Segue os limites de resultados por endpoint, pela cascata.
- **[Versão anterior lendo definições `type: push`]** → Ela marca a definição como inválida, sem monitorar, e preserva o histórico. As tabelas novas são ignoradas.

## Migration Plan

- As tabelas `push_keys` e `endpoint_result_messages` são criadas automaticamente e de forma idempotente. Não há migração de dados.
- A rota antiga dos external endpoints e o YAML atual continuam válidos.
- **Rollback:** voltar para a versão anterior mantém o histórico. Os endpoints Push gerenciados aparecem como inválidos, e as chaves e mensagens ficam guardadas até a volta.
- A documentação traz a migração de scripts do Kuma: criar o endpoint Push com o mesmo token e trocar o endereço do servidor na URL.

## Open Questions

- Nenhuma que bloqueie. O limite de 30 envios rejeitados por minuto e a janela de tolerância depois da carga podem ganhar configuração se o uso mostrar necessidade.
