## Context

Estado do código (master `7f3873d0`, após a v5.36.0), conferido na revisão de QA:

- **Configuração:** endpoints existem apenas em `Config.Endpoints`, carregados por `config.LoadConfiguration` → `parseAndValidateConfigBytes`. As variáveis de ambiente são expandidas antes do parse. A configuração é recusada sem endpoints ou suites (`ErrNoEndpointOrSuiteInConfig`). `ValidateAlertingConfig` roda antes de `ValidateEndpointsConfig`, nunca retorna erro, ignora em silêncio alertas de tipos sem provedor, só registra aviso para overrides inválidos e altera o estado global (`SetAlertingProviderToNil`). `ValidateTunnelingConfig` resolve `client.tunnel` apenas para endpoints e suites do YAML.
- **Watchdog:** `watchdog.Monitor` cria um contexto global e uma goroutine por endpoint, com 222 ms entre partidas (só para endpoints e suites). `executeEndpoint` adquire o semáforo com o contexto **global**, executa `EvaluateHealth()` sem contexto e depois publica métricas, grava o resultado (`InsertEndpointResult` recria a linha em `endpoints` se ela não existir) e trata alertas. O cancelamento só é observado entre execuções. `Shutdown` cancela tudo.
- **Partida e hot-reload:** `start` sobe o HTTP (`go controller.Handle`) antes de `watchdog.Monitor`. O hot-reload (a cada 30 s) faz `stop` (watchdog, HTTP, métricas) → `save` → `load` → `store.Close()` → `initializeStorage` → `start`. Com YAML inválido e `skip-invalid-config-update`, `start` nunca é chamado e o Gatus fica parado.
- **Storage:** `initializeStorage` apaga o histórico das chaves ausentes da configuração (`DeleteAllEndpointStatusesNotInKeys`) e restaura o estado dos alertas disparados (`Triggered`, `ResolveKey`, contadores). O store SQL tem um `writeThroughCache` por chave; a API tem um cache de 10 s para status.
- **Métricas:** os vetores Prometheus são registrados com `GetUniqueExtraMetricLabels()` (apenas endpoints habilitados), e a mesma lista é capturada em `watchdog.Monitor` e `api/external_endpoint.go`. `WithLabelValues` com cardinalidade diferente causa panic.
- **Segurança:** `security.basic` (um usuário; o middleware do Fiber guarda o usuário em `Locals`) ou `security.oidc` (a sessão já guarda o `subject`; cookie `SameSite=Strict`, sem `HttpOnly` nem `Secure`; `allowed-subjects` vazio aceita qualquer usuário do provedor). Com OIDC configurado, o basic é ignorado. `IsAuthenticated` só reconhece OIDC. O Fiber confia em `X-Forwarded-*` porque `EnableTrustedProxyCheck` está desligado.
- **Campos sensíveis do endpoint:** headers, usuário e senha na URL, `client.oauth2.client-secret`, `ssh.password`, `ssh.private-key` e `alerts[].provider-override`. `client.identity-aware-proxy` obtém tokens com as credenciais do servidor, e `client.tls.certificate-file`/`private-key-file` leem arquivos locais.
- **Frontend:** Vue 3 + vue-router + Tailwind (`--radius: 0.5rem`), compilado para `web/static/` (commitado, sem hash nos nomes, build reprodutível com Node 24) e embutido via `go:embed`. Rotas SPA registradas uma a uma em `api/api.go`. Arredondamento em 20 arquivos, no tooltip do Chart.js (padrão do Chart.js) e nos badges SVG (`rx="3"`).
- **CI herdado:** `test.yml` (checkout depois de `setup-go`, `sudo go test -race` por causa de ICMP), `test-ui.yml` sem Node fixo, e workflows que publicam com segredos do mantenedor. O `Dockerfile` usa `golang:alpine` flutuante e compila sem suporte a cross-compile. 15 arquivos Go do upstream não passam no `gofmt`. As tags do upstream estão no fork.
- **Agentes:** o `AGENTS.md` pede `go mod vendor`, mas `vendor/` não é versionado.
- **Referências do dono:** CI/release e skills de `jniltinho/llama-model`; skill `agent-browser` de `jniltinho/go-postfixadmin`; imagem no Docker Hub `jniltinho/gatus`.

