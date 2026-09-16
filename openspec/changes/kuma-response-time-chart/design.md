## Context

- **Gráfico atual** (`web/app/src/components/ResponseTimeChart.vue`):
  - busca `/api/v1/endpoints/{key}/response-times/{24h|7d|30d}/history`, rota pública do upstream com médias por hora de `endpoint_uptimes`;
  - desenha as faixas de queda (pelos eventos) e de Pending (pelos resultados) com `chartjs-plugin-annotation`, usado só nesse componente.
- **O que o storage guarda hoje:**
  - `endpoint_uptimes` tem uma linha por hora (execuções, sucessos, soma do tempo de resposta);
  - depois de 48 horas, as horas viram dias (`mergeHourlyUptimeEntriesOlderThanMergeThresholdIntoDailyUptimeEntries`), e os dados passam de 30 dias são apagados;
  - não há mínimo, máximo nem contagem de Pending (Pending fica em `endpoint_result_messages.pending`).
- **Resultados:** `endpoint_results` guarda até `storage.maximum-number-of-results` (100 por padrão), e o SQL deixa passar até 10 a mais antes de limpar.
- **Status pages:** publicam no máximo `min(50, maximum-number-of-results)` resultados (`statuspage.MaximumPublicResults`).
- **Transação e retry:**
  - `insertEndpointResultWithoutRetry` grava resultado, eventos e uptime numa transação só;
  - no PostgreSQL e no conector MySQL do fork (`mysqlTx`), o primeiro statement que falha aborta a transação e o `Commit` falha;
  - no MySQL/MariaDB, `InsertEndpointResult` repete a transação inteira em deadlock (`retryOnTransientMySQLError`).
- **Suites:** gravam por `InsertSuiteResult`, sem passar por `InsertEndpointResult`.
- **Manutenção:** durante uma janela de manutenção, o `watchdog` grava os resultados normalmente e só pula os alertas.
- **Gráfico do Kuma 2.x** (`src/components/PingChart.vue`, `server/uptime-calculator.js`):
  - **Recent:**
    - os últimos 100 heartbeats, que crescem no cliente até 149;
    - a linha só tem valor em UP, e push sem `ping` fica nulo;
    - DOWN, PENDING e MAINTENANCE viram colunas de altura 1 num eixo escondido (0..1), vermelhas, amarelas ou azuis, com opacidade 0,41.
  - **3h, 6h e 24h:** agregados por minuto (180, 360 e 1.440 minutos, contando o atual) com up, down, avgPing, minPing e maxPing, guardados por 24 h. Os agregados são recarregados a cada 5 min.
  - **1w:** 168 agregados por hora, guardados por 30 dias.
  - **Ordem dos agregados:** a lista chega do mais novo ao mais antigo (`getDataArray`), e o `PingChart` foi escrito para essa ordem. As janelas deslizantes (4 pontos até 6h, 12 acima, avançando metade) começam no agregado mais novo, e os pontos nulos de um buraco ficam em `anterior − intervalo` e `atual + 60 s`.
  - **Cores dos agregados:** vermelha quando `up = 0`; amarela com up e down.
  - **PENDING:** o Kuma conta PENDING como DOWN (`flatStatus`), e o push `pending` já chega como DOWN.
  - **Buracos:** só quebram a linha quando o monitor tem intervalo.
- **Rotas do fork para status pages:** verificam `statuspage.IsEndpointShown` antes de ler o storage, respondem com o 404 idêntico e ficam em cache com `singleflight`.

## Goals / Non-Goals

**Goals:**
- Gráfico visual e funcionalmente equivalente ao do Kuma, no dashboard e na página pública.
- Dados por minuto (24 h) e por hora (7 dias) em todos os storages, sem colocar em risco a gravação do resultado.
- Nada sensível novo na rota pública e sem aumentar o histórico público além de 50 resultados.
- Pending em amarelo, que é funcionalidade do Gatus.

**Non-Goals:**
- Recalcular agregados a partir de dados antigos.
- Mudar os badges (30d/7d/24h/1h), `endpoint_uptimes`, a rota upstream de histórico ou o painel de números.
- Colunas azuis de manutenção: os resultados durante a manutenção aparecem pelo estado deles (uma falha fica vermelha), como nas barras atuais.
- Períodos acima de 1 semana.
- Avisos em tempo real entre instâncias diferentes.

## Decisions

### D1. Tabela nova de agregados, separada de `endpoint_uptimes`
`endpoint_response_time_buckets`, com chave única `(endpoint_id, bucket_seconds, bucket_unix_timestamp)` declarada no `CREATE TABLE`, porque o MySQL não tem `CREATE INDEX IF NOT EXISTS`:

