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

### Requirement: Cabeçalho e histórico iguais nas duas telas de detalhes
A tela de detalhes do endpoint do dashboard (`/endpoints/<key>`) e a pública (`/status/<slug>/endpoints/<key>`) MUST ter o mesmo cabeçalho e a mesma apresentação do histórico de verificações. A ordem dos blocos já é fixada pelo requisito "Destaques e página de detalhes nas telas" de `status-page-highlights`, e a altura das barras e o tooltip que não ocupa espaço no layout pelo requisito "Página pública de status" de `status-page-web-ui`; este requisito cuida do que falta.

**Cabeçalho:** o nome do endpoint MUST ser o título, com o mesmo tamanho e peso nas duas telas e quebra por palavra; o grupo MUST aparecer na linha abaixo do título nas duas; a expiração do certificado, quando a tela a mostrar, MUST usar as mesmas cores por faixa; e o indicador de estado MUST ficar à direita do título nas duas. Cada tela MUST ter um controle de voltar no mesmo lugar e no mesmo formato, com o destino de cada uma.

**Histórico:** o cartão do histórico MUST mostrar as barras dos últimos resultados com 20 px de altura nas duas telas, com os rótulos de tempo do resultado mais antigo e do mais recente nas pontas, e MUST NOT repetir visualmente o nome, o grupo, o host nem o estado do endpoint, que já estão no cabeçalho da página. O resumo textual para leitores de tela continua como está, com o nome do endpoint.

**Diferenças previstas**, que MUST continuar existindo:
- só o dashboard mostra o host, porque o payload público não publica endereço;
- só o dashboard tem os botões de atualizar e de alternar entre média e mínimo-máximo, a paginação da tabela e o tooltip com as condições e os erros da verificação;
- só a tela pública mostra o horário da última atualização, o link de volta para a status page e a tabela de verificações sanitizada, com mensagens apenas quando a página permitir;
- a quantidade de barras pode ser diferente entre as duas.

#### Scenario: Mesmo cabeçalho
- **WHEN** o mesmo endpoint é aberto nas duas telas, na mesma largura de janela
- **THEN** o título tem o mesmo tamanho nas duas
- **AND** as duas mostram o grupo na linha abaixo do nome

#### Scenario: Nada de host na tela pública
- **WHEN** um visitante abre a tela pública de detalhes de um endpoint
- **THEN** o cabeçalho não mostra o host
- **AND** a tela do dashboard do mesmo endpoint mostra o host

#### Scenario: Mesmo histórico
- **WHEN** o mesmo endpoint é aberto nas duas telas
- **THEN** as barras têm 20 px de altura nas duas
- **AND** nenhuma das duas mostra o nome do endpoint em texto visível dentro do cartão do histórico

#### Scenario: Ações só do dashboard
- **WHEN** um visitante abre a tela pública de detalhes
- **THEN** não há botão de atualizar nem de alternar entre média e mínimo-máximo

