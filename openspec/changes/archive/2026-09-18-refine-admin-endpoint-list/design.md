## Context

A lista de endpoints (`views/admin/AdminEndpoints.vue`) é uma tabela `table-fixed` dentro do painel do `AdminListLayout`, com oito colunas e larguras em porcentagem definidas na change `align-details-and-admin-lists`: Name 22%, Group 12%, Type 9%, URL (o resto), Interval 8% (a partir de `lg`), Status 9%, Source 8% (a partir de `lg`) e Actions `15rem`. As células usam `px-3 py-1.5` e a tabela, `text-sm`.

Medições em 1000×800, com um endpoint ativo que recebe push (`push.endpoints` no arquivo de configuração):

| Célula | Largura | Observação |
|---|---|---|
| Name | 213 px | |
| Group | 116 px | |
| **Type** | **87 px** | `HTTP` mais o selo `+ push` **não cabem**: o selo quebra e a linha vai a **53 px** |
| URL | 224 px | |
| Status | 87 px | |
| Actions | 240 px | três botões de texto: View/Edit, Disable/Enable e Remove |

Uma linha sem selo tem **49 px**: os botões de ação são `size="sm"` (36 px de altura) mais os 12 px de `py-1.5` da célula e a borda. As outras duas listas têm a mesma estrutura: `AdminStatusPages.vue` com cinco ações por linha (Open, Copy link, Edit, Disable, Remove) numa coluna de `25rem`, e `AdminPushKeys.vue` com uma ação (Revoke) em `8rem`.

Os roteiros E2E clicam nessas ações **por identificador** (`admin-open-*`, `admin-toggle-*`, `admin-remove-*`, `status-page-open-*`, `status-page-copy-*`, `status-page-edit-*`, `status-page-toggle-*`, `status-page-remove-*`, `push-key-revoke-*`) e nunca pelo texto do botão; os dois `--text` que existem (`removed`, `revoked`) são de toasts, não de botões. `lucide-vue-next` já é dependência e é usada em outras telas do fork.

## Goals / Non-Goals

**Goals:**

- Nenhuma linha da lista com altura diferente das outras por causa de um selo.
- Mais endpoints visíveis na mesma altura de tela, sem perder informação nem legibilidade.
- Ações reconhecíveis, acessíveis por teclado e por leitor de tela, ocupando pouco espaço.
- As três listas com a mesma densidade e o mesmo padrão de ação.

**Non-Goals:**

- Mudar o que a lista mostra, a busca, os filtros, a ordenação ou os diálogos de confirmação.
- Mexer nos formulários, no dashboard ou nas páginas públicas.
- Trocar os identificadores de teste.
- Voltar a permitir rolagem horizontal: o que a change anterior garantiu continua valendo.

## Decisions

### D1 — Densidade da tabela

| Onde | Hoje | Fica |
|---|---|---|
| Fonte da tabela | `text-sm` (14 px) | `text-xs` (12 px), com o nome em `text-sm` para continuar sendo o ponto de entrada da linha |
| Respiro das células | `px-3 py-1.5` | `px-2 py-1` |
| Cabeçalho | `px-3 py-2` | `px-2 py-1.5`, em `text-xs` com `uppercase tracking-wide` |
| Altura da linha | 49 px (53 px com selo) | **29 px**, igual com ou sem selo |
| Célula de ações | `px-3 py-1.5` | `px-2 py-0`: é o botão que define a altura |
| Selos (`push`, origem) | `px-1.5 py-0.5 text-xs` | `px-1 text-[11px] leading-4`, sem quebra |

A conta fecha assim: o nome em `text-sm` (20 px de caixa) mais `py-1` (8 px) e a borda dá 29 px; o botão de ação de 28 px numa célula com `py-0` cabe nesses 29 px. Com `py-1` na célula de ações, como estava escrito antes, a linha iria a 37 px e o teto de 32 px da spec seria impossível — foi medido reproduzindo o markup com o CSS compilado do projeto.

A URL continua em `font-mono`, um ponto abaixo do resto (`text-[11px]`), porque é o campo mais longo e o que menos precisa ser lido por inteiro na lista.

### D2 — O selo de push e a coluna Type

O problema é de largura e de quebra, então os dois são resolvidos:

