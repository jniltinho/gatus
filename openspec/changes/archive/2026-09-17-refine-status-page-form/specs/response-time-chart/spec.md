## MODIFIED Requirements

### Requirement: Gráfico no formato do Uptime Kuma
O cartão **Response Time Trend** dos detalhes do endpoint, no dashboard e na página pública, MUST mostrar o gráfico no formato da página do monitor do Uptime Kuma, com duas exceções registradas: a legenda das séries abaixo do gráfico e o traço das linhas de mínimo e máximo.

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
- linhas de mínimo e máximo translúcidas e com traço próprio (mínimo tracejado, máximo pontilhado), para não se confundirem com a média nem entre si;
- colunas com as cores do Kuma (`rgba(220,53,69,0.41)` e `rgba(245,182,23,0.41)`);
- eixo do tempo com `HH:mm` e `MM-dd HH:mm`;
- eixo Y com o título "Resp. Time (ms)";
- tooltip só da linha da média, com data e hora e o valor em `ms`;
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


## ADDED Requirements

### Requirement: Legenda das séries do Response Time Trend
O cartão **Response Time Trend**, no dashboard e na página pública, MUST mostrar abaixo do gráfico uma legenda com um item por série do período escolhido:

- em Recent: **Response time**;
- em 3h, 6h, 24h e 1w: **Average**, **Minimum** e **Maximum**, nessa ordem;
- **Down** quando o período tiver pelo menos uma coluna vermelha, e **Pending** quando tiver pelo menos uma amarela, sempre depois dos itens das linhas.

Cada item MUST ter uma amostra de cor à esquerda e o nome da série ao lado, e o significado MUST estar no texto: a amostra MUST ser escondida dos leitores de tela. A amostra de uma linha MUST ser um traço na cor cheia da série, com o mesmo estilo de traço da linha no gráfico (média sólida, mínimo tracejado, máximo pontilhado). A amostra de uma coluna MUST ser um quadrado preenchido com a cor cheia e com borda de contraste suficiente para ter forma nos dois temas.

A legenda MUST ficar fora da área de altura fixa do gráfico, sem alterar essa altura, MUST quebrar em várias linhas em telas estreitas e MUST NOT aparecer enquanto o gráfico estiver carregando ou em erro. Clicar num item MUST NOT ligar nem desligar a série: a legenda do Chart.js continua desligada. O contêiner da legenda MUST expor `data-testid="response-time-chart-legend"` e cada item o identificador da série em `data-series` (`response-time`, `average`, `minimum`, `maximum`, `down` e `pending`), e MUST NOT ser descendente do elemento `data-testid="response-time-chart"`, que continua com os atributos `data-*` lidos pelos testes.

#### Scenario: Legenda dos agregados
- **WHEN** o administrador abre os detalhes de `core_api` e escolhe 24h
- **THEN** a legenda mostra Average, Minimum e Maximum, nessa ordem, cada um com o quadrado da cor da linha

#### Scenario: Legenda do Recent
- **WHEN** o período é Recent e não houve queda nem Pending
- **THEN** a legenda mostra só Response time

#### Scenario: Traço das linhas dos agregados
- **WHEN** o período é 1w
- **THEN** a linha da média é sólida, a do mínimo tracejada e a do máximo pontilhada
- **AND** a amostra de cada item da legenda tem o mesmo traço da sua linha

#### Scenario: Legenda com queda
- **WHEN** o período é 6h, um agregado teve só resultados Down (coluna vermelha) e outro teve Pending sem Down (coluna amarela)
- **THEN** a legenda mostra Average, Minimum, Maximum, Down e Pending

#### Scenario: Legenda de um período sem linha
- **WHEN** o período é 24h e nenhum agregado tem média, mas há colunas de queda
- **THEN** a legenda continua mostrando Average, Minimum, Maximum e Down

#### Scenario: Legenda durante a carga
- **WHEN** o cartão ainda está buscando os dados do período
- **THEN** a área do gráfico mostra o spinner e a legenda não aparece

#### Scenario: Legenda na página pública
- **WHEN** um visitante abre `/status/jobs/endpoints/jobs_backup` e escolhe 1w
- **THEN** a legenda aparece com Average, Minimum e Maximum
