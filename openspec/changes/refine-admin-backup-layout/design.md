## Context

- **Aba Backup atual** (`web/app/src/views/admin/AdminBackup.vue`): página de formulário (`meta.admin`, sem `adminList`) com as seções Download e Restore empilhadas, e a prévia e os resultados abaixo delas.
  - **Mensagens:** faixas `role="alert"`/`role="status"`.
  - **Confirmação:** o `ConfirmDialog`.
- **Listas** (`AdminListLayout`, `meta.adminList`):
  - em telas médias e grandes, o `App.vue` fixa a janela (`md:h-screen md:overflow-hidden`), o painel ocupa o resto com rolagem própria, e o rodapé do painel mostra os contadores à esquerda;
  - no celular, a página rola.
- **`ConfirmDialog`:**
  - sobreposição `fixed inset-0 z-50` com `role="dialog"` e Esc local (só com foco dentro), sem foco inicial;
  - usado nas listas de endpoints, status pages e push keys e na aba Backup;
  - os roteiros `admin.sh`, `push.sh`, `status-pages.sh` e `admin-backup.sh` usam `confirm-dialog`, `confirm-accept` e `confirm-cancel`.
- **Toasts:** não existem.
- **Fixos na tela:**
  - `Settings` do dashboard em `fixed bottom-4 left-4 z-50`;
  - cabeçalho do app no topo (título à esquerda, Admin/Logout ou menu à direita).
- **Tema:**
  - `api/spa.go` (`SinglePageApplication`, usado no dashboard, na administração e no `/login`) e `api/spa_render.go` (`themeFromRequest`, usado nas status pages) renderizam `<html class="{{ .Theme }}">`: com o cookie `theme` segue o cookie (qualquer valor diferente de `dark` vira claro); sem ele, segue `ui.dark-mode` (padrão `true`);
  - no navegador, o script inline do `index.html`, `utils/theme.js#wantsDarkMode` e `Settings.vue#wantsDarkMode` trocam para claro sem cookie quando `prefers-color-scheme` é claro;
  - o `meta theme-color` é fixo em `#f7f9fb`;
  - os E2E trocam o tema com `agent-browser set media light|dark` (15 usos em `login.sh`, `push.sh`, `status-pages.sh`, `certificate.sh`, `admin.sh` e `admin-backup.sh`), e o `login.sh` exige que o toggle vá para o escuro.
- **Protótipo:** a branch local `feat/admin-backup-layout-refine` tem uma versão parcial (layout, diálogo e toasts, sem tema), que serve de base.

## Goals / Non-Goals

**Goals:**
- Aba Backup sem rolagem da página em telas médias e grandes, sem cortar conteúdo e com os botões sempre visíveis.
- Diálogo padrão da administração, acessível, usado também pelo `ConfirmDialog`.
- Toasts acessíveis que não cobrem ações e não substituem erros que precisam ficar na tela.
- Tema escuro por padrão, decidido pelo servidor, sem depender do sistema operacional.

**Non-Goals:**
- Migrar as faixas das outras abas para toasts.
- Mudar a API ou as regras do backup e do restore.
- Redesenhar o layout de celular.

## Decisions

### D1. Layout da aba Backup
- **Rota e layout:** `/admin/backup` ganha `meta.adminList`, e a página usa `AdminListLayout` com `active="backup"` e a prop nova `panel=false`.
- **Conteúdo sem painel:** fica num contêiner `mt-3 md:flex md:min-h-0 md:flex-1 md:flex-col`.
- **Grid:** `grid gap-4 md:min-h-0 md:flex-1 md:grid-cols-2 md:grid-rows-1`, com linha `minmax(0, 1fr)` e sem `h-full`, para a altura não depender de percentuais.
- **Cartões:** `flex min-h-0 flex-col border bg-card`, com cabeçalho `shrink-0`, corpo `min-h-0 flex-1 overflow-auto overscroll-contain` e rodapé `shrink-0` com os botões à direita. Com janela baixa, só o corpo rola, e os rodapés continuam visíveis.
- **Celular:** abaixo de `md`, os cartões ficam empilhados e a página rola.
- **Descrição:** a descrição do layout ganha `title` com o texto completo, porque é truncada.

### D2. `AdminDialog`
`components/admin/AdminDialog.vue`:
- **Props:**
  - `open`, `title`, `description`, `size` (`md` = `max-w-md`, `lg` = `max-w-3xl`, `xl` = `max-w-5xl`), `busy` e `testid`;
  - `initialFocus`: função que devolve o elemento a focar ao abrir, com o painel como reserva;
  - `returnFocus`: função que devolve o elemento a focar ao fechar, com reserva no elemento ativo guardado ao abrir, se estiver no documento, visível e habilitado, senão no `h1` da página, que recebe `tabindex="-1"` antes do `focus()` (o `h1` não é focável e o gatilho de uma remoção some do DOM);
  - `describedby`: id de um elemento do corpo que descreve o diálogo, usado no lugar da descrição do cabeçalho (o `ConfirmDialog` liga a mensagem do corpo).
