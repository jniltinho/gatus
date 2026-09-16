## ADDED Requirements

### Requirement: Agregados de tempo de resposta por minuto e por hora
Para cada resultado gravado por `InsertEndpointResult` (verificação, push, heartbeat ou API upstream de external endpoints), o storage MUST somar o resultado no agregado do minuto e no da hora do seu instante, truncados em UTC (inclusive em servidores com fuso local de meia hora). Resultados de suites MUST NOT entrar. Todos os storages MUST fazer isso: memória, SQLite, PostgreSQL, MySQL e MariaDB.

**Conteúdo de cada agregado:**
- a quantidade de resultados Up (`Success` e não Pending), Down (não `Success` e não Pending) e Pending;
- a quantidade de resultados Up com duração de pelo menos 1 ms e a soma, o mínimo e o máximo, em milissegundos inteiros, das durações desses resultados;
- mínimo e máximo nulos quando nenhum Up tiver pelo menos 1 ms;
- um resultado Down, Pending ou Up abaixo de 1 ms MUST NOT alterar o mínimo e o máximo já acumulados.

**Gravação:**
- nos storages SQL, a tabela do fork `endpoint_response_time_buckets` MUST ser criada automaticamente, com chave única por endpoint, tamanho e instante e remoção em cascata junto com o endpoint;
- os agregados MUST ser gravados depois do commit do resultado, numa transação própria, antes de `InsertEndpointResult` retornar;
- uma falha nessa gravação MUST ser registrada no log e MUST NOT mudar o resultado gravado nem o retorno de `InsertEndpointResult`.

**Retenção:**
- agregados por minuto MUST ser mantidos por pelo menos 24 horas, e os por hora por pelo menos 7 dias;
- linhas mais antigas que a retenção mais uma hora MUST ser apagadas;
- a limpeza MUST acontecer no máximo uma vez por hora por endpoint em cada instância, e só MUST ser considerada feita depois do commit.

**Renomeação e remoção:**
- nos storages SQL, renomear um endpoint gerenciado MUST manter os agregados com a chave nova;
- a remoção de endpoints da configuração MUST apagar os agregados deles.

A tabela `endpoint_uptimes` e a rota `/api/v1/endpoints/{key}/response-times/{duration}/history` MUST NOT mudar.

#### Scenario: Minuto com Up, Pending e Down
- **WHEN** o endpoint `jobs_backup` recebe, no mesmo minuto, pushes Up com `ping=10` e `ping=30`, um Up sem `ping`, um `status=pending` e um `status=down`
- **THEN** o agregado desse minuto tem `up=3`, `pending=1`, `down=1`, média de 20 ms, mínimo 10 ms e máximo 30 ms
- **AND** o agregado da hora contém os mesmos valores somados

#### Scenario: Down não apaga o mínimo
- **WHEN** um minuto recebe um Up de 10 ms e depois um Down
- **THEN** o agregado mantém mínimo e máximo de 10 ms em SQLite, PostgreSQL, MySQL, MariaDB e memória

#### Scenario: Paridade entre os bancos
- **WHEN** a mesma sequência de resultados é gravada em SQLite, PostgreSQL, MySQL e MariaDB, e na memória
- **THEN** a leitura dos agregados devolve os mesmos valores em todos

#### Scenario: Falha nos agregados
- **WHEN** a gravação do agregado falha em PostgreSQL ou MySQL
- **THEN** o resultado, os eventos e o uptime continuam gravados e a falha aparece no log

#### Scenario: Limpeza
- **WHEN** um endpoint tem agregados por minuto de 26 horas atrás, grava um resultado novo e a última limpeza foi há mais de uma hora
- **THEN** os agregados por minuto com mais de 25 horas são apagados e os por hora com menos de 7 dias continuam

#### Scenario: Endpoint removido ou renomeado
- **WHEN** um endpoint gerenciado é renomeado e depois removido
- **THEN** os agregados acompanham a chave nova e depois são apagados

#### Scenario: Resultado de suite
- **WHEN** uma suite grava o resultado de um endpoint dela
- **THEN** nenhum agregado é criado ou alterado

### Requirement: API do gráfico de tempo de resposta
O sistema MUST oferecer os dados do gráfico em duas rotas, com o parâmetro `period` igual a `recent`, `3h`, `6h`, `24h` ou `1w`.

**Rota protegida** — `GET /api/v1/endpoints/{key}/response-time-chart?period=...`:
- MUST exigir a mesma autenticação de `/api/v1/endpoints/{key}/statuses`;
- MUST responder 404 JSON para uma chave que não seja de um endpoint do arquivo, de um external endpoint ou de um endpoint gerenciado, sem ler o storage;
- MUST usar `Cache-Control: no-store`.

