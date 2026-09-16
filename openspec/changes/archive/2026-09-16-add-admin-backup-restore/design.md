## Context

- **Endpoints gerenciados:** ficam em `managed_endpoints` (chave, definição YAML sem padrões, versão, datas, autor). `managedendpoint.Service` faz:
  - **escritas:** `Create`, `Update` (com o caminho de renomeação quando nome ou grupo mudam, mesmo com a mesma chave), `SetEnabled` e `Delete`;
  - **proteções das escritas:** `lifecycle.TryBeginChange` (503 durante a recarga), `statesMutex`, validação estrita em `prepare`, conflito de chave com o YAML (409), token de push único (`pushTokens`, `isPushKeyToken`) e início do monitoramento dentro da transação;
  - **tipo:** a troca entre Push e ativo (`ErrTypeChanged`) só é verificada em `update`;
  - **tokens:** `Create` gera o token de um Push sem token (`withGeneratedPushToken`);
  - **segredos:** o detalhe mascara com `MaskSecrets` os headers sensíveis, a senha da URL, os segredos de client/SSH, `provider-override` e os tokens. `prepare` restaura os valores mascarados com `RestoreMaskedSecrets`;
  - **disponibilidade:** `Load` publica um mapa vazio quando o store falha.
- **Status pages gerenciadas:** ficam em `managed_status_pages`. `statuspage.Service` faz `Create`, `Update`, `SetEnabled` e `Delete`, e publica depois do commit.
  - O conflito de slug com o YAML só é verificado na criação.
  - Os avisos de seleção são calculados com os endpoints atuais.
  - Há `managedUnavailable` e `Generation()`.
- **Chaves globais de push:** ficam em `push_keys` (nome, SHA-256 hexadecimal do token, dica, autor e data).
  - `pushkey.Create` gera o token, e o nome é único entre as chaves do YAML e as da web.
  - `push/resolver.go` aceita qualquer chave global para qualquer endpoint.
- **API de administração:**
  - `adminRequestProtection` aplica CSRF, tipo de mídia (JSON ou YAML) e corpo de até 256 KB;
  - o Fiber usa o `BodyLimit` padrão de 4 MiB, e o fasthttp responde 413 em texto antes dos middlewares;
  - o handler de endpoints limpa o cache `endpoint-status-*` depois de alterar;
  - as rotas SPA de `/admin/*` são registradas uma a uma em `api/api.go`.
- **Limitador e IP do cliente:**
  - `security/limiter.go` tem um limitador de falhas privado, usado só pelo login basic;
  - `clientIPMiddleware` só é registrado com login basic.
- **Frontend:**
  - as abas da administração (`AdminTabs`) aparecem nas listas (`AdminListLayout`);
  - `utils/adminApi.js#request` lê só JSON.
- **Deploy:** atrás do nginx do host, com `proxy_read_timeout` padrão de 60 s.

## Goals / Non-Goals

**Goals:**
- Backup dos cadastros feitos pela web, restaurável em outra instalação com qualquer banco suportado.
- Restore seguro:
  - prévia sem efeitos que prevê exatamente a aplicação;
  - mesclagem sem apagar nada;
  - as mesmas regras das telas;
  - resultado por item;
  - repetição idempotente.
- Arquivo com senha opcional, escolhida na hora, sem abrir espaço para esgotamento de recursos.

**Non-Goals:**
- Histórico (resultados, eventos, uptime, agregados) e alertas disparados.
- Cadastros do arquivo de configuração, sessões de login e configuração global.
- "Substituir tudo", remoção de itens ausentes e troca de chave ou slug.
- Backup agendado, envio externo e restore assíncrono.
- Restore atômico de todos os itens.

## Decisions

### D1. Formato do arquivo (versão 1)
JSON UTF-8. O nome sugerido é `gatus-backup-<AAAAMMDD-HHMMSS>.json`, ou `gatus-backup-<AAAAMMDD-HHMMSS>.enc.json` quando cifrado.

```json
{
  "format": "gatus-admin-backup",
  "version": 1,
  "createdAt": "2026-09-16T18:00:00Z",
  "createdBy": "admin",
  "gatusVersion": "v5.36.0-fork.18",
  "endpoints": [{"key": "jobs_backup", "definition": "type: push\nname: backup\ngroup: jobs\ntoken: ...\n"}],
  "statusPages": [{"slug": "jobs", "definition": "slug: jobs\ntitle: Jobs\ngroups: [jobs]\n"}],
  "pushKeys": [{"name": "akamai", "tokenHash": "<64 hex minúsculos>", "hint": "Ab12", "createdAt": "...", "createdBy": "admin"}]
}
```