| Coluna | Conteúdo |
|---|---|
| `endpoint_id` | FK com `ON DELETE CASCADE` |
| `bucket_seconds` | 60 ou 3600 |
| `bucket_unix_timestamp` | `result.Timestamp.UTC().Truncate(time.Minute ou time.Hour).Unix()` (sem reaproveitar o `Truncate` local de `updateEndpointUptime`) |
| `up_count`, `down_count`, `pending_count` | contagens por estado |
| `timed_up_count` | quantos Up têm duração maior que 0 |
| `up_response_time_total` | soma em ms desses Up |
| `up_response_time_min`, `up_response_time_max` | mínimo e máximo em ms desses Up; nulos quando `timed_up_count = 0` |

- **Estados:** Up é `Success && !Pending`; Pending é `Pending`; Down é o resto.
- **Duração em ms:** o critério único é `Duration.Milliseconds() > 0`, na gravação, na API e no frontend. `endpoint_results.duration` guarda nanossegundos, e a consulta leve converte para ms.
- **Up sem duração:** um Up com 0 ms (push sem `ping`, ou check abaixo de 1 ms) conta em `up_count`, mas fica fora da média, do mínimo e do máximo, como o ping nulo do Kuma.
- **Alternativa rejeitada:** colunas em `endpoint_uptimes`. A tabela é do upstream, e a junção em dias apagaria a resolução por hora de 1w.

### D2. Gravação numa transação curta depois do commit do resultado
`InsertEndpointResult` grava os agregados depois do `Commit` da transação principal, numa transação própria (SQL) ou sob o lock do store (memória):
- **Onde roda:** ainda dentro de `InsertEndpointResult`, sob o lock da chave do `watchdog` e antes do `liveupdates.Publish`.
- **Upserts de minuto e hora:** somam as contagens e o total e combinam o mínimo e o máximo por uma fórmula segura com nulos nos três dialetos, porque `LEAST`/`GREATEST` do MySQL e `MIN`/`MAX` escalares do SQLite devolvem `NULL` com qualquer argumento nulo:
  - mínimo = `LEAST(COALESCE(atual, novo), COALESCE(novo, atual))`, com `MIN` escalar no SQLite e `excluded.`/`VALUES()` para o novo valor;
  - máximo = `GREATEST(...)` (ou `MAX`) com a mesma forma;
  - assim, um Down, Pending ou Up de 0 ms no mesmo intervalo (novo valor nulo) mantém o mínimo e o máximo acumulados;
  - no MySQL/MariaDB, `ON DUPLICATE KEY UPDATE ... VALUES()`, como `mysqlUpsertHourlyUptimeQuery`;
  - teste de conformidade: Up de 10 ms e depois um Down no mesmo minuto mantêm o mínimo em 10 nos quatro bancos.
- **Falha:** uma falha nessa transação curta desfaz só os agregados, é registrada no log e não muda o retorno de `InsertEndpointResult`. O resultado já foi gravado. O preço é perder o agregado desse resultado.
- **Retry:** no MySQL/MariaDB, a transação curta usa `retryOnTransientMySQLError`, sem repetir a gravação do resultado. O retry da transação principal não grava agregados, porque eles só são gravados depois do commit.
- **Alternativa rejeitada:** a mesma transação. Com PostgreSQL e MySQL, uma falha do upsert desfaria o resultado. `SAVEPOINT` exigiria mudar `mysqlTx`.

### D3. Retenção e limpeza
- **Retenção:** minuto por 24 h e hora por 7 dias, com mais uma hora de folga.
- **Limpeza:** `DELETE ... WHERE endpoint_id = $1 AND bucket_seconds = $2 AND bucket_unix_timestamp < $3` na transação curta do D2, no máximo uma vez por hora por endpoint.
- **Controle da última limpeza:** um mapa em memória do store com o instante da última limpeza de cada endpoint, atualizado **só depois do commit**. É zerado em `Clear`/`Close`, e a entrada de uma chave sai em `DeleteAllEndpointStatusesNotInKeys`.
- **Relógio:** o store tem `now func() time.Time`, injetável nos testes.
- **Concorrência:** no MySQL/MariaDB, os bloqueios de faixa do `DELETE` podem conflitar com upserts de outros endpoints. O risco fica isolado na transação curta com retry e é coberto por um teste concorrente.
- **Várias instâncias:** cada uma limpa por si, sem conflito.
- **Memória:** poda no mesmo ritmo.
- **Tamanho:** até 1.440 + 168 linhas por endpoint.