## Goals / Non-Goals

**Goals:**
- Criar, editar, habilitar, desabilitar e remover endpoints pela web, com validação pelo menos tão estrita quanto a do YAML e efeito imediato, sem reiniciar os demais endpoints.
- Nunca perder histórico por causa de configuração (admin desligado, conflito, erro temporário).
- Manter o comportamento atual para quem não usa `admin`, exceto o hot-reload mais seguro e o visual.
- Código novo em arquivos novos, para facilitar a sincronização com o upstream.
- CI e release próprios, publicados no Docker Hub, com versionamento sem colisão.
- Skills e regras para agentes; testes pelo navegador com capturas fora do repositório.

**Non-Goals:**
- Gerenciar pela web: provedores de alerta, `security`, `storage`, suites, external-endpoints, janelas de manutenção globais, anúncios e `remote`.
- Vários usuários ou papéis com basic auth; permissões granulares.
- Histórico de auditoria navegável.
- Escrever no arquivo YAML.
- Sincronização em tempo real entre várias instâncias no mesmo PostgreSQL.
- Enviar resolve aos provedores de alerta ao remover um endpoint.
- Endurecer o cookie de sessão OIDC (`HttpOnly`/`Secure`).
- Contribuir a funcionalidade ao upstream; pacotes `.deb`/`.rpm`; binários fora de Linux.

## Decisions

### D1. Persistência no storage
Tabela `managed_endpoints` com colunas explícitas por dialeto: `endpoint_key` único, `definition` (texto YAML), `version` (inteiro, começa em 1), `created_at`, `updated_at`, `updated_by`; sem chave estrangeira para `endpoints`. A tabela é criada com `CREATE TABLE IF NOT EXISTS`, e um erro nesse comando interrompe a inicialização (ao contrário dos `ALTER` silenciosos existentes). Interface `ManagedEndpointStore` implementada pelo store SQL; violações de unicidade viram erro de conflito. A remoção apaga definição e dados da chave na mesma transação.

Alternativas: reescrever o YAML (perde comentários e `${VAR}`, falha com arquivo somente leitura e dispara o hot-reload global); arquivo JSON separado (mais um estado em disco).

### D2. Formato: definição enviada e definição efetiva
A API recebe YAML ou JSON (tipos de mídia interpretados com parâmetros: `application/json`, `application/yaml`, `application/x-yaml`, `text/yaml`), decodificados de forma estrita em `endpoint.Endpoint`. O banco guarda a **definição enviada**, reserializada sem os valores padrão; os padrões (`User-Agent`, `client`, `ui`, método, intervalo, alertas padrão dos provedores) são aplicados só em memória. Assim, mudanças posteriores nos padrões dos provedores valem para os gerenciados. As respostas trazem `definition` (armazenada) e `effective` (com padrões), ambas com segredos mascarados (D11), em YAML e em JSON com as chaves do YAML.

Alternativa: persistir a definição normalizada. Rejeitada porque congela os padrões e polui o formulário.

### D3. Validação estrita dos gerenciados
Pipeline, em arquivos novos:
1. decodificar sem expandir variáveis de ambiente;
2. rejeitar campos que usam credenciais ou arquivos do servidor: `client.identity-aware-proxy`, `client.tls.certificate-file`, `client.tls.private-key-file`, além de `store` e `always-run` (válidos só em suites);
3. resolver `client.tunnel` contra `tunneling` (erro se não existir);
4. validar alertas com uma função pura por endpoint que reaproveita a mescla do alerta padrão, mas retorna erro para tipo inexistente, provedor não configurado ou override inválido (o caminho do YAML continua como está);
5. `ValidateAndSetDefaults()`;
6. chave calculada por `Key()` única contra endpoints, external-endpoints, suites e endpoints de suites do YAML e contra os demais gerenciados (a mensagem cita a origem);
7. `extra-labels` contidos nas labels registradas no ciclo atual (D7).

