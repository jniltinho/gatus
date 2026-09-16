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

### Requirement: Períodos fora do ar no gráfico
O gráfico de tempo de resposta dos detalhes do endpoint, no dashboard e na página pública, MUST destacar cada período fora do ar como uma faixa vermelha translúcida:
- do evento UNHEALTHY até o evento HEALTHY seguinte, ou até o instante atual quando o endpoint continua fora do ar;
- um HEALTHY sem UNHEALTHY anterior MUST fechar uma faixa somente quando não houver START entre os eventos carregados (lista truncada), começando no maior valor entre o início do período e o primeiro ponto do gráfico;
- a sequência START→HEALTHY MUST NOT gerar faixa;
- cada faixa MUST ser recortada no período selecionado (24h, 7d ou 30d), inclusive quando a queda começou antes dele;
- o eixo do tempo MUST ficar fixo no período selecionado.

O gráfico MUST ser mostrado, no dashboard e na página pública, sempre que o endpoint tiver pelo menos um resultado, inclusive com todas as durações zero (envios sem `ping`). As faixas MUST substituir as linhas tracejadas dos eventos. Ao passar o mouse sobre a faixa, o gráfico MUST mostrar o início e a duração da queda. Resultados Pending MUST gerar faixas amarelas translúcidas, do primeiro resultado Pending de uma sequência até o resultado seguinte que não seja Pending, ou até o instante atual quando o último resultado é Pending, calculadas a partir dos resultados recebidos pela tela e recortadas no período, com o início e a duração no tooltip.

#### Scenario: Queda de 10 minutos
- **WHEN** o endpoint ficou fora do ar das 10:00 às 10:10 de hoje e o período selecionado é 24h
- **THEN** o gráfico mostra uma faixa vermelha das 10:00 às 10:10
- **AND** o tooltip da faixa informa o início às 10:00 e a duração de 10 minutos

#### Scenario: Queda em andamento
- **WHEN** o último evento do endpoint é UNHEALTHY às 09:00
- **THEN** a faixa vai das 09:00 até o instante atual

#### Scenario: Queda iniciada antes do período
- **WHEN** o endpoint caiu há 8 dias, voltou há 6 dias e o período selecionado é 7d
- **THEN** a faixa vai do início do período até há 6 dias

#### Scenario: Endpoint novo sem quedas
- **WHEN** os eventos do endpoint são START e HEALTHY, dentro do período
- **THEN** o gráfico não mostra faixas

#### Scenario: Lista de eventos truncada
- **WHEN** os eventos carregados começam com um HEALTHY há 2 horas, sem START, e o primeiro ponto do gráfico é de há 5 horas
- **THEN** a faixa vai de há 5 horas até há 2 horas

#### Scenario: Push sem ping
- **WHEN** um endpoint Push só recebeu envios sem `ping` e teve uma queda
- **THEN** o dashboard e a página pública mostram o gráfico com a faixa da queda

#### Scenario: Queda fora do período
- **WHEN** a única queda ocorreu e terminou há 10 dias e o período selecionado é 7d
- **THEN** o gráfico não mostra faixas

#### Scenario: Faixa amarela de Pending
- **WHEN** o endpoint Push recebe `status=pending` às 10:05 e `status=up` às 10:07, e o período selecionado é 24h
- **THEN** o gráfico, no dashboard e na página pública, mostra uma faixa amarela das 10:05 às 10:07
