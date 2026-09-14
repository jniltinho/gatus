## Context

Estado do código (`master` depois do PR #2, com a administração de endpoints), conferido antes desta proposta e na revisão de QA:

- **Rotas e autenticação:**
  - Em `api/api.go`, as rotas livres são registradas primeiro no grupo `/api`: `/api/v1/config`, badges, uptimes, response-times e o `POST` de external endpoints.
  - Depois vêm as rotas da SPA (`/`, `/endpoints/:key`, `/suites/:key` e `/admin*`), servidas por `SinglePageApplication` sem autenticação, `/health`, `/css/custom.css`, o redirecionamento de `/index.html` e o middleware de arquivos estáticos.
  - Por último, `ApplySecurityMiddleware` com `router.Use` no grupo `/api`. Qualquer `/api/*` sem handler livre anterior passa por ele: com basic, o `Unauthorized` responde 401 com `WWW-Authenticate: Basic` (`security/config.go`), o que abre o prompt do navegador.
  - `SinglePageApplication` faz `template.ParseFS` a cada requisição e força 200 (`api/spa.go`). `app.Get` também atende `HEAD`.
- **Frontend e OIDC:**
  - `main.js` monta o app sem esperar o `vue-router` resolver a rota.
  - `App.vue` busca `/api/v1/config` (livre) em `onMounted`, repete a cada 10 minutos e, se `oidc && !authenticated`, troca todo o conteúdo pela tela de login. O `Tooltip` global espera `duration` em nanossegundos, `conditionResults` e `errors`.
  - `EndpointCard` mostra `hostname` e navega para `/endpoints/:key`.
  - O vue-router decodifica os parâmetros: `/status/a%2Fb` vira `slug = "a/b"`.
  - A Home agrupa por `endpoint.group` no cliente e não tem filtro de grupo na URL. `web/app/public/index.html` usa `lang="en"`.
- **Campos expostos hoje:**
  - `endpoint.Status`: `name`, `group`, `key`, `results` e `events`.
  - `endpoint.Result` em JSON: `status` (código HTTP), `hostname`, `duration` (ns), `errors`, `conditionResults` (condições com valores resolvidos), `success` e `timestamp`.
  - `ui.hide-hostname`, `hide-errors`, `hide-conditions`, `hide-url` e `hide-port` são opcionais por endpoint e aplicados antes de gravar.
  - `EndpointStatuses` acrescenta os endpoints das instâncias `remote`, buscadas por HTTP.
- **Storage:**
  - `api/cache.go` tem um gocache global de 100 entradas FIFO com TTL de 10 s, usado por `EndpointStatuses`.
  - O store SQL só tem `writeThroughCache` com `storage.caching: true` (padrão `false`); na inserção, as entradas da chave são relidas dentro da transação, com teto de 10 000 entradas e TTL de 10 minutos.
  - `GetEndpointStatusByKey` no SQL abre uma transação e faz três consultas (id, 12 colunas de resultados incluindo `hostname`, `ip` e `errors`, e condições). Erros de leitura de resultados são registrados no log e engolidos; endpoint sem registro devolve `ErrEndpointNotFound`.
  - Com SQLite, `SetMaxOpenConns(1)`: todas as leituras disputam a conexão com `InsertEndpointResult` do watchdog.
  - `GetUptimeByKey` devolve `0` tanto sem execuções quanto com 0% de sucesso. O SQL compara `hour_unix_timestamp >= from.Unix()`; a memória trunca `from` na hora. O SQL funde entradas com mais de 48 h em entradas diárias.
  - Janelas usadas pelos badges: `from = now-24h`, `now-7d` e `now-30d`, `to = now` (`api/raw.go`, `api/badge.go`).
- **Implementado no marco 1 antes desta revisão** (branch `feat/public-status-pages`):
  - `storage/store/common/managed_status_page.go`, `storage/store/managed_status_page.go` e `storage/store/sql/managed_status_pages.go`, com testes em SQLite e PostgreSQL;
  - `storage/store/common/endpoint_uptimes.go` (`EndpointUptimes{Last24Hours, Last7Days, Last30Days *float64}`), `storage/store/endpoint_uptime_batch.go` (`EndpointUptimeBatchReader.GetUptimesByKeys(keys, now)`), `storage/store/sql/endpoint_uptime_batch.go` (uma consulta com `SUM(CASE ...)`) e `storage/store/memory/endpoint_uptime_batch.go` (`HourlyStatistics` sob `RLock`, `nil` sem execuções).
- **Fiber:**
  - `fiber.New` não define `ProxyHeader` nem `EnableTrustedProxyCheck`, então `c.IP()` devolve o IP da conexão.
  - `Ctx.Get` e `RequestHeader.Peek` devolvem só a primeira linha de um header repetido; `RequestHeader.PeekAll` devolve todas.
  - O middleware `limiter` do Fiber v2 cria, sem `Storage`, um armazenamento em memória com goroutine de limpeza sem parada e mapa sem teto, e envia `X-RateLimit-*` sem opção de desligar. Não é usado (D8).
  - `controller.Handle` define `ReadTimeout` e `WriteTimeout` de 15 s, que não cancelam o handler.
- **Deploy do usuário:** Docker com a porta publicada em `127.0.0.1` do host e nginx no host (`sqlite/docker-compose.yml`, `postgres/docker-compose.yml`, servidor `gatus-validacao`). Dentro do container, a conexão chega do gateway da bridge do Docker, não de `127.0.0.1`. `web.address` padrão é `0.0.0.0`.
- **Administração (fork):**
  - registro copy-on-write em `managedendpoint` (que importa `config`);
  - `lifecycle.TryBeginChange` (`TryRLock`);
  - `AdminMiddleware` e `adminRequestProtection` (CSRF, 256 KB, tipos de mídia); o CSRF compara `Origin`/`Referer` com `allowed-origins` ou com a origem derivada de `Host` e TLS/`X-Forwarded-Proto`, sem `Hostname()` nem `Protocol()`;
  - `ETag` e `If-Match`; o padrão "apply antes do commit" dos endpoints existe porque iniciar o monitoramento pode falhar;
  - `ErrStorageNotSupported` cai em 500 em `adminServiceError`.
  - A API é recriada a cada `start` (`go controller.Handle(cfg)`); variáveis de pacote sobrevivem ao ciclo. A recarga chama `stop`, `save`, `store.Get().Close()`, `initializeStorage` (que troca a variável global do store sem sincronização) e `start`.
- **Outros tipos de monitoramento:**
  - External endpoints têm `group`, `name`, `Key()` e `Enabled`, e seus resultados são enviados pela API.
  - Suites têm outro modelo de status (`suite.Status`).
  - Watchdog e API não escrevem `Name`, `Group` nem `Enabled` de endpoints.
- **Configuração:** `parseAndValidateConfigBytes` usa `yaml.Unmarshal` sem modo estrito, então a imagem do upstream ignora uma seção desconhecida. A validação roda antes de `managedendpoint.Load` e, na recarga, pode ser descartada.
- **E2E:** o `agent-browser` 0.37.1 tem sessões isoladas (`AGENT_BROWSER_SESSION`), `network route <url> --body <json>` e `network requests`. O desafio basic trava o headless (`test/e2e/admin.sh`).

## Goals / Non-Goals

**Goals:**
- Várias páginas públicas por slug, cada uma com um subconjunto de endpoints selecionado por grupo ou chave, abertas sem login com basic ou OIDC.
- Não expor nada além de nome, grupo, estado, histórico resumido e uptime, e não mudar a proteção das rotas existentes.
- Nenhum caminho público (API, HTML ou link malicioso) pode produzir 401 com `WWW-Authenticate`.
- Custo no storage limitado e independente do número de visitantes; resistência a abuso (enumeração e excesso de requisições) que não derrube a página para todos.
- Páginas definidas no YAML e gerenciadas pela web, com as mesmas garantias da administração de endpoints.
- Código novo em arquivos novos, e o fork independente do caminho de módulo do upstream.

**Non-Goals:**
- Domínio próprio por página, incidentes, manutenções programadas ou avisos por página, anúncios globais, logo ou CSS customizado por página (fase 2).
- Suites, endpoints de suites e instâncias `remote` nas páginas.
- Estado "em manutenção" derivado das janelas de manutenção.
- Página protegida por senha própria, ou controle de acesso por página. O slug não é segredo.
- Indexação por buscadores (`indexable`): as páginas são sempre `noindex`.
- CORS na API pública e CSP completa (fase 2).
- Métricas Prometheus novas.
- Mudar as rotas públicas já existentes (badges, uptime, response-times e `/api/v1/config`).
- Sincronização em tempo real entre várias instâncias no mesmo PostgreSQL.

## Decisions

### D0. Decisões do dono e caminho do módulo
Tomadas antes do marco 1 (achado 13 da revisão):
- `status-pages.enabled` tem padrão `true`. Mitigação: uma página gerenciada criada sem `enabled` nasce **desabilitada**, pela API e pela UI; as páginas do YAML continuam com padrão `true`.
- Com `admin.enabled: false`, as páginas gerenciadas **continuam publicadas**, como os endpoints gerenciados. A carga registra no log os slugs publicados por origem. `status-pages.enabled: false` despublica todas.
- Seleção **por grupo e por chave**. Endpoints novos de um grupo selecionado entram sozinhos; o formulário de endpoints mostra em quais páginas públicas o endpoint vai aparecer (D10).
- Um endpoint incluído só pela chave publica o **nome real do grupo**; endpoints sem grupo ficam numa seção com `name: ""`, exibida como "Outros serviços".
- Página pública mínima: logo e `ui.header`, título e descrição; sem anúncios, `Social`, `ui.buttons` nem "Powered by".
- O módulo Go do fork passa de `github.com/TwiN/gatus/v5` para `gatus/v5`, num PR próprio antes do marco 1 (marco 0). Os caminhos de import citados nesta change usam `gatus/v5`. Consequências aceitas: `go install`/`go get` do fork deixam de funcionar, o `goimports` trata o caminho como biblioteca padrão ao agrupar imports, e a sincronização com o upstream passa a exigir reescrita de imports.

### D1. Modelo da página
Campos da página:

| Campo | Regra |
|-------|-------|
| `slug` | `^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`; único entre YAML e gerenciadas; imutável; reservados: `options`, `validate`, `new`, `preview`, `exposure` |
| `title` | obrigatório; 1 a 100 runas depois de `TrimSpace` |
| `description` | opcional; até 1000 runas; texto puro |
| `groups` | até 50 nomes de grupo, cada um com 1 a 200 runas, comparados exatamente depois de `TrimSpace` dos dois lados (como a `key` do upstream); sem duplicados |
| `endpoints` | até 200 chaves no formato de `key.ConvertGroupAndNameToKey`, normalizadas para minúsculas; sem duplicados |
| `enabled` | booleano; padrão `true` no YAML e `false` nas gerenciadas (D0) |

- A página precisa ter pelo menos um item em `groups` ou em `endpoints`.
- A descrição é texto puro, renderizada por interpolação do Vue (nunca `v-html` nem markdown).
- Um grupo ou chave sem correspondência no momento não invalida a página, porque o endpoint pode ser criado depois. Os avisos saem da validação da administração e de `statuspage.Load` (D4), nunca da validação do arquivo.

**Ordem de exibição:**
- As seções seguem a ordem de `groups`.
- Grupos que só aparecem por `endpoints` vêm depois, em ordem alfabética, com o nome real.
- Endpoints sem grupo ficam numa última seção com `name: ""` ("Outros serviços" na tela).
- Dentro de cada seção, os endpoints são ordenados pelo nome, sem diferenciar maiúsculas.

**Alternativas:**
- Seções nomeadas com uma lista manual de monitores, como no Kuma. Adiada: os grupos já existem e incluem endpoints novos automaticamente. Um campo `sections` pode ser acrescentado depois sem quebrar o formato.
- Comparar grupos sem diferenciar maiúsculas. Rejeitada: o dashboard trata `Core` e `core` como grupos diferentes, e o formulário escolhe grupos de uma lista.
- `indexable`. Removido (achado 22): título e descrição do HTML vêm da UI global e o conteúdo é renderizado no cliente, e responder com cabeçalhos diferentes por página publicada revelaria a existência da página na rota HTML (D5).

### D2. Endpoints publicáveis
A seleção usa um índice imutável de chave, nome e grupo, montado a partir de:
- `cfg.Endpoints` e `cfg.ExternalEndpoints` habilitados, na carga do ciclo;
- o snapshot copy-on-write de `managedendpoint` (gerenciados válidos, habilitados e sem conflito), lido a cada montagem (a leitura do ponteiro atômico é barata).

Só são lidos `Name`, `Group`, `Enabled` e `Key()`, que o watchdog não altera. Objetos em monitoramento nunca são serializados (a data race corrigida na administração vinha disso).

Ficam de fora:
- suites e endpoints de suites (outro modelo de status);
- instâncias `remote` (recurso alpha, com busca HTTP síncrona);
- endpoints desabilitados.

Alternativa: selecionar pelas chaves existentes no store (`GetAllEndpointStatuses`). Rejeitada: incluiria históricos órfãos e exigiria ler tudo a cada montagem.

### D3. Configuração no YAML
```yaml
status-pages:
  enabled: true                   # padrão true; false despublica todas (o catch-all 404 continua)
  trusted-proxies: ["172.30.0.1/32"] # conexões cujo X-Forwarded-For é aceito (ver Migration Plan)
  rate-limit: 120                 # respostas 404 por minuto por IP; 0 desliga
  pages:
    - slug: infra
      title: Infraestrutura
      description: Serviços públicos da plataforma
      groups: [core, database]
      endpoints: [external_cdn]
```
- Pacote `config/statuspage` com `Config{Enabled *bool, TrustedProxies []string, RateLimit *int, Pages []*Page}`.
- `ValidateStatusPagesConfig` roda depois de `ValidateAdminConfig` e `ValidateUniqueKeys` e é **só estrutural**.
- A configuração é inválida com:
  - slug inválido, reservado ou duplicado;
  - título vazio ou longo demais;
  - página sem seleção ou acima dos limites de D1;
  - entrada de `trusted-proxies` que não seja IP nem CIDR (um IP sem máscara vira `/32` ou `/128`);
  - `rate-limit` negativo.
- A seção é opcional; sem ela, valem os padrões e não há páginas no YAML.
- As variáveis de ambiente são expandidas como no restante do arquivo.

Alternativa: uma lista direto em `status-pages:`. Rejeitada porque não deixaria lugar para as opções globais sem quebrar o formato depois.

### D4. Páginas gerenciadas, registro e carga
**Tabela `managed_status_pages`** (já implementada):
- `slug` único;
- `definition`: YAML da definição enviada, sem os valores padrão;
- `version`, começando em 1;
- `created_at` e `updated_at`, em milissegundos, como em `managed_endpoints`;
- `updated_by`.

É criada com `CREATE TABLE IF NOT EXISTS`, chamada a partir de `createSchema`; um erro interrompe a inicialização. Vale para SQLite e PostgreSQL.

**Store:** a interface `ManagedStatusPageStore` tem listar, obter, criar, alterar com versão esperada e remover com versão esperada, com o hook `apply` do `ManagedEndpointStore`. As páginas não usam `apply` para publicar (D9); o hook fica disponível para validações dentro da transação. O store em memória não a implementa.

**Registro:**
- O pacote `statuspage` guarda um snapshot copy-on-write (`atomic.Pointer`) com o estado de cada página: origem `config` ou `admin`, definição validada, versão do banco, conflito, erro e **revisão**.
- A revisão vem de um contador de pacote monotônico (`atomic.Uint64`), atribuída a cada página publicada numa carga ou numa escrita; nunca é a versão do banco (D7).
- A geração do ciclo (`atomic.Uint64`) é incrementada a cada `Load`.
- `statuspage.Load(cfg)` roda em `initializeStorage` logo depois de `managedendpoint.Load`, **independentemente de `admin.enabled`** (D0).
- Na carga, depois de montar o índice de D2, são registrados:
  - os avisos de grupo ou chave sem correspondência (achado 11);
  - uma linha com os slugs publicados por origem.
- `status-pages.enabled: false` despublica todas; o registro continua carregado para a administração.

**Situações especiais:**
- **Conflito de slug:** o YAML prevalece. A gerenciada fica marcada como em conflito e não é publicada; removê-la apaga só a definição. Se o YAML deixar de usar o slug, ela volta a ser publicada na próxima carga.
- **Gerenciada inválida na carga** (por exemplo, limites alterados): fica marcada com o erro, não é publicada e aparece na administração.
- **Falha ao listar as gerenciadas:** as páginas do YAML são publicadas, a origem gerenciada fica marcada como indisponível, o erro vai para `logr.Errorf` e a lista da administração mostra um banner. Nenhuma gerenciada é publicada até a próxima carga.
- **Storage sem suporte** (memória): só as páginas do YAML; a administração de status pages responde 501 nas escritas (a administração já exige SQL, então isso só ocorre em testes).

Alternativas:
- Escrever no YAML. Rejeitada pelos mesmos motivos da D1 do admin: perde comentários e `${VAR}`, falha com arquivo somente leitura e dispara o hot-reload global.
- Despublicar as gerenciadas quando `admin` é desligado. Rejeitada pelo dono (D0).

### D5. Rotas públicas fora da autenticação
Registradas em `api/api.go`, no bloco livre, antes do middleware de arquivos estáticos e de `ApplySecurityMiddleware`:

**`GET /api/v1/status-pages/:slug`** (só com `status-pages.enabled`):
- responde 200 com o payload de D6 ou o 404 idêntico;
- aplica o limitador de D8.

**Catch-all `unprotectedAPIRouter.All("/v1/status-pages")` e `All("/v1/status-pages/*")`, sempre registrado:**
- inclusive com `status-pages.enabled: false`;
- cobre slug vazio, `/a/b`, `/infra/extra` e outros métodos;
- responde o mesmo 404 da rota específica, passando pelo limitador como um 404;
- impede que qualquer caminho de `/api/v1/status-pages` chegue ao middleware de segurança.

**`GET/HEAD /status/:slug` e `/status/*`, no bloco da SPA, sempre registrados:**
- respondem **sempre 200** com o HTML da SPA, qualquer que seja o slug; a rota HTML não revela se a página existe, por isso não passa pelo limitador;
- usam um helper novo `renderSPA` (`api/spa_render.go`) que faz o parse do template uma vez por ciclo, sem mexer em `SinglePageApplication`;
- enviam `Cache-Control: no-cache` (o template depende do cookie de tema) e os cabeçalhos de D5.1.

**D5.1 Cabeçalhos das rotas públicas:** `X-Robots-Tag: noindex, nofollow`, `X-Content-Type-Options: nosniff` e `Referrer-Policy: strict-origin-when-cross-origin`. A API pública também envia `Vary: Accept-Encoding` explicitamente em todas as respostas, com ou sem compressão, para o valor não depender do tamanho do corpo. Sem `frame-ancestors`: a página pode ser embutida em iframe (TV de NOC); uma CSP completa exigiria mexer no script inline do template do upstream. Sem CORS na API pública (fase 2).

Os handlers não leem sessão nem credenciais, e a resposta é a mesma para anônimos e autenticados. Nenhuma rota protegida muda.

**No frontend:**
- Rotas `/status/:slug([a-z0-9-]{1,64})` e `/status/:pathMatch(.*)*`, ambas com `meta.public`; a segunda mostra "Página não encontrada" sem chamar a API.
- A view valida o slug com a regex de D1 antes do `fetch` e usa `encodeURIComponent`.
- Layout reativo (achado 14):
  - `isPublic = computed(() => route.meta.public === true)`, com `Loading` até `router.isReady()`;
  - `<PublicLayout>` com `v-if="isPublic"`; o dashboard e a tela de login do OIDC ficam em `v-else`, então o login só aparece fora das rotas públicas;
  - `/api/v1/config` só é buscada na primeira rota não pública (`watch(isPublic, ...)`), e só então começa o intervalo de 10 minutos;
  - o layout público define `document.documentElement.lang = 'pt-BR'` e restaura o valor ao sair;
  - não mostra `ui.buttons`, o link Admin, anúncios, `Social` nem `Settings`.
- A view pública só chama `/api/v1/status-pages/:slug`, não navega para `/endpoints/:key` e usa um tooltip próprio em `components/public/`.
- `watch(() => route.params.slug)` limpa os dados, refaz a busca e reinicia o intervalo.

Alternativas:
- Renderizar a página pública no servidor, fora da SPA. Rejeitada por duplicar tema, Tailwind e componentes.
- Responder 404 na rota HTML para página não publicada. Rejeitada na revisão (achado 10): seria um oráculo de enumeração, inclusive por `HEAD`, e exigiria o limitador também no HTML.
- Não registrar nada com `enabled: false`. Rejeitada (achado 2): os caminhos cairiam no middleware de segurança e responderiam 401.

### D6. Payload público
A resposta usa tipos próprios em `statuspage`, nunca `endpoint.Status` ou `endpoint.Result`:
```json
{
  "slug": "infra", "title": "Infraestrutura", "description": "...",
  "status": "operational", "updatedAt": "2026-09-14T15:00:00Z", "truncated": false,
  "groups": [{
    "name": "core", "status": "operational",
    "endpoints": [{
      "name": "api", "status": "up",
      "uptime": {"24h": 0.9993, "7d": 0.998, "30d": null},
      "results": [{"timestamp": "2026-09-14T14:59:00Z", "success": true, "durationMs": 123}]
    }]
  }]
}
```

**Fica fora:** chave, URL, hostname, IP, porta, código HTTP, `dns_rcode`, erros, condições, eventos, expiração de certificado ou domínio, alertas, `extra-labels` e origem. O código HTTP e os erros podem revelar detalhes internos, e o `ui.hide-*` do upstream é opcional.

**Regras:**
- `updatedAt`: instante da montagem, no relógio do servidor (não o horário do último resultado).
- Resultados: os últimos `min(50, storage.maximum-number-of-results)`, do mais antigo para o mais recente.
- Estado do endpoint: `up` se o último resultado teve sucesso, `down` se não teve, `unknown` sem resultados. Um endpoint selecionado sem nenhum registro no store (por exemplo, gerenciado recém-criado) é `unknown`, com `results` vazio e uptime `null`.
- Estado do grupo e da página, ignorando `unknown`:
  - `operational` se todos estão `up`;
  - `down` se todos estão `down`;
  - `degraded` se há dos dois;
  - `unknown` se nenhum é conhecido.
- Uptime: `null` quando não há execuções no período. Com SQL, depois de 48 h as entradas horárias viram diárias, então as bordas de 7d e 30d são aproximadas por dia, como nos badges.
- No máximo 200 endpoints por página, na ordem de D1. Acima disso, vão os 200 primeiros com `truncated: true` e um aviso no log a cada montagem.
- Qualquer erro de leitura do storage (exceto a ausência do endpoint) resulta em `503 {"error":"status page temporarily unavailable"}`, sem o texto do erro; o erro vai para o log com o slug.
- 404: `{"error":"status page not found"}`.

Alternativa: incluir a chave para ligar a página aos badges. Rejeitada pelo princípio de expor o mínimo; a chave pode entrar depois se houver uso.

### D7. Montagem, cache e custo
**Captura da requisição (achados 3 e 4):**
- Na entrada, a requisição lê uma única vez do snapshot atômico `(slug, revisão, geração, definição)`.
- Página não publicada: 404 sem tocar no storage.
- A chave do cache e do `singleflight` é `slug|revisão|geração`, e a montagem usa **só** a definição capturada. Uma escrita gera revisão nova, então a requisição seguinte nunca se junta a uma montagem anterior nem lê o cache antigo.

**Cache:**
- gocache próprio (não o de `api/cache.go`), com TTL de 30 s para payloads e de 5 s para o 503 de erro de leitura (cache negativo por revisão);
- `Load` e as escritas não precisam invalidar (chave nova); `DeleteKeysByPattern` do slug roda só para liberar memória;
- nunca serve payload de revisão anterior.

**Montagem:**
- síncrona, na goroutine da requisição, com `singleflight.Do` (sem refresh em segundo plano);
- dentro do `Do`, só a montagem líder pede vaga no semáforo global de 4 montagens públicas, com `select` e timeout de 5 s; no timeout, todos os participantes daquela chave recebem 503 genérico **sem** cache. Assim, 100 requisições simultâneas da mesma revisão ocupam uma única vaga;
- resolve o leitor **uma vez por montagem**, no início, por `var getSummaryReader = func() (summaryReader, bool)` (por padrão, `store.Get()` convertido em `EndpointSummaryBatchReader`), substituível nos testes. Nenhum leitor fica guardado entre montagens, porque a recarga fecha e troca o store; `summaryReader` é a interface estreita com `GetEndpointSummaries`.

**Leitura em lote (achados 5 e 6):**
- Interface `EndpointSummaryBatchReader` em `storage/store/endpoint_summary_batch.go`:
  ```go
  GetEndpointSummaries(keys []string, maximumResults int, now time.Time) (map[string]*common.EndpointSummary, error)
  ```
  com `common.EndpointSummary{Results []common.ResultSummary, Uptimes common.EndpointUptimes}` e `common.ResultSummary{Timestamp time.Time, Success bool, Duration time.Duration}`. Chaves sem registro ficam fora do mapa (→ `unknown`); qualquer outro erro é devolvido. A presença no mapa é decidida pela existência do endpoint no store (status ou resultados), nunca pelo uptime: um endpoint com resultados e sem execuções no período vem com uptimes `nil`. (`GetUptimesByKeys` omite chaves sem execuções em 30 dias; `GetEndpointSummaries` não pode herdar essa regra.)
- **SQL:** uma transação de leitura por montagem (sempre desfeita com `Rollback`, nada é gravado), com três consultas:
  - ids das chaves existentes em `endpoints`, que decidem a presença no mapa;
  - resultados, só com `success`, `duration` e `timestamp`, por `ROW_NUMBER() OVER (PARTITION BY endpoint_id ORDER BY endpoint_result_id DESC)` com `rn <= maximumResults` e `endpoint_key IN (...)` (window functions existem no SQLite embutido do `modernc.org/sqlite` e no PostgreSQL);
  - uptime, a mesma consulta `SUM(CASE ...)` de `GetUptimesByKeys`, extraída para um helper que recebe a transação.
- **Memória:** sob um único `RLock`, copia os últimos resultados resumidos e soma `HourlyStatistics` pelo helper `uptimesOf`, compartilhado com `GetUptimesByKeys` (sem `RLock` recursivo).
- `EndpointUptimeBatchReader` continua (já implementado) e é usado nos testes que comparam com `GetUptimeByKey` **no mesmo store**, com as mesmas bordas (`>= now-Δ` e `<= now` no SQL; hora truncada na memória).
- 200 chaves ficam abaixo do limite de parâmetros do SQLite e do PostgreSQL.
- A pré-visualização da administração usa um semáforo próprio de 1 vaga, com o mesmo timeout e sem cache, para não tirar vagas da página pública.

**Custo e cabeçalhos:**
- No máximo uma montagem por revisão de página a cada 30 s, qualquer que seja o número de visitantes; três consultas numa transação por montagem no SQL.
- Com página publicada: `Cache-Control: no-cache`. O cache é do servidor; um cache HTTP no navegador ou num proxy não pode manter no ar, por até 30 s, uma página que acabou de ser desabilitada.
- Em 404, 429 e 503: `Cache-Control: no-store`.

Alternativas:
- Filtrar `/api/v1/endpoints/statuses`. Rejeitada: expõe o `Result` completo, busca `remote` e tem cache por paginação.
- `GetEndpointStatusByKey` e `GetUptimeByKey` por endpoint. Rejeitada: cerca de 200 transações e 1200 consultas para 200 endpoints, disputando a única conexão do SQLite com o watchdog e carregando em memória os campos sensíveis.
- `singleflight.DoChan` com timeout ou montagem desacoplada da requisição. Rejeitada (achado 16): poderia ler o store durante `store.Initialize` na recarga (data race) ou usar um `*sql.DB` fechado.

### D8. Limitador e IP do cliente
**Limitador próprio** (`statuspage/limiter.go`), no lugar do middleware do Fiber (achado 8):
- janela deslizante aproximada por dois contadores (minuto atual e anterior);
- mapa com teto de 50 000 chaves, com descarte das mais antigas e limpeza preguiçosa, **sem goroutine**;
- instância de pacote reaproveitada entre ciclos; a cada `start` só o limite é reconfigurado;
- chave: IPv4 por `/32`, IPv6 agregado por `/64`, IPv4 mapeado em IPv6 normalizado;
- limite de `status-pages.rate-limit` por minuto (padrão 120; `0` desliga);
- **conta só as respostas 404** (achado 1 e validação do grok), da rota específica e do catch-all. Uma página publicada, servida do cache, montada ou em 503, não consulta nem incrementa o limitador: o custo dela já é limitado a uma montagem por revisão a cada 30 s (cache, cache negativo de 5 s, `singleflight` e semáforo), e um balde compartilhado atrás de proxy, esgotado por 404, nunca a bloqueia;
- ao exceder: `429 {"error":"too many requests"}` com `Retry-After` e `Cache-Control: no-store`, sem `X-RateLimit-*`.

**IP do cliente (chave do limite):**
- Por padrão, é o IP da conexão.
- Se esse IP estiver em `trusted-proxies`, todas as linhas de `X-Forwarded-For` (`PeekAll`) são concatenadas na ordem de chegada e percorridas da direita para a esquerda; vale o primeiro IP que não esteja em `trusted-proxies`.
- Aceita `IP`, `IP:porta` e `[v6]:porta`. O IP da conexão e cada entrada são normalizados (IPv4 mapeado em IPv6 vira IPv4) **antes** da comparação com `trusted-proxies`.
- Com mais de 20 entradas, linha acima de 1 KB, entrada inválida ou header ausente, vale o IP da conexão.

**Outros pontos:**
- `ProxyHeader` e `EnableTrustedProxyCheck` do `fiber.Config` não mudam, então `c.IP()` continua como está para o resto do código (OIDC e logs). O CSRF do admin não depende disso.
- **Aviso em tempo de execução:** na primeira requisição de cada ciclo cuja conexão venha de IP privado (RFC 1918, `100.64.0.0/10`, `fc00::/7`), loopback ou link-local **não** confiável e traga `X-Forwarded-For`. Como a instância do limitador é reaproveitada entre ciclos, o controle não é um `sync.Once`: um `atomic.Uint64` guarda a geração em que o aviso foi emitido e é comparado e trocado (`CompareAndSwap`) com a geração atual; `logr.Warnf` explica que todos os clientes compartilham o limite e cita o IP a colocar em `trusted-proxies`. O mesmo sinal aparece em `GET /api/v1/admin/status-pages` (`sharedRateLimitWarning`). A heurística por `web.address` foi removida (não dispara em containers).

Alternativas:
- Middleware `limiter` do Fiber. Rejeitada: goroutine vazada a cada recarga, mapa sem teto e `X-RateLimit-*` quebrando o 404 idêntico.
- Habilitar `EnableTrustedProxyCheck` globalmente. Rejeitada por mudar o comportamento do código do upstream.

### D9. API de administração
As rotas ficam no grupo admin existente (`AdminMiddleware` e `adminRequestProtection`), em `api/admin_status_pages.go`, registradas antes de `/:slug`:

| Rota | Função |
|------|--------|
| `GET /status-pages` | Lista páginas do YAML e gerenciadas: slug, título, origem, `enabled`, versão, conflito, erro, número de endpoints resolvidos e caminho público; mais `publicationEnabled` (`status-pages.enabled`), `managedUnavailable` e `sharedRateLimitWarning` |
| `GET /status-pages/options` | Grupos (nome e quantidade) e endpoints publicáveis (chave, nome e grupo) |
| `GET /status-pages/exposure?group=<g>&key=<k>` | Páginas (slug, título, origem, `enabled`, motivo `group` ou `key`) em que um endpoint com esse grupo e chave apareceria; basta um dos dois parâmetros, e sem nenhum responde 400 |
| `POST /status-pages/validate` | Devolve a definição normalizada e avisos (grupos e chaves sem correspondência) |
| `POST /status-pages` | Cria (201, com `ETag`); `enabled` ausente vira `false` |
| `GET /status-pages/:slug` | Obtém (com `ETag`) |
| `PUT /status-pages/:slug`, `POST /status-pages/:slug/enable`, `POST /status-pages/:slug/disable`, `DELETE /status-pages/:slug` | Exigem `If-Match` (428 sem ele, 412 com versão antiga) |
| `GET /status-pages/:slug/preview` | Payload de D6 para qualquer página, inclusive desabilitada, em conflito ou do YAML, sem cache público e sem limitador, com o semáforo próprio de 1 vaga de D7 |

**Escritas:**
- `lifecycle.TryBeginChange` antes de validar ou gravar: 503 sem nada gravado.
- Mutex no serviço.
- Decodificação estrita (JSON ou YAML): campo desconhecido → 400.
- Slug do corpo diferente do da URL → 400; slug reservado → 400.
- Slug em uso pelo YAML ou por outra gerenciada → 409, citando a origem.
- Página do YAML → 409, como os endpoints somente leitura.
- Auditoria no log com autor, slug e operação, sem o corpo.

**Publicação depois do commit (achado 3):** com o mutex do serviço e o `TryBeginChange` segurados, o serviço grava no banco e, só depois do commit, publica o snapshot novo com revisão nova. As escritas já são serializadas entre si e com os ciclos, então não há corrida; se o commit falhar, nada foi publicado.

**Mapeamento de erros próprio** (`api/admin_status_pages_errors.go`, sem reutilizar `adminServiceError`): storage sem suporte → 501; slug em uso ou página do YAML → 409; slug reservado, definição inválida ou troca de slug → 400; versão → 412; sem `If-Match` → 428; ciclo em andamento → 503; página inexistente → 404; criação → 201.

**Com `status-pages.enabled: false`:** a administração e a pré-visualização continuam funcionando, e a lista informa `publicationEnabled: false`.

`GET /api/v1/config` não muda: o link Admin já leva às telas novas.

Alternativa: rotas separadas fora do grupo admin. Rejeitada por duplicar autorização e CSRF.

### D10. Telas de administração
- Rotas SPA `/admin/status-pages`, `/admin/status-pages/new` e `/admin/status-pages/:slug/edit`, registradas no servidor só com a administração habilitada. As rotas do `vue-router` são estáticas, e a view trata a ausência da administração.
- Abas "Endpoints" e "Status pages" nas views de administração.
- **Lista:**
  - slug, título, origem, estado (publicada, desabilitada, em conflito ou inválida) e número de endpoints;
  - ações abrir (`<a href="/status/<slug>" target="_blank" rel="noopener">`), copiar link, habilitar/desabilitar, editar e remover;
  - banners para `publicationEnabled: false`, `managedUnavailable` e `sharedRateLimitWarning`.
- **Copiar link:** usa `navigator.clipboard`, com um campo selecionado como alternativa.
- **Formulário:**
  - slug somente leitura na edição;
  - título, descrição e `enabled` (desmarcado ao criar);
  - grupos marcados a partir de `/options`;
  - endpoints com busca;
  - avisos de `/validate` e painel de pré-visualização (`/preview` depois de salvar, `/validate` antes).
- **Aviso de exposição no formulário de endpoints** (`AdminEndpointForm.vue`): ao mudar grupo ou nome, consulta `/status-pages/exposure` e mostra "Este endpoint aparecerá publicamente nas páginas: …".
- Remoção com o `ConfirmDialog` existente e tratamento de 412 igual ao dos endpoints.
- Páginas do YAML ficam somente leitura.

### D11. Página pública
- View `views/public/StatusPage.vue` e componentes em `components/public/`.
- **Cabeçalho:** logo e `ui.header` do template, título e descrição da página.
- **Faixa de estado geral:** texto e cor ("Todos os sistemas operacionais", "Degradação parcial", "Indisponível", "Sem dados"), nunca só a cor, com `role="status"` só nela.
- **Seções por grupo,** com o estado do grupo; a seção sem grupo aparece como "Outros serviços".
- **Página truncada:** com `truncated: true`, um aviso "Mostrando os primeiros 200 serviços".
- **Linha por endpoint:**
  - nome e indicador circular (exceção prevista em `ui-square-style`);
  - uptime de 24h, 7d e 30d ("—" quando `null`);
  - barras quadradas dos últimos resultados (50 em telas largas, 25 abaixo de 640 px), com `aria-hidden`;
  - resumo textual `sr-only`, por exemplo "api: no ar, 48 de 50 verificações com sucesso, uptime 24h 99,9%";
  - tooltip próprio com horário, sucesso e duração em ms, acessível por teclado e toque.
- **Atualização:**
  - "Atualizado há X", calculado a partir de `updatedAt` do servidor (desvio de relógio limitado a ≥ 0) e fora de `aria-live`;
  - nova busca a cada 60 s, pausada com a aba oculta (`visibilitychange`);
  - em 429 ou 503 com dados anteriores, mantém os dados, avisa a falha e tenta de novo no ciclo seguinte, respeitando `Retry-After` quando maior que 60 s;
  - na primeira carga com 429, 503, erro de rede ou resposta não JSON (por exemplo, 502 HTML do nginx), mostra uma mensagem de indisponibilidade com nova tentativa.
- **Tema, título e textos:**
  - alternância de tema pelo cookie `theme` existente;
  - `document.title` com o título da página;
  - 404 e slug inválido mostram "Página não encontrada";
  - textos em pt-BR;
  - visual quadrado, variantes `dark:` e `prefers-reduced-motion` respeitado.

Alternativa: reutilizar `EndpointCard` e o `Tooltip` global. Rejeitada porque dependem de `hostname`, `errors` e duração em ns e navegam para a rota protegida.

### D12. Observabilidade
Esta mudança não cria métricas Prometheus. Os vetores são registrados por ciclo com labels congeladas (D7 do admin), e uma label por slug criaria outra fonte de cardinalidade e de registro. O log de acesso do nginx cobre o volume.

Logs:
- auditoria das escritas;
- slugs publicados por origem e avisos de correspondência na carga;
- falha ao listar as gerenciadas;
- aviso de página truncada;
- aviso de limite compartilhado atrás de proxy (uma vez por ciclo);
- erros de leitura com o slug;
- 429 em nível debug.

Alternativa: contador `gatus_status_page_requests_total{slug,code}`. Adiada.

### D13. Relação com o upstream
- **Proposta do upstream** (#1328, com o apoio do mantenedor em #638): marca endpoints com `visibility.public` e mostra só esses a anônimos, no mesmo dashboard. Isso exige filtrar as rotas protegidas por autenticação e expõe a anônimos o `Result` completo (hostname, erros e condições).
- **Status pages por slug:** atendem públicos diferentes com páginas diferentes, usam payload sanitizado, não mudam as rotas protegidas e ficam mais próximas do Kuma.
- **Convivência:** se o #1328 for aceito, `visibility.public` passa a governar o dashboard `/` e as status pages continuam separadas. A montagem do payload ignora `visibility`.
- O #1311 foi fechado como *not planned*, e a UI de status pages é o diferencial do gatus.io. A funcionalidade fica no fork.
- Com o módulo em `gatus/v5` (D0), trazer mudanças do upstream exige reescrever os imports; o código em arquivos novos continua reduzindo conflitos no restante.

### D14. Testes
**Testes Go:**
- validação estrutural da configuração (slugs reservados, runas, `TrimSpace`);
- store em SQLite e PostgreSQL (`GATUS_TEST_POSTGRES_URL`);
- leitura em lote: SQL × `GetUptimeByKey` no mesmo store SQL e memória × `GetUptimeByKey` na memória, comparando os valores quando há execuções no período (sem execuções, o lote devolve `nil` e `GetUptimeByKey` devolve 0); resultados em lote × `GetEndpointStatusByKey`; endpoint com resultados e sem execuções no período presente no mapa com uptimes `nil`; chave sem registro fora do mapa; erro propagado;
- registro: conflito, recarga, gerenciada inválida, falha de listagem, administração desligada e avisos na carga;
- seleção, ordem e seção sem grupo;
- estados agregados.
- **Sanitização por lista de permitidos:** grava resultados com hostname, erros e condições, decodifica o JSON em structs espelho com `DisallowUnknownFields` e procura os **valores** sensíveis (`10.0.0.5`, `dial tcp`).
- **404 idêntico** (status, corpo e os cabeçalhos `Content-Type`, `Cache-Control`, `X-Robots-Tag`, `X-Content-Type-Options`, `Referrer-Policy` e `Vary`, ignorando `Date`) para página inexistente, desabilitada, em conflito, inválida, slug malformado, slug vazio, `/a/b` e `enabled: false`, com `GET` e `HEAD`, sem consulta ao storage (`summaryReader` de contagem).
- **Sem credenciais:** 200 com basic e com OIDC (emissor falso com `httptest.Server` servindo `/.well-known/openid-configuration` e JWKS), sem `WWW-Authenticate`; `/status/qualquer` sempre 200; rotas protegidas continuam com 401.
- **Cache e montagem:** requisições simultâneas geram uma montagem e ocupam uma vaga do semáforo; o leitor é resolvido a cada montagem (troca do store entre montagens); um `summaryReader` bloqueante prova que uma escrita durante a montagem faz a requisição seguinte disparar outra montagem; cache negativo (50 requisições sequenciais com leitor em falha → no máximo 1 montagem por 5 s); timeout do semáforo → 503 sem cache.
- **Limitador e proxy:** 429 com `Retry-After` e sem `X-RateLimit-*`; página publicada (cache, montagem e 503) não conta nem é bloqueada, mesmo com o balde esgotado por 404; teto de chaves; agregação IPv6 por /64; `X-Forwarded-For` ignorado de conexão não confiável, usado de proxy confiável, em duas linhas, com 10 000 entradas, com porta e com a conexão em `::ffff:172.30.0.1`; aviso em tempo de execução uma única vez por ciclo e de novo depois de uma recarga; contagem de goroutines estável depois de N ciclos de `api.New`.
- **API de administração:** 401, 403, 413, 415, 428, 412, 409, 400 (campo desconhecido, troca e reserva de slug), 501 e 503; `enabled` ausente cria desabilitada; publicação só depois da gravação (store falso injetável no serviço que devolve erro na gravação: nada publicado); exposição por grupo, por chave e 400 sem parâmetros; pré-visualização com semáforo próprio.
- **Concorrência:** `-race` com `Load` concorrente, escritas da administração e 100 leitores públicos.
- `go test ./... -race` com PostgreSQL.

**E2E** (`test/e2e/status-pages.sh`):
- sobe o binário com SQLite, basic auth e administração habilitada;
- cria uma página pelas telas, confere a pré-visualização e publica;
- abre `/status/<slug>` numa sessão do `agent-browser` sem `set credentials` e confere por `network requests` que nenhuma requisição foi a `/api/v1/config` nem recebeu 401;
- abre `/status/a%2Fb` e `/status/nao-existe` na mesma sessão: "Página não encontrada", nenhuma requisição com 401;
- confere página desabilitada (API 404);
- simula OIDC sem sessão com `network route` em `/api/v1/config` (`{"oidc":true,"authenticated":false}`): a página pública não busca a config e não mostra a tela de login; controle positivo navegando para `/`, que mostra "Login with OIDC";
- temas claro e escuro, viewport de 390 px;
- capturas em `dist/prints/status-pages/`.

## Risks / Trade-offs

- **Exposição de nomes de endpoints e grupos:** podem revelar nomes internos, e endpoints novos de um grupo entram sozinhos. → Página gerenciada nasce desabilitada, pré-visualização, aviso de exposição no formulário de endpoints e documentação.
- **Páginas publicadas com a administração desligada:** desligar o admin num incidente não despublica. → Log dos slugs publicados na carga; `status-pages.enabled: false` despublica tudo.
- **Slug adivinhável:** → não é controle de acesso (documentado); rota HTML sempre 200, 404 idêntico na API e limitador contando os 404.
- **Badges, uptime e response-times já públicos por chave:** comportamento do upstream. → Fora do escopo, registrado na documentação.
- **Carga no storage por abuso:** → cache de 30 s por revisão, cache negativo de 5 s, `singleflight`, semáforo com timeout, 404 sem storage, leitura em lote, limite de 200 endpoints e limitador.
- **Limite compartilhado atrás de proxy:** um balde único para todos os visitantes. → Só 404 contam e página publicada nunca é limitada; aviso em tempo de execução; `trusted-proxies` documentado para Docker e nginx.
- **`X-Forwarded-For` forjado:** → só aceito de conexões em `trusted-proxies`, todas as linhas, da direita para a esquerda, com limites de tamanho.
- **Dados até 30 s atrasados:** → aceitável para página pública; o admin vê pela pré-visualização sem cache.
- **Iframe permitido:** clickjacking não se aplica (página só leitura, sem ações). → Registrado; `frame-ancestors` configurável na fase 2.
- **Várias instâncias no mesmo PostgreSQL:** mudanças nas páginas só valem nas outras depois de recarregar ou reiniciar; atrás de um balanceador, o visitante pode alternar entre a página e "Página não encontrada". → Documentado, com o mesmo aviso do admin.
- **Módulo `gatus/v5`:** sincronização manual com o upstream e imports agrupados como biblioteca padrão pelo `goimports`. → Decisão do dono (D0); `make fmt` usa `gofmt`.
- **Divergência do upstream:** → arquivos novos; toques mínimos em `api/api.go`, `config/config.go`, `main.go`, `storage/store/sql` (`createSchema`), `App.vue`, no router e em `AdminEndpointForm.vue`.
- **Rollback para o upstream:** a seção `status-pages` e a tabela são ignoradas e as páginas somem. → Backup do banco antes.

## Migration Plan

Entrega em marcos, cada um num pull request no fork, com CI verde antes do merge:
0. **Caminho do módulo:** `gatus/v5` em `go.mod`, imports e documentação.
1. **Configuração, store e registro:** `config/statuspage`, tabela, leitura em lote, `statuspage.Load` e seleção.
2. **API pública:** payload sanitizado, estados, cache por revisão, `singleflight`, semáforo, limitador, IP do cliente, rotas públicas, catch-all e cabeçalhos.
3. **API de administração** das status pages.
4. **Página pública** no frontend e layout público reativo no `App.vue`.
5. **Telas de administração** das status pages e aviso de exposição no formulário de endpoints.
6. **Documentação e E2E;** release pela skill `create-release` e atualização do servidor local de validação.

**`trusted-proxies` nas instalações:**
- **Docker com porta publicada em `127.0.0.1` e nginx no host** (padrão do usuário): dentro do container, a conexão chega do gateway da bridge. Fixar a sub-rede da rede do compose e confiar no gateway:
  ```yaml
  networks:
    default:
      ipam:
        config:
          - subnet: 172.30.0.0/24
  ```
  com `status-pages.trusted-proxies: ["172.30.0.1/32"]`.
- **Alternativa:** `network_mode: host` com `web.address: 127.0.0.1` e `trusted-proxies: ["127.0.0.1/32", "::1/128"]`.
- **Binário direto no host atrás do nginx:** `trusted-proxies: ["127.0.0.1/32", "::1/128"]`.
- O vhost de exemplo já envia `X-Forwarded-For`. Se o aviso de limite compartilhado aparecer no log, ele cita o IP a acrescentar.

Rollback: voltar para a release anterior do fork. A tabela `managed_status_pages` é ignorada por versões antigas, e as páginas deixam de existir.

## Open Questions

1. Vale mostrar um estado "em manutenção" durante as janelas de manutenção (fase 2)?
2. O padrão de 120 respostas 404 por minuto por IP é adequado para uma sala de monitoramento atrás de NAT (páginas publicadas não contam)?

## Ajustes da revisão de QA

Relatório: 23 achados (1 crítico, 4 altos, 12 médios e 6 baixos). O que mudou por achado:

| Achado | Ajuste |
|--------|--------|
| 1 (crítico) Limite único atrás do Docker | D8: só 404 contam, página publicada nunca é limitada; aviso em tempo de execução por ciclo no lugar da heurística de `web.address`; Migration Plan com sub-rede fixa e gateway; tarefa 6.5 |
| 2 (alto) 401 fora do handler | D5: catch-all sempre registrado; rotas de frontend com regex e catch-all público; validação e `encodeURIComponent` na view |
| 3 (alto) Publicar antes do commit | D9: publicação depois do commit; D4 e D7: revisão em memória no lugar da versão do banco |
| 4 (alto) `singleflight` só por slug | D7: captura única e chave `slug|revisão|geração` |
| 5 (alto) Custo real no SQL | D7: `EndpointSummaryBatchReader` com uma transação e três consultas; menção ao `writeThroughCache` removida; preview no semáforo |
| 6 (médio) Uptime `null` na memória | Leitor em lote na memória (já implementado); D6 documenta as bordas diárias; testes no mesmo store |
| 7 (médio) Erros engolidos | D6: ausência → `unknown`, outros erros → 503; D7: cache negativo de 5 s |
| 8 (médio) `limiter` do Fiber | D8: limitador próprio sem goroutine, com teto, /64 e só `Retry-After`; D14 define os cabeçalhos comparados |
| 9 (médio) `X-Forwarded-For` em várias linhas | D8: `PeekAll`, limites de tamanho, porta e `::ffff:` |
| 10 (médio) Oráculo na rota HTML | D5: HTML sempre 200 com template parseado uma vez por ciclo |
| 11 (médio) Avisos no lugar errado | D1, D3 e D4: avisos só em `statuspage.Load` e na validação da administração |
| 12 (médio) Comportamentos indefinidos | D4: falha de listagem e storage sem suporte; D9: administração com `enabled: false` e mapeamento de erros próprio |
| 13 (médio) Decisões de produto | D0 com as respostas do dono; API cria desabilitada; nome real do grupo; aviso de exposição (D9 e D10); página mínima |
| 14 (médio) `App.vue` | D5: layout reativo, config sob demanda, login só fora das rotas públicas, tooltip próprio, `watch` do slug; D10: link "abrir" |
| 15 (médio) Testabilidade | D14: `network requests` com controle positivo, emissor OIDC falso, `summaryReader` injetável, allowlist na sanitização |
| 16 (médio) Ciclo de vida | D7: montagem síncrona, semáforo com timeout, store capturado uma vez |
| 17 (médio) Tarefas | `tasks.md` reorganizado com marco 0 e as tarefas novas |
| 18 (baixo) Slugs reservados | D1: reservados, runas e `TrimSpace` dos dois lados |
| 19 (baixo) Acessibilidade | D5 e D11: `sr-only`, `aria-hidden`, tooltip por teclado, `role="status"`, `lang`, movimento reduzido e estados de primeira carga |
| 20 (baixo) Cabeçalhos | D5.1: `nosniff` e `Referrer-Policy`; iframe permitido; sem CORS; `HEAD` e `Vary` nos testes |
| 21 (baixo) Afirmações imprecisas | Context, D5, D7, D8 e D10 corrigidos |
| 22 (baixo) `indexable` | Removido; páginas sempre `noindex` |
| 23 (baixo) Várias instâncias | Risks e documentação (tarefa 6.1) |

## Ajustes da validação com o grok

A CLI do grok revisou a change ajustada contra o relatório de QA e o código (saída em `scratchpad/grok-status-pages.txt`). Aplicado:
- **Leitor do store:** resolvido a cada montagem por `getSummaryReader`, sem guardar ponteiro entre recargas (D7).
- **Aviso por ciclo:** geração com `CompareAndSwap` no lugar de `sync.Once`, porque o limitador é reaproveitado (D8).
- **Cache HTTP:** `Cache-Control: no-cache` no 200 da API, para "desabilitar tira do ar imediatamente" valer também atrás de caches HTTP (D7).
- **Limitador:** só 404 contam; uma página publicada nunca é limitada, nem depois de expirar o cache com o balde esgotado (D8).
- **Semáforo:** pedido só pela montagem líder, dentro do `singleflight`; pré-visualização com semáforo próprio de 1 vaga (D7).
- **Presença no lote:** decidida pela existência do endpoint, não pelo uptime; testes de uptime comparam valores só com execuções (D7, D14, tarefas 1.3 e 1.4).
- **Normalização de `::ffff:`** antes da comparação com `trusted-proxies`, e faixas privadas definidas para o aviso (D8).
- **Definições:** `updatedAt` como instante da montagem (D6); `Vary: Accept-Encoding` explícito (D5.1); `exposure` sem parâmetros → 400 (D9); aviso de página truncada (D11); falha de gravação testada com store falso (D14); catch-all sem `/*` citado na proposal; a pergunta sobre markdown saiu das Open Questions (já decidida em D1).

Rejeitado: marcar a tarefa 0.2 como concluída. A troca do módulo está em andamento num PR próprio e só conta como feita depois do merge.
