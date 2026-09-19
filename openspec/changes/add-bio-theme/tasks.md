## 1. Base: escala de cinza por variáveis, sem mudar nada

- [ ] 1.1 Roteiro de cores computadas (D4): instância local com dados semeados, todas as telas, temas claro e escuro, JSON por tela; rodar no `master` e guardar a referência.
- [ ] 1.2 `gray` por variáveis em `theme.extend.colors` (50 a 900, canais RGB, `<alpha-value>`), mesmos valores em `:root` e `:root.dark`; `make frontend-build`; rodar 1.1 de novo: nenhuma diferença, incluindo utilidades com opacidade, `hover:`, `dark:` e `@apply`. Diferença → rever D3 antes de seguir.
- [ ] 1.3 Os 2 `bg-white` e as 7 classes `gray-950` mortas de `LoginPage.vue`, só com 1.1 igual.

## 2. Três temas de ponta a ponta

- [ ] 2.1 `ui.default-theme` em `internal/config/ui`: validação, presença registrada antes dos padrões, precedência sobre `dark-mode` com aviso; testes, inclusive `gatus config validate` e recarregamento da configuração.
- [ ] 2.2 `ViewData` com identificador, classe e `ThemeColor`; `themeFromRequest` e `defaultTheme` com três valores; template de `index.html` sem decisão própria.
- [ ] 2.3 `Cache-Control: no-cache` e `Vary: Cookie` no HTML dos dois manipuladores, GET e HEAD, preservando `private, no-store` das páginas com login; contrato HTTP atualizado.
- [ ] 2.4 `theme.cases.json` e os três testes que o consomem: Go, `theme.test.mjs` e o do script inline extraído de `index.html`; `theme.js` com escolha de tema no lugar de `toggleTheme`, classes mutuamente exclusivas e `theme-color` por tema.
- [ ] 2.5 `ResponseTimeChart.vue` por identificador de tema e variáveis, redesenhando entre quaisquer dois temas; regra do tema `bio` em `SuiteCard.vue`; busca por outros seletores `.dark` em CSS e cores fixas em componentes.

## 3. O tema

- [ ] 3.1 Variáveis de `:root.theme-bio` (as do tema e a escala de cinza) com os papéis da D2; capturas do cabeçalho claro e do marinho para o dono escolher.
- [ ] 3.2 Blocos de `:root.theme-bio` para a barra de rolagem e para `prefers-contrast: more`; `forced-colors` conferido.
- [ ] 3.3 Componente de seletor de tema em `Settings.vue`, `PublicLayout.vue` e `LoginPage.vue`, com teclado e foco.
- [ ] 3.4 `custom.css` representativo (com `!important` e com sobrescrita de variáveis) nos três temas.
- [ ] 3.5 Script de contraste sobre as telas no tema `bio`: texto do tema ≥ 4,5:1, bordas de controle e foco ≥ 3:1, e nenhum par de cor de estado abaixo do mesmo par no tema claro.

## 4. Verificação

- [ ] 4.1 Migração dos helpers de tema das suítes E2E (`status-pages.sh`, `status-page-groups.sh`, `push.sh`, `admin-backup.sh`, `certificate.sh`, `login.sh`) e de `docs/screenshots/capture.sh` para três temas com classes exclusivas e `theme-color`; `login.sh` usando o seletor; casos `dark` e `light` preservados.
- [ ] 4.2 E2E do tema: `theme=bio` → HTML cru já com a classe; escolha pelo seletor lembrada; `ui.default-theme: bio` sem cookie; cookie inválido com cada padrão; seletor pelo teclado; transições entre todos os pares; todas as telas no tema `bio`, a 390 px, e a página pública a 360 px com logo, título longo e o menu aberto, sem rolagem horizontal; seletor nas configurações do dashboard com ponteiro grosso.
- [ ] 4.3 `make frontend-build` final, `go test ./... -race`, `make lint`, testes de unidade do frontend, contrato HTTP, todas as suítes E2E e `openspec validate add-bio-theme --strict`.

## 5. Entrega

- [ ] 5.1 `docs/README.md` (tabela `ui`), `config.yaml` de exemplo, screenshots do tema, `AGENTS.fork.md`.
- [ ] 5.2 PR com CI verde; release com notas em inglês, `test/e2e/upgrade.sh` antes da tag, imagem, `mariadb/`, exemplos; arquivar a change.
