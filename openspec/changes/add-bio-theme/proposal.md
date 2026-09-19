## Why

A interface tem dois temas, claro e escuro. O dono quer um terceiro, com a paleta que o logo já usa desde a `v6.0.2` (verde-água, azul e marinho), para que o painel e as páginas de status publicadas tenham identidade própria sem depender de um `custom.css` mantido à parte. O nome do tema é **`bio`**.

## What Changes

1. **Tema `bio`**, de base clara: marinho `#1d3c55` no texto, azul `#285783` nos elementos interativos, verde-água `#34cdd7`/`#2cb7c0` nos realces. As cores de estado (verde, vermelho, amarelo) e as demais cores fixas não mudam.
2. **Três temas em todo lugar onde hoje há dois**: o cookie `theme` aceita `bio`; o HTML entregue pelo servidor já vem com a classe certa; `theme-color` acompanha; as classes de tema são mutuamente exclusivas.
3. **`ui.default-theme: dark | light | bio`**. `ui.dark-mode` continua valendo quando `default-theme` não está definido; com os dois, `default-theme` vale e o log avisa.
4. **Seletor de tema** no lugar do botão de alternar, nas configurações do dashboard, no cabeçalho das páginas públicas e na tela de login.
5. **Escala `gray` do Tailwind por variáveis CSS**, com os valores de hoje nos temas claro e escuro. É a única forma de as classes de cinza fixo responderem a um terceiro tema sem reescrever centenas de classes. `white` e `black` **não** são remapeados.
6. **Cabeçalhos do HTML:** `Cache-Control: no-cache` e `Vary: Cookie` em todo HTML da interface, porque o que é entregue depende do cookie de tema.

Sem mudança para quem não pedir: sem `ui.default-theme` e sem o cookie novo, a interface fica como está — e isso é verificado (ver Impact).

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `ui-theme`: três temas, tema padrão, seletor, cores do tema `bio`, barra de rolagem, e a garantia de que os temas existentes não mudam.
- `basic-login-page`: a tela de login lê e oferece os três temas.
- `admin-web-ui`, `response-time-chart`, `ui-square-style`, `status-page-web-ui`: requisitos que falavam em "temas claro e escuro" ou "dois temas" passam a cobrir o terceiro.

## Impact

- **Configuração:** `ui.default-theme` (novo, opcional).
- **Servidor:** `internal/config/ui` (campo, validação e a precedência decidida antes de os padrões serem aplicados), `internal/api/spa_render.go` e `spa.go` (tema, `theme-color` e cabeçalhos), `web/app/public/index.html`.
- **Frontend:** `tailwind.config.js`, `src/index.css`, `src/utils/theme.js`, um componente de seletor, `ResponseTimeChart.vue` e a regra `.dark .suite-header` de `SuiteCard.vue`.
- **Testes existentes:** o helper `set_theme` de seis suítes E2E e de `docs/screenshots/capture.sh` só liga e desliga a classe `dark`, e `login.sh` clica num botão de alternar: todos migram.
- **Risco principal:** o remapeamento da escala de cinza toca toda a interface. A porta de entrada é uma comparação das **cores computadas** de todos os elementos de todas as telas, nos temas claro e escuro, antes e depois — determinística, ao contrário de comparar capturas.
- **Contrato HTTP:** `Vary: Cookie` e `Cache-Control: no-cache` no HTML do dashboard são cabeçalhos novos; o contrato gravado é atualizado.
