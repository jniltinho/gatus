# ui-theme Specification

## Purpose
TBD - created by archiving change refine-admin-backup-layout. Update Purpose after archive.
## Requirements
### Requirement: Tema escuro por padrão
Sem uma escolha de tema válida salva pelo visitante, a interface web MUST usar o tema definido por `ui.dark-mode`, que é escuro por padrão, sem seguir a preferência de tema do sistema operacional. Isso vale para o dashboard, as páginas de detalhes, a administração, as páginas públicas e a tela de login.

**Entrega pelo servidor:**
- o HTML entregue MUST já ter o tema inicial, para a página não trocar de tema ao carregar;
- MUST informar o tema padrão configurado;
- MUST ter a cor de tema do navegador (`theme-color`) coerente com o tema.

**Escolha do visitante:**
- a escolha feita pelo botão de tema MUST continuar salva e MUST prevalecer sobre `ui.dark-mode` nas visitas seguintes;
- um valor salvo inválido MUST ser ignorado;
- com `ui.dark-mode: false` e sem escolha salva, a interface MUST usar o tema claro.

#### Scenario: Primeira visita com o sistema em modo claro
- **WHEN** um visitante sem escolha de tema salva, com o sistema operacional em modo claro, abre o dashboard ou a tela de login com a configuração padrão
- **THEN** o HTML entregue já tem o tema escuro e a interface continua escura depois de carregar

#### Scenario: Tema claro escolhido
- **WHEN** o visitante escolhe o tema claro e recarrega a página
- **THEN** a interface continua no tema claro

#### Scenario: Escolha salva inválida
- **WHEN** o cookie de tema do visitante tem um valor diferente de `dark` e `light`
- **THEN** a interface usa o tema de `ui.dark-mode`

#### Scenario: Claro por configuração
- **WHEN** a configuração tem `ui.dark-mode: false` e o visitante não tem escolha salva
- **THEN** a interface é exibida no tema claro

### Requirement: Cores da barra de rolagem pelo tema
As cores da barra de rolagem MUST vir das variáveis do tema, com um tom de repouso e, onde o navegador permite estilizar o realce da barra, um tom mais forte ao passar o mouse. As cores MUST mudar junto com o tema claro e escuro, inclusive quando o visitante troca pelo botão de tema.

O tom de repouso MUST ter pelo menos 3:1 de contraste com a superfície de menor contraste de cada tema (os fundos claros das áreas com rolagem no tema claro e o fundo das tabelas no escuro). Com `prefers-contrast: more`, o repouso e o realce MUST ficar mais fortes, com o realce acima do repouso. Com `forced-colors: active`, a barra MUST voltar às cores do sistema.

#### Scenario: Troca de tema
- **WHEN** o visitante troca do tema escuro para o claro numa página com rolagem
- **THEN** a variável de cor da barra passa a ter o valor do tema claro

#### Scenario: Alto contraste do sistema
- **WHEN** o sistema operacional está no modo de alto contraste
- **THEN** a barra usa as cores do sistema, e não as do tema

#### Scenario: Contraste do polegar
- **WHEN** as cores da barra são medidas sobre o fundo da página no tema claro e sobre o fundo das tabelas no tema escuro
- **THEN** o contraste do tom de repouso é de pelo menos 3:1 nos dois casos

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

