## MODIFIED Requirements

### Requirement: Tema escuro por padrão
A interface web MUST oferecer três temas: escuro (`dark`), claro (`light`) e Bionexo (`bionexo`), este de base clara com a paleta da marca. Sem uma escolha de tema válida salva pelo visitante, ela MUST usar o tema de `ui.default-theme` (`dark`, `light` ou `bionexo`) e, na falta dele, o definido por `ui.dark-mode`, que é escuro por padrão, sem seguir a preferência de tema do sistema operacional. Isso vale para o dashboard, as páginas de detalhes, a administração, as páginas públicas e a tela de login.

**Entrega pelo servidor:**
- o HTML entregue MUST já ter o tema inicial, para a página não trocar de tema ao carregar;
- MUST informar o tema padrão configurado;
- MUST ter a cor de tema do navegador (`theme-color`) coerente com o tema.

**Escolha do visitante:**
- a escolha feita pelo seletor de tema MUST continuar salva e MUST prevalecer sobre o tema padrão configurado nas visitas seguintes;
- um valor salvo inválido MUST ser ignorado;
- com `ui.dark-mode: false` e sem escolha salva, a interface MUST usar o tema claro;
- com `ui.default-theme` e `ui.dark-mode` presentes, `ui.default-theme` MUST valer; um valor de `ui.default-theme` fora dos três MUST invalidar a configuração.

Todo requisito que fala dos temas, em qualquer capacidade, MUST valer para todos os temas oferecidos.

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

#### Scenario: Tema Bionexo escolhido
- **WHEN** o visitante escolhe o tema Bionexo e recarrega a página
- **THEN** o HTML entregue pelo servidor já tem a classe do tema Bionexo, sem a classe do tema escuro
- **AND** a cor de tema do navegador é a do tema Bionexo

#### Scenario: Tema padrão Bionexo
- **WHEN** a configuração tem `ui.default-theme: bionexo` e o visitante não tem escolha salva
- **THEN** a interface é exibida no tema Bionexo, no dashboard, na administração, nas páginas públicas e na tela de login

#### Scenario: default-theme e dark-mode juntos
- **WHEN** a configuração tem `ui.default-theme: light` e `ui.dark-mode: true`
- **THEN** a interface sem escolha salva é exibida no tema claro

#### Scenario: Tema padrão inválido
- **WHEN** a configuração tem `ui.default-theme: azul`
- **THEN** a configuração é inválida, no início e em `gatus config validate`

#### Scenario: Temas existentes inalterados
- **WHEN** a mesma tela é capturada nos temas claro e escuro antes e depois desta mudança
- **THEN** as capturas são idênticas pixel a pixel

### Requirement: Cores da barra de rolagem pelo tema
As cores da barra de rolagem MUST vir das variáveis do tema, com um tom de repouso e, onde o navegador permite estilizar o realce da barra, um tom mais forte ao passar o mouse. As cores MUST mudar junto com o tema claro e escuro, inclusive quando o visitante troca pelo botão de tema.

O tom de repouso MUST ter pelo menos 3:1 de contraste com a superfície de menor contraste de cada tema (os fundos claros das áreas com rolagem nos temas claro e Bionexo e o fundo das tabelas no escuro). Com `prefers-contrast: more`, o repouso e o realce MUST ficar mais fortes, com o realce acima do repouso. Com `forced-colors: active`, a barra MUST voltar às cores do sistema.

#### Scenario: Troca de tema
- **WHEN** o visitante troca do tema escuro para o claro numa página com rolagem
- **THEN** a variável de cor da barra passa a ter o valor do tema claro

#### Scenario: Alto contraste do sistema
- **WHEN** o sistema operacional está no modo de alto contraste
- **THEN** a barra usa as cores do sistema, e não as do tema

#### Scenario: Contraste do polegar
- **WHEN** as cores da barra são medidas sobre o fundo da página no tema claro e sobre o fundo das tabelas no tema escuro
- **THEN** o contraste do tom de repouso é de pelo menos 3:1 nos dois casos

## ADDED Requirements

### Requirement: Seletor de tema
A troca de tema MUST ser feita por um seletor que mostra o tema em uso e oferece os três temas, no lugar de um botão que alterna: no dashboard, nas páginas públicas e na tela de login. O seletor MUST ser um botão que abre um menu com uma opção por tema, operável por teclado (abrir com Enter ou Espaço, setas entre as opções, Esc fecha e devolve o foco ao botão), com a opção em uso marcada para leitores de tela. Escolher um tema MUST aplicá-lo sem recarregar a página e MUST salvar a escolha.

#### Scenario: Escolher pelo teclado
- **WHEN** o visitante foca o seletor, abre o menu com Enter, vai até "Bionexo" com as setas e confirma
- **THEN** a interface passa ao tema Bionexo sem recarregar, o menu fecha e o foco volta ao seletor

#### Scenario: Tema em uso
- **WHEN** o visitante abre o seletor no tema escuro
- **THEN** a opção do tema escuro aparece marcada como a escolhida

### Requirement: Contraste do tema Bionexo
No tema Bionexo, o texto MUST ter pelo menos 4,5:1 de contraste com a superfície em que aparece, e os componentes de interface e seus estados de foco pelo menos 3:1. O verde-água da marca MUST NOT ser usado como cor de texto nem como única indicação de foco sobre superfícies claras. As cores de estado (no ar, fora, pendente, sem dados) MUST ser as mesmas dos outros temas e MUST manter esses mínimos sobre as superfícies do tema.

#### Scenario: Medição
- **WHEN** o contraste é medido por script sobre as superfícies do tema Bionexo
- **THEN** todo par de texto e superfície tem pelo menos 4,5:1, e todo componente, anel de foco e cor de estado pelo menos 3:1

### Requirement: Componentes sem cores próprias por tema
Nenhum componente MUST escolher cores fixas conferindo se o tema é escuro: o gráfico de tempo de resposta e os cartões de suíte MUST ler suas cores das variáveis do tema e MUST ser redesenhados quando o tema muda.

#### Scenario: Gráfico no tema Bionexo
- **WHEN** o visitante troca para o tema Bionexo numa página com o gráfico de tempo de resposta
- **THEN** a grade, o texto e as marcas do gráfico passam às cores do tema Bionexo, sem recarregar