### D4. Leitura: interface opcional do fork
`storage/store/response_time_chart.go` define `ResponseTimeChartReader`, obtido por `GetResponseTimeChartReader()`, no padrão de `EndpointSummaryBatchReader`:
- **`GetResponseTimeBuckets(key string, bucketSeconds int, from, to time.Time) ([]common.ResponseTimeBucket, error)`:**
  - devolve os agregados com pelo menos um resultado, em ordem crescente de instante;
  - chave inexistente devolve lista vazia, sem erro.
- **`GetRecentResponseTimeResults(key string, limit int) ([]common.RecentResponseTimeResult, error)`:**
  - consulta leve (`timestamp`, `success`, `duration` e `pending` por `LEFT JOIN` em `endpoint_result_messages`) com `suite_result_id IS NULL`;
  - pega os `limit` mais recentes por `endpoint_result_id` e devolve em ordem crescente de `timestamp`;
  - não usa nem renova o write-through cache;
  - chave inexistente devolve lista vazia.
- **Memória:**
  - os mapas dos agregados ficam no pacote `memory`, por chave, sob o lock do store, sem mudar `endpoint.Uptime`;
  - `DeleteAllEndpointStatusesNotInKeys` apaga os agregados das chaves removidas e passa a pegar o lock;
  - a memória não implementa `RenameManagedEndpoint`, e nada muda nisso.
- **SQL:** a renomeação (`RenameManagedEndpoint`) mantém o `endpoint_id`, e os agregados seguem junto (teste em `conformance_rename_test.go`).
- **Esquema MySQL:** `mysql_schema_test.go` passa a listar a tabela nova e a esperar 9 chaves estrangeiras com `ON DELETE CASCADE`.

### D5. API do gráfico
**Rotas:**
- **Protegida:** `GET /api/v1/endpoints/{key}/response-time-chart?period=...`. A existência é verificada como em `endpointEventsHandler` (404 JSON), depois vem o `period`, e a resposta leva `Cache-Control: no-store`.
- **Pública:** `GET /api/v1/status-pages/{slug}/endpoints/{key}/response-time-chart?period=...`, no bloco público antes dos catch-alls.
  - `IsEndpointShown` vem primeiro: 404 idêntico por `statusPageNotFound`, contando no limitador.
  - Depois vem o `period`: 400 por `sendStatusPageError`, com cabeçalhos públicos e `no-store`.
  - A resposta de sucesso usa os cabeçalhos públicos e o `Cache-Control` da rota pública de detalhes.
- **`period`:** um de `recent`, `3h`, `6h`, `24h` ou `1w`; ausente ou inválido responde 400 JSON.

**Limite de Recent:**
- **protegida:** `min(100, maximum-number-of-results)`;
- **pública:** `min(50, maximum-number-of-results)`, igual às status pages.

**Janela de cada período** (`to` é o instante da resposta):

| Período | Agregados | Janela |
|---|---|---|
| `3h` | por minuto | 180 minutos, contando o atual |
| `6h` | por minuto | 360 minutos, contando o atual |
| `24h` | por minuto | 1.440 minutos, contando o atual |
| `1w` | por hora | 168 horas, contando a atual |
| `recent` | — | do primeiro ao último resultado devolvido; com lista vazia, `from = to` |

Nos agregados, a janela é `toBucket = now.UTC().Truncate(bucket)` e `from = toBucket − (n − 1) × bucket`, com n = 180, 360, 1.440 ou 168. Assim, a resposta tem no máximo n agregados, e os testes verificam esse máximo.

**Resposta:**
```json
{"period":"recent","intervalSeconds":60,"from":"...","to":"...",
 "results":[{"timestamp":"...","status":"up","durationMs":12}]}
```
```json
{"period":"24h","intervalSeconds":60,"bucketSeconds":60,"from":"...","to":"...",
 "buckets":[{"timestamp":"...","up":3,"down":0,"pending":1,"avgMs":20,"minMs":10,"maxMs":35}]}
```
- **`status`:** `up`, `down` ou `pending`.
- **`durationMs`:** `Duration.Milliseconds()`, sempre presente (sem `omitempty`), inclusive 0.
- **`avgMs`, `minMs` e `maxMs`:** nulos sem Up com duração maior que 0.
- **Proibido na resposta:** mensagem, erro, hostname ou condição.

**`intervalSeconds`:** calculado por `endpointIntervalSeconds(cfg, key)`, usado pelas duas rotas, com nulo quando o valor é 0 ou não existe:
1. endpoint do arquivo: `Interval`;
2. external endpoint do arquivo: `Heartbeat.Interval`;
3. endpoint gerenciado: `Endpoint.Interval` ou `Push.Heartbeat.Interval` (nulo em conflito ou inválido).

