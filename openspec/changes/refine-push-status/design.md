## Context

- **Resultado:** `endpoint.Result` só tem `Success bool`, e todas as camadas perguntam só "sucesso ou falha":
  - alertas (`watchdog/alerting.go`);
  - eventos: `InsertEndpointResult` grava START e o evento do primeiro resultado, e depois compara `success` com o último resultado (`getLastEndpointResultSuccessValue`); o store em memória faz o mesmo em `memory/util.go`;
  - uptime (`endpoint_uptimes`) e métricas (`PublishMetricsForEndpoint`);
  - resumo das status pages (`ResultSummary`, `statuspage/payload.go`, `aggregateStatus`);
  - frontend: `EndpointCard.vue`, `RecentChecksTable.vue`, `StatusBadge.vue`, `Tooltip.vue`, `Home.vue`, `public/EndpointRow.vue` e `views/public/StatusPageEndpoint.vue`.
- **Push (`api/push.go`):** `status=up` (padrão) é sucesso e qualquer outro valor é falha, com `msg` padrão `OK`. Uma falha copia a mensagem para `Errors`.
- **Heartbeat e envios:**
  - o heartbeat (`watchdog/external_endpoint.go`) grava uma falha com o erro `heartbeat: no update received within <intervalo>`, sem mensagem nem origem, a cada intervalo sem envio aceito;
  - `lastPush` é gravado para todo envio aceito;
  - `processExternalEndpointResult` e `SubmitEndpointResult` gravam o resultado, publicam métricas, pulam alertas em manutenção, chamam `HandleAlerting` e copiam os contadores;
  - tudo isso serializado por chave.
- **Mensagem e origem:** ficam na tabela do fork `endpoint_result_messages` (`endpoint_result_id`, `message`, `origin`, `ON DELETE CASCADE`), gravada na mesma transação, depois de `endpoint_results`.
- **API protegida de status (`api/endpoint_status.go`):** só lê o storage e não diz se o endpoint é Push. As rotas `/uptimes/:duration` e `/response-times/:duration` devolvem `text/plain` e `0` num período sem execuções, sem cache no servidor.
- **Payload público:** já calcula `uptime` e `responseTime` nulos sem execução, a partir das somas horárias (`GetEndpointSummaries`, SQL e memória).
- **Detalhes do endpoint no dashboard (`EndpointDetails.vue`), em ordem:**
  1. barras;
  2. quatro cartões (Current Status, Avg Response Time, Response Time Range, Last Check);
  3. "Uptime Statistics" com imagens de badge;
  4. gráfico (`ResponseTimeChart.vue`, linhas tracejadas nos eventos UNHEALTHY e seletor 24h/7d/30d);
  5. tabela de verificações recolhível;
  6. cartões de badges de tempo de resposta;
  7. "Current Health";
  8. "Events".
- **Página pública de detalhes (`StatusPageEndpoint.vue`):**
  - segue o mesmo leiaute, com o mesmo `ResponseTimeChart`;
  - a tabela mostra Status, Date and time e Response time;
  - estados diferentes de `up` e `down` viram "No data".
- **Uptime Kuma 2.5.4, conferido localmente em 2026-09-16:**
  - `status=pending` registra Down, e o painel só tem verde e vermelho;
  - a página do monitor mostra barras, uma linha de números (Response ou Ping atual, Avg. 24h, Uptime 24h/30d/1y) e o gráfico com faixas vermelhas nas quedas.
- **Decisões do dono:**
  - Pending em amarelo e tentativas são funcionalidades do Gatus;
  - a tabela de verificações do dashboard não muda, fora o badge Pending;
  - a página pública ganha a mesma tabela, com mensagens.

## Goals / Non-Goals

**Goals:**
- Status Pending persistido nos quatro storages, sem mudar colunas do upstream.
- `status=pending` e tentativas nos endpoints Push, com Pending em amarelo em todas as telas que colorem resultados.
- Tabela de verificações da página pública igual à do dashboard, com mensagens atrás de uma opção por página e sem erros das verificações ativas.
- Painel de números e gráfico com faixas fora do ar nas páginas de detalhes do dashboard e pública.
- Nada muda para quem não usa `status=pending`, `heartbeat.retries` nem `show-messages`.