Sem expansão de env porque a API é acessada pela web e `${VAR}` permitiria enviar segredos do processo para qualquer URL.

### D4. Ciclo de vida por endpoint no watchdog
Registro em arquivo novo do `watchdog`, com entrada por chave contendo origem (`config`/`admin`), contexto próprio, função de cancelamento e canal `done`:
- `executeEndpoint` recebe o contexto do endpoint: usa-o no `Acquire` do semáforo e confere `ctx.Err()` depois de `EvaluateHealth()`, antes de métricas, store e alertas. Resultado de execução cancelada é descartado.
- `StopEndpoint` cancela, espera `done` por até `client.timeout` + 5 s e só então chama `ep.Close()` (fechar as conexões enquanto a execução cria o client HTTP sob demanda é uma data race). Mesmo que o prazo estoure, a checagem de `ctx.Err()` impede gravações posteriores. A verificação em andamento não é interrompida, porque o client HTTP não recebe contexto: ela termina no próprio timeout e o resultado é descartado.
- `RestartEndpoint` = `StopEndpoint` síncrono + `StartEndpoint` com o objeto novo.
- O registro tem estado `closed` depois de `Shutdown`; `StartEndpoint` falha nesse estado. `Shutdown` espera as goroutines do registro por até 5 s, para que execuções antigas não usem o estado do ciclo seguinte.
- `Monitor` inicia os endpoints pelo registro. Como as escritas da administração ficam bloqueadas durante todo o ciclo (D6), não há alteração concorrente a conferir durante a partida.
- Operações de parada só atuam em entradas da origem pedida (um gerenciado em conflito nunca para o endpoint do YAML).

Alternativa: disparar o hot-reload completo a cada alteração. Rejeitada por derrubar HTTP e todo o monitoramento.

### D5. Propriedade dos endpoints
`cfg.Endpoints` fica imutável depois do load e contém só o YAML. Os gerenciados ativos ficam num registro próprio (pacote novo), com snapshot copy-on-write via `atomic.Pointer`. Objetos em execução nunca são alterados no lugar: uma alteração cria um objeto novo. A API lê definições do banco, não os objetos vivos. As poucas buscas por chave em tempo de execução (`api/badge.go`) consultam `cfg.Endpoints` e depois o snapshot dos gerenciados.

Alternativa: mutex dentro de `config.Config` e gerenciados em `cfg.Endpoints`. Rejeitada: o campo exportado continuaria acessado sem lock (inclusive por código novo do upstream), os objetos vivos são alterados pelas goroutines e cópias de `Config` gerariam avisos de `copylocks`.

### D6. Carga, conflito e serialização
- Com storage SQL, `initializeStorage` sempre lê `managed_endpoints`, **independentemente de `admin.enabled`**: valida cada definição contra a configuração carregada, monitora as válidas e preserva as chaves de todas na limpeza. Com `admin` desligado, os gerenciados continuam monitorados; só a API e as telas ficam indisponíveis.
- Conflito de chave com o YAML: o YAML prevalece; o gerenciado fica marcado como em conflito e não é monitorado. Remover um gerenciado em conflito apaga só a definição. Se o YAML deixar de definir a chave, o gerenciado volta a ser monitorado herdando o histórico da chave (comportamento documentado).
- Um serviço de administração com mutex serializa validar → gravar → aplicar.
- Um lock de ciclo de vida (pacote `lifecycle`) é mantido em modo exclusivo da carga inicial até o fim do `Monitor` e, no hot-reload, de antes do `stop` até o fim do `Monitor` seguinte. As escritas da administração tentam o modo compartilhado **antes** de validar ou gravar qualquer coisa e respondem 503 se não conseguirem, então um 503 nunca deixa nada gravado. Isso inclui a partida inicial: escritas feitas enquanto o `Monitor` ainda inicia endpoints recebem 503.
- `parseAndValidateConfigBytes` valida os pré-requisitos do `admin` e só então dispensa endpoints ou suites quando `admin.enabled` é verdadeiro.

