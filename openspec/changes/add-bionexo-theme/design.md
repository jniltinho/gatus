## Context

O tema é decidido em três lugares que precisam concordar:

- **Servidor** (`internal/api/spa_render.go`): `themeFromRequest` lê o cookie `theme` (`dark` → classe `dark`, `light` → classe vazia) e, sem cookie válido, usa `defaultTheme`, que vem de `ui.dark-mode` (escuro por padrão) e devolve `dark` ou a string vazia — não existe um valor `light` hoje. O `<html>` sai com `class="{{ .Theme }}"` e `data-default-theme="{{ .DefaultTheme }}"`, os dois com esse mesmo par de valores, e a `theme-color` sai do mesmo template.
- **Script inline** de `web/app/public/index.html`, que aplica a mesma regra antes da primeira pintura.
- **`web/app/src/utils/theme.js`**, usado pelos três botões de alternar: `Settings.vue`, `PublicLayout.vue` e `LoginPage.vue`.

O Tailwind (3.1.8) está com `darkMode: 'class'`. Contei, nos 47 componentes: **720** variantes `dark:`, **477** classes por variável (`bg-background`, `text-muted-foreground`, ..., sobre as 48 variáveis HSL de `src/index.css`) e **578** classes de cinza fixo (`bg-white`, `text-gray-900`, `border-gray-800`, ...). Só o segundo grupo responde a um tema hoje.

Contraste das cores da marca, medido (WCAG): marinho `#1d3c55` sobre branco 11,5:1; azul `#285783` sobre branco 7,6:1; verde-água `#34cdd7` sobre branco **1,9:1** e `#2cb7c0` **2,4:1**; marinho sobre verde-água `#34cdd7` 5,9:1.

## Goals / Non-Goals

**Goals:** um terceiro tema completo, em todas as telas; nenhum pixel de diferença nos temas claro e escuro; nenhuma piscada de tema na carga; texto com contraste AA.

**Non-Goals:** temas definidos pelo usuário em configuração; um tema Bionexo escuro; mudar as cores de estado; mudar o logo; seguir `prefers-color-scheme` (o projeto decidiu não seguir).

## Decisions

### D1 — Base clara, sem a classe `dark`

O tema é a classe `theme-bionexo` no `<html>`, **sem** `dark`: as 720 variantes `dark:` ficam desligadas e o ponto de partida é o tema claro, que é também a cara do site da empresa. Um tema Bionexo de base escura exigiria uma segunda dimensão (marca × claro/escuro) no cookie, no servidor e no seletor; fica fora (Non-Goals) até alguém pedir.

### D2 — O verde-água não é cor de texto

Com 1,9:1 a 2,4:1 sobre branco, o verde-água não passa nem no mínimo de 3:1 de componentes gráficos. Papéis no tema:

- **texto e títulos:** marinho `#1d3c55`; texto secundário num marinho dessaturado que mantenha 4,5:1;
- **botão primário:** fundo verde-água com texto marinho (5,9:1), que é o que o site faz; ou fundo marinho com texto branco (11,5:1) onde o botão precisa pesar mais;
- **links e foco:** azul `#285783` (7,6:1); o anel de foco nunca é só verde-água;
- **verde-água** fica para fundos de realce, bordas decorativas e o cabeçalho, sempre acompanhado de outra pista.

As cores de estado continuam as de hoje; a tarefa 3.3 mede o contraste delas sobre as superfícies novas.

### D3 — As 578 classes de cinza passam por variáveis

Reescrever 578 classes para tokens semânticos seria a solução limpa e um diff enorme, em componentes que acabaram de estabilizar. Em vez disso, o `tailwind.config.js` redefine `colors.gray` (50 a 900, a escala do Tailwind 3.1.8 instalado) e `colors.white` como `rgb(var(--gray-N) / <alpha-value>)`. Em `:root` e `:root.dark` as variáveis recebem **exatamente** os valores atuais da escala do Tailwind; em `:root.theme-bionexo`, uma escala tingida de marinho, com o "branco" podendo virar um off-white azulado.

- `<alpha-value>` preserva as classes com opacidade (`bg-gray-900/50`).
- O CSS gerado muda de forma, não de valor: por isso a regressão visual da tarefa 1.2 é a porta de entrada — se os temas claro e escuro não saírem idênticos, a abordagem é revista antes de qualquer cor nova.
- `ui.custom-css` de quem usa continua funcionando: a especificidade das utilidades não muda.

