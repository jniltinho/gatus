## Context

**Telas de detalhes.** As duas montam os mesmos blocos, na mesma ordem, com `Card` e `CardHeader` do mesmo conjunto, e já compartilham `DetailsSummary`, `ResponseTimeChart`, `RecentChecksTable` e `EventsTimeline`. O que difere:

| | `views/EndpointDetails.vue` (dashboard) | `views/public/StatusPageEndpoint.vue` (pública) |
|---|---|---|
| Voltar | `Button` fantasma "Back to Dashboard" | `RouterLink` "Back to \<título da página\>" |
| Título | `text-2xl font-semibold tracking-tight` | `text-4xl font-bold tracking-tight break-words` |
| Linha abaixo | `Group: <grupo> • <host>` | `Group: <grupo> · Updated …` — o grupo **já está lá**, o que falta é o host |
| Certificado | dias **e** data | só os dias, porque o payload público publica `certificateExpiresInDays` e nada mais |
| Estado | `StatusBadge` à direita | `StatusBadge` à direita |
| Cartão do histórico | `EndpointCard` completo: repete nome, grupo e host, mais os botões de atualizar e de média/mínimo-máximo no cabeçalho | `EndpointRow` com `show-header` falso: só as barras |
| Barras | `h-6 sm:h-8` com `gap-0.5` | `h-5` com `gap-px` |
| Tooltip da barra | `Tooltip.vue` no nível do app, posicionado por `top`/`left`, com horário, tempo de resposta, **condições** e erros, e seleção por clique | tooltip do próprio `EndpointRow`, com horário, resultado e duração |
| Tabela de verificações | `RecentChecksTable` + `Pagination` | `RecentChecksTable` com `sanitized` e `show-message` conforme a página |
| Badges | quatro períodos, rótulos "Last 30 days", "Last 7 days", "Last 24 hours", "Last hour" | os mesmos quatro, com os mesmos rótulos |
| Eventos | `endpoint-events` | `status-endpoint-events` |

**Lista da administração.** `views/admin/AdminEndpoints.vue` tem oito colunas: Name, Group, Type, URL, Interval, Status, Source e Actions. Todas são de conteúdo, com `whitespace-nowrap` em Name, Group, Type e Actions, e a URL limitada por `max-w-xs` (320 px). O contêiner que rola é o `admin-list-scroll` do `AdminListLayout`, com `overflow-x-auto` abaixo de `md` e `overflow-auto` a partir dali.

Medições na instalação de validação (9 endpoints, URLs curtas), com a diferença entre `scrollWidth` e `clientWidth` do `admin-list-scroll`:

| Largura da janela | 1280 | 1150 | 1024 | 900 | 800 | 768 | 700 | 640 |
|---|---|---|---|---|---|---|---|---|
| Transbordo | 0 | 0 | 0 | 0 | 5 px | 37 px | 105 px | 165 px |

Larguras das colunas em 800 px: Name 114, Group 65, Type 61, **URL 226**, Interval 74, Status 77, Source 71, Actions 81.

`AdminStatusPages.vue` (Slug, Title, Source, Status, Endpoints, Actions) e `AdminPushKeys.vue` (Name, Key, Source, Created, Actions) têm a mesma estrutura, com uma coluna larga cada (`Title` e `Key`).

## Goals / Non-Goals

**Goals:**

- Quem conhece uma das telas de detalhes reconhece a outra: mesmo cabeçalho, mesmo histórico, mesma ordem.
- Nenhuma lista da administração com rolagem horizontal, em nenhuma largura a partir de 360 px.
- Manter o que é legitimamente diferente entre as duas telas, e dizer o que é.

**Non-Goals:**

- Unificar os dois componentes de barras num só: eles consomem payloads diferentes (o público é sanitizado) e a fusão traria mais risco que ganho.
- Mudar o que a API devolve, a paginação do dashboard ou o que a página pública esconde.
- Mexer no dashboard em si (a grade de cartões), no gráfico ou nos badges.