A ordem segue a precedência do arquivo sobre o gerenciado. A rota pública recebe `cfg` no registro.

**Cache da rota pública:**
- cache próprio do gráfico (`chartCache`), separado de `publicCache`. O `publicCache` tem só 1.000 entradas, e uma página pode ter 200 endpoints × 5 períodos: o gráfico tiraria do cache os payloads das páginas. O `chartCache` é LRU, com no máximo 2.000 entradas e 32 MB de memória (`WithMaxMemoryUsage`);
- TTL de 30 s, com `singleflight` próprio e o semáforo existente;
- chave de Recent: `slug|revision|generation|key|recent|sequence`;
- chave dos agregados: `slug|revision|generation|key|period|minuto atual`, sem a sequência, para não invalidar a cada resultado nem encher o LRU;
- `intervalSeconds` não entra na chave, então uma edição do endpoint aparece em até 30 s.

**Várias instâncias:** a sequência é por processo, então um resultado gravado por outra instância só aparece quando a entrada expira, como nos detalhes.

### D6. Gráfico no frontend
`ResponseTimeChart.vue` é reescrito com a API nova.

**Props:**
- `chartUrl`: base da rota, protegida ou pública;
- `period`, controlado pelo pai;
- `refreshKey`;
- `publicRoute`: busca com `credentials: 'omit'`; sem ele, com `include` e `PROTECTED_API_HEADERS`, e um 401 chama `notifyUnauthorized()`.

**Funções puras** em `utils/responseTimeChart.js`, testadas com `node --test`.
- **`recentDatasets(results, intervalSeconds)`**, porta de `getChartDatapointsFromHeartbeatList` na ordem crescente:
  - **linha:** `durationMs` quando `status = up` e `durationMs > 0`, senão nulo;
  - **ordem:** percorre a lista em ordem crescente e marca a posição dos nulos para essa ordem, o que é diferente dos agregados;
  - **coluna:** altura 1, vermelha no Down e amarela no Pending; no Up, altura 0;
  - **buracos:** com `intervalSeconds` e um intervalo maior que 10 × intervalo, entram pontos nulos em `anterior + intervalo` e `atual − intervalo`.
- **`bucketDatasets(buckets, period, intervalSeconds)`**, porta fiel de `getChartDatapointsFromStats`:
  - percorre a lista do mais novo ao mais antigo, como o Kuma, e inverte as séries no fim, para o eixo ficar crescente;
  - **junção:** janela de 4 agregados até 6h e de 12 em 24h e 1w, avançando metade, só com Up e com mais que o dobro da janela;
  - **buracos:** só com `intervalSeconds`, maiores que `max(10 min, 10 × intervalo)` até 24h e `max(10 h, 10 × intervalo)` em 1w. Nesse caso, os pontos nulos ficam nas abscissas do Kuma na ordem do mais novo ao mais antigo, `anterior − intervalo` e `atual + 60 s` (60 s também em 1w), com teste da posição em 24h e em 1w;
  - **linha:** média com `up > 0 && avgMs > 0`, e as mesmas regras nas séries de mínimo e máximo;
  - **coluna:** valor 1 quando `down + pending > 0`, senão 0, para a coluna ter sempre a altura toda no eixo de 0 a 1;
    - vermelha quando `up = 0 && down > 0`;
    - amarela quando `pending > 0 && down = 0`, ou quando `up > 0 && down + pending > 0`;
    - transparente quando `down + pending = 0`.
- **Período guardado:** `readStoredPeriod()`/`storePeriod()` em `localStorage` (`gatus:response-time-chart-period`) com try/catch. Um valor inválido volta para `recent`. As duas views usam essas funções.

**Visual do Kuma:**

| Série | Cor da linha | Preenchimento |
|---|---|---|
| média | `#5CDD8B` | `#5CDD8B38` no Recent, `#5CDD8B06` nos agregados |
| mínimo | `#3CBD6B38` | `#5CDD8B06` |
| máximo | `#7CBD6B38` | `#5CDD8B06` |

- **Linhas:** `fill: "origin"` e `tension 0.2` em todas; pontos com `radius 0` e `hitRadius 100`.
- **Barras:** borda transparente, `barThickness: "flex"`, `barPercentage` e `categoryPercentage` 1, `inflateAmount: 0.05`, eixo `y1` escondido de 0 a 1.
- **Eixos:**
  - X de tempo com `minUnit: "minute"`, `round: "second"`, formatos `HH:mm`/`MM-dd HH:mm` e `ticks` com `sampleSize 3`, `maxRotation 0`, `autoSkipPadding 30`, `padding 3`;
  - Y com o título "Resp. Time (ms)";
  - `bounds: "ticks"` e `layout.padding` {10, 30, 30, 10}.