**Non-Goals:**
- Status Maintenance (azul) do Kuma.
- Intervalo próprio para as tentativas: a tentativa segue o intervalo do heartbeat.
- Pending em verificações ativas e tentativas em endpoints ativos.
- Mudar a regra de mensagem ou as colunas da tabela do dashboard.
- Uptime de 1 ano, que o Gatus não guarda.
- Suites (`SuiteDetails.vue`).

## Decisions

### D1. Pending como marca do fork sobre um resultado sem sucesso

`endpoint.Result` ganha `Pending bool` (JSON `pending,omitempty`). Um resultado Pending sempre tem `Success: false`, então o código do upstream que só lê `Success` o trata como indisponível no uptime e como falha nas métricas.

A marca é persistida em `endpoint_result_messages.pending`:

| Dialeto | Coluna |
|---------|--------|
| SQLite | `pending INTEGER NOT NULL DEFAULT 0` |
| PostgreSQL | `pending BOOLEAN NOT NULL DEFAULT FALSE` |
| MySQL/MariaDB | `pending BOOLEAN NOT NULL DEFAULT FALSE` |

- **Instalações novas:** a coluna já vem no `CREATE TABLE`.
- **Instalações existentes:** um `ALTER TABLE ... ADD COLUMN` idempotente:
  - SQLite: consulta `PRAGMA table_info`;
  - PostgreSQL: usa `ADD COLUMN IF NOT EXISTS`;
  - MySQL/MariaDB: consulta `information_schema.columns` e tolera o erro 1060 (coluna duplicada) quando duas instâncias sobem juntas.
- **Gravação e leitura:** a linha é gravada quando há mensagem, origem ou Pending. `loadEndpointResultMessages` lê `pending`.
- **Leitura nula:** a leitura usa `sql.NullBool`, porque resultados sem linha de mensagem não têm a coluna. Nenhuma consulta filtra por `pending`.
- **Memória:** o store em memória guarda o `Result` inteiro.
- **Resumo:** `ResultSummary` ganha `Pending`, `Message`, `Origin`, `HTTPStatus` e `Errors` (campo interno, fora de qualquer JSON). O SQL lê esses campos em `endpoint_summary_batch.go` com `LEFT JOIN endpoint_result_messages` e `status` e `errors` de `endpoint_results`, e a memória copia do `Result`. Os `Errors` só servem para reconhecer o prefixo do heartbeat de resultados antigos. Só o payload de detalhes com `show-messages` publica mensagem e origem.
- **Índice:** `endpoint_events` ganha o índice `(endpoint_id, endpoint_event_id)` onde não existir: `CREATE INDEX IF NOT EXISTS` no SQLite e no PostgreSQL. No MySQL/MariaDB, a chave estrangeira já indexa `endpoint_id`. Ele atende a consulta do último evento (D3).

**Alternativas consideradas:**
- **Coluna em `endpoint_results`:** rejeitada, porque muda uma tabela do upstream.
- **`Status string` no lugar de `Success`:** rejeitada, porque exigiria mudar todas as leituras do upstream.
- **Pending fora do uptime:** rejeitada, porque mudaria a contagem do upstream nos quatro storages e esconderia um job que nunca termina.

### D2. `status=pending` no push

`status=pending` (exatamente esse valor) grava um resultado Pending com a mensagem do envio, sem copiar a mensagem para `Errors`. O envio conta como recebido para o heartbeat (`lastPush`) e não mexe nas tentativas (D4). Vale para endpoints Push e para pushes em endpoints ativos. A documentação registra que, no Kuma, `pending` vira Down.

### D3. Alertas e eventos ignoram Pending

- **Alertas:** em `processExternalEndpointResult` e em `SubmitEndpointResult`, um resultado Pending grava e publica métricas, mas não chama `HandleAlerting` nem copia contadores. `NumberOfFailuresInARow`, `NumberOfSuccessesInARow` e os alertas disparados ficam como estavam. Manutenção e `lastPush` não mudam.
- **Eventos:** a decisão compara o resultado com o **tipo do último evento HEALTHY ou UNHEALTHY** do endpoint, e não com o último resultado:
  - um resultado Pending nunca cria evento;
  - um resultado com sucesso cria HEALTHY quando o último evento desse tipo é UNHEALTHY ou não existe;
  - uma falha cria UNHEALTHY quando o último é HEALTHY ou não existe;
  - o START continua sendo gravado quando o endpoint não tem eventos;
  - a ausência de HEALTHY/UNHEALTHY é um resultado vazio válido, e não um erro. Hoje `getLastEndpointResultSuccessValue` devolve `errNoRowsReturned` e o chamador não cria evento, e esse padrão não pode ser copiado. Com o primeiro resultado Pending, o único evento é START, e o sucesso seguinte precisa criar HEALTHY.