**Rota pública** — `GET /api/v1/status-pages/{slug}/endpoints/{key}/response-time-chart?period=...`:
- MUST responder o 404 idêntico das status pages, contando no limitador e sem ler o storage, quando a página não estiver publicada ou não mostrar a chave;
- MUST usar os cabeçalhos das rotas públicas.

**Validação:** a existência MUST ser verificada antes do `period`. Um `period` ausente ou inválido MUST responder 400 JSON com `Cache-Control: no-store`.

**Conteúdo da resposta:**
- **`recent`:** os resultados mais recentes do endpoint, sem resultados de suite, em ordem crescente de instante, cada um só com `timestamp`, `status` (`up`, `down` ou `pending`) e `durationMs` (milissegundos inteiros, sempre presente):
  - na rota protegida, no máximo `min(100, storage.maximum-number-of-results)`;
  - na pública, no máximo `min(50, storage.maximum-number-of-results)`.
- **`3h`, `6h` e `24h`:** os agregados por minuto dos últimos 180, 360 e 1.440 minutos, contando o atual (no máximo 180, 360 e 1.440 agregados).
- **`1w`:** os agregados por hora das últimas 168 horas, contando a atual (no máximo 168 agregados).
- **Campos de cada agregado:** `timestamp`, `up`, `down`, `pending`, `avgMs`, `minMs` e `maxMs`, em ordem crescente e sem agregados vazios; `avgMs`, `minMs` e `maxMs` são nulos sem Up com duração maior que 0.
- **Campos gerais:** `period`, `from`, `to`, `bucketSeconds` (nos agregados) e `intervalSeconds`. O intervalo é o do endpoint do arquivo, o do heartbeat do external endpoint do arquivo, ou o do endpoint gerenciado, nesta ordem, e fica nulo quando é 0 ou não existe.
- **Dados proibidos:** a resposta MUST NOT conter mensagem, erros, hostname, condições ou qualquer outro campo do resultado.
- **Endpoint sem dados no storage:** responde 200 com a lista vazia e, no Recent, `from` igual a `to`.
- **Recent:** MUST NOT usar nem renovar o cache de escrita do storage.

**Cache da rota pública:** a resposta MUST ficar no máximo 30 segundos num cache próprio do gráfico, limitado em entradas e memória, que MUST NOT expulsar do cache as respostas das páginas e dos detalhes.
- Recent: a chave MUST incluir a sequência de resultados do endpoint na instância, para um resultado novo gravado por ela gerar uma resposta nova.
- Demais períodos: a chave MUST incluir o minuto atual no lugar da sequência.

#### Scenario: Recent protegido
- **WHEN** `storage.maximum-number-of-results` é 200, o endpoint tem 120 resultados e o administrador pede `/api/v1/endpoints/jobs_backup/response-time-chart?period=recent`
- **THEN** a resposta tem os 100 mais recentes em ordem crescente, só com instante, estado e duração

#### Scenario: Recent público limitado
- **WHEN** um visitante pede `period=recent` na rota pública de um endpoint com 100 resultados
- **THEN** a resposta tem os 50 mais recentes

#### Scenario: Agregados da semana
- **WHEN** um visitante pede `/api/v1/status-pages/jobs/endpoints/jobs_backup/response-time-chart?period=1w`
- **THEN** a resposta tem os agregados por hora das últimas 168 horas, sem mensagens nem erros

#### Scenario: Endpoint fora da página
- **WHEN** um visitante pede `/api/v1/status-pages/infra/endpoints/database_pg/response-time-chart?period=30d` e `database_pg` não está na página `infra`
- **THEN** a resposta é o 404 idêntico das status pages, e não 400

#### Scenario: Período inválido
- **WHEN** o administrador pede `period=30d` para um endpoint existente
- **THEN** a resposta é 400 com corpo JSON

#### Scenario: Endpoint ainda sem resultados
- **WHEN** o administrador pede o gráfico de um endpoint Push recém-criado
- **THEN** a resposta é 200 com a lista vazia

#### Scenario: Push sem heartbeat
- **WHEN** o gráfico de um external endpoint sem `heartbeat.interval` é pedido
- **THEN** `intervalSeconds` é nulo

#### Scenario: Resultado novo renova o Recent público
- **WHEN** a resposta pública de Recent está em cache e o endpoint grava um resultado na mesma instância
- **THEN** a próxima requisição de Recent já contém o resultado novo

### Requirement: Gráfico no formato do Uptime Kuma
O cartão **Response Time Trend** dos detalhes do endpoint, no dashboard e na página pública, MUST mostrar o gráfico no formato da página do monitor do Uptime Kuma.

