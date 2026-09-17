## Why

A barra de rolagem da interface é a do navegador, sem estilo: larga no Chrome e no Edge do Windows e do Linux, cinza-claro fixa, igual nos temas claro e escuro. Ela aparece na página inteira, nos painéis das listas da administração, nos corpos dos diálogos, nos cartões da aba Backup e nas tabelas com rolagem horizontal, e destoa do layout quadrado e do tema escuro padrão.

O dono pediu uma barra de rolagem **bem fina**, **igual nos navegadores** e **com o tema do layout**.

## What Changes

- **Barra fina e no tema, em toda a interface:**
  - 10 px, sem cantos arredondados, com trilho transparente e polegar nas cores do tema, mais forte ao passar o mouse;
  - as cores saem de variáveis novas do tema, no padrão do projeto, então claro e escuro trocam junto com o resto da interface;
  - o polegar tem pelo menos 3:1 de contraste sobre os fundos de cada tema (`muted-foreground`: 4,55:1 no claro e 5,73:1 no escuro).
- **Mesmo resultado nos navegadores de desktop:**
  - Chrome, Edge e Safari usam as regras `::-webkit-scrollbar`, que dão largura, forma e hover;
  - o Firefox, que não tem essas regras, usa as propriedades padrão (`scrollbar-width: thin` e `scrollbar-color`), isoladas por `@supports`, porque defini-las junto faria o Chrome ignorar as regras acima;
  - em telas de toque (celular e tablet sem trackpad), a barra do sistema continua como é hoje (`@media (pointer: fine)`).
- **Alto contraste:** com o alto contraste do sistema (`forced-colors`) a barra volta às cores do sistema, e com `prefers-contrast: more` o polegar fica mais forte.
- **Onde vale:** dashboard, detalhes do endpoint, administração (listas, formulários, diálogos, aba Backup e modal de passos das suites), páginas públicas e tela de login, tanto na página quanto nas dez áreas com rolagem própria, na vertical e na horizontal.
- **Testes:** E2E no Chrome (com as barras visíveis e rolagem forçada) medindo a espessura real, de 9 a 10 px, na vertical e na horizontal, e a troca das cores por tema, com prints; conferência manual no Firefox.
- **Documentação:** `docs/README.md` (quem usa `ui.custom-css` para estilizar a barra passa a precisar de `!important`) e `AGENTS.fork.md`.

## Capabilities

### New Capabilities

(nenhuma)

### Modified Capabilities

- `ui-square-style`: a barra de rolagem entra no estilo da interface, fina e quadrada, com espessura verificável.
- `ui-theme`: as cores da barra vêm das variáveis do tema e trocam junto com ele.

## Impact

- **Frontend:** `web/app/src/index.css` (variáveis e regras) e o build versionado em `web/static`.
- **Backend:** nenhum.
- **macOS, iPad com trackpad e notebooks com tela de toque:** a barra sobreposta do sistema passa a ser clássica nas áreas com rolagem, ocupando espaço. É o preço de ter a mesma barra nos navegadores de desktop, e fica documentado.
- **Risco:** baixo. É só CSS, sem mudança de estrutura nem de comportamento, e a rolagem continua funcionando onde o navegador não aceita os estilos.
