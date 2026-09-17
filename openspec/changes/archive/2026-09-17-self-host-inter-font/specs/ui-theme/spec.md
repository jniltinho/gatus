## ADDED Requirements

### Requirement: Tipografia entregue pelo próprio serviço
A interface MUST usar a fonte Inter entregue pelo próprio Gatus, em arquivos `woff2` variáveis servidos em `/fonts/`, versionados no repositório e embutidos no binário junto do restante dos estáticos. O frontend MUST NOT pedir fontes a nenhum domínio externo, nas telas autenticadas e nas páginas públicas.

Os `@font-face` MUST declarar `font-display: swap` e um `unicode-range` por subconjunto, e a pilha de fontes MUST manter as fontes do sistema como reserva, de modo que a interface continue legível se o arquivo não carregar. A licença da fonte MUST ser distribuída junto dos arquivos.

Os números da interface MUST usar algarismos tabulares, inclusive dentro de campos de formulário, para que um valor que muda sozinho não mude a largura do que está em volta e para que colunas de números fiquem alinhadas. O gráfico de tempo de resposta desenha texto em `canvas` e fica de fora dos algarismos tabulares, mas MUST usar a mesma família de fontes do restante da interface. Os badges SVG gerados no backend ficam de fora das duas regras.

#### Scenario: Fonte aplicada sem sair do servidor
- **WHEN** um visitante abre uma página pública sem credenciais
- **THEN** a fonte Inter é carregada de `/fonts/`
- **AND** nenhuma requisição vai para `fonts.googleapis.com` ou `fonts.gstatic.com`

#### Scenario: Números com a mesma largura
- **WHEN** a interface mostra dois valores de mesmo comprimento e dígitos diferentes, como `111` e `999`
- **THEN** os dois ocupam exatamente a mesma largura

#### Scenario: Fonte indisponível
- **WHEN** o arquivo da fonte não carrega
- **THEN** a interface continua legível com a fonte do sistema, sem erro na tela

#### Scenario: Fonte trocada pelo administrador
- **WHEN** o administrador define `body { font-family: … }` em `ui.custom-css`
- **THEN** a interface passa a usar a família escolhida, sem precisar de `!important`

#### Scenario: Algarismos tabulares desligados
- **WHEN** o administrador define `body { font-variant-numeric: normal !important }` em `ui.custom-css`
- **THEN** a interface volta aos algarismos proporcionais

#### Scenario: Arquivo da fonte entregue pelo binário
- **WHEN** um cliente pede o arquivo da fonte ao Gatus
- **THEN** a resposta é `200` com `Content-Type: font/woff2`
