## Why

A aba **Backup** da administração (`/admin/backup`, entregue na `v5.36.0-fork.18`) não segue o padrão das outras abas:
- **Rolagem:** a página rola inteira, com a prévia e os resultados do restore empilhados abaixo dos formulários. As listas da administração ocupam a altura da janela e só rolam dentro do painel.
- **Mensagens:** sucessos e erros aparecem como faixas dentro da página, longe do botão que as causou, e empurram o conteúdo.
- **Prévia e resultados:** ficam no fluxo da página, e para vê-los é preciso rolar. Na administração, as confirmações usam diálogo (`ConfirmDialog`).

O dono pediu para refinar a página de backup: sem rolagem da página, mensagens em toast e diálogos seguindo o padrão do layout.

Ele também pediu o **modo escuro por padrão**. O servidor já marca a página como escura (`ui.dark-mode`, padrão `true`), mas o script do `index.html`, as telas públicas e o `Settings` trocam para o tema claro quando o sistema operacional do visitante prefere claro e não há escolha salva.

## What Changes

- **Aba Backup sem rolagem da página** em telas médias e grandes:
  - usa o layout das listas (`AdminListLayout`, `meta.adminList`), com o cabeçalho e as abas compactas;
  - Download e Restore ficam em dois cartões lado a lado, cada um com cabeçalho, corpo com rolagem própria e rodapé com os botões;
  - no celular, a página rola normalmente, como nas listas.
- **Prévia e resultados do restore em diálogos:**
  - um componente de diálogo da administração (`AdminDialog`), com o visual do `ConfirmDialog`: título, descrição, botão de fechar, corpo com rolagem e rodapé;
  - renderizado no `body` na ordem de abertura, com uma pilha: Esc e o botão de fechar agem só no diálogo do topo e nunca durante uma aplicação em andamento;
  - o foco vai para o diálogo ao abrir, fica preso nele (inclusive depois de um clique fora) e volta a um alvo estável ao fechar; a rolagem do fundo fica bloqueada;
  - a **prévia** mostra avisos, filtro e tabela e tem Cancel e Restore no rodapé; fechá-la descarta a prévia;
  - Restore abre a confirmação (`ConfirmDialog`) por cima e, depois de aplicar, a prévia dá lugar ao **diálogo de resultados**;
  - o `ConfirmDialog` passa a usar o `AdminDialog`, mantendo a API e os `data-testid`, para todos os diálogos terem o mesmo comportamento.
- **Mensagens em toast:**
  - componente de toasts da administração (`AdminToasts`) com uma fila reativa (`utils/toast.js`): sucesso, informação, aviso e erro, com título opcional, botão de dispensar, tempo por tipo e no máximo 4 ao mesmo tempo;
  - centralizado, no topo em telas médias e grandes e logo abaixo do cabeçalho no celular, onde não há botões de ação: os rodapés dos cartões e dos diálogos, o `Settings` do dashboard, os contadores das listas e os botões do cabeçalho ficam nas bordas;
  - duas regiões ARIA fixas (status e alerta), pausa com hover/foco e enquanto houver diálogo aberto, e limpeza ao trocar de rota;
  - na aba Backup, viram toasts o download concluído, os erros de download, de leitura do arquivo, da prévia e do restore, o erro das quantidades e o resumo do restore;
  - avisos que fazem parte do formulário continuam na página: texto claro, validação da senha e avisos de monitoramento da prévia;
  - erros que explicam o estado da tela também ficam persistentes: arquivo recusado (no status do arquivo) e senha errada (junto do campo, com `aria-invalid`);
  - erros com instrução (409, 413, 422 e 429) ficam até serem dispensados.
- **Modo escuro por padrão:**
  - sem escolha salva no cookie `theme` (ou com um valor inválido), a interface usa o tema definido por `ui.dark-mode` (escuro por padrão), entregue pelo servidor no HTML, sem seguir a preferência do sistema operacional;
  - o `meta theme-color` acompanha o tema;
  - a escolha feita pelo botão de tema continua salva e respeitada;
  - vale para o dashboard, a administração, as páginas públicas e a tela de login;
  - `ui.dark-mode: false` mantém o claro como padrão.
- **Testes:** unitários da fila de toasts e do tema, testes Go da renderização do SPA, os E2E passam a trocar o tema por cookie (em vez de `set media`), e o E2E `test/e2e/admin-backup.sh` atualizado fica mais rigoroso: sem rolagem nem corte em 1280×900, 1280×720 e 1024×600, diálogos, Esc, prisão de foco, toasts que não cobrem os botões em várias larguras e prévia descartada quando algo muda.

## Capabilities

### New Capabilities

- `ui-theme`: tema inicial da interface pelo servidor e escolha do visitante.

### Modified Capabilities

- `admin-backup-restore`: a aba Backup ocupa a janela sem rolagem da página, com a prévia e os resultados em diálogos e as mensagens em toasts.
- `admin-web-ui`: toasts e diálogo padrão da administração, com o `ConfirmDialog` usando o mesmo diálogo.

## Impact

- **Frontend:**
  - novos `components/admin/AdminDialog.vue`, `components/admin/AdminToasts.vue` e `utils/toast.js` (com testes);
  - `ConfirmDialog.vue` sobre o `AdminDialog`;
  - `AdminListLayout.vue` com a opção de conteúdo sem painel;
  - `views/admin/AdminBackup.vue` reescrita no layout novo;
  - rota `/admin/backup` com `meta.adminList`;
  - `App.vue` com os toasts;
  - tema: `public/index.html`, `utils/theme.js` e `components/Settings.vue` usam o tema entregue pelo servidor quando não há cookie válido;
  - build em `web/static`.
- **Documentação:** `docs/README.md` (descrição de `ui.dark-mode`).
- **Backend:** só a renderização do SPA (`ui.ViewData.DefaultTheme`, `themeFromRequest` único em `api/spa.go` e `api/spa_render.go`), com testes.
- **Outras telas:** nenhuma mudança visual além do `ConfirmDialog`, que ganha foco, prisão de foco e o cabeçalho com botão de fechar. Os avisos em faixa das outras abas ficam como estão.
- **E2E:** `login.sh`, `push.sh`, `status-pages.sh`, `certificate.sh` e `admin-backup.sh` trocam o tema por cookie; `admin-backup.sh` e `admin.sh` ganham passos novos; os `data-testid` do `ConfirmDialog` continuam.