- **Definições:** as definições completas do store, sem máscara, ordenadas por chave, slug e nome.
- **`createdAt`/`createdBy` das chaves:** são só informativos. O restore cria a chave com a data atual e o autor do restore.
- **`gatusVersion`:** é opcional e vem de `debug.ReadBuildInfo` (pode ser "(devel)").
- **Leitura estrita:** chaves desconhecidas, `format` diferente, `version` maior que a suportada, lista ausente ou item duplicado (chave, slug, nome ou hash) dão 400.
- **Política de versão:** campos novos exigem versão nova, e o leitor aceita as versões de 1 até a suportada.
- **Limites:** o JSON em texto claro tem até 2 MiB, com até 1.000 endpoints, 200 status pages e 500 chaves.
  - Com isso, o envelope cifrado (base64, cerca de 2,7 MiB) e o corpo JSON do restore cabem nos 4 MiB do Fiber, sem mudar o `BodyLimit` global.
  - `Build` responde 422 ("backup exceeds 2 MiB or the item limits") quando os cadastros passam desses limites.
  - Atrás de um proxy, o corpo do restore precisa de `client_max_body_size 4m` no nginx, cujo padrão é 1m (documentado).

### D2. Cifragem opcional
O arquivo cifrado é um envelope JSON:

```json
{"format":"gatus-admin-backup-encrypted","version":1,
 "kdf":{"name":"argon2id","time":2,"memoryKiB":19456,"threads":1,"salt":"<base64 padrão, 16 bytes>"},
 "cipher":{"name":"aes-256-gcm","nonce":"<base64 padrão, 12 bytes>"},
 "data":"<base64 padrão do texto cifrado com a tag>"}
```

- **Parâmetros:** são os mínimos recomendados pela OWASP para Argon2id (19 MiB, t=2, p=1). A versão 1 aceita **somente** esses valores de `time`, `memoryKiB` e `threads`, o que limita o custo de um envelope forjado.
- **Chave e aleatoriedade:** a chave é `argon2.IDKey(senha, salt, 2, 19456, 1, 32)`, e o salt e o nonce saem de `crypto/rand` a cada backup.
- **AAD:** o dado adicional autenticado do GCM é o JSON canônico do cabeçalho: struct Go `{format, version, kdf, cipher}`, serializado por `encoding/json`, com esta ordem de campos. Um vetor de teste fixo garante os bytes.
- **Leitura estrita:**
  - campos desconhecidos são recusados;
  - `kdf.name` e `cipher.name` são fixos;
  - o salt tem exatamente 16 bytes e o nonce exatamente 12;
  - o base64 é o padrão, com padding.
- **Erro único:** senha errada e arquivo alterado dão o mesmo erro, "invalid password or corrupted file" (400).
- **Senha:** de 12 a 1.024 bytes UTF-8, contados assim também na tela (`TextEncoder`).
  - Senha enviada com arquivo sem cifra dá 400 ("file is not encrypted").
  - Arquivo cifrado sem senha dá 400 ("password required").
- **DoS:** um semáforo global permite no máximo 2 derivações ao mesmo tempo. Com o semáforo ocupado por mais de 5 s, a resposta é 429 com `Retry-After: 5`.
- **Tentativas:** um limitador de falhas **próprio**, separado do login, conta por IP do cliente as senhas erradas das rotas de restore (10 falhas em 15 minutos bloqueiam por 15 minutos, com 429).
  - A janela do limitador atual é a constante `failureLimiterWindow` (1 minuto). O construtor exportado recebe a janela (`window time.Duration`), guardada no struct, e o login continua com 1 minuto.
  - O IP vem de `status-pages.trusted-proxies`, com o middleware de IP registrado nessas rotas em qualquer modo de autenticação.
- **Semáforo compartilhado:** backup e restore usam o mesmo semáforo de 2 derivações. A derivação nunca acontece com o lock de `TryBeginChange` seguro.
- **Sigilo:** a senha nunca é guardada nem vai para o log.

### D3. Backup
`POST /api/v1/admin/backup` com `Content-Type: application/json` obrigatório e corpo `{}` ou `{"password": "..."}`:
- **Recarga:** monta o JSON sob `lifecycle.TryBeginChange` (503 durante a recarga), solta o lock e só então deriva a chave e cifra.
- **Indisponibilidade:** responde 503 quando qualquer um dos três registros está indisponível.
- **Leitura:** lê as listas dos stores.
- **Resposta:** `200` com `Content-Disposition: attachment; filename="..."` e `Cache-Control: no-store`.
- **Log:** a operação, as quantidades, a cifragem e o autor.
- **Modo de desenvolvimento:** o CORS expõe `Content-Disposition`.

