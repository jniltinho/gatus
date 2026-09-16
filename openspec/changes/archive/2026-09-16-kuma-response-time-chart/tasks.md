## 1. Storage: agregados por minuto e por hora

- [x] 1.1 Esquema `endpoint_response_time_buckets` (chave única no `CREATE TABLE`, `ON DELETE CASCADE`) em SQLite, PostgreSQL e MySQL/MariaDB, com `mysql_schema_test.go` atualizado (tabela nova e 9 FKs em cascata).
- [x] 1.2 SQL:
  - depois do commit de `InsertEndpointResult`, transação curta com os upserts de minuto e hora por dialeto:
    - instante `UTC().Truncate`;
    - mínimo e máximo pela fórmula segura com nulos `LEAST(COALESCE(atual, novo), COALESCE(novo, atual))`;
    - só Up com `Milliseconds() > 0` no mínimo, na média e no máximo;
  - falha registrada no log, sem mudar o retorno;
  - retry de MySQL só nessa transação;
  - limpeza no máximo por hora por endpoint, marcada depois do commit, com a entrada do mapa removida em `DeleteAllEndpointStatusesNotInKeys`;
  - relógio injetável (`now`).
- [x] 1.3 Memória:
  - mapas no pacote `memory`, por chave e sob o lock, com poda;
  - `DeleteAllEndpointStatusesNotInKeys` apaga os agregados e passa a pegar o lock.
- [x] 1.4 Interface opcional `ResponseTimeChartReader` (`GetResponseTimeBuckets` e `GetRecentResponseTimeResults`):
  - ordem crescente;
  - chave inexistente devolve lista vazia;
  - Recent SQL por consulta leve com `suite_result_id IS NULL` e `LEFT JOIN` de Pending, sem o cache de escrita.
- [x] 1.5 Testes:
  - conformidade SQL nos quatro bancos e teste equivalente da memória: Up, Down, Pending, durações de 0 ms, abaixo de 1 ms e de 1 ms, e Down depois de Up mantendo mínimo e máximo;
  - falha injetada nos agregados sem perder o resultado (PostgreSQL e MySQL);
  - limpeza com relógio injetado;
  - renomeação SQL (`conformance_rename_test.go`) e cascata;
  - `InsertSuiteResult` sem agregados;
  - concorrência em MySQL/MariaDB;
  - `-race`.

## 2. API do gráfico

- [x] 2.1 `endpointIntervalSeconds(cfg, key)`: arquivo, depois external do arquivo, depois gerenciado (ativo ou push); nulo quando é 0, em conflito ou inválido. Testes com os cinco tipos.
- [x] 2.2 Rota protegida `GET /api/v1/endpoints/{key}/response-time-chart?period=`:
  - existência sem storage e depois o período (400);
  - Recent com `min(100, maximum)`;
  - janelas truncadas (`toBucket − (n − 1) × bucket`) com no máximo 180, 360 e 1.440 minutos e 168 horas;
  - `durationMs` em ms sempre presente e `from = to` no Recent vazio;
  - `no-store`.
- [x] 2.3 Rota pública `GET /api/v1/status-pages/{slug}/endpoints/{key}/response-time-chart?period=`:
  - `IsEndpointShown` antes (404 idêntico) e depois o período (400 por `sendStatusPageError`);
  - Recent com `min(50, maximum)`;
  - cabeçalhos públicos;
  - `chartCache` próprio (LRU com 2.000 entradas e 32 MB), TTL de 30 s e `singleflight` (sequência no Recent, minuto atual nos agregados);
  - registro antes dos catch-alls, com `cfg`.
- [x] 2.4 Testes de API:
  - decodificação estrita sem dados sensíveis;
  - 401, 404, 400 e a ordem entre 404 e 400;
  - endpoint sem dados;
  - limites 100 e 50;
  - Recent público renovado por resultado novo;
  - `intervalSeconds` nulo.

## 3. Frontend

- [x] 3.1 `utils/responseTimeChart.js`, com testes em `node --test`:
  - `recentDatasets`;
  - `bucketDatasets`:
    - porta fiel do mais novo ao mais antigo, com as séries invertidas no fim;
    - janelas;
    - buracos só com intervalo, com os nulos em `anterior − intervalo` e `atual + 60 s`;
    - coluna de valor 1;
    - cores com Pending;
  - `readStoredPeriod`/`storePeriod` com try/catch;
  - testes da posição dos nulos em Recent, 24h e 1w, do agrupamento a partir do mais novo e do valor guardado inválido.
- [x] 3.2 `ResponseTimeChart.vue` reescrito:
  - opções visuais do Kuma (linhas, barras, eixos, tooltip e alturas) para os temas claro e escuro;
  - `data-*` para os testes;
  - spinner só na primeira carga e na troca de período ou endpoint;
  - refresh pendente durante a carga;
  - Recent a cada `refreshKey` e agregados no máximo a cada 60 s;
  - geração contra respostas atrasadas;
  - `credentials: 'omit'` na pública e `PROTECTED_API_HEADERS` com `notifyUnauthorized` na protegida.
- [x] 3.3 `EndpointDetails.vue` e `StatusPageEndpoint.vue`: seletor Recent/3h/6h/24h/1w no cabeçalho do cartão, rotas protegida e pública, sem `events`, `results` e `duration` no gráfico, com os badges sem mudança.
- [x] 3.4 Remover `utils/downtime.js` e seus testes, `chartjs-plugin-annotation` (`package.json` e `package-lock.json`) e `RESPONSE_TIME_DURATIONS`.
- [x] 3.5 Lint, `npm run test:unit` e `make frontend-build`.

## 4. Documentação, E2E e entrega

- [x] 4.1 Documentação:
  - `docs/push-monitoring.md` e `docs/status-pages.md`: gráfico, períodos, cores, rotas e limites, períodos vazios logo depois da atualização, várias instâncias e a tabela nova para quem voltar ao upstream;
  - `AGENTS.fork.md`.
- [x] 4.2 E2E:
  - reescrever o passo de `7d` de `test/e2e/status-pages.sh` e o de faixa de queda de `test/e2e/push.sh`;
  - verificar por `data-*` a coluna amarela de um push Pending em Recent e a coluna vermelha de um Down;
  - trocar para 24h e 1w e verificar a URL com `?period=`;
  - verificar que a escolha fica depois do reload;
  - rota pública e 404;
  - prints claro e escuro, conferidos com print normal depois da animação.
- [x] 4.3 `go test ./... -race` com PostgreSQL, MySQL e MariaDB, `make lint` e `openspec validate kuma-response-time-chart --strict`.
- [x] 4.4 Entrega:
  - PR no `jniltinho/gatus` com CI verde e merge;
  - release `v5.36.0-fork.17` com imagem no Docker Hub;
  - pacote `mariadb`;
  - arquivamento da change.