## Decisions

### D1 — Cabeçalho comum das telas de detalhes

Os dois cabeçalhos passam a ter a mesma forma, dentro do que o payload de cada tela permite:

```
[voltar]
<h1 class="text-2xl font-semibold tracking-tight break-words">nome</h1>   [StatusBadge] [Updated … (só na pública)]
<p class="mt-1 text-sm text-muted-foreground">Group: <grupo>[ · <host>, só no dashboard]</p>
<p class="mt-1 text-xs …">Certificate expires in N days[ · <data>, só no dashboard]</p>
```

- o título vai para `text-2xl font-semibold tracking-tight break-words` nas duas: a pública encolhe de `text-4xl` e o dashboard ganha o `break-words`;
- o grupo aparece nas duas — a pública **já mostra**, então o que muda é só a ordem visual, com o grupo à esquerda e o `Updated …` à direita;
- o certificado mostra os dias nas duas, com as mesmas cores por faixa;
- o `StatusBadge` fica à direita do título nas duas;
- o controle de voltar continua com textos diferentes (Dashboard × título da status page) e passa a ter o mesmo formato: ícone de seta e texto, no mesmo lugar.

**Host: continua só no dashboard.** O payload público não carrega chave, URL, hostname nem endereço — está escrito em `statuspage/payload.go`, a spec `status-page-highlights` exige a rota pública "sem chave, URL, hostname, erros ou condições" e `test/e2e/status-pages.sh` reprova o roteiro se o JSON público contiver a chave ou um endereço. Fica como diferença prevista.

**Data do certificado: passa a ser publicada.** A pedido do dono, a página pública de detalhes mostra a data de vencimento ao lado dos dias, como o dashboard. Isso exige um campo novo no payload público, `certificateExpiresAt`, e por isso esta change passa a tocar no backend. É seguro: a data não revela endereço, host nem detalhe do certificado, e já era deduzível dos dias que o payload publica desde a fork.6 — quem lê "vence em 39 dias" sabe a data com um dia de erro. O campo:

- é calculado do **mesmo** resultado de onde saem os dias (o mais recente com certificado entre os resultados publicados), como `timestamp + certificateExpiration`, em UTC e no formato RFC 3339;
- só existe quando a página liga `show-certificate-expiration`, junto do campo dos dias, e é omitido quando não houver resultado com certificado;
- vale para `groups[].endpoints[]`, `featured[]` e a API pública de detalhes, como os dias.

**Onde a data aparece:** só na página pública **de detalhes**, ao lado dos dias. As linhas das listas públicas continuam com os dias apenas: ali a linha do certificado divide espaço com nome, estado e colunas de uptime, e o teto de 72 px por linha da spec `status-page-web-ui` não comporta um texto que quebra em telas estreitas.

### D2 — Cartão do histórico igual nas duas

- **Prop `compact` no `EndpointCard`, opcional e `default: false`.** A grade do dashboard (`Home.vue`) não passa a prop e **não muda**. Com `compact`, e só com ela: o cartão não repete nome, grupo, host e badge de estado (já estão no cabeçalho da página), as barras vão para 20 px e o realce `hover:scale-[1.01]` é desligado (hoje o cartão inteiro pula na tela de detalhes). O tooltip continua sendo o do `App.vue`, emitido como hoje: ele já não ocupa espaço no layout, e é dele que vêm as condições e os erros da verificação e a seleção por clique, que a spec lista como diferença prevista. Desenhar um tooltip local duplicaria essa lógica e arriscaria dois tooltips ao mesmo tempo.
- **Mesma altura e mesmas cores:** 20 px nas duas telas de detalhes, com as cores de estado que já existem. O número de barras continua diferente (o dashboard mostra a página de resultados, a pública mostra 50 ou 25 conforme a largura) e isso não muda.
- **Tooltip:** o requisito é que ele não ocupe espaço no layout nas duas, o que já é verdade. O conteúdo continua diferente de propósito: o do dashboard mostra também as condições e os erros da verificação, que o payload público não tem, e mantém a seleção por clique.
- **Rótulos de tempo nas pontas** ("X minutes ago" e "Y seconds ago") passam a existir nas duas telas de detalhes. No `EndpointRow` eles ficam presos ao modo sem cabeçalho (`show-header` falso): ligá-los na lista da status page estouraria o teto de 72 px por linha que a spec `status-page-web-ui` fixa.
- **Ações:** os botões de atualizar e de alternar média/mínimo-máximo continuam **só** no dashboard. A página pública recebe atualizações em tempo real e não tem escolha de média.