### D4. Plano do restore (prévia)
As rotas de restore aceitam só `application/json`, com corpo de até 3,5 MiB verificado no handler:

```json
{"file": <objeto do arquivo ou do envelope>, "password": "...", "overwrite": false, "disableEndpoints": false}
```

**Sem efeitos:** `POST /api/v1/admin/restore/preview` monta o plano com `adminbackup.Plan`, sem gravar nada e sem iniciar monitoramento. O plano **simula o estado acumulado** na ordem da aplicação (chaves de push, endpoints, status pages):

1. **Registros indisponíveis:** qualquer um dos três indisponível dá 503. `managedendpoint` ganha o sinal `IsManagedUnavailable()`, marcado no `Load` que publica um mapa vazio depois de falhar ao listar, como `statuspage` e `pushkey` já fazem.
2. **Tokens em uso** (antes das chaves), com os tokens de:
   - external endpoints e `push.endpoints` do arquivo de configuração;
   - **todos** os endpoints gerenciados, lidos da definição armazenada (inclusive em conflito ou inválidos, cujo `Parsed` está vazio);
   - todos os endpoints do arquivo de backup.
3. **Chaves de push:**
   - **formato:** hash com 64 hexadecimais minúsculos, nome de 1 a 64 caracteres sem espaços nas pontas e dica de 0 a 4 caracteres `[A-Za-z0-9_-]`; fora disso, `skip`;
   - **mesma chave:** mesmo nome e hash de uma chave existente dá `unchanged`;
   - **nome em uso:** nome de uma chave do YAML ou de outra chave com hash diferente dá `skip` ("name in use");
   - **hash em uso:** o hash de uma chave do YAML, de outra chave existente ou planejada, ou de qualquer token do passo 2 dá `skip` ("token hash in use"). Uma chave global nunca pode aceitar o token de um endpoint.
4. **Endpoints**, com um `RestoreContext` que acumula chaves e tokens planejados:
   - **chave divergente:** `key` diferente da chave calculada da definição dá `skip`;
   - **segredo mascarado:** `HasMaskedSecret` percorre os mesmos lugares em que `MaskSecrets` escreve e dá `skip` ("masked secret") quando encontra `managedendpoint.Mask`. A validação do restore **não** chama `RestoreMaskedSecrets`. Os lugares são:
     - headers sensíveis;
     - a senha e os parâmetros sensíveis da query da `url` (o campo inteiro não é `********`);
     - `client.oauth2.client-secret`, `ssh.password` e `ssh.private-key`;
     - `token` e `push.token`;
     - qualquer folha de `alerts[].provider-override`;
   - **Push sem token:** `type: push` sem `token` dá `skip` ("push endpoint without token"). O restore nunca gera tokens;
   - **validação:** `ValidateRestore` chama `Prepare(raw, Context)` diretamente, com os tokens e chaves planejados (sem `prepare(raw, key)`, que restaura máscaras, e sem `withGeneratedPushToken`). Erro dá `skip` com a mensagem, inclusive para token repetido dentro do arquivo e token igual ao hash de uma chave planejada;
   - **conflito com o YAML:** chave do arquivo de configuração dá `skip`;
   - **item existente:** compara as definições normalizadas (parse e serialização sem padrões, depois de aplicar `disableEndpoints`). Igual dá `unchanged`. Diferente dá `update` com `overwrite` e `skip` ("already exists") sem ele;
   - **update com restrição:** uma troca de tipo dá `skip` ("type cannot change"); nome ou grupo diferentes com a mesma chave dão `update` com o motivo "name or group changes", e seguem o caminho atual de atualização de nome e grupo sem trocar a chave.
   - **`disableEndpoints`:** grava `enabled: false` nas definições criadas ou atualizadas.
5. **Status pages:**
   - **conflito com o YAML:** `ValidateRestore` verifica o conflito de slug com o YAML explicitamente, porque `parseSubmitted(raw, slug)` não o faz na atualização. Slug do arquivo de configuração dá `skip`, inclusive para uma página gerenciada existente;
   - **validação:** a validação de `parseSubmitted(raw, "")` dá `skip` com a mensagem em caso de erro;
   - **item existente:** compara a página normalizada (parse, padrões e serialização dos dois lados, como faz o store), com `unchanged`, `update` ou `skip` como nos endpoints;
   - **avisos de seleção:** calculados com os endpoints existentes mais os planejados.
