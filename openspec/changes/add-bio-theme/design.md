## Context

O tema é decidido em três lugares que precisam concordar:

- **Servidor** (`internal/api/spa_render.go`): `themeFromRequest` lê o cookie `theme` (`dark` → classe `dark`; `light` → classe vazia) e, sem cookie válido, usa `defaultTheme`, que vem de `ui.dark-mode` e devolve `dark` ou a string vazia — não existe um valor `light`. `ViewData.Theme` é a **classe**, não um identificador, e o template de `index.html` decide a `theme-color` com `eq .Theme "dark"`. `internal/config/ui` preenche `DarkMode` com o padrão, de modo que depois dos padrões não dá mais para saber se o campo veio do arquivo.
- **Script inline** de `web/app/public/index.html`, que aplica a mesma regra antes da primeira pintura.
- **`web/app/src/utils/theme.js`**, usado pelos botões de `Settings.vue`, `PublicLayout.vue` e `LoginPage.vue`. `THEME_COLORS` tem só `dark` e `light`.

Cabeçalhos do HTML hoje: o das páginas públicas sai com `Cache-Control: no-cache`; o do dashboard (`spa.go`) não define política de cache; nenhum dos dois tem `Vary: Cookie`. As páginas com login próprio saem com `private, no-store`.

Tailwind 3.1.8, `darkMode: 'class'`, cores do projeto em `theme.extend.colors`. Medido nos 47 componentes (`grep -rho` sobre `web/app/src/**/*.vue`): **720** ocorrências de `dark:`; **578** classes de cinza, branco ou preto fixos; **477** classes por variável do tema. Em `src/index.css`, o bloco `:root` declara 23 variáveis e `:root.dark` 21; os blocos de `prefers-contrast: more` e da barra de rolagem só existem para `:root` e `:root.dark`. Todas as três famílias respondem aos temas de hoje — o cinza fixo responde pelo seu par `dark:`. O problema é só o terceiro tema: **sem a classe `dark`, ele herda os valores claros das classes fixas**.

Outros fatos que pesam:

- `text-white` aparece 14 vezes, quase sempre sobre uma cor de estado (`bg-green-500 text-white` em `FlowStep.vue`); `bg-white`, 2 vezes. `gray-950` aparece 7 vezes em `LoginPage.vue` e **não existe** no Tailwind 3.1.8: essas classes não geram CSS hoje.
- `ResponseTimeChart.vue` guarda `isDark` (um booleano de `classList.contains('dark')`) e desenha com cores fixas em `rgba` e hexadecimal. `SuiteCard.vue` não faz isso: tem uma regra CSS `.dark .suite-header` com `rgba`.
- `manifest.json` tem `theme_color` fixo, e `index.html` tem `apple-mobile-web-app-status-bar-style`.
- Contraste (WCAG) das cores do tema: marinho sobre branco 11,5:1; azul `#285783` sobre branco 7,6:1; verde-água `#34cdd7` sobre branco **1,9:1** e `#2cb7c0` **2,4:1**; marinho sobre verde-água 5,9:1. Dívidas que já existem no tema claro: branco sobre `green-500` tem cerca de 2,3:1, e `yellow-400` sobre branco cerca de 1,5:1.

## Goals / Non-Goals

**Goals:** um terceiro tema em todas as telas; temas claro e escuro comprovadamente inalterados; nenhuma piscada de tema; texto do tema com contraste AA.

**Non-Goals:** temas definidos por configuração; um tema `bio` escuro; remapear `white`, `black`, `blue` ou as cores de estado; pagar as dívidas de contraste das cores de estado (o tema não pode piorá-las, e só); seguir `prefers-color-scheme`; mudar `manifest.json`, que é estático e continua com a cor do tema claro.

## Decisions

### D1 — Base clara, classe `theme-bio`, sem `dark`

O tema é a classe `theme-bio` no `<html>`, **sem** `dark`: as 720 variantes `dark:` ficam desligadas e o ponto de partida é o tema claro. As classes de tema são mutuamente exclusivas: aplicar um tema remove as outras. Um tema de marca escuro exigiria uma segunda dimensão (marca × claro/escuro) no cookie, no servidor e no seletor; fica fora.

### D2 — Papéis das cores, e o que o tema não promete

- **texto e títulos:** marinho; texto secundário num marinho dessaturado com pelo menos 4,5:1;
- **botão primário:** fundo verde-água com texto marinho (5,9:1), ou fundo marinho com texto branco (11,5:1) onde precisa pesar mais;
- **links, anel de foco e `--primary`/`--ring`:** azul `#285783`; o foco nunca é só verde-água;
- **verde-água:** fundos de realce e bordas decorativas, sempre com outra pista.

O que **não** muda: as utilidades `blue-*` fixas (anéis `ring-blue-200`, por exemplo) continuam no azul do Tailwind, próximo do da marca; as cores de estado continuam as do tema claro. O requisito de contraste cobre o que o tema define; para as cores de estado, a regra é não piorar: o contraste de cada par no tema `bio` tem de ser maior ou igual ao do mesmo par no tema claro — o que obriga as superfícies do tema a não serem mais escuras que as do claro onde há texto de estado.

### D3 — Só a escala `gray` passa por variáveis

Em `theme.extend.colors`, `gray` (50 a 900, a escala do 3.1.8) vira `rgb(var(--gray-N) / <alpha-value>)`, com as variáveis em canais RGB separados por espaço. `:root` e `:root.dark` recebem **os mesmos valores**, os da escala atual do Tailwind (o tema escuro já troca de cinza pelas variantes `dark:`, não pela escala); `:root.theme-bio` recebe uma escala tingida de marinho.

