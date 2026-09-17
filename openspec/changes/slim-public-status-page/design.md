## Context

- `views/public/StatusPage.vue`: contêiner `container mx-auto px-4 py-8 max-w-5xl`, cabeçalho `mb-6` com título `text-3xl`, `StatusSummary`, seção de destaques `mt-8` com `grid gap-4 md:grid-cols-2`, e uma seção por grupo `mt-8` com cabeçalho `border-b pb-2` e `<ul class="divide-y">`.
- `components/public/EndpointRow.vue`: cada linha é um `<li>` com `py-3` (destaque: `border bg-card p-4`), contendo:
  - cabeçalho `flex` com ponto de estado, nome, rótulo do estado e, à direita, um `<dl class="flex gap-4">` com `24h 100% 7d 100% 30d 100%` — larguras variáveis, sem alinhamento entre linhas;
  - linha opcional da expiração do certificado (`mt-0.5`);
  - as barras: `mt-2 flex gap-px` com `span` de `h-6`, `role="group"`, `tabindex="0"`, navegação por setas, `@mouseenter`/`@click` por barra;
  - `<p class="mt-1 min-h-[1rem] text-xs" aria-live="polite" data-testid="status-endpoint-detail">` com o detalhe da barra ativa — **sempre presente**, reservando 16 px + 4 px por endpoint;
  - um `<p class="sr-only">` com o resumo acessível.
- O mesmo componente serve a página de detalhes (`show-header` falso), onde só as barras e o detalhe aparecem.
- A spec já chama esse detalhe de "tooltip" (`Acessibilidade da página pública`), mas hoje ele é uma linha fixa no fluxo.
- Medida atual de uma linha de grupo em 1280×900: 12 + **24** + 8 + 24 + 4 + 16 + 12 ≈ **100 px**. O nome não tem classe de tamanho, então herda `text-base` (16 px) e ocupa 24 px de caixa de linha, não 20 px.
- O E2E lê `[data-testid="status-endpoint-detail"]` depois de focar o histórico pelo teclado, e confere 25 barras em 390 px.

## Goals / Non-Goals

**Goals:**

- Cerca de 65 px por endpoint de grupo, sem perder nenhuma informação da tela.
- Uptimes alinhados na vertical, comparáveis de relance.
- Detalhe da verificação sem ocupar espaço fixo e sem empurrar a lista quando aparece.
- Mesmo comportamento no celular, sem rolagem horizontal a partir de 360 px.

**Non-Goals:**

- Mudar o dashboard, as telas de administração ou a página pública de detalhes além do que vier do componente compartilhado.
- Mudar as cores de estado, a quantidade de barras ou os dados do payload público.
- Trocar a fonte ou o visual quadrado.

## Decisions

### D1 — Ritmo vertical

| Onde | Hoje | Fica |
|---|---|---|
| Contêiner da página | `py-8` | `py-6` |
| Cabeçalho da página | `mb-6` | `mb-4` |
| Seções (destaques e grupos) | `mt-8` | `mt-6` |
| Cabeçalho do grupo | `pb-2` | `pb-1.5` |
| Linha do endpoint | `py-3` | `py-2` |
| Espaço antes das barras | `mt-2` | `mt-1.5` |
| Altura das barras | `h-6` (24 px) | `h-5` (20 px) |
| Linha do detalhe | `mt-1` + `min-h-[1rem]` | removida (D3) |
| Cartão do destaque | `p-4`, tabela `mt-3`, células `py-1` | `p-3`, tabela `mt-2`, células `py-0.5` |
| Grade dos destaques | `gap-4` | `gap-3` |

Conta da linha de grupo: 8 + **24** + 6 + 20 + 8 = **66 px** (com a linha do certificado, quando ligada, 66 + 18 = 84 px; a partir da segunda linha do grupo, mais 1 px da borda do `divide-y`). As barras continuam com `gap-px` e `flex-1`.

A altura de 20 px das barras (`h-5`) faz parte do requisito, não só do design: um `h-6` esquecido ainda caberia no teto de 72 px da spec.

Também entram na tabela o `mb-3` do título "Featured", que vai a `mb-2`. Ficam **fora** desta change, de propósito: a faixa do `StatusSummary` (`px-4 py-3`), que é um elemento só e não afeta a densidade da lista, e a página pública de detalhes (`py-8 space-y-6`), que não é uma lista de serviços — lá o que muda vem do componente compartilhado.