- o selo ganha `whitespace-nowrap`, que impede o `+` de se separar do `push`, **e a célula também**, para o selo inteiro não descer para uma segunda linha;
- a coluna Type passa de 9% para `7rem` fixos, que comportam `HTTP` mais o selo compacto em 1280, 1000 e 768 px — medido reproduzindo o markup. O tipo mais longo do produto é `WEBSOCKET`, que com o selo precisa ser conferido em 768 px na implementação; se não couber, a coluna vai para `8rem` e a tabela de D3 é refeita;
- o selo passa a dizer `push` apenas, sem o `+`, com a dica "Also receives push" no `title` — o `+` era decoração e custava 8 px;
- o selo **mantém** o `data-testid="admin-accepts-push-<chave>"`, que `test/e2e/push.sh` espera.

### D2.1 — O aviso de conflito vira ícone

O aviso "Conflicts with YAML" e o de definição inválida ocupam hoje uma segunda linha dentro da célula do nome, justamente para não somarem largura. Isso deixa a linha em 53 px — o mesmo defeito do selo, por outro caminho. Eles passam a ser um ícone `AlertTriangle` de 14 px, `shrink-0`, ao lado do nome, num `flex` com o nome em `min-w-0 truncate`, com a origem do conflito ou o erro no `title` e nome acessível equivalente. Como a tabela é `table-fixed`, o ícone não soma largura, e a linha fica com a mesma altura das outras — o que permite exigir na spec que **todas** as linhas tenham a mesma altura.

### D3 — Proporção das colunas

Com a coluna de ações encolhendo (D4), a largura recuperada vai para o conteúdo. Larguras finais dos endpoints:

**Endpoints:**

| Coluna | Largura | Alinhamento |
|---|---|---|
| Name | `26%` | esquerda |
| Group | `12%` | esquerda |
| Type | `7rem` | esquerda |
| URL | resto | esquerda |
| Interval | `6rem` (só `lg`) | direita |
| Status | `7rem` | esquerda |
| Source | `5rem` (só `lg`) | esquerda |
| Actions | `7rem` | direita |

**Status pages:**

| Coluna | Largura | Alinhamento |
|---|---|---|
| Slug | `20%` | esquerda |
| Title | resto | esquerda |
| Source | `5rem` (só `lg`) | esquerda |
| Status | `8rem` | esquerda |
| Endpoints | `6rem` (só `lg`) | direita |
| Actions | `11rem` | direita |

**Chaves de push:**

| Coluna | Largura | Alinhamento |
|---|---|---|
| Name | resto | esquerda |
| Key | `7rem` | esquerda |
| Source | `5rem` (só `lg`) | esquerda |
| Created | `30%` | esquerda |
| Actions | `4rem` | direita |

Intervalo e quantidade de endpoints alinhados à direita porque são numéricos, o que os algarismos tabulares da fork.23 já favorecem. **Orçamento de largura:** a soma das colunas fixas dividida por `1 −` (soma das porcentagens) tem de caber na largura mínima da tabela (768 px menos o respiro do painel), e a célula de ações leva `whitespace-nowrap` com os ícones ocupando no máximo a largura declarada — cinco ícones de 28 px com `px-2` dão 172 px, dentro dos 176 px de `11rem`. A célula de ações da lista de chaves perde o `h-12` que ela tem hoje para igualar linhas com e sem botão: com `py-0` as duas ficam iguais sem isso.

### D4 — Ações como ícones

Cada ação vira um alvo quadrado de 28 px com ícone de 14 px, `title` e `aria-label` com o texto que hoje está escrito, mantendo o `data-testid` atual:

| Ação | Ícone (`lucide-vue-next`) | Nome acessível |
|---|---|---|
| Editar | `Pencil` | "Edit \<nome\>" |
| Ver (origem YAML) | `Eye` | "View \<nome\>" |
| Desabilitar | `CirclePause` | "Disable \<nome\>" |
| Habilitar | `CirclePlay` | "Enable \<nome\>" |
| Remover | `Trash2` | "Remove \<nome\>" |
| Abrir página pública | `ExternalLink` | "Open \<slug\>" |
| Copiar link | `Link2` | "Copy link of \<slug\>" |
| Revogar chave | `Trash2` | "Revoke \<nome\>" |