- **Evento:** `close`.
- **Renderização:**
  - `<Teleport to="body">`, montado na ordem de abertura;
  - sobreposição `fixed inset-0 bg-black/50 dark:bg-black/70 p-4`, com `z-index` `50 + posição na pilha` (estilo inline), para o visual e a pilha coincidirem;
  - o painel tem `role="dialog"`, `aria-modal="true"`, `aria-labelledby` (título), `aria-describedby` (a prop `describedby` ou a descrição do cabeçalho, quando existe), `tabindex="-1"` e `max-h-[calc(100dvh-2rem)]`;
  - os ids de título e descrição são incrementais, não aleatórios;
  - cabeçalho com botão de fechar (`aria-label="Close"`), corpo `min-h-0 flex-1 overflow-auto overscroll-contain` e rodapé `justify-end`.
- **Pilha:** o módulo (`utils/dialogStack.js`) mantém uma pilha de ids de diálogos abertos e exporta o `computed` reativo `dialogOpen`, usado pelos toasts para pausar.
  - Abrir empilha; fechar remove pelo id, não com "pop", porque uma prévia fecha e os resultados abrem no mesmo tick.
  - `onBeforeUnmount` remove o id (ex.: botão Voltar do navegador com o diálogo aberto).
- **Teclado e foco:** um único listener no `document`, em captura, registrado quando a pilha deixa de estar vazia e removido quando esvazia (não fica listener sobrando depois de HMR ou de desmontagem), age só no diálogo do topo:
  - Esc emite `close`, exceto com `busy`;
  - Tab e Shift+Tab circulam pelos focáveis do painel do topo, e com `busy`, sem focáveis, o foco fica no painel;
  - `focusin` fora do painel do topo devolve o foco a ele, inclusive depois de um clique na sobreposição.
- **Fechar:** clicar na sobreposição não fecha.
- **Foco ao fechar:**
  - quando a pilha esvazia, o foco é devolvido via `returnFocus`, ignorando alvos desabilitados ou fora do documento;
  - quando um diálogo fecha e **outro continua no topo** (ex.: Cancel da confirmação sobre a prévia), o foco vai para o `initialFocus` do novo topo ou, com `busy`, para o próprio painel, porque o botão focado foi desmontado e o `focusin` não dispara ao cair no `body`.
- **Rolagem do fundo:** enquanto a pilha não está vazia, o `body` recebe `overflow-hidden` (importante no celular), removido quando a pilha esvazia.
- **Sem `inert`:** o conteúdo por trás não recebe `inert`. A prisão de foco já cobre o teclado, e `inert` no app bloquearia os toasts.

### D3. `ConfirmDialog` sobre o `AdminDialog`
- **API:** mantém `open`, `title`, `message`, `confirmLabel`, `confirm` e `cancel`, e os `data-testid` `confirm-dialog`, `confirm-cancel` e `confirm-accept`.
- **Implementação:**
  - renderiza `AdminDialog` `size="md"`, com a mensagem no corpo e o id dela passado em `describedby`, sem repetir a mensagem no cabeçalho;
  - o foco inicial vai para **Cancel**, como o APG recomenda para ações destrutivas;
  - Esc e o X emitem `cancel`.
- **Nas outras telas:** o `ConfirmDialog` ganha cabeçalho com X, foco inicial, prisão de foco e bloqueio da rolagem do fundo.

### D4. Toasts
- **Fila** (`utils/toast.js`, com `reactive` do Vue):
  - `showToast(type, message, { title, duration })`, os atalhos `toast.success/info/warning/error`, `dismissToast(id)`, `clearToasts()`, `pauseToasts()` e `resumeToasts()`;
  - tipos `success`, `info`, `warning` e `error`, com desconhecido virando `info`;
  - duração padrão de 5 s (sucesso, informação), 8 s (aviso) e 10 s (erro), e `duration: 0` mantém até dispensar;
  - no máximo 4 ao mesmo tempo, removendo os mais antigos e os timers deles;
  - a pausa congela o tempo restante de todos e a retomada recomeça do restante.