- **Implementação:**
  - SQL: uma consulta `SELECT event_type FROM endpoint_events WHERE endpoint_id = $1 AND event_type IN ('HEALTHY','UNHEALTHY') ORDER BY endpoint_event_id DESC LIMIT 1` substitui `getLastEndpointResultSuccessValue` em `InsertEndpointResult`;
  - memória: percorre `Events` do fim para o começo, ignorando START.
- **Resultado:**
  - sem Pending, o comportamento é o mesmo do upstream, porque o último evento desse tipo sempre reflete o último resultado;
  - com Pending, a regra continua certa depois de um primeiro resultado Pending e depois de mais Pending do que `maximum-number-of-results`;
  - a limpeza de eventos antigos mantém os mais recentes.

**Alternativa considerada:** comparar com o último resultado não Pending. Rejeitada no QA: com o primeiro resultado Pending ou com Pending além do limite de resultados, não há resultado para comparar, e os eventos se perdem ou se repetem.

### D4. Tentativas em `heartbeat.retries`

- **Onde:** endpoints Push gerenciados e external endpoints do arquivo com heartbeat ganham `heartbeat.retries`, inteiro de 0 a 100, padrão 0. Fica em `heartbeat` porque só existe onde há heartbeat. O campo é rejeitado nos endpoints ativos e num external endpoint do arquivo sem `heartbeat.interval`.
- **Rotas cobertas:** as tentativas valem para todo resultado de um endpoint com `retries` que passe por `processExternalEndpointResult`, inclusive a API upstream `POST /api/v1/endpoints/:key/external`.
- **Contagem:** um contador por chave, em memória, no mesmo mapa de `lastPush` (sobrevive a recargas e é serializado pelo lock da chave).
  - Um resultado down (envio com status diferente de `up` e `pending`, ou intervalo sem envio) com contador menor que `retries` vira Pending e incrementa o contador.
  - Com o contador esgotado, é gravado como falha normal, com alertas e evento. O contador fica esgotado até um sucesso, então as falhas seguintes são down.
  - Um envio `up` zera o contador. Um `status=pending` explícito não o altera.
  - Em janela de manutenção, o resultado é gravado como hoje, sem alertas, e o contador não avança.
- **Descarte:** o contador e o `lastPush` da chave são apagados:
  - nos fluxos de renomear e remover de `managedendpoint`, depois do commit;
  - na recarga da configuração, depois de `managedendpoint.Load`, para as chaves que deixam de existir entre `cfg.ExternalEndpoints` e os endpoints Push gerenciados. Hoje o ciclo de recarga não tem esse gancho, e ele é criado nesta change.

  Hoje `lastPushes` nunca é apagado.
- **Conversão em Pending:** quando um resultado down vira Pending, `Errors` é esvaziado. Se não há mensagem, os erros unidos por `; ` viram a mensagem.
- **Mensagem do heartbeat:** todo resultado de heartbeat, down ou Pending, grava o texto `heartbeat: no update received within <intervalo>` também em `Message`. Nos down, `Errors` continua com o texto para os alertas. Resultados antigos, só com `Errors`, são reconhecidos pelo prefixo exato `heartbeat: no update received within ` ao montar a mensagem pública.
- **Reinício do processo:** o contador recomeça em 0, o que pode gerar até `retries` Pending a mais antes do próximo down. Isso fica documentado.
- **Alertas:** o `failure-threshold` conta a partir do primeiro down, então as tentativas atrasam o alerta em `retries` intervalos.
- **Formulário:** campo "Retries" ao lado de "Heartbeat interval", com a ajuda "Missed heartbeats or down pushes are recorded as pending this many times before down".

**Alternativas consideradas:**
- **Calcular as tentativas lendo os últimos resultados:** rejeitada, por custar uma consulta a mais por envio e depender do limite de resultados.
- **Campo `retries` na raiz:** rejeitada, porque poderia colidir com campos futuros do upstream nos external endpoints.

### D5. Telas com Pending