### D4 — Um identificador de tema, do cookie ao HTML

`dark`, `light` e `bionexo` são os três valores do cookie e de `ui.default-theme`. O servidor os traduz em classes (`dark`, vazio, `theme-bionexo`) num único lugar, e `ViewData.Theme` continua sendo a lista de classes. `DefaultTheme` hoje é `dark` ou vazio, e `theme.js` (`defaultThemeIsDark`) só distingue `dark` do resto: ele passa a carregar o identificador (`dark`, `light` ou `bionexo`), com o vazio ainda lido como claro para um HTML em cache de uma versão anterior. O script inline e `theme.js` aplicam a mesma tabela — hoje as duas cópias da regra já existem, e o teste de unidade de `theme.js` ganha o caso novo. Valor desconhecido no cookie é ignorado, como hoje.

`ui.default-theme` inválido é erro de configuração na inicialização (e no `gatus config validate`). Com `default-theme` e `dark-mode` juntos, `default-theme` vale e um aviso no log diz isso; `dark-mode` sozinho se comporta como hoje, sem aviso: não é depreciação.

### D5 — Seletor, não alternância

Um botão que alterna entre três estados não diz qual é o próximo. Os três pontos de troca passam a usar um componente único: um botão que mostra o tema atual e abre um menu com as três opções (`role="menuitemradio"`, setas, Esc, foco devolvido ao botão). Nas páginas públicas ele substitui o botão "Dark mode"/"Light mode".

### D6 — O que mais lê cor do tema

`theme-color` (uma cor por tema) e as variáveis da barra de rolagem (o requisito vigente exige 3:1 em cada tema). Dois componentes **não** leem variáveis: decidem por `document.documentElement.classList.contains('dark')` e usam cores fixas — `ResponseTimeChart.vue` (grade, texto e marcas do gráfico, em `rgba` e hexadecimais) e `SuiteCard.vue`. No tema novo eles cairiam nas cores do claro, com texto quase preto no lugar do marinho. Passam a ler as cores de variáveis CSS do tema, o que também tira deles a segunda cópia da paleta. A tarefa 2.4 procura outros casos (os SVG de badges não contam: são servidos prontos e não seguem o tema).

### D7 — Ordem em relação a `collapse-status-page-groups`

O requisito "Página pública de status" de `status-page-web-ui` diz que a contagem fica "legível nas duas versões do tema", e o requisito dos testes E2E manda capturar "nos temas claro e escuro". A change `collapse-status-page-groups`, ainda não arquivada, modifica o primeiro. Para não ter duas changes ativas modificando o mesmo requisito, esta proposta não o toca: a regra geral entra em `ui-theme` (todo requisito que fala dos temas vale para todos os temas oferecidos), e a atualização daquelas duas frases é uma tarefa desta change, feita depois de a outra ser arquivada.

## Risks / Trade-offs

- **Regressão visual nos temas existentes** → D3 e a tarefa 1.2; é o critério de parada.
- **Contraste** → D2, medido por script nas superfícies reais (tarefa 3.3), não no olho.
- **Piscada de tema** → o servidor já entrega a classe; o E2E confere o HTML cru, antes do JavaScript.
- **Três cópias da regra** (servidor, script inline, `theme.js`) → uma tabela de casos compartilhada pelos testes do servidor e de `theme.js`.
- **Cache**: o HTML varia pelo cookie `theme`; conferir que os cabeçalhos de cache da SPA já tratam isso (tarefa 2.2).

## Migration Plan

Campo opcional e cookie com valor novo: nada a migrar. Voltar de versão: a versão anterior ignora `ui.default-theme` (o YAML é lido de forma tolerante) e um cookie `theme=bionexo` (valor inválido → tema padrão).

## Open Questions

- **Nome do tema.** O repositório e a imagem são públicos. Um tema chamado `bionexo`, com a paleta da empresa, é uma decisão do dono: a alternativa é um nome neutro (`ocean`, `teal`) com a mesma paleta, e `bionexo` só no uso interno. A proposta usa `bionexo`, como pedido.
- **Cabeçalho marinho ou claro?** O site usa cabeçalho branco; um cabeçalho marinho daria mais identidade ao painel. A proposta deixa claro, como o site, e mostra as duas opções em captura na tarefa 3.1 para o dono escolher.
- **`bionexo` como tema padrão da instalação do dono** é só configuração (`ui.default-theme: bionexo`); o padrão do projeto continua escuro.