### D3 — Ordem e blocos, escritos no requisito

A ordem — barras, painel de números, gráfico, tabela de verificações, badges, saúde atual e eventos — **já é requisito** em `status-page-highlights` ("a página pública de detalhes MUST seguir o layout da página do dashboard, nesta ordem"), e o painel de números entre as barras e o gráfico já é requisito do próprio `endpoint-details-summary`. O delta desta change **não repete** isso: referencia e cuida só do que é novo — cabeçalho comum, ausência de repetição visual no cartão do histórico, rótulos de tempo e a lista de diferenças previstas.

Para o teste conseguir comparar as duas telas, os blocos que hoje não têm identificador ganham um: `details-badges` e `details-health` nas duas telas.

Diferenças que **continuam**, registradas na spec: host e data do certificado só no dashboard; paginação na tabela do dashboard; botões de atualizar e de média/mínimo-máximo só no dashboard; `sanitized`, mensagens condicionais, `Updated …` e link de volta para a status page só na pública; número de barras.

### D4 — Listas da administração sem rolagem horizontal

Elasticizar só a coluna larga **não resolve**: as colunas `whitespace-nowrap` (Name, Group, Type e Actions nos endpoints; Slug nas status pages) e o selo de conflito dentro da célula do nome definem uma largura mínima que cresce com os dados. Um nome longo empurra a tabela do mesmo jeito. Então:

1. **`table-layout: fixed`** nas três tabelas, com as colunas que podem crescer marcadas com `truncate` e o valor completo no `title`: nos endpoints, `Name` e `URL`; nas status pages, `Slug` e `Title`; nas chaves de push, `Name` (a coluna `Key` mostra só as últimas letras e já é curta). Com layout fixo, a largura da tabela deixa de depender do conteúdo.
2. **Selo de conflito** sai de dentro da célula do nome e vira uma segunda linha da mesma célula, para não somar largura.
3. **`Actions`** continua `whitespace-nowrap`, com largura de conteúdo, porque é o que precisa estar sempre alcançável.
4. **Colunas por largura:** de `lg` para cima, todas as colunas; entre `md` e `lg`, sem `Interval` e sem `Source` (nos endpoints) e sem as equivalentes nas outras listas; abaixo de `md`, a tabela dá lugar a um cartão por item, **com todos os campos**, inclusive o intervalo, e com as mesmas ações.
5. **Identificadores:** os cartões reusam os mesmos `data-testid` das ações de hoje (`admin-open-*`, `admin-toggle-*`, `admin-remove-*`, `status-page-edit-*`, `push-key-revoke-*`), porque quatro roteiros E2E dependem deles. Cada lista mantém o identificador da própria tabela (`admin-table`, `status-pages-table`, `push-keys-table`) e ganha o do cartão (`admin-card-*`, `status-page-card-*`, `push-key-card-*`).

O `overflow-x-auto` do `AdminListLayout` continua onde está, como rede de segurança para conteúdo inesperado.

### D5 — Testes

Cada lista é medida no roteiro que tem dado de verdade para ela. O `push.sh` nunca abre `/admin/status-pages` e a configuração dele não define nenhuma página; o `admin.sh` não toca em status pages. Quem tem a lista de endpoints **e** uma status page criada pela web é o `status-pages.sh`, então as duas primeiras listas são medidas lá, e a lista de chaves de push continua no `push.sh`, que é onde a chave `akamai` é criada e revogada.