- **Posição:** `fixed z-[70]`, **centralizado**, largura `w-[calc(100%-2rem)] sm:w-[28rem]`:
  - em telas médias e grandes, no topo (`md:top-3`), sobre o centro do cabeçalho do app;
  - abaixo de `md`, logo abaixo do cabeçalho (`top-14`), para não cobrir o menu, Admin ou Logout, que ficam no cabeçalho do celular.
  - No topo central não há botões de ação: os rodapés dos cartões e dos diálogos, o `Settings` do dashboard, os contadores das listas e o menu do cabeçalho ficam nas bordas.
  - Em telas médias e grandes, o centro do cabeçalho do app não tem controles: o título fica à esquerda e Admin/Logout à direita, fora dos 28rem centrais a partir de 768 px.
  - No celular, pode cobrir o título de um diálogo alto, mas nunca o rodapé com as ações.
  - A região tem `pointer-events-none` e os toasts `pointer-events-auto`.
- **Acessibilidade:**
  - a pilha visual dos toasts não tem `role` e mostra todos em ordem cronológica;
  - duas regiões **visualmente ocultas** (`sr-only`), sempre montadas antes das mensagens, com `aria-atomic="true"`: uma `role="status"` (`aria-live="polite"`), que recebe o texto do último sucesso, informação ou aviso, e uma `role="alert"` (`aria-live="assertive"`), que recebe o texto do último erro;
  - hover ou foco dentro de um toast pausa todos (WCAG 2.2.1);
  - enquanto `dialogOpen` for verdadeiro, os timers ficam pausados, porque o botão de dispensar fica fora da prisão de foco, e voltam a contar quando a pilha esvazia.
- **Visual:** quadrado, borda, barra colorida à esquerda por tipo, ícone lucide, título opcional, mensagem, botão de dispensar e variantes `dark:`. Testes E2E por `data-testid="toast"`, `data-type` e `toast-message`.
- **Montagem:** `AdminToasts` fica no `App.vue` quando o app autenticado está visível (`routerReady && !isPublic && !isLogin && (!config.oidc || config.authenticated)`). `router.afterEach` chama `clearToasts()` quando a rota muda, para uma mensagem de uma aba não aparecer em outra.

### D5. Fluxo da aba Backup
- **Download:** o sucesso vira toast ("Backup downloaded as …"), e os erros viram toast de erro (413/422 com `duration: 0`). Os avisos de texto claro e de senha continuam no cartão.
- **Quantidades:** a falha vira toast de erro, e as caixas mostram "—".
- **Arquivo:** arquivo grande demais, JSON inválido, formato desconhecido ou falha de leitura geram toast de erro **e** um estado persistente no `restore-file-status` (texto em vermelho, `role="alert"` só na mudança), que continua explicando por que Preview está desabilitado.
- **Senha errada:** a prévia com resposta "Invalid password or corrupted file." mostra o toast e marca o campo com `aria-invalid="true"` e `aria-describedby` apontando para a mensagem persistente junto do campo (`restore-password-error`). Editar a senha limpa a marca.
- **Preview:**
  - abre o diálogo **Restore preview** (`size="xl"`), com o resumo na descrição, os avisos de monitoramento, o filtro por ação, a tabela com cabeçalho fixo, Cancel (`restore-plan-close`) e Restore (`restore-apply`);
  - `returnFocus` aponta para o botão Preview;
  - erros viram toast de erro (409, 413, 422 e 429 com `duration: 0`), sem abrir o diálogo.
- **Prévia em andamento:** trocar ou reler o arquivo, a senha ou as opções durante a prévia descarta a resposta (geração), e nenhum diálogo abre. Fechar a prévia (Cancel, X, Esc) descarta o plano.
- **Restore:**
  - abre o `ConfirmDialog` por cima;
  - durante a aplicação, a prévia fica `busy`;
  - no sucesso, a prévia fecha e o diálogo **Restore results** abre com o resumo na descrição, a tabela e Close (`restore-results-close`), com `returnFocus` no botão Preview;
  - o toast é "Restore finished", ou aviso "Restore finished with failures";
  - no erro, toast de erro; com 409, o plano também é descartado.

### D6. Tema escuro por padrão
- **Servidor:**
  - `ui.ViewData` ganha `DefaultTheme` (`dark` quando `ui.dark-mode` é verdadeiro, senão vazio);
  - uma função única `themeFromRequest` passa a servir `SinglePageApplication` e `renderSPA`: cookie `theme` com `dark` ou `light` vale; ausente ou inválido usa `DefaultTheme`;
  - o `index.html` renderiza `<html class="{{ .Theme }}" data-default-theme="{{ .DefaultTheme }}">` e `<meta name="theme-color" content="{{ if eq .Theme "dark" }}#030712{{ else }}#f7f9fb{{ end }}">`.