- **Cor:** amarelo, com as classes do `degraded` do `StatusBadge` (`bg-yellow-400`/`dark:bg-yellow-500`). Rótulo "Pending".
- **Dashboard:**
  - barras de `EndpointCard.vue` e `Tooltip.vue`;
  - `StatusBadge` ganha o estado `pending` (validator, `currentHealthStatus` em `EndpointDetails.vue` e `EndpointCard.vue`);
  - badge de `RecentChecksTable.vue`;
  - resumo de `Home.vue`, com a contagem de Pending em amarelo;
  - no resumo, um endpoint Pending conta só em Pending, e não em down;
  - contadores de falha de grupo e filtros tratam Pending como falha.
- **Público:**
  - barras, ponto e tooltip de `public/EndpointRow.vue`;
  - estado e tabela de `StatusPageEndpoint.vue`, que deixa de transformar `pending` em "No data";
  - `STATUS_LABELS` com `pending`.
- A tabela de verificações do dashboard não muda além do badge.

### D6. Mesma tabela na página pública, com `show-messages`

- **Opção:** `show-messages` por página, booleana, padrão falso, no YAML (`config/statuspage`) e na administração (checkbox "Show messages" ao lado de "Show certificate expiration", com a mesma validação e versão).
- **Payload de detalhes:** `page` ganha `showMessages`, que reflete a opção. A tela escolhe as colunas por ele, e não pela presença de mensagens (uma página só com verificações TCP não teria nenhuma). Somente com a opção, o payload de detalhes do endpoint (`GET /api/v1/status-pages/:slug/endpoints/:key`) inclui em cada resultado:
  - `message`, montada no servidor a partir de `ResultSummary` pela mesma regra do dashboard, sem os erros:
    - a mensagem do resultado (envio ou heartbeat);
    - senão, o texto de heartbeat reconhecido pelo prefixo nos `Errors` de resultados antigos;
    - senão, `HTTP <código>` de uma verificação ativa;
    - senão, ausente.

    Os demais `Errors` nunca entram;
  - `origin` (`push` quando vier de envio).
- **Payload da página:** `GET /api/v1/status-pages/:slug` continua sem mensagens.
- **Tabela pública:**
  - com a opção, `RecentChecksTable` recebe `show-message` e fica igual à do dashboard (Status, Date and time, Message, Origin);
  - na página pública, o componente usa só `message` e `origin` do payload, sem a montagem do dashboard (mensagem, erros, status HTTP), para nunca publicar erros por engano;
  - sem a opção, continua com Response time.
- **Cache:** mudar a opção cria uma nova revisão, então o cache não serve respostas antigas.

**Alternativa considerada:** publicar sempre. Rejeitada, porque uma mensagem de envio pode ter texto interno. A opção deixa a decisão com o administrador, como `show-certificate-expiration`.

### D7. Painel de números nos detalhes

- **Dados:**
  - a API protegida `GET /api/v1/endpoints/:key/statuses` ganha os campos do fork:
    - `push`: verdadeiro quando a chave é de um endpoint Push gerenciado em qualquer estado (`managedendpoint`) ou de `cfg.ExternalEndpoints` com qualquer `enabled`. Não usa o filtro de `statuspage/registry.go`, que só lista os habilitados;
    - `uptime` e `responseTime` (`24h`, `7d`, `30d`, nulos sem execução);
    - `currentResponseTime`: duração em ms do último resultado, nulo com zero, independente da paginação;
  - os valores são calculados com `GetEndpointSummaries` e as funções de uptime e média do `statuspage`, exportadas, numa leitura a mais só nessa rota;
  - a página pública usa o `uptime` e o `responseTime` que o payload de detalhes já tem.
- **Painel:** único, entre as barras e o gráfico, no dashboard e na página pública, no lugar dos quatro cartões e das imagens de "Uptime Statistics":

  | Número | Valor |
  |--------|-------|
  | Response (Current) | `currentResponseTime` no dashboard e duração do último resultado na página pública ("—" com 0 ou sem resultado) |
  | Avg. Response (24h) | `responseTime.24h` ("—" se nulo) |
  | Uptime (24h) | `uptime.24h` |
  | Uptime (7d) | `uptime.7d` |
  | Uptime (30d) | `uptime.30d` |

