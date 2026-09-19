## MODIFIED Requirements

### Requirement: Interface sem cantos arredondados
Todos os elementos retangulares da interface web MUST ser exibidos com raio de borda zero, em todos os temas (claro, escuro e Bio), no dashboard, nas páginas de detalhes e nas telas de administração. Isso inclui cards, botões, campos de texto, selects, badges, contadores, banners de anúncio, modais, tooltips (inclusive o tooltip do gráfico de tempo de resposta), a barra e os botões de configurações, popovers, barras de progresso, tabelas e contêineres de gráficos.

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
