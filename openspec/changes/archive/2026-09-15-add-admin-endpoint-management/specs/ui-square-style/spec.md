## ADDED Requirements

### Requirement: Interface sem cantos arredondados
Todos os elementos retangulares da interface web MUST ser exibidos com raio de borda zero, nos temas claro e escuro, no dashboard, nas páginas de detalhes e nas telas de administração. Isso inclui cards, botões, campos de texto, selects, badges, contadores, banners de anúncio, modais, tooltips (inclusive o tooltip do gráfico de tempo de resposta), a barra e os botões de configurações, popovers, barras de progresso, tabelas e contêineres de gráficos.

#### Scenario: Card de endpoint
- **WHEN** o dashboard exibe um card de endpoint
- **THEN** o card, seus botões e seus badges têm `border-radius` computado igual a `0px`

#### Scenario: Barra de configurações e contador
- **WHEN** o dashboard exibe a barra de configurações, seus botões e o contador de endpoints com falha
- **THEN** todos têm `border-radius` computado igual a `0px`

#### Scenario: Barra de progresso de suite
- **WHEN** a página de uma suite exibe a barra de progresso do fluxo
- **THEN** a barra e seu preenchimento têm extremidades retas

#### Scenario: Tooltip do gráfico
- **WHEN** o usuário passa o mouse sobre o gráfico de tempo de resposta
- **THEN** o tooltip é desenhado com cantos retos

### Requirement: Indicadores circulares preservados
Indicadores circulares pequenos (pontos de status, marcadores de etapa do fluxo de suites, marcadores da linha do tempo de anúncios e o indicador de carregamento) MUST continuar circulares.

#### Scenario: Ponto de status
- **WHEN** o dashboard exibe o ponto de status de um endpoint
- **THEN** o ponto continua circular

### Requirement: Raio de borda centralizado no tema
O raio de borda MUST ser definido no tema do Tailwind de forma que `rounded`, `rounded-sm`, `rounded-md`, `rounded-lg` e as variantes maiores resultem em zero, mantendo valor não zero apenas em `rounded-full`, reservado a indicadores circulares.

#### Scenario: Componente novo com classe de arredondamento
- **WHEN** um componente novo usa a classe `rounded-lg`
- **THEN** o componente é exibido com cantos retos sem nenhuma alteração adicional

### Requirement: Badges SVG sem cantos arredondados
Os badges SVG servidos pela API (saúde, uptime e tempo de resposta) MUST ser gerados sem cantos arredondados. O endpoint `badge.shields`, que devolve dados para o shields.io renderizar, MUST permanecer inalterado.

#### Scenario: Badge de saúde
- **WHEN** um cliente requisita `GET /api/v1/endpoints/core_api/health/badge.svg`
- **THEN** nenhum elemento `rect` do SVG tem atributo `rx` maior que zero