### D7. Labels Prometheus congeladas por ciclo
`metrics` passa a expor a lista registrada no ciclo atual (ex.: `RegisteredExtraLabels()`), calculada como hoje a partir dos endpoints habilitados do YAML. Watchdog, external endpoints e gerenciados usam essa lista e nunca a recalculam fora de `InitializePrometheusMetrics`. Gerenciados só podem declarar `extra-labels` contidos nela; na carga, um gerenciado com label fora da lista fica inválido. Ao remover ou alterar um endpoint, as séries da chave são apagadas com `DeletePartialMatch` em todos os vetores.

### D8. Autorização
Pacote `config/admin` (`enabled`, `allowed-subjects`, `allowed-origins`). Pré-requisitos: `security` com `basic` ou `oidc`; storage `sqlite` ou `postgres`; com OIDC, `allowed-subjects` não vazio (evita que qualquer conta de um provedor público vire administrador). Rotas `/api/v1/admin/*` no grupo protegido, com middleware adicional:
- com OIDC (que tem prioridade sobre basic, como hoje), o `subject` da sessão (claim `sub`) precisa estar em `admin.allowed-subjects`, com comparação sem diferenciar maiúsculas; aviso no log se um subject do admin não estiver em `security.oidc.allowed-subjects` quando essa lista existir;
- só com basic, o usuário basic é administrador; o autor vem de `Locals`.

`GET /api/v1/config` passa a devolver `admin: {enabled, authorized}`. Com apenas basic, a rota não recebe credenciais, mas o único usuário basic é sempre administrador, então `authorized` é `true` sempre que `enabled`; o frontend usa `enabled && authorized` em todos os casos, e a API pede as credenciais. O 401 do middleware de autenticação mantém o formato atual (texto).

Alternativa: credenciais separadas de admin no basic. Rejeitada porque o navegador guarda um único par basic por origem.

### D9. CSRF
O vetor principal é o basic (o navegador reenvia credenciais em requisições de outros sites); o cookie OIDC já é `SameSite=Strict`. Para `POST`, `PUT` e `DELETE`:
- `Sec-Fetch-Site: cross-site` → 403;
- `Origin` (ou `Referer`, se não houver `Origin`) precisa casar com `admin.allowed-origins`, se definido, ou com a origem derivada do header `Host` e do esquema (TLS da conexão ou `X-Forwarded-Proto`); caso contrário, 403. `X-Forwarded-Host` e `X-Forwarded-Port` não são usados, nem os helpers `Hostname()`/`Protocol()` do Fiber, que confiam em qualquer `X-Forwarded-*` porque `EnableTrustedProxyCheck` está desligado. Páginas não conseguem definir `Host`, `Origin` nem `Sec-Fetch-*`, e um header como `X-Forwarded-Proto` numa requisição vinda de outro site exige preflight de CORS, que o Gatus não autoriza; por isso a derivação é segura contra CSRF. Atrás de proxy que publique uma porta diferente da repassada no `Host`, use `admin.allowed-origins`;
- requisições sem `Origin` e sem `Referer` são aceitas (clientes que não são navegador);
- requisições com corpo exigem um dos tipos de mídia de D2, senão 415;
- com `ENVIRONMENT=dev`, `http://localhost:8081` é aceito (servidor de desenvolvimento do Vue).

### D10. Alteração, remoção e alertas
- **Concorrência otimista:** `GET` devolve `ETag` com a versão; `PUT`, `DELETE`, `enable` e `disable` exigem `If-Match` (428 sem ele, 412 com versão antiga).
- **Alteração:** validar → gravar (versão + 1) → `RestartEndpoint`. O bloco de restauração de alertas disparados de `initializeStorage` é extraído para uma função reutilizável e aplicado ao objeto novo, preservando `Triggered`, `ResolveKey` e contadores dos alertas cujo checksum não mudou; os demais são limpos pela rotina existente.
- **Remoção:** `StopEndpoint` → transação que apaga definição, status, resultados, eventos e alertas disparados → invalidação do `writeThroughCache` da chave e do cache de status da API → remoção das séries Prometheus. Nenhum resolve é enviado aos provedores; a resposta informa quantos alertas estavam disparados, e a tela avisa antes de confirmar.
- **Falha ao aplicar:** a transação no banco só é confirmada depois de aplicar a mudança no watchdog. Com o lock de ciclo obtido antes de gravar (D6), o registro não pode ser fechado durante a escrita; se ainda assim a aplicação falhar, a transação é desfeita e a API responde 500.
- **Chave imutável:** qualquer mudança no texto de `name` ou `group` é rejeitada. O servidor aplica `url.PathUnescape` à chave, e o frontend usa `encodeURIComponent`.