- **Grade:** `rgba(0,0,0,0.1)` no claro e `rgba(255,255,255,0.1)` no escuro.
- **Tooltip:**
  - `mode nearest`, `intersect false`, `padding 10`, só a série 0;
  - título `yyyy-MM-dd HH:mm:ss` e valor ` N ms` com `Intl.NumberFormat`;
  - fundo `rgba(212,232,222,1)` no claro e `rgba(32,42,38,1)` no escuro.
- **Legenda:** nenhuma.
- **Chart.js:** registra `LineController`, `BarController`, `LineElement`, `BarElement`, `PointElement`, `LinearScale`, `TimeScale`, `Tooltip` e `Filler`, com o adaptador `date-fns` que já existe.
- **Altura:** 250, 300, 320 ou 275 px pelos cortes 992/768/576, usando a largura da janela (`window.innerWidth`, recalculada no `resize`) em vez do `screen.width` do Kuma.
- **Gancho de teste:** o contêiner expõe `data-period`, `data-line-points` (pontos com valor), `data-down-columns` e `data-pending-columns`, calculados das séries, para o E2E verificar sem ler pixels.
- **Carga:** o spinner aparece só na primeira carga e na troca de período ou endpoint.
- **Remoções:**
  - `utils/downtime.js` e seus testes;
  - `chartjs-plugin-annotation` do `package.json` e do `package-lock.json`;
  - `RESPONSE_TIME_DURATIONS` de `utils/statusPage.js`.

**Diferenças conscientes do Kuma:**
- Recent com 100 resultados (50 na página pública) em vez de crescer até 149;
- agregados atualizados a cada 60 s em vez de 5 min;
- Pending em amarelo com cores próprias nos agregados;
- altura pela largura da janela;
- cartão "Response Time Trend" do fork com o seletor no cabeçalho.

### D7. Atualização
- **Recent:** a cada `refreshKey` novo, busca em seguida (os avisos chegam juntados em 500 ms por `liveUpdates`).
- **3h a 1w:** no máximo uma busca a cada 60 s, com a busca adiada para o fim do intervalo.
- **`refreshKey` durante uma carga:** marca um refresh pendente, feito quando a carga termina.
- **Respostas atrasadas:** descartadas por geração.
- **Nas atualizações:** o gráfico não é apagado nem mostra spinner.

### D8. Dashboard e página pública
- **Seletor:** as duas views mostram Recent/3h/6h/24h/1w no cabeçalho do cartão, com o valor inicial de `readStoredPeriod()`.
- **Dashboard:** usa a rota protegida.
- **Página pública:** usa a rota pública com `publicRoute`.
- **Badges** (30d/7d/24h/1h): não mudam.
- **Props antigas:** as duas views deixam de passar `events`, `results` e `duration` ao gráfico.

## Risks / Trade-offs

- **Períodos vazios logo depois da atualização.** Mitigação: Recent é o padrão e funciona na hora; a documentação avisa.
- **Agregado perdido numa falha da transação curta.** É aceito e registrado no log: o resultado vale mais que o gráfico.
- **Queda do processo entre o commit e a transação curta.** O agregado de um resultado se perde, sem inconsistência no resto.
- **Escrita a mais (uma transação curta por resultado).** Mitigação: dois upserts na chave única e limpeza no máximo uma vez por hora. Conformidade e `-race` nos quatro bancos, e teste concorrente em MySQL/MariaDB.
- **Fuso.** Instantes truncados em UTC, como o Kuma (`dayjs.utc()`). Em fusos de 30 ou 45 minutos, as horas de 1w não caem na hora cheia local (aceito).
- **`VALUES()` obsoleto no MySQL 8.** Já é usado no fork e testado no CI com 8.4 e 9.x.
- **Colunas no lugar das faixas.** Uma queda longa vira várias colunas, como no Kuma, e o balão "Down for…" some. Os eventos continuam na lista.
- **Resultados na manutenção.** Aparecem pelo estado, sem cor própria.

## Migration Plan

1. A criação automática do esquema adiciona a tabela nos quatro bancos, sem migração de dados.
2. **Rollback para uma versão anterior do fork, ou volta ao upstream:** a tabela é ignorada. A documentação cita o nome dela para quem quiser apagá-la.

## Open Questions

Nenhuma.
