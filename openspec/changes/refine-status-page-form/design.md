## Context

`AdminStatusPageForm.vue` ainda usa o casco antigo `container mx-auto px-4 py-8 max-w-5xl`: o `App.vue` só dá o casco de altura total (`md:flex md:h-screen md:flex-col md:overflow-hidden` no contêiner, `md:flex md:min-h-0 md:flex-1 md:flex-col` no `<main>`) para as rotas com `meta.adminList === true`, hoje `/admin`, `/admin/status-pages`, `/admin/push-keys` e `/admin/backup`. As três seções do formulário (General, Groups, Endpoints), a lista de endpoints com `max-h-80 overflow-y-auto`, a faixa de validação e a seção de pré-visualização ficam empilhadas na mesma página, então a janela rola muito e ainda há uma segunda barra dentro da lista. A tela de Backup já resolveu isso com colunas `md:min-h-0 md:flex-1` e corpo `overflow-auto overscroll-contain`, e a administração já tem `AdminDialog`, `dialogStack` e `utils/toast.js` desde a fork.19.

O formulário tem quatro estados de tela, não um: carga (`loading`, com o spinner), somente leitura (página do YAML, um `<pre>` com o `yamlText`), edição e criação. `showPreview()` numa página ainda não salva não abre pré-visualização nenhuma: valida e responde "Valid definition. Save the page to preview it…". O `save()` de uma página nova navega para a rota de edição (`router.push`), e o `App.vue` chama `clearToasts()` em todo `afterEach` com caminho diferente.

No gráfico, `ResponseTimeChart.vue` registra `legend: { display: false }` por paridade com o Uptime Kuma e desenha, nos períodos agregados, três datasets: `avg-ping` (`#5CDD8B`), `min-ping` (`#3CBD6B38`) e `max-ping` (`#7CBD6B38`). Os dois últimos têm alfa `0x38` (22%), então sobre o cartão claro as três linhas são praticamente o mesmo verde claro: nem a cor nem o tooltip — filtrado para o dataset 0 — dizem qual é qual. O contêiner de altura fixa já expõe `data-testid="response-time-chart"`, `data-period`, `data-loading`, `data-line-points`, `data-down-columns` e `data-pending-columns`, lidos por `test/e2e/push.sh`.

## Goals / Non-Goals

**Goals:**

- O formulário de status page cabe na janela em telas médias e grandes, nos quatro estados de tela, com uma barra de rolagem por coluna e nenhuma rolagem dentro de outra.
- As ações disponíveis na tela ficam sempre visíveis, sem depender de rolar até o fim.
- A pré-visualização e as mensagens usam os padrões que a administração já tem (diálogo e toasts), sem perder nada que precise ficar na tela.
- O cartão Response Time Trend diz o que é cada linha **e** deixa as linhas distinguíveis entre si, no dashboard e na página pública.

**Non-Goals:**

- Mudar o que a API do formulário envia ou recebe, a validação do backend ou o formato das definições salvas.
- Mudar as cores, a altura, o tooltip ou os dados do gráfico além do necessário para distinguir mínimo e máximo.
- Refazer o formulário de endpoints (`AdminEndpointForm.vue`), que continua com a rolagem da página.
- Trocar a barra de rolagem em si: a espessura veio da change `style-thin-scrollbars`.

## Decisions

### D1 — O formulário entra no casco de altura total, com um corpo por estado de tela

As rotas `AdminStatusPageNew` e `AdminStatusPageEdit` recebem `meta: { admin: true, adminList: true }`. Como o casco `md:h-screen md:overflow-hidden` recorta o que passar da janela, **os quatro estados** precisam do próprio corpo que rola — não só a grade de edição.

Casco comum, com os prefixos `md:` para preservar o comportamento de hoje no celular:

```
<div class="container mx-auto flex max-w-7xl flex-col px-4 py-4 md:min-h-0 md:flex-1">
  <header class="shrink-0">…título, slug, Back…</header>
  <div class="shrink-0">…avisos que ficam: somente leitura, definição salva inválida, conflito de versão…</div>
  …corpo do estado…
</div>
```

Corpos:

- **Carga:** `<div class="flex justify-center py-12 md:min-h-0 md:flex-1 md:items-center md:py-0">` com o `Loading`, para o spinner ficar centrado na área em vez de colado no topo.
- **Somente leitura:** `<div class="border bg-card p-6 md:min-h-0 md:flex-1 md:overflow-auto md:overscroll-contain">` com o `<pre>` dentro. O `<pre>` deixa de ser o elemento recortado e passa a rolar; o aviso de somente leitura fica no bloco `shrink-0` acima, então continua visível enquanto o YAML rola.
- **Edição e criação:** grade de duas colunas, cada uma no formato do Backup (cabeçalho fixo e corpo que rola), mais a barra de ações no rodapé:

```
<div class="mt-3 grid gap-4 md:min-h-0 md:flex-1 md:grid-cols-2 md:grid-rows-1">
  <div class="flex min-h-0 flex-col gap-4 md:overflow-auto md:overscroll-contain">…General, Groups, faixa da validação…</div>
  <section class="flex min-h-0 flex-col border …">
    <header class="shrink-0 …">Endpoints…</header>
    <div class="shrink-0 …">busca e "Only selected"</div>
    <div class="max-h-80 overflow-y-auto border md:max-h-none md:min-h-0 md:flex-1 …" data-testid="status-page-endpoint-list">…cabeçalho sticky e linhas…</div>
  </section>
</div>
<div class="mt-3 shrink-0 border-t pt-3">…Validate, Preview, Save…</div>
```

A largura passa de `max-w-5xl` para `max-w-7xl`, a mesma das listas e do Backup, porque agora são duas colunas. O cabeçalho `sticky top-0` da lista **continua dentro** de `status-page-endpoint-list`: tirá-lo de lá desalinharia as colunas `grid-cols-[1fr_auto_auto]` do cabeçalho e das linhas assim que a barra de rolagem aparecesse. Quem rola na coluna da esquerda é o `div` de `flex min-h-0 flex-col`, não o `space-y-4` das seções.

Duas correções de largura na coluna estreita (~360 px em `md`):

- a linha do slug empilha: o prefixo `/status/` e o `Input` numa linha e os botões Copy link e Open abaixo, até `lg` (`flex-wrap` com `lg:flex-nowrap`);
- os cartões de grupo passam de `sm:grid-cols-2 lg:grid-cols-3` para `lg:grid-cols-2` — trocando, não acumulando.

### D2 — Pré-visualização no `AdminDialog`, com o contrato que ele tem

O `AdminDialog` emite `close`; não existe `update:open`, então `v-model:open` deixaria Esc e o botão de fechar sem efeito. O uso é o mesmo do `AdminBackup.vue`:

```
<AdminDialog :open="preview !== null" size="xl" testid="status-page-preview"
             title="Preview of the saved version" :description="…"
             :return-focus="previewButton" @close="preview = null">
```

O corpo leva o conteúdo de hoje (Featured e grupos), que já rola sozinho, e o rodapé um botão Close que chama o mesmo `@close`. `size="xl"` porque uma página com vários grupos comprime no `lg` padrão. `:return-focus` aponta para o botão Preview, senão o foco volta para o `h1`. Os identificadores `status-page-preview` e `status-page-preview-featured` continuam, e `status-page-preview-toggle`, `previewExpanded` e os ícones `ChevronDown`/`ChevronRight` saem.

Na **criação**, `showPreview()` continua sem abrir diálogo: valida e emite um toast de informação dizendo para salvar a página antes de pré-visualizar. É o comportamento atual, agora escrito na spec.

### D3 — Toasts para o que é transitório, faixa para o que precisa ficar

- **Sucesso ao salvar:** `toast.success(...)`. Numa página nova, `save()` navega para a rota de edição e o `App.vue` chama `clearToasts()` no `afterEach`, então o toast disparado antes da navegação some. A mensagem continua guardada em `pendingSuccess` e é emitida no `watch` de `props.slug`, **depois** do `load()` — é o que mantém "Status page created" na tela de edição e o passo do roteiro E2E que espera esse texto.
- **Uma mensagem por ação:** Validate emite só o toast de informação com o resumo ("The page will show N endpoints"); a mensagem "Valid definition." deixa de existir como toast separado, para não empilhar dois avisos da mesma ação.
- **Faixa da validação:** `status-page-validation` continua no topo da coluna da esquerda e só aparece **quando há avisos**. Quando aparece, a coluna rola até ela (`scrollIntoView({ block: 'nearest' })`), senão ela nasce fora da vista de quem estava olhando a seção Groups.
- **Erros:** falha de operação vai para toast de erro, **menos** dois casos que ficam em faixa fixa acima do conteúdo: o conflito de versão (412), num estado próprio `conflictError` que só é limpo por `reloadCurrentVersion()` ou por um save bem-sucedido — o `clearMessages()` de cada ação não pode apagá-lo, senão o "sem perder o que foi digitado" vira sorte —, e a falha ao carregar a página, que deixaria a tela muda se sumisse em 10 segundos.
- Os avisos de somente leitura e de definição salva inválida continuam fixos, como hoje.

### D4 — Legenda em HTML, com as linhas distinguíveis

Nomear três verdes iguais não resolve o problema: sobre o cartão claro, `#3CBD6B38` e `#7CBD6B38` renderizam quase o mesmo tom. Então a legenda vem junto com uma diferença de traço nas linhas:

- `min-ping` ganha `borderDash: [6, 4]` e `max-ping` `borderDash: [2, 3]`; a média continua sólida. As cores e a translucidez não mudam.
- A legenda é um `<ul>` do próprio componente, abaixo do `<Line>` e **fora** do `div` de altura fixa, com `flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground`, `role="list"` e `aria-label="Chart series"`.
- Item de linha: uma amostra de 14 px de largura com `border-top: 2px` na cor cheia da série e o mesmo estilo de traço (`solid`, `dashed`, `dotted`), `aria-hidden`, e o nome ao lado. Item de coluna (Down, Pending): um quadrado de 10 px preenchido com a cor cheia e `border border-muted-foreground`, porque `--border` no tema claro tem contraste de ~1,2:1 e não daria forma nenhuma ao quadrado. O significado está sempre no texto do item.