### D11. Segredos mascarados
Toda resposta com definição (origem `config` ou `admin`) substitui por `********`: valores de headers cujo nome contém `authorization`, `cookie`, `token`, `secret`, `password` ou `key` (sem diferenciar maiúsculas); a senha do userinfo da URL; os valores de parâmetros de query da URL cujo nome contém `token`, `secret`, `password`, `key` ou `authorization`; `client.oauth2.client-secret`; `ssh.password`; `ssh.private-key`; e os valores de `alerts[].provider-override`. Em `PUT` e `validate`, um valor igual à máscara mantém o valor armazenado. Os logs de auditoria nunca incluem esses valores.

### D12. Teste de endpoint com limites
`POST /test` valida e executa `EvaluateHealth()` numa cópia, fora do watchdog, sem gravar, alertar ou publicar métricas; timeout de `min(client.timeout, 10 s)`; no máximo 2 testes simultâneos (429 além disso); `CloseIdleConnections` ao final; valores resolvidos das condições truncados em 512 caracteres. Rotas admin limitadas a 256 KB de corpo (413).

### D13. Frontend de administração
Views em `web/app/src/views/admin/`, rotas no `vue-router` e rotas SPA em `api/api.go`. Componentes seguem o `AGENTS.md` e reaproveitam `components/ui/*`; editor YAML em `<textarea>`; conversão formulário → YAML com um serializador próprio e YAML → formulário com `POST /parse`, que só decodifica e devolve o documento enviado (`POST /validate` devolve a definição mascarada, e um segredo digitado viraria `********`); tratamento de 412 (recarregar e reaplicar) e de segredos mascarados.

### D14. Visual quadrado
`--radius: 0` e `theme.borderRadius` substituído no Tailwind (todas as variantes 0, `full` mantido). Passam a `rounded-none`: `Badge.vue`, contador de falhas em `Home.vue`, barra e botões de `Settings.vue`, barras de progresso de `SequentialFlowDiagram.vue`. Tooltip do Chart.js com `cornerRadius: 0`. Continuam circulares: pontos de status, marcadores de etapa e da linha do tempo e o spinner de `Loading.vue`. Badges SVG sem `rx`. O endpoint `badge.shields` devolve JSON para o shields.io, que define o próprio estilo, e fica fora do escopo.