- Um componente `AdminActionButton.vue` embala o alvo com ícone, `title`, `aria-label`, estado desabilitado e o `data-testid` recebido por prop, para as três listas não repetirem o mesmo bloco e para o foco e o tamanho ficarem iguais. Detalhes que precisam estar nele:
  - **`type="button"`**, porque a barra da lista de chaves tem um `<form>`;
  - **âncora quando for link:** "Abrir página pública" é um `<a target="_blank" rel="noopener">` e continua sendo — o componente aceita `href` e renderiza `<a>` com as mesmas classes, o mesmo tamanho e o mesmo foco, e o nome acessível diz que abre em nova aba;
  - **foco visível sem estourar a linha:** o `Button` do projeto usa `ring-offset-2`, que desenha 4 px para fora e seria cortado pela área que rola; o componente usa anel interno (`ring-inset`, sem deslocamento);
  - **desabilitado com explicação:** o `Button` aplica `disabled:pointer-events-none`, então o `title` não apareceria; o componente usa `aria-disabled` com o clique inerte, mantendo a dica — hoje o administrador pelo menos lê "Disable" quando o botão está desabilitado por conflito;
  - **vermelho no destrutivo** com variante escura (`text-red-600 dark:text-red-400`), como hoje, e não o `variant="destructive"` de fundo cheio.
- A confirmação continua a mesma: o ícone só abre o diálogo.
- Nos **cartões** das telas estreitas, os mesmos ícones aparecem, mas com alvo de 36 px: ali a tela é de toque, e a uniformidade que interessa é a do símbolo, não a do tamanho.

**Acessibilidade:** botão só com ícone precisa de nome; o `aria-label` com o nome do item também resolve a ambiguidade de "qual linha" para quem navega por teclado, e o `title` dá a dica ao ponteiro. Os ícones ficam `aria-hidden`.

### D5 — Testes

**E2E**, aproveitando os roteiros que já abrem as três listas:

1. a conferência da lista de endpoints fica em **`push.sh`**, que já tem `push.endpoints` na configuração e já espera os selos `admin-accepts-push-_cdn` e `admin-accepts-push-core_health`: é onde o defeito se reproduz sem mexer em fixture. A medida é feita em **900 e 1000 px**, onde a coluna Type é apertada — em 1280 px o selo cabe e o defeito não aparece;
2. a altura de **todas** as linhas da lista MUST ser igual, incluindo a do endpoint com selo de push e a do endpoint em conflito, e MUST ficar em 32 px ou menos (o desenho fica em 29 px; a folga cobre a variação de fonte entre plataformas);
3. cada ação MUST continuar acessível pelo mesmo identificador e MUST ter nome acessível: `aria-label` não vazio contendo o nome do item **e** `textContent` vazio — as duas asserções juntas, porque só a segunda passaria com um rótulo `sr-only` e só a primeira passaria com o texto ainda visível;
4. clicar no ícone de remover MUST abrir a confirmação, como hoje (o roteiro já faz isso por identificador);
5. a mesma conferência de altura e de nome acessível nas listas de status pages (em `status-pages.sh`) e de chaves de push (em `push.sh`);
6. **passo em risco:** `push.sh` mede a barra de rolagem fina com a janela em 1280×420, e falha de propósito se a lista **não** transbordar. Com as linhas em 29 px as cinco linhas passam a caber, e o passo quebraria; a altura da janela desse trecho desce para 1280×300, que volta a produzir transbordo com folga.

**Conferência manual:** as três listas em 1280×900 e 1100×800, nos dois temas, com um endpoint que recebe push, um em conflito e um nome longo; e os cartões em 390 px.

## Risks / Trade-offs

- **Texto de 12 px** é menor que o padrão do restante da administração. É uma tela de varredura, com o nome em 14 px como âncora, e o formulário continua no tamanho normal.
- **Ações como ícones** exigem aprender três símbolos; o `title` e o `aria-label` cobrem a dúvida, e Editar/Remover são convenções conhecidas. Quem usa leitor de tela ganha, porque o nome do item passa a fazer parte do rótulo da ação.
- **Alvo de 28 px** na tabela é menor que os 36 px de hoje: passa o mínimo de 24 px da WCAG 2.5.8, mas exige mais pontaria. Nos cartões das telas de toque o alvo continua com 36 px.
- **Três listas mudando juntas** é mais superfície de revisão; em troca, a administração não fica com duas linguagens visuais.
