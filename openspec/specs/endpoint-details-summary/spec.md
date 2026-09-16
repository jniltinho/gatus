# endpoint-details-summary Specification

## Purpose
TBD - created by archiving change refine-push-status. Update Purpose after archive.
## Requirements
### Requirement: Resumo na API protegida de status do endpoint
`GET /api/v1/endpoints/:key/statuses` MUST incluir os campos do fork:
- `push`: verdadeiro para endpoints Push gerenciados em qualquer estado (habilitados, desabilitados ou em conflito) e para external endpoints do arquivo, habilitados ou não;
- `currentResponseTime`: duração em milissegundos do último resultado, `null` quando é zero, independente da página de resultados pedida;
- `uptime`: razões de 0 a 1 em `24h`, `7d` e `30d`, `null` sem execução no período;
- `responseTime`: médias em milissegundos em `24h`, `7d` e `30d`, `null` sem execução no período.

Os valores MUST ser calculados como o uptime e o tempo médio do payload das status pages, com qualquer storage.

#### Scenario: Endpoint Push com execuções
- **WHEN** o endpoint Push `jobs_backup` teve 3 sucessos e 1 falha nas últimas 24 horas, com 100 ms de média
- **THEN** a resposta tem `push: true`, `uptime.24h` 0.75 e `responseTime.24h` 100

#### Scenario: Endpoint recente
- **WHEN** um endpoint ativo começou a ser monitorado há 2 horas
- **THEN** a resposta tem `push: false`, e `uptime` e `responseTime` de `24h`, `7d` e `30d` calculados sobre as execuções existentes

#### Scenario: Período sem execuções
- **WHEN** um endpoint Push teve execuções há 3 dias e nenhuma nas últimas 24 horas
- **THEN** `uptime.24h` e `responseTime.24h` são `null` e `uptime.7d` tem valor

#### Scenario: Segunda página de resultados
- **WHEN** o dashboard pede `GET /api/v1/endpoints/jobs_backup/statuses?page=2`
- **THEN** `currentResponseTime` é a duração do último resultado do endpoint, e não da página pedida

### Requirement: Painel de números dos detalhes do endpoint
A página de detalhes do endpoint no dashboard e a página pública de detalhes MUST mostrar, entre as barras de resultados e o gráfico de tempo de resposta, um único painel com cinco números em texto, nesta ordem:
1. Response (Current), a duração do último resultado (`currentResponseTime` no dashboard);
2. Avg. Response (24h);
3. Uptime (24h);
4. Uptime (7d);
5. Uptime (30d).

No dashboard, com `push: true`, os dois primeiros rótulos MUST ser Ping (Current) e Avg. Ping (24h). O uptime MUST ser mostrado em porcentagem com até duas casas decimais, sem zeros à direita. Um valor `null`, e a duração atual zero ou sem resultado, MUST ser mostrado como "—". O painel MUST substituir os cartões Current Status, Avg Response Time, Response Time Range e Last Check e as imagens de "Uptime Statistics". MUST ser atualizado junto com os resultados e MUST quebrar em duas colunas em telas estreitas.

#### Scenario: Endpoint HTTP
- **WHEN** o endpoint `core_site` tem último resultado de 5 ms, média de 24 horas de 12 ms e uptime de 1 em 24h, 0.995 em 7d e 0.9925 em 30d
- **THEN** o painel mostra `Response (Current) 5 ms`, `Avg. Response (24h) 12 ms`, `Uptime (24h) 100%`, `Uptime (7d) 99.5%` e `Uptime (30d) 99.25%`

#### Scenario: Endpoint Push no dashboard
- **WHEN** o endpoint Push `jobs_backup` é aberto nos detalhes do dashboard
- **THEN** os dois primeiros números têm os rótulos `Ping (Current)` e `Avg. Ping (24h)`

#### Scenario: Endpoint da página pública sem execuções
- **WHEN** um endpoint de uma status page ainda não tem resultados e o visitante abre seus detalhes
- **THEN** os cinco números mostram "—"

#### Scenario: Página pública
- **WHEN** um visitante abre `/status/services/endpoints/core_site`
- **THEN** o painel mostra os cinco números com os valores de `uptime` e `responseTime` do payload de detalhes

