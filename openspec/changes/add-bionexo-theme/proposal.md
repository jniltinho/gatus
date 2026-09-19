## Why

A interface tem dois temas, claro e escuro. O dono quer um terceiro, com a identidade visual da Bionexo — a mesma paleta que o logo já usa desde a `v6.0.2` —, para que o painel e as páginas de status publicadas tenham a cara da empresa sem depender de um `custom.css` mantido à parte.

## What Changes

1. **Tema `bionexo`**, de base clara, com a paleta medida no site da empresa: marinho `#1d3c55` no texto e nas superfícies de destaque, verde-água `#34cdd7`/`#2cb7c0` nos realces, azul `#285783` nos elementos interativos. As cores de estado (verde, vermelho, amarelo) não mudam: são semânticas, não de marca.
2. **Três temas em todo lugar onde hoje há dois**: o cookie `theme` aceita `bionexo`, o HTML entregue pelo servidor já vem com o tema certo (sem piscar), e `theme-color` acompanha.
3. **Tema padrão configurável com três valores:** `ui.default-theme: dark | light | bionexo`. `ui.dark-mode` continua aceito, como hoje; quando os dois estão presentes, `default-theme` vale.
4. **Seletor de tema no lugar do botão de alternar**, no dashboard (`Settings`), nas páginas públicas (`PublicLayout`) e na tela de login: com três opções, um botão que alterna deixa de dizer para onde vai.
5. **Escala de cinza por variáveis CSS.** 578 classes da interface usam cinzas fixos do Tailwind (`bg-white`, `text-gray-900`, ...), fora do alcance de um tema. A escala `gray` e o `white` passam a ser definidos por variáveis no `tailwind.config.js`, com os valores de hoje nos temas claro e escuro — que não podem mudar um pixel — e valores próprios no `bionexo`.

Sem mudança para quem não pedir: sem `ui.default-theme` e sem o cookie novo, tudo fica como está.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `ui-theme`: três temas, tema padrão por `ui.default-theme`, seletor de tema, e as regras de contraste valendo para o tema novo.
- `basic-login-page`: a tela de login lê e oferece os três temas.

## Impact

- **Configuração:** `ui.default-theme` (novo, opcional). `ui.dark-mode` segue valendo.
- **Servidor:** `internal/config/ui` (campo e validação), `internal/api/spa_render.go` (`themeFromRequest`, `defaultTheme`, `ViewData`), `web/app/public/index.html` (script inline e `theme-color`).
- **Frontend:** `tailwind.config.js` (escala de cinza por variáveis), `src/index.css` (variáveis dos três temas, barra de rolagem), `src/utils/theme.js`, um componente de seletor de tema usado por `Settings.vue`, `PublicLayout.vue` e `LoginPage.vue`, e os dois componentes que hoje escolhem cores fixas olhando a classe `dark` (`ResponseTimeChart.vue` e `SuiteCard.vue`).
- **Risco principal:** o remapeamento da escala de cinza toca a aparência de toda a interface. A garantia é um teste de regressão visual: capturas dos temas claro e escuro antes e depois, idênticas pixel a pixel.
- **Nome do tema num projeto público:** ver Open Questions do design.
- **Documentação e testes:** `docs/README.md` (tabela `ui`), screenshots do tema novo, testes de unidade de `theme.js`, testes do servidor, E2E em todas as telas.
