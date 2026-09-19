## 1. Base: escala de cinza por variáveis, sem mudar nada

- [ ] 1.1 Capturas de referência dos temas claro e escuro em todas as telas (`docs/screenshots/capture.sh` estendido para os dois temas), antes de qualquer mudança.
- [ ] 1.2 `colors.gray` e `colors.white` por variáveis no `tailwind.config.js`, com os valores atuais em `:root` e `:root.dark`; `make frontend-build`; capturas de novo e comparação pixel a pixel com as de 1.1. Diferença → rever D3 antes de seguir.

## 2. Três temas de ponta a ponta

- [ ] 2.1 `ui.default-theme` em `internal/config/ui` (validação, precedência sobre `dark-mode`, aviso), com testes, e `gatus config validate` recusando valor inválido.
- [ ] 2.2 `themeFromRequest`, `defaultTheme` e `ViewData` com os três identificadores; `theme-color` por tema; conferência dos cabeçalhos de cache do HTML; testes do servidor com a tabela de casos compartilhada.
- [ ] 2.3 Script inline de `index.html` e `utils/theme.js` (`themeFromCookie`, `applyTheme`, escolha de tema no lugar de `toggleTheme`), com testes de unidade.
- [ ] 2.4 `ResponseTimeChart.vue` e `SuiteCard.vue` lendo as cores de variáveis do tema no lugar de `classList.contains('dark')` com cores fixas, redesenhando ao trocar de tema; busca por outros casos (`classList.contains('dark')`, cores em `rgba`/hexadecimal nos componentes); barra de rolagem, toasts e diálogos conferidos no tema novo.

## 3. O tema

- [ ] 3.1 Variáveis de `:root.theme-bionexo` (as 48 HSL e a escala de cinza), com os papéis da D2; capturas do cabeçalho claro e do marinho para o dono escolher.
- [ ] 3.2 Componente de seletor de tema (menu acessível) em `Settings.vue`, `PublicLayout.vue` e `LoginPage.vue`.
- [ ] 3.3 Script de contraste sobre as superfícies reais do tema: texto ≥ 4,5:1, componentes e barra de rolagem ≥ 3:1, cores de estado incluídas.

## 4. Verificação

- [ ] 4.1 E2E: cookie `theme=bionexo` → HTML cru já com a classe; escolha pelo seletor lembrada; `ui.default-theme: bionexo` sem cookie; cookie inválido; teclado no seletor; todas as telas no tema novo, também a 390 px.
- [ ] 4.2 `go test ./... -race`, `make lint`, testes de unidade do frontend, contrato HTTP e as demais suítes E2E.
- [ ] 4.3 Depois de `collapse-status-page-groups` arquivada: atualizar em `status-page-web-ui` as frases "nas duas versões do tema" e "nos temas claro e escuro" (D7).

## 5. Entrega

- [ ] 5.1 `docs/README.md` (tabela `ui`), `config.yaml` de exemplo, screenshots do tema, `AGENTS.fork.md`.
- [ ] 5.2 PR com CI verde; release com notas em inglês, `test/e2e/upgrade.sh` antes da tag, imagem, `mariadb/`, exemplos; arquivar a change.