- `white` e `black` ficam literais: `text-white` sobre uma cor de estado tem de continuar branco. Os 2 usos de `bg-white` migram para o token de superfície só se as cores computadas não mudarem (D4).
- `gray-950` não é acrescentado: ativá-lo mudaria a tela de login. As 7 classes mortas de `LoginPage.vue` são removidas ou trocadas por `gray-900`, conforme o que a D4 mostrar que já é o resultado de hoje.
- As utilidades com opacidade (`gray-800/20`, `/40`, `/50`, `/60`), `hover:`, `dark:` e os `@apply` de `index.css` entram na verificação.
- **CSS próprio:** os blocos de `index.css` que hoje só existem para `:root` e `:root.dark` (barra de rolagem, `prefers-contrast: more`) ganham o de `:root.theme-bio`; `.bg-success` e semelhantes ficam como estão. `ui.custom-css` continua sendo carregado depois: a tarefa 3.4 testa um `custom.css` representativo, com `!important` e com sobrescrita de variáveis, nos três temas.

### D4 — A porta de entrada: cores computadas, não capturas

Comparar capturas pixel a pixel não é determinístico aqui: `capture.sh` consulta hosts externos, usa `RANDOM`, os textos "há X segundos" andam, a fonte carrega com `font-display: swap`. A verificação da D3 é outra: um roteiro abre cada tela com **dados fixos** (instância local com endpoints de push e resultados semeados, sem hosts externos), nos temas claro e escuro, e coleta de **todos os elementos** as cores computadas — `color`, `background-color`, as quatro `border-*-color`, `outline-color`, `fill`, `stroke`, `box-shadow`, mais `::before`/`::after` —, em JSON, antes e depois da mudança. Qualquer diferença reprova. É determinístico, independe de anti-aliasing e de fonte, e aponta o elemento exato.

Isso é feito com a D3 **isolada**, antes de qualquer cor do tema novo e antes do seletor (que muda a interface de propósito). Capturas de tela ficam como conferência humana, não como critério.

### D5 — Um identificador de tema, do cookie ao HTML, com uma tabela só

`dark`, `light` e `bio` são os valores do cookie e de `ui.default-theme`. `ViewData` passa a ter o identificador do tema, a classe (`dark`, vazio, `theme-bio`) e a `ThemeColor`, para o template não decidir nada. `data-default-theme` passa a levar o identificador; o vazio continua sendo lido como claro, e o atributo ausente ou com o template cru (`{{ .DefaultTheme }}`, servidor de desenvolvimento) como escuro, como hoje.

As três implementações — Go, script inline e `theme.js` — são testadas com **a mesma tabela de casos**, um JSON em `web/app/src/utils/theme.cases.json`: cookie (válido, inválido, ausente) × padrão configurado (`dark`, `light`, `bio`, vazio, ausente, template cru) → tema, classes e `theme-color` esperados, mais as transições entre todos os pares de temas. O teste do script inline extrai o script de `index.html` e o executa num `document` falso, para que a terceira cópia não fique de fora.

Em `internal/config/ui`, a presença de `dark-mode` e de `default-theme` é registrada **antes** de os padrões serem aplicados, para que a precedência e o aviso funcionem; o recarregamento da configuração com `default-theme` alterado é testado.

### D6 — Cache do HTML

O HTML varia com o cookie, então todo HTML da interface sai com `Cache-Control: no-cache` e `Vary: Cookie`, em GET e HEAD, nos dois manipuladores (`spa.go` e `spa_render.go`). As páginas com login próprio mantêm `private, no-store`. Com `no-cache`, um HTML antigo — com o script inline antigo, que não conhece `bio` — é sempre revalidado, o que também resolve a atualização de versão. `Vary: Accept-Encoding`, que o gzip já acrescenta, convive com `Vary: Cookie`. O requisito de cabeçalhos das rotas públicas já exige `no-cache` no HTML e não é contrariado; o cabeçalho novo entra por um requisito de `ui-theme`.

### D7 — Seletor, não alternância

Um componente único: botão que mostra o tema em uso e abre um menu com as três opções (`role="menuitemradio"`, setas, Esc, foco devolvido). No cabeçalho das páginas públicas, onde o rótulo já some abaixo de 640 px, o seletor fechado e o menu aberto têm de caber a 360 px com logo e título longo. `apple-mobile-web-app-status-bar-style` fica como está.

### D8 — O gráfico e o cartão de suíte

`ResponseTimeChart.vue` troca o booleano `isDark` pelo identificador do tema, lê grade, texto e marcas de variáveis CSS e é redesenhado em qualquer troca de tema (hoje só reage a claro↔escuro). `SuiteCard.vue` ganha a regra do tema `bio` ao lado de `.dark .suite-header`. A tarefa 2.5 procura outros seletores `.dark` em CSS e outras cores fixas em componentes.

## Risks / Trade-offs

- **Regressão nos temas existentes** → D4, critério de parada.
- **Contraste** → D2, medido por script nas telas reais, com a regra de não piorar para as cores de estado.
- **Três cópias da regra de tema** → D5, uma tabela para as três.
- **Piscada de tema e HTML antigo** → D6.
- **E2E que passam a testar menos do que parecem** → a migração dos helpers é tarefa, não detalhe: um `set_theme` que só mexe em `dark` deixaria `theme-bio` ligada.

## Migration Plan

Campo opcional e cookie com valor novo: nada a migrar. Voltar de versão: a versão anterior ignora `ui.default-theme` (o YAML é lido de forma tolerante) e trata `theme=bio` como valor inválido, caindo no tema padrão.

## Open Questions

- **Cabeçalho claro ou marinho?** O site de onde a paleta vem usa cabeçalho claro; um marinho daria mais identidade. A tarefa 3.1 entrega as duas capturas para o dono escolher.