### D15. CI, release e imagem
- **Makefile:** `UPSTREAM_BASE` guarda o commit do upstream em que o fork se baseia (hoje `7f3873d0`, posterior à tag v5.36.0; uma tag não serve porque o master do upstream já tem commits depois dela). `fmt` e a checagem de `gofmt` do `lint` atuam só nos arquivos Go adicionados ou alterados desde esse commit (sem reformatar o upstream); `vet` em todo o módulo; `build` e `release-cross` com `CGO_ENABLED=0` apenas nas receitas (o `-race` precisa de cgo); `release-cross` gera os tarballs e `dist/pkg/linux_<arch>/gatus`; `docker-release VERSION=...` executa `release-cross` e publica `jniltinho/gatus:v<versão>` para `linux/amd64` e `linux/arm64` com `docker buildx build --push`, recusando `VERSION=dev`.
- **Imagem:** `Dockerfile.release` do fork (o `Dockerfile` do upstream fica intacto): estágio `FROM --platform=$BUILDPLATFORM alpine:3.24.1` para `ca-certificates` e `tzdata`, imagem final `FROM scratch` com `COPY dist/pkg/linux_${TARGETARCH}/gatus` e `config.yaml`. Sem `RUN` na plataforma de destino, então não precisa de QEMU; usa os mesmos binários dos tarballs. Contexto controlado por `Dockerfile.release.dockerignore`.
- **Publicação da imagem:** por decisão do dono, nesta mudança a imagem é publicada a partir de uma máquina com `docker login` no Docker Hub (`make docker-release`). Publicar pelo workflow, com secrets, fica para uma mudança futura.
- **`ci.yml`** (push e PR para `master`), CI básico do código Go: checkout com histórico completo antes de `setup-go` (`go-version-file: go.mod`); `make lint`; `make build`; testes com `sudo env "PATH=$PATH" "GOROOT=$GOROOT" "GOCACHE=$(go env GOCACHE)" "GOMODCACHE=$(go env GOMODCACHE)" go test ./... -race`; serviço PostgreSQL para os testes do store (variável `GATUS_TEST_POSTGRES_URL`), adicionado quando esses testes existirem (marco 4).
- **`release.yml`** (tags `v*-fork.*`): testes → `make release-cross VERSION=${GITHUB_REF_NAME#v}` → `gh release create` com notas geradas a partir da tag anterior do fork (`--notes-start-tag`), sem pré-release e marcada como latest, com os dois tarballs; `permissions: contents: write`.
- **Diferenças do llama-model:** arm64, `config.yaml` no pacote, sem UPX, sem injeção de versão, notas via `gh` para não puxar o histórico do upstream.
- **Adiado:** checagem de `web/static` no CI (o build foi verificado como reprodutível com Node 24; `web/app/.nvmrc` fixa a versão para builds locais) e publicação da imagem pelo workflow.
- **Versionamento:** `v<base>-fork.<N>`. Pela precedência do SemVer, `v5.36.0-fork.1` ordena abaixo de `v5.36.0` em ferramentas como Renovate e `sort -V`; isso fica documentado no README.
- **Dependabot:** apenas `github-actions` (atualizações de `gomod` conflitariam com a sincronização).

### D16. Skills, licenças e regras do fork
As 15 skills `golang-*` e a `agent-browser` são copiadas sem alterações, acompanhadas das licenças de origem (`samber/cc-skills-golang`, MIT; `vercel-labs/agent-browser`, Apache-2.0). A `create-release` é reescrita para o Gatus. As regras do fork ficam em `AGENTS.fork.md` (skills obrigatórias, release, alvos do Makefile, regras do admin, testes E2E, não usar `go mod vendor`, sincronização com o upstream), e o `AGENTS.md` ganha apenas uma linha apontando para ele, reduzindo conflitos com o upstream.

**Sincronização com o upstream:** conflitos em `web/static/` são resolvidos regenerando com `make frontend-build`; workflows removidos continuam removidos; `UPSTREAM_BASE` passa a apontar para o novo commit do upstream e a base das próximas tags para a nova versão.

### D17. Testes ponta a ponta
`test/e2e/admin.sh` sobe o binário local com SQLite temporário, `security.basic` e `admin.enabled`, e usa o `agent-browser` para percorrer: lista, criar, validar, testar, salvar, editar, desabilitar, remover e acesso sem credenciais (401), nos temas claro e escuro, com capturas em `dist/prints/` (ignorado pelo git). O 403 por subject OIDC é coberto por teste Go com sessão injetada. O roteiro não roda no CI nesta mudança porque exige Chrome.

Alternativa: Playwright ou Cypress em `web/app`. Rejeitada para seguir o padrão do `jniltinho/go-postfixadmin` sem dependências npm novas.