- **Rótulos:** com `push: true`, "Response" vira "Ping" no dashboard. Na página pública fica "Response", porque o payload público não diz se o endpoint é Push.
- **Valores:** uptime com até duas casas, sem zeros à direita; "—" se nulo. A formatação reutiliza `formatUptime` de `utils/statusPage.js`. O que faltar fica numa função pura em `utils/detailsSummary.js`, com testes unitários. A média de 24h inclui as durações zero dos envios sem `ping`, como o payload público já faz. Isso fica documentado.
- **Sem resultados:** um endpoint sem resultados continua com 404 na API protegida, e o dashboard mostra a tela de não encontrado como hoje. Os "—" valem para períodos sem execução e para a página pública.
- **Atualização e leiaute:** o painel atualiza com os resultados e quebra em duas colunas em telas estreitas.
- **Cartões que continuam:** badges de tempo de resposta, "Current Health" e "Events". O estado atual fica no "Current Health" e a última verificação na barra.

**Alternativa considerada:** usar `/uptimes/:duration` e `/response-times/:duration`. Rejeitada no QA: devolvem `text/plain`, `0` sem execução (não distinguem de 0%), não têm cache e seriam quatro requisições.

### D8. Faixas fora do ar no gráfico

- **Intervalos:** `ResponseTimeChart.vue` calcula os intervalos a partir dos eventos ordenados:
  - de cada UNHEALTHY até o HEALTHY seguinte, ou até agora;
  - um HEALTHY sem UNHEALTHY anterior só fecha uma queda quando não há START entre os eventos carregados, sinal de que a lista foi truncada. Nesse caso, a faixa começa no maior valor entre o início do período e o primeiro ponto do gráfico;
  - a sequência normal START→HEALTHY não gera faixa.
- **Recorte:** cada intervalo que se sobrepõe ao período selecionado é recortado nele, inclusive quedas iniciadas antes do período.
- **Desenho:** anotações `box` do `chartjs-plugin-annotation`, em vermelho translúcido, com `scales.x.min` e `max` fixados no período, para as faixas não esticarem o eixo.
- **Tooltip:** a faixa mostra o início e a duração.
- **Substituição:** as faixas substituem as linhas tracejadas, no dashboard e na página pública, que usam o mesmo componente.
- **Testes:** o cálculo fica numa função pura em `utils/downtime.js`, com testes unitários.
- **Condições:** Pending não gera faixa.
- **Visibilidade:** hoje o dashboard esconde o gráfico quando nenhum resultado tem duração maior que zero (`EndpointDetails.vue`), e a página pública mostra com qualquer resultado. As duas passam a mostrar o gráfico sempre que houver pelo menos um resultado, para as faixas de queda aparecerem também em endpoints Push sem `ping`.

### D9. Status pages públicas

- **Payload da página:** cada resultado ganha `pending` (só quando verdadeiro), e o estado do endpoint passa a ser `pending` quando o último resultado é Pending.
- **`aggregateStatus`:** `operational` só com todos `up`, `down` só com todos `down`, e `degraded` em qualquer outra mistura de conhecidos, inclusive só `pending`. Hoje estados que não sejam `up` nem `down` são ignorados.
- **Sanitização:** as structs espelho do teste de sanitização passam a conhecer `pending`, e `message` e `origin` só no payload de detalhes.

## Risks / Trade-offs

- **[Pending conta como indisponível no uptime e como falha nas métricas]** → Escolha conservadora e documentada.
- **[Contador de tentativas em memória]** → Um reinício pode gerar até `retries` Pending a mais antes do down, o que só atrasa o alerta. Documentado.
- **[`status=pending` diverge do Kuma]** → Documentado como extensão.
- **[Mensagens públicas]** → Opt-in por página, sem erros de verificações ativas. A documentação avisa que a mensagem do envio é publicada como foi enviada.
- **[Eventos pelo último evento]** → Muda a consulta do upstream em `InsertEndpointResult`. Um teste compara as sequências sem Pending com o comportamento anterior, nos quatro storages.
- **[Leitura extra na API protegida de status]** → Só na rota de um endpoint, usando o resumo em lote que já existe.
- **[Binário anterior]** → Mostra Pending como falha e ignora `heartbeat.retries` e `show-messages` no YAML do arquivo. As definições gerenciadas são decodificadas com campos estritos, então a versão anterior rejeita endpoints e páginas com esses campos.

## Migration Plan

- A coluna `pending` é criada de forma idempotente na inicialização. Não há migração de dados.
- **Rollback:**
  1. Desligar `heartbeat.retries` e `show-messages` nas definições gerenciadas pela administração, removendo os campos.
  2. Voltar ao binário anterior. A coluna, o índice e os campos do YAML do arquivo ficam sem uso.

## Open Questions

Nenhuma que bloqueie.