6. **Avisos gerais:** quantos endpoints habilitados vão começar a ser monitorados e quantos têm alertas.

**Resposta:**
```json
{"summary":{"create":3,"update":1,"unchanged":5,"skip":2},
 "notices":{"monitoringStarts":3,"withAlerts":2},
 "fingerprint":"<hex>",
 "items":[{"type":"endpoint","id":"jobs_backup","action":"update","reason":"name or group changes","warnings":[]}]}
```

**Fingerprint:** SHA-256 dos bytes de uma struct Go serializada por `encoding/json` (campos em ordem fixa, sem mapas), com vetor de teste. Os itens seguem a ordem fixa de tipo `pushKey`, `endpoint`, `statusPage` e, dentro do tipo, o id. A struct contém:
- o SHA-256 dos bytes do texto claro: o `json.RawMessage` de `file` sem cifra, ou os bytes decifrados, sem remarshal;
- `overwrite` e `disableEndpoints`;
- `statuspage.Generation()`;
- para cada item: tipo, id, ação, versão atual e SHA-256 da definição atual do destino.

**Definição do plano:** cada item `create`/`update` guarda a definição normalizada que será aplicada (já com `disableEndpoints`). A aplicação usa só essa definição, nunca o YAML cru do arquivo.

### D5. Aplicação do restore
`POST /api/v1/admin/restore` recebe o corpo da prévia mais `"fingerprint"`:
- **Antes de aplicar:** recalcula o plano, com 409 e nada aplicado se o fingerprint mudou, e 503 se um registro estiver indisponível.
- **Aplicação:** percorre os itens `create`/`update` na ordem do plano, com a definição guardada no plano, chamando os serviços com o autor do restore:
  - **chaves de push:** `pushkey.Restore`;
  - **endpoints:** `managedendpoint.Service.RestoreCreate`, igual a `Create` sem `withGeneratedPushToken` (o plano já recusou Push sem token), ou `Update` com a versão do plano. O `Update` com uma definição sem máscaras torna o `RestoreMaskedSecrets` interno inócuo;
  - **status pages:** `statuspage.Service.Create`, ou `Update` com a versão do plano.
- **Lock:** não há lock externo. Cada chamada usa o próprio `TryBeginChange`.
  - Com o primeiro `ErrCycleInProgress`, os itens restantes são `skipped` ("configuration reload in progress").
  - A resposta é 200 com o resultado parcial, e repetir o restore depois da recarga completa o que faltou.
- **Resultado de cada item:**
  - `created`, `updated`, `unchanged` ou `skipped`, com o motivo do plano;
  - `failed` com a mensagem, por exemplo `ErrManagedEndpointVersionMismatch` ("changed during the restore") ou erro do store;
  - a falha de um item não interrompe os demais.
- **Depois dos itens:** os avisos de seleção das páginas aplicadas são recalculados e voltam no resultado, e o log registra o resumo com o autor. `adminbackup` não importa `api`: é o handler em `api/admin_backup.go` que limpa o cache `endpoint-status-*` depois de `Apply`, como `api/admin.go` já faz.
- **`pushkey.Restore(name, tokenHash, hint, author, isEndpointToken func(hash) bool)`:**
  - segura `pushkey.mutex` durante a verificação, a gravação e a publicação do snapshot;
  - a verificação de tokens de endpoints recebe um callback injetado a partir de `managedendpoint`, que segura o `statesMutex` durante a verificação e a publicação, evitando a corrida com um `managedendpoint.Create` simultâneo;
  - a ordem dos locks é sempre `pushkey.mutex` → `statesMutex`, e `managedendpoint` nunca pega `pushkey.mutex`.
- **Tempo:** com os limites de D1, um restore em SQLite fica bem abaixo do timeout de 60 s do nginx. Se o cliente receber 504 mesmo assim, uma nova prévia mostra o que já foi aplicado como `unchanged`.

### D6. Tela Backup
Rota `/admin/backup` (`meta.admin`), com a rota SPA registrada no servidor. A página mostra `AdminTabs` no topo, como nas listas, com a aba nova **Backup** depois de "Push keys", e o conteúdo usa o layout dos formulários.

- **Download backup:**
  - quantidades (endpoints, status pages, chaves web);
  - **Encrypt with a password** desligado por padrão, com senha, confirmação e contagem de bytes;
  - sem senha, aviso de segredos em texto claro;
  - **Download** usa `adminApi.download` (Blob, nome do `Content-Disposition` com nome de reserva, 401 com `notifyUnauthorized`, erro JSON ou texto no 413 ou 422).