**Seletor e visibilidade:**
- o seletor MUST ficar no cabeçalho do cartão, com **Recent, 3h, 6h, 24h e 1w**;
- Recent MUST ser o padrão, e a última escolha MUST ser lembrada no navegador;
- um valor guardado inválido ou ilegível MUST voltar para Recent;
- o gráfico MUST aparecer sempre que o endpoint tiver pelo menos um resultado, inclusive só com pushes sem `ping`.

**Recent:**
- a linha MUST ter a duração dos resultados Up com pelo menos 1 ms e nenhum valor nos demais;
- cada resultado Down MUST gerar uma coluna vermelha translúcida da altura toda, e cada Pending uma coluna amarela translúcida;
- com `intervalSeconds` presente, um intervalo entre resultados maior que 10 vezes esse valor MUST quebrar a linha.

**3h, 6h, 24h e 1w:**
- MUST mostrar as linhas de média, mínimo e máximo, e a linha MUST ficar sem valor nos agregados sem Up ou com média nula;
- os agregados com Up MUST ser juntados, a partir do mais novo, em janelas deslizantes de 4 agregados até 6h e de 12 em 24h e 1w, avançando metade da janela, quando houver mais que o dobro de agregados da janela;
- cada agregado com Down ou Pending MUST gerar uma coluna translúcida da altura toda, qualquer que seja a quantidade:
  - vermelha quando não houver Up e houver Down;
  - amarela quando houver Pending sem Down, ou Up junto com Down ou Pending;
- com `intervalSeconds` presente, um intervalo sem dados maior que o maior valor entre 10 minutos e 10 vezes esse intervalo (até 24h), ou entre 10 horas e 10 vezes esse intervalo (em 1w), MUST quebrar a linha;
- sem `intervalSeconds`, a linha MUST NOT ser quebrada.

**Visual:**
- linha da média verde (`#5CDD8B`), preenchida, suavizada e sem pontos visíveis;
- linhas de mínimo e máximo translúcidas;
- colunas com as cores do Kuma (`rgba(220,53,69,0.41)` e `rgba(245,182,23,0.41)`);
- eixo do tempo com `HH:mm` e `MM-dd HH:mm`;
- eixo Y com o título "Resp. Time (ms)";
- tooltip só da linha da média, com data e hora e o valor em `ms`;
- sem legenda;
- altura de 250 px em janelas a partir de 992 px, 300 px abaixo de 992 px, 320 px abaixo de 768 px e 275 px abaixo de 576 px;
- cores de grade e de tooltip próprias para os temas claro e escuro.

**Carga e testes:**
- o gráfico MUST mostrar o spinner só na primeira carga e na troca de período ou de endpoint;
- o contêiner MUST expor `data-period`, `data-line-points`, `data-down-columns` e `data-pending-columns` para os testes.

**Remoções:** as faixas contínuas de queda e de Pending e o plugin de anotações MUST ser removidos. Os badges de tempo de resposta e de saúde MUST NOT mudar.

#### Scenario: Push Pending no Recent
- **WHEN** o administrador abre os detalhes de `jobs_backup` em Recent e os últimos resultados são Up com `ping=40` e Pending
- **THEN** a linha termina em 40 ms e o último resultado aparece como uma coluna amarela da altura toda

#### Scenario: Queda nos agregados
- **WHEN** o período é 24h e um minuto teve só resultados Down
- **THEN** o minuto aparece como uma coluna vermelha, sem valor na linha

#### Scenario: Pending seguido de Down na mesma hora
- **WHEN** o período é 1w e uma hora teve um Pending e depois só resultados Down, sem Up
- **THEN** a hora aparece como uma coluna vermelha

#### Scenario: Minuto misto
- **WHEN** o período é 3h e um minuto teve um resultado Up e um Down
- **THEN** o minuto aparece como uma coluna amarela e a linha tem a média do resultado Up

#### Scenario: Escolha lembrada
- **WHEN** o administrador escolhe 6h e recarrega a página
- **THEN** o gráfico abre em 6h

#### Scenario: Valor guardado inválido
- **WHEN** o navegador tem `30d` guardado como período do gráfico
- **THEN** o gráfico abre em Recent

#### Scenario: Push sem ping
- **WHEN** um endpoint Push só recebeu envios sem `ping`, alguns Down
- **THEN** o cartão aparece com as colunas vermelhas e sem linha

#### Scenario: Página pública
- **WHEN** um visitante abre `/status/jobs/endpoints/jobs_backup` e escolhe 1w
- **THEN** o gráfico busca `/api/v1/status-pages/jobs/endpoints/jobs_backup/response-time-chart?period=1w`, sem credenciais, sem chamar `/api/v1/config` e sem 401