- **Navegador:**
  - `utils/theme.js` concentra a regra em `defaultThemeIsDark()`, que lê `data-default-theme` e trata `{{ .DefaultTheme }}` literal (servidor de desenvolvimento do Vue) como escuro, e em `wantsDarkMode()` (cookie `dark`/`light`, senão o padrão), sem `prefers-color-scheme`;
  - o `Settings.vue` passa a usar `utils/theme.js`;
  - o script inline do `index.html` aplica a mesma regra antes da primeira pintura, na classe **e** no `meta theme-color`, porque o servidor de desenvolvimento do Vue não executa o template Go;
  - ao trocar o tema, o `meta theme-color` é atualizado;
  - no `npm run serve`, o padrão é sempre escuro: `ui.dark-mode: false` só aparece no HTML servido pelo Go.
- **Configuração:** `ui.dark-mode: false` dá o claro por padrão. A descrição de `ui.dark-mode` no `docs/README.md` deixa de citar o sistema operacional.

### D7. E2E com o tema por cookie
- **Helper:** um helper nos roteiros que hoje chamam `set media light|dark` (`login.sh`, `push.sh`, `status-pages.sh`, `certificate.sh` e `admin-backup.sh`) substitui essas chamadas: grava o cookie `theme=light|dark` pelo `eval` (`document.cookie`) e recarrega. O `admin.sh` troca o tema por classe e não muda.
- **Prints:** os nomes `-light`/`-dark` continuam corretos.
- **`login.sh`:** o passo do tema é reescrito:
  - sem cookie e com `set media light`, a tela de login e o dashboard ficam escuros (classe `dark` e `data-default-theme="dark"` no HTML entregue);
  - o toggle vai para o claro, e recarregar mantém o claro.
- **Teste Go:** `api/spa_test.go` e o teste do SPA das status pages verificam `data-default-theme`, a classe e o `theme-color` com e sem `ui.dark-mode`, com cookie `dark`, `light` e inválido.

### D8. Testes do layout, dos diálogos e dos toasts
- **Unitários:**
  - `utils/toast.test.mjs`: durações, tipo desconhecido, `duration: 0`, limite (com os timers removidos), dispensa com timer pendente, pausa e retomada, `clearToasts`;
  - `utils/theme.test.mjs`, com `document` falso: padrão pelo atributo, cookie válido e inválido, literal de desenvolvimento.
- **E2E `admin-backup.sh`:**
  - **Janelas:** 1280×900, 1280×720 e 1024×600, sem cifragem, com cifragem e com arquivo cifrado com a dica de senha visível, e com a prévia aberta;
  - **Sem rolagem nem corte:** `document.documentElement.scrollHeight <= innerHeight + 1` e `scrollWidth <= innerWidth + 1` (a mesma tolerância de subpixel das listas), e `getBoundingClientRect().bottom <= innerHeight` de `backup-download`, `restore-preview` e dos botões do rodapé do diálogo;
  - **Diálogos:** `role="dialog"` no painel; Cancel e Esc fecham a prévia; Tab repetido mantém `document.activeElement` dentro do painel;
  - **Toasts:** de download, de restore e de senha errada, com `aria-invalid` no campo de senha; clique em `toast-dismiss`;
  - **Toasts sobre ações:** com um toast visível, clicar em Download, Preview, nos botões do rodapé do diálogo e, em 390×844, no botão de menu do cabeçalho, em 1280×900, 800×600 e 390×844;
  - **Foco aninhado:** com a prévia aberta, abrir a confirmação, apertar Cancel e verificar que `document.activeElement` está dentro da prévia;
  - **Prévia em andamento:** segurar `POST /api/v1/admin/restore/preview` com `agent-browser network route`, clicar em Preview, marcar Overwrite, liberar a rota e verificar que a prévia não abre;
  - **Arquivo recusado:** um arquivo que não é backup deixa a mensagem persistente no status do arquivo.
- **E2E `admin.sh`:** na confirmação de remoção de um endpoint, Esc fecha e a API mostra que o endpoint continua; o foco inicial está em Cancel.

## Risks / Trade-offs

- **O `ConfirmDialog` muda em todas as listas.** Ganho de consistência e acessibilidade; API e `data-testid` se mantêm, e os E2E existentes validam.
- **Toast no topo central** cobre o centro do cabeçalho do app por alguns segundos. Não há controles ali, e os toasts pausam com hover/foco e podem ser dispensados.
- **Visitante com o sistema em modo claro** passa a ver o escuro na primeira visita. É o pedido do dono; o botão de tema continua.
- **Mudança dos E2E para cookie:** necessária, porque `prefers-color-scheme` deixa de valer.
- **Primeiro teste em `utils/` que importa `vue`** (`toast.js`). O Vue funciona no `node --test`.

## Migration Plan

Frontend e a renderização do SPA, sem migração de dados. Voltar à versão anterior restaura o layout antigo e o tema pela preferência do sistema.

## Open Questions

Nenhuma.