`utils/responseTimeChart.js` ganha as cores cheias e a montagem dos itens:

```js
export const CHART_LEGEND_COLORS = Object.freeze({ line: '#5CDD8B', minLine: '#3CBD6B', maxLine: '#7CBD6B', down: '#DC3545', pending: '#F5B617' })

// chartLegend returns the legend items of the period, in the order they are drawn
export const chartLegend = (period, summary) => { … }
```

Em `recent`: `[{ id: 'response-time', label: 'Response time', color: line, dash: 'solid' }]`. Nos agregados: `average` (sólido), `minimum` (tracejado), `maximum` (pontilhado). Em ambos, `down` entra quando `summary.downColumns > 0` e `pending` quando `summary.pendingColumns > 0`, nessa ordem, no fim. Os `id` são os valores de `data-series`, em minúsculas.

O componente só desenha a legenda com `!loading && !error && payload`, para ela não vazar durante o spinner. O invólucro novo **não** recebe identificador: `data-testid="response-time-chart"` e todos os `data-*` continuam no `div` de altura fixa, que `test/e2e/push.sh` já lê por `dataset`, e a legenda é irmã dele, nunca filha.

### D5 — Testes

Em `test/e2e/status-pages.sh`:

1. o `config.yaml` do roteiro passa a ter ~25 endpoints extras (`interval: 1h`, apontando para `$BASE/health`), porque com os 4 de hoje a lista `flex-1` não transborda em 1280×900 e a asserção de rolagem falharia com o código certo;
2. copiar o `layout_ok` de `test/e2e/admin-backup.sh` (documento sem rolagem na vertical **e** na horizontal, mais `top >= 0 && bottom <= innerHeight + 1` para cada elemento) e aplicá-lo na edição com `status-page-save`, `status-page-validate`, `status-page-preview-button` e `admin-back`. A asserção de `scrollHeight` do documento sozinha não serve: com `md:overflow-hidden` ela é verdadeira mesmo com a tela recortada;
3. rolagem por dentro: `status-page-endpoint-list` com `scrollHeight > clientHeight` **e** `clientHeight > 320`, o que também detecta um `max-h-80` esquecido em `md`; a coluna da esquerda com `scrollHeight > clientHeight` no elemento que rola;
4. Preview: clicar, esperar `status-page-preview`, conferir `status-page-preview-featured`, apertar Esc, conferir que o diálogo sumiu e que o campo Title continua com o valor digitado — e só então seguir para os cliques em `status-page-field-enabled` e `status-page-save`, que hoje vêm logo depois e bateriam no overlay do diálogo;
5. Validate: conferir `[data-testid="toast"][data-type="info"]` com o resumo (o `toast_text` do backup), porque o mesmo texto já aparece na faixa e um `wait --text` passaria sem toast nenhum; e conferir que a faixa `status-page-validation` só existe quando há avisos;
6. revisar os cliques já existentes do roteiro (seleção de grupo e de endpoints) acrescentando `scrollintoview`, porque agora eles ficam dentro de colunas que rolam;
7. `not_covered` (do backup) em `admin-back` e em `status-page-save` com um toast visível;
8. legenda: na página pública em 24h, `[...document.querySelectorAll('[data-testid="response-time-chart-legend"] [data-series]')].map((item) => item.dataset.series).join(',')` igual a `average,minimum,maximum`; em Recent, `response-time`; e `data-loading="true"` sem nenhum nó da legenda;
9. no `push.sh` (dashboard, `_kuma-backup`, que tem Pending), conferir `data-series="pending"` na legenda e que a legenda **não** é descendente de `[data-testid="response-time-chart"]`, cuja altura continua em 250 px na janela de 1280×900.

Testes de unidade de `chartLegend`: Recent sem colunas, Recent só com Down, Recent com Pending, agregado com Down e Pending, agregado sem ponto de linha (os três itens continuam) e a ordem dos itens.

## Risks / Trade-offs

- **Duas colunas em telas médias:** entre 768 px e ~1024 px as colunas ficam estreitas; mitigado pelo empilhamento da linha do slug e pelos grupos em `lg:grid-cols-2`, e verificado pelo `scrollWidth` do `layout_ok`.
- **Janelas baixas:** com pouca altura cada coluna vira uma área de rolagem curta. É o mesmo compromisso já aceito nas listas e no Backup, e abaixo de `md` o layout antigo continua valendo.
- **Identificadores de teste:** mover a pré-visualização para o diálogo muda onde `status-page-preview` aparece. Só `status-pages.sh` usa esses identificadores, e o roteiro é atualizado junto.
- **Legenda e traço diferentes do Uptime Kuma:** o Kuma não tem legenda e desenha as três linhas sólidas. O fork passa a ter legenda e a tracejar mínimo e máximo; é um desvio consciente da paridade, registrado no requisito, porque três linhas iguais e sem nome não se explicam.
- **Toast sobre o botão Back:** a pilha de toasts fica no topo e pode encostar no cabeçalho do formulário; o E2E confere com `not_covered`, e se cobrir, Back vai para a barra inferior.
