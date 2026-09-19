## Context

A página pública (`web/app/src/views/public/StatusPage.vue`) lista os destaques e, abaixo, uma seção por grupo, cada uma com uma linha por endpoint (`components/public/EndpointRow.vue`) e 50 barras por linha. O payload vem de `internal/statuspage` (`Payload`, `GroupPayload`, `SummaryPayload`), é sanitizado, e é atualizado em tempo real por SSE. O limite de 200 endpoints é a constante `statuspage.MaximumEndpoints` de `internal/config/statuspage`, usada na validação (chaves escolhidas uma a uma) e na montagem (`truncated`).

Medido no servidor de validação, com 50 resultados por endpoint: **4,1 KB por endpoint, 0,65 KB com gzip**. Com 200 endpoints, 820 KB (130 KB comprimido) e 10 mil barras no DOM; com 1.000, 4,1 MB (650 KB) e 50 mil barras.

## Goals / Non-Goals

**Goals:** recolher grupos na página pública sem esconder problemas; deixar o dono escolher o estado inicial por página; tirar o 200 do código sem tirar a proteção.

**Non-Goals:** paginar o payload ou carregar grupos sob demanda (muda o contrato da API e o SSE; fica para quando 1.000 não bastar); recolher os destaques; recolher grupos no dashboard interno (já existe); limite por página individual.

## Decisions

### D1 — Recolhido não renderiza

Um grupo recolhido usa `v-if`, não `v-show`: as linhas e suas barras não existem no DOM. É o que faz o recolhimento ser também a resposta de desempenho para páginas grandes, e é como o dashboard já faz (`uncollapsedGroups`). O custo é redesenhar ao expandir, imperceptível para um grupo.

### D2 — Um problema nunca nasce escondido

O risco de recolher numa página de status é óbvio: o visitante olha, vê tudo fechado e conclui que está tudo bem. Três defesas, todas na spec:

- o cabeçalho recolhido mostra o estado agregado e a contagem `N up · N down` do grupo;
- com `groups-collapsed: true`, só os grupos operacionais começam recolhidos;
- um grupo recolhido **pelo visitante** que deixa de estar operacional é aberto, inclusive por uma atualização em tempo real, e volta a fechar quando se recupera. A escolha não é apagada: ela passa a valer de novo quando o grupo está operacional.

Alternativa rejeitada: respeitar sempre a escolha do visitante. É mais simples e mais previsível, mas troca a função da página (avisar) por uma preferência de layout.

### D3 — Onde fica a escolha do visitante

`localStorage`, numa chave por página: `gatus:status-page:<slug>:groups` com um objeto `{ "<grupo>": "collapsed" | "expanded" }`. É o padrão que `RecentChecksTable` e o dashboard já usam (`gatus:*`). Só são gravados os grupos em que o visitante mexeu, para que uma mudança do padrão da página continue valendo para os demais. Grupos que não existem mais são descartados na leitura. `localStorage` indisponível (modo privado, bloqueio) é tratado com `try/catch`: a página funciona sem lembrar.

Nada disso vai ao servidor: a página continua sem estado por visitante, e o cache de 30 s do payload continua valendo para todos.

### D4 — `groups-collapsed` na página, e não global

O estado inicial é uma decisão de quem publica cada página: uma página de 8 serviços quer tudo aberto, uma de 300 quer fechado. Global seria um segundo lugar para procurar. O campo vai em `pageconfig.Page`, passa pela mesma validação e normalização das páginas gerenciadas, e sai no payload como `groupsCollapsed`. Como o backup guarda a definição inteira, ele carrega o campo sem mudança de formato nem de versão.

### D5 — `summary` por grupo calculado no servidor

O frontend poderia contar os endpoints do grupo, mas a regra do que é `up`, `down` e `pending` já mora no servidor (`SummaryPayload`, que trata `Pending` do push). Repetir a regra em JavaScript criaria duas fontes. `GroupPayload` ganha `Summary`, calculado na mesma passada que calcula o estado agregado do grupo. Os destaques ficam fora da contagem do grupo porque não são listados nele (o `summary` da página continua contando tudo).

### D6 — O limite vira configuração global, com teto

`status-pages.maximum-endpoints-per-page`, padrão 200, de 1 a 1000. Global, e não por página, porque o que ele protege é o servidor (storage, memória, banda), não a página. O teto de 1.000 é o ponto em que o payload passa de 4 MB e o DOM, tudo expandido, de 50 mil barras: acima disso a resposta certa é paginar (Non-Goal), não subir o número.

A constante `MaximumEndpoints` continua existindo como **padrão**; validação e montagem passam a receber o limite em vigor. Uma página gerenciada gravada com um limite maior não é recusada quando o limite cai — isso tiraria do ar uma página por causa de uma mudança noutro lugar —: ela é truncada na exibição, e a listagem da administração ganha o aviso, no mesmo mecanismo dos avisos de seleção (`Warning`).

### D7 — Expandir tudo numa página grande

Com o limite alto e o visitante expandindo todos os grupos, o DOM cresce. Duas medidas baratas: não há botão "expandir tudo" (cada grupo é uma decisão), e as barras de um grupo recém-expandido entram no próximo quadro (`nextTick`), para o clique responder antes do desenho. Se a medição da tarefa 4.3 mostrar travamento com 1.000 endpoints expandidos, o teto da D6 cai para o maior valor que passar.

## Risks / Trade-offs

- **Problema escondido** → D2.
- **Payload grande com o limite alto** → gzip já ativo, cache de 30 s e `singleflight` já existem; o limitador por IP não muda; o teto da D6.
- **Estado do visitante divergente do da página** → D3 grava só o que ele mexeu.
- **Acessibilidade** → botão de verdade (`<button>`), `aria-expanded`, `aria-controls`, foco visível nos dois temas; teste E2E por teclado.
- **Contrato da API** → só campos novos; o contrato HTTP gravado ganha os casos novos.

## Migration Plan

Sem migração de dados: os dois campos são opcionais e os padrões são o comportamento de hoje.

**Voltar de versão exige um passo.** As definições das páginas gerenciadas são decodificadas de forma estrita (`KnownFields(true)` em `internal/statuspage/definition.go`): uma versão anterior **recusa** uma página gravada com `groups-collapsed`, que fica em conflito em vez de publicada. Antes de voltar, é preciso tirar o campo das páginas gerenciadas (ou restaurar um backup anterior), e as notas da release devem dizer isso. No arquivo de configuração é o contrário: o YAML é lido de forma tolerante (`yaml.Unmarshal`), e uma versão anterior **ignora** `groups-collapsed` e `maximum-endpoints-per-page` sem erro (conferido com `gatus config validate` da v6.0.1) — a página volta a abrir expandida e o limite volta a 200, em silêncio. `test/e2e/upgrade.sh` cobre a ida; a volta fica documentada, não automatizada.

## Open Questions

- **Teto de 1.000** (D6): confirmar com a medição da tarefa 4.3.
- **`groups-collapsed: true` deve ser o padrão para páginas novas criadas na administração?** A proposta diz não (padrão `false` em todo lugar), para não mudar o que o dono vê sem ele pedir.