### Decisões tomadas após a revisão de QA
As perguntas levantadas na revisão foram decididas assim (podem ser revistas pelo dono):
1. Com `admin` desligado, os gerenciados continuam monitorados e com histórico (D6).
2. Com OIDC, `admin.allowed-subjects` é obrigatório (D8).
3. A correção do hot-reload entra nesta mudança (`config-hot-reload`).
4. `client.tunnel` é permitido (resolvido); `identity-aware-proxy`, arquivos TLS, `store` e `always-run` são bloqueados (D3).
5. Remover endpoint com alerta disparado não envia resolve; a tela avisa (D10).
6. Imagem via `Dockerfile.release` com os binários do release (D15).
7. Lint de formatação só no código do fork (D15).
8. Indicadores circulares continuam redondos; sufixo `-fork.N` mantido com a ordenação documentada.
9. `v5.36.0-fork.1` é publicado ao fim do marco 1, antes do admin.

### Ajustes após a validação do grok
1. O lock de ciclo é obtido antes de validar ou gravar: 503 nunca deixa nada gravado; falha ao aplicar desfaz a transação e responde 500 (D6, D10).
2. A origem derivada para CSRF usa só `Host` e o esquema, sem `X-Forwarded-Host`/`X-Forwarded-Port` (D9).
3. Com apenas basic, `authorized` é `true` sempre que o admin está habilitado (D8).
4. Escritas durante a partida inicial recebem 503; o `Monitor` não precisa conferir alterações concorrentes (D4, D6).
5. Parâmetros de query sensíveis na URL também são mascarados (D11).
6. `CREATE TABLE IF NOT EXISTS`, falhando só em erro real (D1).

## Risks / Trade-offs

- [Divergência do upstream] → código novo em arquivos novos; alterações mínimas em `main.go`, `watchdog/endpoint.go`, `api/api.go`, `metrics/metrics.go` e `storage/store/sql`; procedimento de sincronização em `AGENTS.fork.md`.
- [Execução lenta atrasa `StopEndpoint`] a verificação em andamento não é interrompida → espera limitada a `client.timeout` + 5 s e descarte por `ctx.Err()`; interromper a requisição exigiria um client HTTP com contexto, fora do escopo.
- [`Shutdown` mais lento] espera de até 5 s pelas goroutines do registro no desligamento e no hot-reload → evita que execuções antigas usem o estado do ciclo seguinte.
- [Data race em estruturas compartilhadas] → objetos imutáveis, registro copy-on-write e `-race` no CI.
- [Várias instâncias no mesmo PostgreSQL] uma instância pode recriar a linha de um endpoint removido em outra e continuar alertando → documentado; aviso no log na partida quando o storage é PostgreSQL e `admin` está ativo.
- [Segredos no banco] ficam em texto, como no YAML → mascarados na API, fora dos logs; proteger backups.
- [Requisições a redes internas via `/test`] → só administradores, limites de D12 e auditoria.
- [Ordenação SemVer das tags do fork] → documentada; sufixo pode ser revisto (Open Questions).
- [Rollback para a imagem do upstream] a tabela é ignorada, os gerenciados deixam de ser monitorados e o histórico deles é apagado; se o YAML não tiver endpoints, a imagem do upstream não sobe → backup do banco antes e YAML com pelo menos um endpoint.

## Migration Plan

Entrega em marcos, cada um num pull request no fork, com CI verde antes do merge:
1. **CI, release e skills**; publicar `v5.36.0-fork.1` (binários pelo workflow, imagem com `make docker-release`).
2. **Visual quadrado.**
3. **Fundação:** hot-reload seguro, registro do watchdog, labels congeladas, lock de ciclo de vida e restauração de alertas reutilizável.
4. **Backend admin** atrás de `admin.enabled`.
5. **Frontend admin, E2E, documentação** e nova release.

Nas instalações com Docker Compose, trocar a imagem para `jniltinho/gatus:<tag>`.

Rollback: voltar para `twinproduction/gatus:v5.36.0` com backup do banco e YAML contendo ao menos um endpoint.

## Open Questions

1. O sufixo `-fork.N` atende mesmo com a ordenação SemVer, ou prefere outro esquema?
2. Vale sincronizar periodicamente várias instâncias no mesmo PostgreSQL (ex.: comparar `max(version)` a cada 30 s)?
3. Vale oferecer `GET /api/v1/admin/export` (YAML dos gerenciados) para facilitar o rollback para o upstream?