Os 20 px de altura das barras continuam acima do mínimo confortável de toque quando somados ao `py-2` da linha, e a área clicável de cada barra continua sendo a largura toda dela. O alvo de 24 px do WCAG 2.5.8 não era atendido antes nem depois: as barras são um histórico, e o mesmo dado está no resumo acessível e na página de detalhes.

### D2 — Uptimes em colunas alinhadas

O `<dl>` continua com os pares `<dt>`/`<dd>` de hoje — `<dd>` solto é HTML inválido — e o que muda são as classes, um único markup servindo as duas larguras:

```
<dl class="flex shrink-0 gap-4 text-xs text-muted-foreground sm:grid sm:grid-cols-3 sm:gap-0" aria-hidden="true">
  <div v-for="period in periods" class="flex gap-1 sm:w-16 sm:justify-end">
    <dt class="sm:hidden">{{ period.label }}</dt>
    <dd class="font-medium text-foreground">{{ formatUptime(endpoint.uptime[period.key]) }}</dd>
  </div>
</dl>
```

- A partir de `sm` vira uma grade de três colunas de 64 px alinhadas à direita, e o rótulo some (`sm:hidden`). Abaixo de `sm` continua o que existe hoje: rótulo e valor juntos, em linha. `grid-cols-3` e `w-16` nos filhos andam juntos: os dois são normativos.
- **Rótulos uma vez por grupo:** o cabeçalho do grupo ganha, à direita, uma grade igual com `24h`, `7d` e `30d` (`hidden sm:grid sm:grid-cols-3`, filhos `w-16 text-right`), também `aria-hidden`, porque o leitor de tela já recebe o resumo acessível de cada endpoint.
- **Estrutura do cabeçalho do grupo:** o `justify-between` continua com **dois** filhos — um `<div class="flex min-w-0 items-baseline gap-3">` com o nome (`truncate`) e o estado do grupo, e a grade dos períodos. Com três filhos soltos, o estado do grupo iria parar no meio do cabeçalho.
- Os destaques mantêm a tabela própria, que já é alinhada.
- `formatUptime` continua devolvendo `—` quando não há dado, e as colunas continuam com `aria-hidden`, porque o resumo acessível já traz o uptime de 24h.

### D3 — Detalhe da verificação em tooltip flutuante

O `<p>` fixo sai do fluxo e vira um tooltip posicionado sobre as barras:

- o contêiner das barras ganha `relative`;
- o tooltip é `absolute z-10 w-max border bg-card px-2 py-1 text-xs shadow-sm pointer-events-none`, renderizado só quando há barra ativa, com `aria-hidden="true"` e mantendo `data-testid="status-endpoint-detail"`;
- **posição horizontal sem medir nada:** a barra ativa de índice `i` entre `n` dá `x = i / n * 100`. Até a metade, o tooltip é ancorado pela esquerda (`style="left: x%; max-width: calc(100% - x%)"`); da metade em diante, pela direita (`style="right: (100 − x − largura de uma barra)%; max-width: calc(x% + largura de uma barra)"`). Como a borda ancorada nunca sai do contêiner e o `max-width` fecha a outra ponta, o tooltip **não pode** vazar da linha. `translateX(-50%)` com `max-w-full` não resolveria: o CSS não conhece a largura de um `w-max` para clampar;
- **posição vertical:** `bottom-full mb-1` na linha de grupo, onde acima só existe o espaço entre as linhas; e `top-full mt-1` quando o componente está sem cabeçalho (página de detalhes) ou em destaque, porque nesses dois casos o que está acima das barras é conteúdo (o título do cartão e a linha "Last response / View details");
- `pointer-events-none` evita que o tooltip roube o `mouseleave` das barras. Em troca, o ponteiro atravessa o tooltip e chega ao que está atrás dele — na prática, às barras da linha de cima, que então mostram o próprio detalhe. É o comportamento normal de quem sai da área das barras, e está registrado em Risks;
- o texto continua idêntico (`horário · resultado · duração`), e o anúncio para leitores de tela passa para um `<span class="sr-only" aria-live="polite" aria-atomic="true">` irmão, **sempre presente**, vazio quando não há barra ativa: um `aria-live` que só aparece junto com o texto não anuncia nada, e deixar o texto visível sem `aria-hidden` faria o leitor anunciar duas vezes.

### D4 — Cabeçalho da página

Título continua `text-3xl` em `sm` e acima, e passa a `text-2xl` abaixo disso, com `mb-4` no bloco e `mt-1` na descrição (hoje `mt-2`). O aviso de truncamento e a mensagem de página sem serviços passam de `mt-3`/`mt-8` para `mt-2`/`mt-6`.

