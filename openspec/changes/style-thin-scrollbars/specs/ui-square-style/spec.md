## ADDED Requirements

### Requirement: Barra de rolagem fina e quadrada
A barra de rolagem da interface web MUST ser fina e sem cantos arredondados, com trilho transparente e sem os botões de seta, valendo no dashboard, nas páginas de detalhes, na administração, nas páginas públicas e na tela de login, tanto na rolagem da página quanto nas áreas com rolagem própria, na vertical e na horizontal.

**Espessura:** nos navegadores baseados em Chromium e no WebKit, a barra MUST ocupar exatamente 10 px. No Firefox, que não permite definir a espessura, ela MUST usar a barra fina do navegador.

**Estilo por navegador:** o estilo MUST usar as regras `::-webkit-scrollbar` para Chromium e WebKit e as propriedades padrão apenas onde essas regras não existem, porque definir as duas formas ao mesmo tempo faz o Chromium ignorar as regras específicas.

**Telas de toque:** em dispositivos de ponteiro grosso, como celulares e tablets sem trackpad, a barra do sistema MUST ser mantida, inclusive com o alto contraste do sistema ligado.

#### Scenario: Espessura da barra vertical
- **WHEN** a lista de endpoints da administração tem mais itens que a altura do painel, no Chrome, num dispositivo de ponteiro fino
- **THEN** a diferença entre a largura visível e a largura interna do painel é de 9 a 10 px

#### Scenario: Espessura da barra horizontal
- **WHEN** a tabela de checks tem mais colunas que a largura disponível, no Chrome
- **THEN** a diferença entre a altura visível e a altura interna da área é de 9 a 10 px

#### Scenario: Barra quadrada
- **WHEN** a interface é exibida num navegador que aceita as regras `::-webkit-scrollbar`
- **THEN** o polegar da barra é desenhado sem cantos arredondados e sem botões de seta
