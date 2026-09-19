## ADDED Requirements

### Requirement: Grupos recolhíveis na página pública

A página pública MUST permitir recolher e expandir cada grupo de endpoints por um botão no cabeçalho do grupo, acionável por mouse e por teclado, com `aria-expanded` e `aria-controls`. Um grupo recolhido MUST NOT renderizar as linhas dos seus endpoints. Recolhido ou não, o cabeçalho MUST mostrar o nome do grupo, o estado agregado e a contagem de endpoints do grupo por estado. A seção de destaques MUST NOT ser recolhível.

#### Scenario: Recolher um grupo

- **WHEN** o visitante aciona o cabeçalho do grupo `sites`, que está expandido
- **THEN** as linhas dos endpoints de `sites` deixam de existir no documento, o botão passa a `aria-expanded="false"` e o cabeçalho continua mostrando o estado e a contagem do grupo

#### Scenario: Teclado

- **WHEN** o visitante foca o cabeçalho de um grupo e pressiona Enter ou Espaço
- **THEN** o grupo alterna entre recolhido e expandido

#### Scenario: Destaques

- **WHEN** a página tem endpoints em destaque
- **THEN** a seção `Featured` não tem botão de recolher

### Requirement: Estado inicial dos grupos definido pela página

A definição de uma página MUST aceitar o campo opcional `groups-collapsed` (booleano, padrão `false`), no arquivo de configuração e nas páginas gerenciadas, e o payload público MUST devolvê-lo como `groupsCollapsed`. Com `false`, todos os grupos MUST começar expandidos. Com `true`, os grupos cujo estado agregado é operacional MUST começar recolhidos, e os demais MUST começar expandidos.

#### Scenario: Padrão

- **WHEN** uma página não define `groups-collapsed`
- **THEN** o payload tem `groupsCollapsed: false` e todos os grupos começam expandidos

#### Scenario: Grupos recolhidos por padrão, com um problema

- **WHEN** uma página tem `groups-collapsed: true`, o grupo `sites` está operacional e o grupo `apis` tem um endpoint fora
- **THEN** `sites` começa recolhido e `apis` começa expandido

#### Scenario: Valor inválido

- **WHEN** a definição de uma página tem `groups-collapsed: "sim"`
- **THEN** a validação recusa a página, como recusa os demais campos de tipo errado

### Requirement: Escolha do visitante lembrada no navegador

A escolha de recolher ou expandir um grupo MUST ser lembrada no navegador do visitante, por página (slug) e por grupo, e MUST prevalecer sobre o estado inicial definido pela página nas visitas seguintes. Um grupo que o visitante recolheu e cujo estado agregado deixa de ser operacional MUST ser expandido, na carga da página e nas atualizações em tempo real, sem apagar a escolha: quando o grupo volta a operacional, volta a recolhido. Sem armazenamento disponível no navegador, a página MUST funcionar, só sem lembrar.

#### Scenario: Lembrar entre visitas

- **WHEN** o visitante recolhe o grupo `sites` de uma página com `groups-collapsed: false` e recarrega a página
- **THEN** `sites` começa recolhido

#### Scenario: Problema num grupo recolhido pelo visitante

- **WHEN** o visitante recolheu `sites`, e um endpoint de `sites` passa a falhar enquanto a página está aberta
- **THEN** `sites` é expandido, e volta a ficar recolhido quando o grupo volta a operacional

#### Scenario: Páginas diferentes

- **WHEN** o visitante recolhe `sites` na página `services`
- **THEN** o grupo `sites` da página `internal` não é afetado

### Requirement: Contagem por grupo no payload público

Cada grupo do payload público MUST trazer `summary` com `total`, `up`, `down` e `pending`, contando os endpoints publicados daquele grupo, no mesmo formato e com as mesmas regras do `summary` da página. Os endpoints em destaque MUST NOT entrar na contagem de nenhum grupo, já que não são listados neles.

#### Scenario: Grupo com uma falha

- **WHEN** o grupo `apis` publica três endpoints, um deles fora
- **THEN** o grupo tem `summary: {"total": 3, "up": 2, "down": 1, "pending": 0}`

#### Scenario: Endpoint em destaque

- **WHEN** `sites_website` está em destaque e o grupo `sites` tem mais dois endpoints
- **THEN** o `summary` de `sites` tem `total: 2`

### Requirement: Limite de endpoints por página configurável

A seção `status-pages` MUST aceitar `maximum-endpoints-per-page`, inteiro de `1` a `1000`, com padrão `200`. Esse valor MUST ser o limite de chaves de endpoint que a definição de uma página pode selecionar uma a uma e o número máximo de endpoints que uma página mostra; acima dele, a página MUST devolver os primeiros na ordem de exibição, com `truncated: true` e o aviso no log citando o limite em vigor. Um valor fora do intervalo MUST invalidar a configuração na inicialização. Uma página gerenciada gravada quando o limite era maior MUST continuar válida e ser truncada na exibição, e não recusada.

#### Scenario: Padrão

- **WHEN** `maximum-endpoints-per-page` não é definido e uma página seleciona 250 endpoints por grupo
- **THEN** o payload traz 200 endpoints e `truncated: true`

#### Scenario: Limite maior

- **WHEN** `maximum-endpoints-per-page: 500` e uma página seleciona 250 endpoints por grupo
- **THEN** o payload traz os 250 endpoints e `truncated: false`

#### Scenario: Fora do intervalo

- **WHEN** `maximum-endpoints-per-page: 5000`
- **THEN** a configuração é inválida e o Gatus não inicia, com a mensagem citando o intervalo aceito

#### Scenario: Limite reduzido depois de gravar uma página

- **WHEN** uma página gerenciada seleciona 300 chaves, gravada com o limite em 500, e o limite passa a 200
- **THEN** a página continua publicada, mostra os 200 primeiros com `truncated: true`, e a administração avisa que ela excede o limite