### D5 — Testes

As medidas ficam em `/status/messages`, não em `/status/services`: `messages` seleciona o grupo `core` sem destaques, então tem **dois** endpoints (`health` e `offline`) na mesma lista, com o divisor entre eles. Em `services`, `health` é destaque e cada grupo fica com um único endpoint, o que tornaria o alinhamento e o "a linha seguinte não se moveu" encenação.

Toda comparação numérica é resolvida **dentro** do `eval`, devolvendo `true`/`false` (ou com `Math.round`), como o roteiro já faz em `top_public`: `[ "66.4" -le 70 ]` é erro de sintaxe no `test` do bash.

Passos, na sessão pública em 1280×900:

1. **altura:** `Math.round(...getBoundingClientRect().height)` de `status-endpoint-offline` (linha com divisor) MUST ficar entre 56 e 72 px — reprova os 100 px de hoje e cabe nos 66 px do desenho, com folga para a variação de fonte entre plataformas;
2. **colunas existem e estão alinhadas:** para as três células de uptime dos dois endpoints, exigir `width > 0` e texto casando com `/\d|—/` **antes** de comparar os `left` com tolerância de 1 px. Sem isso, apagar as colunas deixa todos os retângulos em zero e o teste passa;
3. **rótulos:** dentro de `[data-testid="status-group-core"] li`, todo `<dt>` MUST estar com `display: none` em 1280 px, e a grade do cabeçalho do grupo MUST ter os três rótulos visíveis. Contar elementos não serve: o fallback do celular mantém os `<dt>` no DOM;
4. **tooltip sem empurrar e sem vazar:** medir a altura da linha e o topo da linha seguinte, passar o ponteiro pela **última** barra (`[role=group] > span:last-child`, a que mais tende a vazar; a primeira costuma ser um preenchimento sem resultado), conferir que o detalhe aparece com ` ms`, que altura e topo não mudaram, que `getComputedStyle(tooltip).pointerEvents === 'none'`, que o retângulo do tooltip cabe no da linha (`left >= linha.left - 1 && right <= linha.right + 1`) e que `document.documentElement.scrollWidth <= innerWidth + 1`. Repetir na primeira barra com resultado. Depois, tirar o ponteiro dali (`hover` no título da página), senão os prints seguintes saem com o tooltip aberto;
5. **região aria-live:** com a página recém-carregada, sem barra ativa, `[data-testid="status-endpoint-offline"] [aria-live="polite"]` MUST existir e estar vazio, e `[data-testid="status-endpoint-detail"]` MUST NOT existir;
6. o tooltip por teclado continua funcionando (o passo que já existe, com `[data-testid="status-endpoint-detail"]`);
7. **telas estreitas:** em 390×844 e também em 360×800 (a largura mínima do requisito publicado), conferir `document.documentElement.scrollWidth <= innerWidth + 1`, as 25 barras e que os `<dt>` estão visíveis numa linha de grupo (`status-endpoint-offline`), não no cartão de destaque, que não tem colunas de uptime;
8. prints da página nos dois temas, para comparar o antes e o depois.

Sem teste de unidade: é layout.

## Risks / Trade-offs

- **Tooltip cobrindo a linha de cima:** consequência de tirar a linha reservada. Some ao sair do ponteiro e tem fundo opaco com borda. Como ele é transparente ao ponteiro, mover o mouse para cima dele leva o ponteiro às barras da linha anterior, que passam a mostrar o próprio detalhe — é o mesmo que sair da área das barras.
- **Barras de 20 px** são um alvo menor para o ponteiro e para o toque do que os 24 px de hoje. O histórico continua acessível pelo teclado e pela página de detalhes.
- **Página de detalhes:** o mesmo componente desenha as barras lá (sem cabeçalho), então o tooltip e a altura menor valem também naquela tela. É desejável, mas precisa ser conferido no roteiro.
- **Colunas de largura fixa (`w-16`)** cortam valores muito longos. `formatUptime` usa o `Intl` do navegador, com até duas casas decimais, então os valores esperados (`100%`, `99,95%`, `—`) cabem nos 64 px; um locale com separador longo pode apertar, e a conferência manual olha isso.
- **Nomes longos de endpoint** continuam com `truncate`, e agora dividem a linha com colunas de largura fixa, o que reduz um pouco o espaço do nome em telas médias.