- **Restore:**
  - **arquivo:** qualquer arquivo, lido no navegador com limite de `2.7 * 1024 * 1024` bytes (o envelope de um texto claro de 2 MiB tem cerca de 2,67 MiB); o formato é detectado pelo `format` depois do `JSON.parse`;
  - **opções:** senha quando cifrado, **Overwrite existing items** e **Restore endpoints as disabled**;
  - **prévia:** **Preview** mostra os avisos (monitoramento e alertas), o resumo, a tabela (tipo, id, ação com cor, motivo, avisos) e o filtro por ação;
  - **aplicação:** **Restore** fica desabilitado durante a prévia e sem prévia válida. A confirmação traz "N items will be created and M updated", com as quantidades do plano. Depois vem a tabela de resultados;
  - **invalidação:** trocar ou reler o arquivo, a senha ou as opções invalida a prévia.
- **Erros:** um mapeador próprio de erros de backup e restore: 413 (texto do Fiber ou do proxy), 422 (limites) e 429 (senha ou semáforo, com `Retry-After`). O mapeador atual transforma 429 em "Too many endpoint tests".
- **Rota:** `/admin/backup` em `web/app/src/router/index.js` (`meta.admin`) e na lista de `AdminTabs.vue`.
- **Funções puras** em `utils/adminBackup.js` (formato, nome, bytes da senha, resumo, filtros, contagens e mensagens de erro), testadas com `node --test`.

### D7. Organização do código
- **Pacote `adminbackup`:**
  - `format.go`, com tipos e decodificação estrita;
  - `crypto.go`, com envelope, semáforo e AAD;
  - `backup.go`, com `Build`;
  - `restore.go`, com `Plan` e `Apply`;
  - dependências por interfaces pequenas (leitores dos stores, validadores e escritores dos serviços), para os testes.
- **`managedendpoint`:**
  - `ValidateRestore(raw, ctx *RestoreContext)`, chamando `Prepare` com o contexto de restore, sem `RestoreMaskedSecrets`;
  - `RestoreCreate` (sem geração de token), `IsManagedUnavailable()` e `StoredPushTokens()` (tokens lidos das definições armazenadas de todos os gerenciados);
  - `HasMaskedSecret(document)`, a partir dos caminhos de `MaskSecrets`;
  - `WithPushTokensLocked(func(hashes map[string]bool) error)`.
- **`statuspage`:** `ValidateRestore(raw)`, com o conflito de slug com o YAML, e `SelectionWarningsWith(page, refs)`.
- **`pushkey`:** `Restore`, `IsNameInUse` e `IsHashInUse`.
- **`security`:** construtor exportado de um limitador de falhas independente, com a janela como parâmetro.
- **`api/admin_backup.go`:**
  - rotas de restore num grupo `/v1/admin/restore` com autorização e CSRF, mas **sem** o limite de 256 KB de `adminRequestProtection` (a proteção ganha uma variante com o limite como parâmetro, 3,5 MiB aqui); `POST /backup` continua com 256 KB;
  - verificação de `application/json` e do tamanho;
  - middleware de IP do cliente;
  - rota SPA `/admin/backup` em `api/api.go`.

## Risks / Trade-offs

- **Arquivo sem senha com segredos.** Mitigação: aviso na tela, na documentação e no nome do arquivo; `no-store`; senha opcional oferecida em destaque.
- **Monitoramento e alertas começam logo no destino** (checagens para a rede do destino, alertas para webhooks reais). Mitigação: avisos na prévia e a opção "Restore endpoints as disabled".
- **Restore parcial** (falha ou recarga no meio). Aceito: resultado por item e repetição idempotente.
- **Mudança entre prévia e aplicação.** Mitigação: fingerprint com o conteúdo, as versões e a geração da configuração (409), mais a versão esperada por item.
- **Hash de chave vindo de token fraco.** A chave de outra instalação do fork foi gerada com 32 caracteres aleatórios. Um arquivo editado à mão pode trazer um hash qualquer, que só autoriza quem conhece o token. Isso fica documentado, e o requisito de geração passa a valer só para as chaves criadas pela web.
- **Padrões de status page mudando entre versões do Gatus** podem transformar `unchanged` em `update`. Aceito, e a prévia mostra.
- **Várias instâncias no mesmo banco.** As outras só veem os itens depois de recarregar ou reiniciar, como nas escritas atuais.

## Migration Plan

Sem migração de esquema. O recurso só existe com `admin.enabled`. Voltar para uma versão anterior remove a aba, e os cadastros restaurados continuam normais.

## Open Questions

Nenhuma.
