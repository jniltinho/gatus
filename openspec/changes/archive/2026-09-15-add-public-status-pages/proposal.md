## Why

O Gatus tem um único dashboard, e com `security` configurado ele fica inteiro atrás de login: não existe uma página que clientes ou outras equipes possam abrir sem senha mostrando só parte dos serviços. No upstream, o pedido de status pages públicas foi fechado como *not planned* ([TwiN/gatus#1311](https://github.com/TwiN/gatus/issues/1311)); a UI de status pages é o diferencial da versão paga (gatus.io). A alternativa em andamento ([#638](https://github.com/TwiN/gatus/issues/638), PR [#1328](https://github.com/TwiN/gatus/pull/1328)) só separa endpoints públicos e privados no mesmo dashboard, expondo o resultado completo das verificações.

O fork já tem administração pela web (`add-admin-endpoint-management`). Com isso, pode oferecer várias páginas de status públicas por grupo, como os *Status Pages* do Uptime Kuma (`/status/<slug>`), sem abrir o dashboard nem a API de status atual.

## What Changes

- **Marco 0 (PR próprio, antes de tudo):** o caminho do módulo Go do fork passa de `github.com/TwiN/gatus/v5` para `gatus/v5`. Os links para issues e PRs do upstream continuam como URL.
- Nova seção `status-pages` no YAML, com `enabled` (padrão `true`), `trusted-proxies`, `rate-limit` e `pages`.
- Cada página tem `slug`, `title`, `description`, `groups`, `endpoints` e `enabled`.
  - Os endpoints entram pelo `group` (endpoints novos do grupo aparecem sozinhos) e pela chave.
  - Valem endpoints do YAML, external-endpoints e endpoints gerenciados. Suites e instâncias `remote` ficam de fora.
  - O formulário de endpoints da administração avisa em quais páginas públicas o endpoint vai aparecer.
- Rotas públicas, sem autenticação mesmo com `security.basic` ou `security.oidc`:
  - `GET/HEAD /status/<slug>`: sempre 200 com a SPA, sem revelar se a página existe;
  - `GET /api/v1/status-pages/<slug>`: decide entre 200 e 404;
  - um *catch-all* em `/api/v1/status-pages` e `/api/v1/status-pages/*` responde o mesmo 404, sempre registrado (inclusive com `enabled: false`), para nenhum caminho cair no middleware de segurança e abrir o prompt de login.
- Payload próprio e sanitizado: nome, grupo, estado, últimos resultados (horário, sucesso e duração) e uptime de 24h, 7d e 30d.
  - Nunca expõe URL, hostname, IP, código HTTP, erros, condições, eventos ou chaves.
- Página inexistente, desabilitada, em conflito ou inválida responde com um 404 idêntico. As respostas públicas levam sempre `X-Robots-Tag: noindex, nofollow`, `X-Content-Type-Options: nosniff` e `Referrer-Policy`.
- Custo controlado:
  - cache de 30 s chaveado por uma revisão em memória, com cache negativo de 5 s para 503;
  - uma montagem por revisão de página (`singleflight`), síncrona, com semáforo global de 4 e timeout de 5 s;
  - resultados e uptime lidos em lote numa única transação de leitura por montagem;
  - limite de 200 endpoints por página.
- Limitador próprio por IP (padrão de 120 por minuto) que conta só as respostas 404; uma página publicada nunca é limitada, porque o custo dela já é limitado pelo cache e pela montagem única.
  - `X-Forwarded-For` só é aceito de `trusted-proxies`, lido de todas as linhas, da direita para a esquerda.
  - Um aviso no log, uma vez por ciclo, aponta conexões de IP privado não confiável com `X-Forwarded-For` (limite compartilhado atrás de proxy).
- Páginas gerenciadas pela administração ficam na tabela `managed_status_pages` (SQLite e PostgreSQL).
  - Usam versão com `ETag` e `If-Match`, o lock de ciclo de vida e as proteções do admin (CSRF, 256 KB e tipos de mídia).
  - Uma página criada sem `enabled` nasce **desabilitada**, também pela API.
  - A gravação acontece antes da publicação em memória: nada não gravado fica visível.
  - As páginas do YAML ficam somente leitura. Em conflito de slug, o YAML prevalece.
  - Continuam publicadas com `admin.enabled: false`; o log da carga lista os slugs publicados por origem.
- API `/api/v1/admin/status-pages/*` para listar, obter, criar, alterar, habilitar, desabilitar, remover, validar, pré-visualizar, consultar opções e consultar a exposição de um endpoint.
- Novas telas:
  - página pública mínima (logo, cabeçalho, título, descrição, estado geral, grupos, barras de histórico e uptime), acessível, com tema claro e escuro, layout responsivo, visual quadrado e atualização automática;
  - telas de administração "Status pages", com lista, formulário, pré-visualização e botão de copiar o link.
- `App.vue`: o layout reage à rota (`meta.public`); as rotas públicas não buscam `/api/v1/config`, não passam pela tela de login do OIDC e não mostram o cabeçalho do dashboard.
- Documentação (`docs/status-pages.md`, com Docker, nginx e `trusted-proxies`) e roteiro E2E `test/e2e/status-pages.sh`, com capturas em `dist/prints/status-pages/`.
- Entrega em 7 marcos (0 a 6), cada um num pull request, com release ao final.

Sem páginas definidas, a diferença visível é que `/status/<slug>` mostra "Página não encontrada" e `/api/v1/status-pages/<slug>` responde 404. As rotas de status atuais continuam protegidas. A troca do caminho do módulo afeta só quem importa o código Go do fork como biblioteca (nenhum uso conhecido); para instalações, nenhuma mudança é **BREAKING**.

## Capabilities

### New Capabilities
- `public-status-pages`:
  - seção `status-pages` e páginas do YAML;
  - seleção de endpoints;
  - rotas públicas e payload sanitizado;
  - estados agregados;
  - respostas 404 indistinguíveis;
  - cache, montagem, leitura em lote, limite de requisições e IP do cliente atrás de proxy.
- `status-page-management`: persistência, API de administração, concorrência otimista, conflito com o YAML, lock de ciclo de vida, pré-visualização, exposição de endpoints e auditoria das páginas gerenciadas.
- `status-page-web-ui`: página pública, layout público, telas de administração de status pages e testes ponta a ponta com `agent-browser`.

### Modified Capabilities
Nenhuma. `openspec/specs/` ainda está vazio, e a change `add-admin-endpoint-management` não foi arquivada, então os requisitos novos ficam em capabilities novas.

## Impact

- **Módulo Go (marco 0):** `go.mod`, todos os imports `.go`, `README.md`, `docs/` e referências em arquivos de build e do frontend passam a usar `gatus/v5`.
- **Go:**
  - `config/config.go`: campo `StatusPages` e `ValidateStatusPagesConfig` (só estrutural);
  - novo `config/statuspage`;
  - novo pacote `statuspage`: registro, seleção, montagem do payload, cache, limitador, IP do cliente e serviço de administração;
  - `storage/store/`: interfaces `ManagedStatusPageStore`, `EndpointUptimeBatchReader` e `EndpointSummaryBatchReader`;
  - `storage/store/sql/` e `storage/store/memory/`: tabela e leituras em lote;
  - `main.go`: `statuspage.Load` em `initializeStorage`;
  - `api/api.go`: rotas públicas antes dos arquivos estáticos e do middleware de segurança, e rotas de administração;
  - novos `api/status_page.go`, `api/spa_render.go`, `api/admin_status_pages.go` e `api/admin_status_pages_errors.go`.
- **Frontend:**
  - rotas públicas e de administração no `vue-router`;
  - `App.vue`: layout reativo à rota e configuração buscada só fora das rotas públicas;
  - novas views `views/public/StatusPage.vue`, `views/admin/AdminStatusPages.vue` e `views/admin/AdminStatusPageForm.vue`, componentes em `components/public/` e aviso de exposição em `AdminEndpointForm.vue`;
  - `utils/adminApi.js`;
  - `web/static/` regenerado.
- **Banco:** tabela `managed_status_pages` criada automaticamente.
- **API:** `GET /api/v1/status-pages/:slug` (pública) e `/api/v1/admin/status-pages/*`.
- **Dependências:** nenhuma nova. `golang.org/x/sync/singleflight` já faz parte de módulo em uso; o limitador é próprio.
- **Segurança:** é a primeira rota pública do fork que lê o storage para vários endpoints. Badges, uptime, response-times e `/api/v1/config` continuam públicos como no upstream.
- **Documentação e testes:** `docs/status-pages.md`, seção no `README.md` e `test/e2e/status-pages.sh`.
- **Upstream:** código novo em arquivos novos. Se o #1328 for aceito, `visibility.public` e as status pages convivem. A troca do caminho do módulo torna a sincronização com o upstream manual.
