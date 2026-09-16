## 1. Componentes

- [x] 1.0 `utils/dialogStack.js`: pilha por id com ids incrementais e o `computed` `dialogOpen`, com testes.
- [x] 1.1 `utils/toast.js`:
  - fila com durações de 5/8/10 s e `duration: 0`;
  - limite de 4 com timers removidos;
  - `dismissToast`, `clearToasts` e `pauseToasts`/`resumeToasts`;
  - testes em `utils/toast.test.mjs` (dispensa com timer pendente, limite, pausa e retomada).
- [x] 1.2 `components/admin/AdminToasts.vue`:
  - centralizado, `md:top-3` e `top-14` no celular, `z-[70]`, `pointer-events` só nos toasts;
  - pilha visual sem `role` em ordem cronológica, e duas regiões `sr-only` fixas (`status` polite e `alert` assertive, `aria-atomic`) com o texto da última mensagem de cada tipo;
  - pausa com hover/`focus-within` e com `dialogOpen`;
  - variantes `dark:`;
  - montagem no `App.vue` só com o app autenticado visível, e `clearToasts` no `router.afterEach`.
- [x] 1.3 `components/admin/AdminDialog.vue`:
  - `Teleport` para o `body`, pilha por id com `z-index` pela posição;
  - `role`/`aria` no painel;
  - `initialFocus`, `returnFocus` (reserva no `h1` com `tabindex="-1"`) e `describedby`;
  - foco no novo topo quando um diálogo fecha sobre outro;
  - listener único no `document` (Esc, Tab, `focusin`) registrado só com pilha não vazia, agindo no topo e respeitando `busy`;
  - sem `inert`;
  - `overflow-hidden` no `body`;
  - limpeza no `onBeforeUnmount`.
- [x] 1.4 `ConfirmDialog.vue` sobre o `AdminDialog` (mesma API e `data-testid`, mensagem ligada por `describedby`, foco inicial em Cancel, Esc e X como Cancel).
- [x] 1.5 `AdminListLayout.vue` com a prop `panel` (conteúdo `md:flex md:min-h-0 md:flex-1 md:flex-col`) e descrição com `title`; rota `/admin/backup` com `meta.adminList`.

## 2. Aba Backup

- [x] 2.1 `AdminBackup.vue` no layout sem painel:
  - grid `md:min-h-0 md:flex-1 md:grid-cols-2 md:grid-rows-1`;
  - cartões com corpo `overflow-auto overscroll-contain` e rodapés fixos.
- [x] 2.2 Diálogos:
  - prévia (avisos, filtro, tabela com cabeçalho fixo, Cancel e Restore) descartada ao fechar, com `returnFocus` no Preview;
  - confirmação por cima e prévia `busy` durante a aplicação;
  - diálogo de resultados.
- [x] 2.3 Mensagens:
  - toasts de download, quantidades, arquivo, prévia e restore, com 409/413/422/429 persistentes;
  - status persistente do arquivo recusado;
  - senha errada com `aria-invalid` e `aria-describedby` até editar;
  - resposta de prévia descartada quando arquivo, senha ou opções mudam durante o pedido.
- [x] 2.4 Lint, `npm run test:unit` e `make frontend-build`.

## 3. Tema escuro por padrão

- [x] 3.1 Servidor:
  - `ui.ViewData.DefaultTheme`;
  - `themeFromRequest` único para `SinglePageApplication` e `renderSPA` (cookie `dark`/`light`, senão o padrão);
  - `index.html` com `data-default-theme` e `theme-color` pelo tema;
  - testes em `api/spa_test.go` e no teste do SPA das status pages (com e sem `ui.dark-mode`, cookie `dark`, `light` e inválido).
- [x] 3.2 Navegador:
  - `utils/theme.js` com `defaultThemeIsDark` (literal de desenvolvimento = escuro) e `wantsDarkMode` sem `prefers-color-scheme`;
  - `Settings.vue` usando `utils/theme.js`;
  - script inline do `index.html` com a mesma regra na classe e no `theme-color`;
  - `theme-color` atualizado no toggle;
  - testes em `utils/theme.test.mjs`.
- [x] 3.3 `docs/README.md`: descrição de `ui.dark-mode`.

## 4. Testes e entrega

- [x] 4.1 Helper de tema por cookie nos E2E que usam `set media` (`login.sh`, `push.sh`, `status-pages.sh`, `certificate.sh`, `admin-backup.sh`), e passo do tema do `login.sh` reescrito:
  - escuro sem cookie com sistema claro;
  - toggle para claro;
  - claro mantido depois de recarregar.
- [x] 4.2 `test/e2e/admin-backup.sh`:
  - sem rolagem nem corte (`scrollHeight`/`scrollWidth` do `documentElement` com tolerância de 1 px e `bottom` dos botões) em 1280×900, 1280×720 e 1024×600, com cifragem, arquivo cifrado e prévia aberta;
  - foco dentro da prévia depois de cancelar a confirmação;
  - `role="dialog"`, Cancel, Esc e Tab preso;
  - toasts de download, restore e senha errada (com `aria-invalid`);
  - `toast-dismiss`;
  - cliques em Download, Preview, no rodapé do diálogo e (no celular) no menu do cabeçalho com toast visível em 1280×900, 800×600 e 390×844;
  - prévia descartada ao marcar Overwrite com o pedido segurado por `agent-browser network route`;
  - arquivo que não é backup com status persistente;
  - prints claro e escuro.
- [x] 4.3 `test/e2e/admin.sh`: Esc na confirmação de remoção (endpoint continua pela API) e foco inicial em Cancel.
- [x] 4.4 Rodar todos os E2E alterados, `go test ./api/... -race`, `make lint` e `openspec validate refine-admin-backup-layout --strict`.
- [x] 4.5 Entrega:
  - PR no `jniltinho/gatus` com CI verde e merge;
  - release `v5.36.0-fork.19` com imagem no Docker Hub;
  - pacote `mariadb`;
  - arquivamento da change.
