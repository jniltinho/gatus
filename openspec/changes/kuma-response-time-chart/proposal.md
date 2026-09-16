## Why

O gráfico **Response Time Trend** dos detalhes do endpoint não se parece com o gráfico da página do monitor do Uptime Kuma:
- **Períodos:** mostra só médias por hora em 24h, 7d e 30d.
- **Visual:** linha azul com pontos.
- **Quedas e Pending:** faixas contínuas calculadas pelos eventos.

No Kuma, o gráfico abre em **Recent**, com um ponto por heartbeat, e oferece 3h, 6h, 24h e 1w com médias por minuto ou por hora e linhas de mínimo e máximo. A linha é verde e preenchida, sem pontos, e cada check fora do ar (ou pendente) vira uma coluna translúcida vermelha (ou amarela). O dono do fork pediu o gráfico igual ao do Kuma, com os mesmos períodos e as colunas por check.

O Gatus não tem dados para esses períodos:
- guarda a média por hora e, depois de 48 horas, junta as horas em dias;
- guarda só os últimos resultados (100 por padrão);
- não guarda mínimo, máximo nem a contagem de Pending por intervalo.

## What Changes

- **Agregados por minuto e por hora (tabela nova do fork):**
  - depois de gravar cada resultado, o storage soma o resultado num agregado por minuto (guardado por 24 horas) e num agregado por hora (guardado por 7 dias), numa transação curta separada, para uma falha no gráfico nunca desfazer o resultado;
  - cada agregado conta os resultados Up, Down e Pending e guarda a média, o mínimo e o máximo do tempo de resposta dos Up com `ping`;
  - vale para memória, SQLite, PostgreSQL, MySQL e MariaDB.
- **API do gráfico:**
  - `GET /api/v1/endpoints/{key}/response-time-chart?period=recent|3h|6h|24h|1w` (protegida);
  - `GET /api/v1/status-pages/{slug}/endpoints/{key}/response-time-chart?period=...` (pública, só para endpoints mostrados por página publicada, com o 404 idêntico);
  - `recent` devolve os últimos resultados (instante, estado e tempo de resposta): até 100 na rota protegida e até 50 na pública, o mesmo limite das status pages;
  - os demais devolvem os agregados do período (por minuto em 3h, 6h e 24h, por hora em 1w);
  - nenhum dos dois leva mensagem, erro ou condição.
- **Gráfico igual ao do Kuma**, no dashboard e na página pública:
  - seletor **Recent / 3h / 6h / 24h / 1w**, que abre em Recent e lembra a última escolha no navegador;
  - linha verde preenchida sem pontos e, nos agregados, linhas finas de mínimo e máximo, com a mesma junção por janela deslizante do Kuma;
  - colunas translúcidas da altura toda: vermelhas para checks Down e intervalos com Down sem Up, amarelas para Pending e intervalos mistos;
  - intervalos longos sem dados quebram a linha;
  - eixo do tempo `HH:mm` / `MM-dd HH:mm`, eixo Y "Resp. Time (ms)", tooltip com data e hora e `N ms`, alturas do Kuma pela largura da janela.
- **Remoção:** as faixas contínuas de queda e de Pending, com balão de detalhes, saem do gráfico, e o plugin `chartjs-plugin-annotation` sai do projeto. A rota upstream `/api/v1/endpoints/{key}/response-times/{duration}/history` e os badges (30d/7d/24h/1h) continuam.
- **Tempo real:** a cada aviso de resultado novo, o período **Recent** atualiza em seguida. Os agregados atualizam no máximo uma vez por minuto.
- **Documentação e E2E.**

## Capabilities

### New Capabilities

- `response-time-chart`: agregados por minuto e por hora, API do gráfico (protegida e pública) e gráfico no formato do Uptime Kuma nos detalhes do endpoint.

### Modified Capabilities

- `endpoint-details-summary`: as faixas de queda e de Pending do gráfico são substituídas pelas colunas por check ou intervalo.
- `status-page-highlights`: a página pública de detalhes usa o gráfico novo, com o seletor Recent/3h/6h/24h/1w e a API pública do gráfico.
- `realtime-endpoint-updates`: o gráfico atualiza pelo período escolhido (Recent na hora, agregados no máximo a cada 60 segundos).

## Impact

- **Backend:**
  - tabela nova do fork `endpoint_response_time_buckets` (`ON DELETE CASCADE`), criada nos quatro bancos com migração automática, sem tocar em `endpoint_uptimes`;
  - gravação no `InsertEndpointResult`, depois do commit do resultado, com upsert por dialeto e limpeza das linhas antigas;
  - memória com mapas por endpoint;
  - interface `store` com a leitura dos agregados;
  - rotas novas em `api/`;
  - leituras numa interface opcional do fork (`ResponseTimeChartReader`), com Recent por consulta leve, sem o cache de escrita;
  - cache curto da rota pública (sequência de `liveupdates` no Recent, minuto atual nos agregados).
- **Frontend:** `ResponseTimeChart.vue` reescrito, `EndpointDetails.vue` e `StatusPageEndpoint.vue` com o seletor novo, `utils/responseTimeChart.js` (funções puras testadas), remoção de `utils/downtime.js`, do plugin de anotações e de `RESPONSE_TIME_DURATIONS`.
- **Histórico:** os períodos 3h a 1w começam vazios depois da atualização e se preenchem com o tempo. Recent funciona na hora, com os resultados já gravados.
- **Custo de escrita:** uma transação curta a mais por resultado, e a limpeza das linhas antigas fica limitada a uma vez por hora por endpoint.
- **E2E:** os passos atuais do gráfico (`7d` e faixa de queda) são reescritos.
- **Volta ao Gatus original:** a tabela nova é ignorada.
