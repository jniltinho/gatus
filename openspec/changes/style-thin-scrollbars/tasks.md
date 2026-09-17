## 1. Estilo

- [x] 1.1 Variáveis no `index.css`, na convenção da tripla HSL: `--scrollbar-size` (10px) só em `:root`, e `--scrollbar-thumb` e `--scrollbar-thumb-hover` em `:root` e `:root.dark`, com o contraste conferido (≥ 3:1 sobre o branco no claro e sobre `dark:bg-gray-800` no escuro).
- [x] 1.2 Regras **fora de `@layer base`** (ao lado de `html { height: 100% }`, para o `custom.css` sem layer não vencer), no bloco de D2:
  - `@media not all and (pointer: coarse)` envolvendo tudo: `::-webkit-scrollbar`, `-track`, `-corner`, `-button` (`display: none` com `width/height: 0`), `-thumb` com hover, o `@supports not selector(::-webkit-scrollbar)` do Firefox e o `forced-colors` do WebKit;
  - `html { scrollbar-color: auto }` global em `forced-colors: active`;
  - `prefers-contrast: more` redefinindo repouso e realce nos dois temas.
- [x] 1.3 Lint, `npm run test:unit`, `make frontend-build` e o build de `web/static` no commit.

## 2. Testes e entrega

- [x] 2.1 E2E em `test/e2e/push.sh` (usa o helper `set_theme`):
  - sessão com `--hide-scrollbars false` (o agent-browser 0.37.1 esconde as barras por padrão);
  - `matchMedia('(pointer: coarse)').matches === false` antes de medir (o Chromium headless responde `pointer: none`);
  - reduzir a janela até haver transbordo no painel da lista e na tabela de checks, medir e restaurar 1280×900;
  - espessura de 9 a 10 px na vertical e na horizontal, sem faixa larga que esconda o caminho padrão do Chromium;
  - `--scrollbar-thumb` com `trim()` igual a `215.4 16.3% 46.9%` no claro e `215 20.2% 65.1%` no escuro;
  - prints da lista com rolagem nos dois temas.
- [x] 2.2 Conferir na tela, nos dois temas: dashboard, lista da administração, corpo de um diálogo, cartões da aba Backup, modal de passos das suites, `<pre>` do formulário de endpoint (rolagem horizontal em 40 px de altura), tabela de checks na página pública e janela estreita (conferido pelos prints do `push.sh` e do `admin-backup.sh`: painel da lista nos dois temas e tabela de checks em janela estreita; as demais áreas não transbordaram nos roteiros).
- [ ] 2.3 Conferência manual no Firefox (barra fina e na cor do tema, e `CSS.supports('selector(::-webkit-scrollbar)') === false`) registrada no PR. **Pendente:** não há Firefox neste ambiente WSL; precisa ser feita num desktop.
- [x] 2.4 Documentação: nota em `docs/README.md` sobre `ui.custom-css` e `!important`, e seção em `AGENTS.fork.md`.
- [x] 2.5 `make lint` e `openspec validate style-thin-scrollbars --strict`.
- [ ] 2.6 Entrega:
  - PR no `jniltinho/gatus` com CI verde e merge;
  - release `v5.36.0-fork.21` com imagem no Docker Hub;
  - pacote `mariadb`;
  - arquivamento da change.