**Larguras e transbordo**, para cada lista, em 1100×800, 900×700, 700×800, 390×844 e 360×800 — larguras longe dos pontos de quebra (`lg` = 1024, `md` = 768), porque com a barra de rolagem visível a largura da media query fica alguns pixels abaixo do viewport pedido:

1. `scrollWidth - clientWidth` do `admin-list-scroll` MUST ser 0 **e** `document.documentElement.scrollWidth <= innerWidth + 1`;
2. a última ação de uma linha **gerenciada pela web** (a que tem todas as ações: `admin-row-jobs_backup`, uma status page com origem Web, `push-key-row-admin-akamai`) MUST estar dentro da janela. Medir "a primeira linha" não serve: a lista pode começar por um item do YAML, que tem menos ações;
3. abaixo de `md`, a tabela da lista em questão MUST ter sumido (`admin-table`, `status-pages-table` ou `push-keys-table`, conforme a aba — usar sempre `admin-table` daria verde nas outras duas sem nada ter mudado) e os cartões MUST estar presentes, com as ações pelos mesmos identificadores;
4. o item medido MUST ter nome **e** URL longos: o roteiro passa a criar um endpoint com nome de uns 60 caracteres e URL de uns 120, que é o caso que o requisito promete cobrir;
5. o truncamento MUST ser conferido: a célula da URL com `scrollWidth > clientWidth` e o `title` com o endereço inteiro;
6. ao fim do bloco, restaurar `1280×900`, senão os passos seguintes do roteiro rodam em modo cartão e não acham os identificadores de linha.

**Comparação das duas telas de detalhes**, em `status-pages.sh`, no mesmo endpoint (`_panel`, que está na página `services` e existe em `/endpoints/_panel`), com a mesma janela nas duas sessões:

7. `font-size` computado do `h1` igual nas duas;
8. altura de uma barra do histórico igual nas duas e igual a 20 px;
9. grupo presente nas duas; host e data de expiração **ausentes** na pública, presentes no dashboard;
10. o cartão do histórico MUST NOT mostrar o nome do endpoint em texto **visível** — a comparação ignora `sr-only` e `aria-*`, porque o resumo acessível da página pública é obrigatório por spec e contém o nome;
11. a ordem dos blocos, comparada por um mapa explícito de identificadores (histórico, `details-summary`, gráfico, tabela, `details-badges`, `details-health`, eventos), ordenados por `getBoundingClientRect().top`, e não pela string dos identificadores, que são diferentes nas duas telas.

Prints das duas telas nos dois temas para a conferência manual. Não há teste de unidade envolvido: o projeto só tem testes de unidade dos utilitários (`node --test src/utils/*.test.mjs`), então a verificação desta change é toda de ponta a ponta.

## Risks / Trade-offs

- **`table-layout: fixed`** muda a distribuição das colunas: larguras deixam de seguir o conteúdo, então as proporções precisam ser conferidas na tela em cada largura, e uma coluna nova exige rever o conjunto.
- **Esconder Interval e Source entre `md` e `lg`** tira informação da tela em janelas médias. São os dois dados menos usados para achar um endpoint, e continuam no cartão do celular e no formulário.
- **Cartões abaixo de `md`** criam um segundo layout para manter; é o preço de não ter rolagem horizontal num celular.
- **Encolher o título da página pública** de `text-4xl` para `text-2xl` deixa a tela menos imponente; em troca, as duas telas ficam iguais, que é o pedido.
- **Host e data do certificado continuam só no dashboard.** As telas ficam parecidas, não idênticas, e o motivo é de produto: a página pública não publica endereço nem detalhe do certificado.
- **Os cartões abaixo de `md` precisam carregar os mesmos identificadores de ação das linhas**, senão quatro roteiros E2E quebram na primeira largura estreita.
- **A tela do dashboard perde a repetição do nome dentro do cartão**, que hoje serve de âncora quando se rola a página. O cabeçalho continua visível no topo e o nome está no título do documento.
