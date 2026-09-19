## Context

A página pública (`web/app/src/views/public/StatusPage.vue`) lista os destaques e, abaixo, uma seção por grupo, com uma linha por endpoint e até 50 barras por linha. Ela **não tem tempo real**: busca o payload a cada 60 s (`REFRESH_INTERVAL_MS`) e quando a aba volta a ficar visível, e o servidor guarda o payload em cache por 30 s (`publicCacheTTL`, chave `slug|revisão|geração`). O SSE existe só na página de detalhes de um endpoint, um stream por endpoint, com `data: {}` e teto de 10 streams por IP — não serve para atualizar a listagem.

O payload vem de `internal/statuspage/payload.go`. `SummaryPayload` tem cinco campos (`total`, `up`, `down`, `pending`, `unknown`). O estado agregado de um grupo (`aggregateStatus`) é `operational`, `degraded`, `down` ou `unknown`, este último quando nenhum endpoint do grupo tem dados; endpoints sem dados são ignorados quando há outros com dados. Os destaques não são listados nos grupos. Os testes do payload decodificam rejeitando campos desconhecidos, contra uma lista de campos permitidos que a spec fixa.

As definições das páginas gerenciadas são decodificadas de forma estrita (`KnownFields(true)`, `internal/statuspage/definition.go`); o arquivo de configuração, de forma tolerante (`yaml.Unmarshal`).

## Goals / Non-Goals

**Goals:** recolher grupos na página pública sem nunca esconder um problema; deixar quem publica escolher o estado inicial; lembrar a escolha do visitante sem guardar nada legível no navegador.

**Non-Goals:** tempo real na listagem; recolher os destaques; mexer no dashboard interno; **o limite de endpoints por página** (D6); paginar o payload.

## Decisions

### D1 — Recolhido não renderiza, e o cabeçalho é um botão de verdade

`v-if`, não `v-show`: as linhas de um grupo recolhido não existem no DOM. O cabeçalho é um `<button>` com `aria-expanded` e `aria-controls`. O dashboard interno **não** é o modelo: ele guarda os grupos numa chave global (`gatus:uncollapsed-groups`), começa com tudo fechado e usa `<div @click>` sem atributos de acessibilidade. A página pública é lida por gente de fora, com leitor de tela e teclado, e a spec exige o botão.

### D2 — Precedência contínua, reavaliada a cada payload

A pergunta que importa numa página de status é "tem algo errado?", e um grupo fechado responde "não" sem dizer nada. A regra, na spec, é uma ordem fixa aplicada a cada payload: não operacional → aberto; senão, a escolha lembrada; senão, o padrão da página. Consequências que a revisão cobrou e que ficam explícitas:

- **"Reabrir" acontece no próximo payload**, não em tempo real: até 60 s do *polling* mais até 30 s do cache. É o mesmo atraso com que a página já mostra qualquer mudança; recolher não o piora, porque o cabeçalho recolhido exibe o estado e a contagem do mesmo payload.
- **Recolher durante um incidente é permitido, mas não pega:** vale até o próximo payload e não é gravado. Proibir o clique seria pior numa página com um grupo de 100 linhas fora do ar.
- **A escolha lembrada sobrevive ao incidente:** expandir à força não a apaga, e ela volta a valer quando o grupo se recupera.
- **Um grupo sem dados (`unknown`) fica aberto**, porque não é operacional, e fecha quando os primeiros resultados chegam. É o comportamento certo para um grupo recém-criado: quem publica vê que ainda não há dados.
- **Escolha da visita e escolha lembrada são duas coisas.** A da visita vive em memória e vale até fechar a página; lembrar entre visitas é o que depende do navegador (D3). Sem essa distinção, num navegador sem armazenamento um grupo voltaria ao padrão a cada *polling*.
- **Destaque fora do ar não abre o grupo dele**, porque o destaque não é listado no grupo: ele já está no topo, sempre visível.

### D3 — Lembrar sem gravar nomes

Uma página pode ter login próprio, e a resposta autenticada sai com `private, no-store` justamente para não deixar rastro. Gravar `{"Clientes VIP": "collapsed"}` no `localStorage` de um navegador compartilhado deixaria. Em vez de distinguir páginas com e sem login — o frontend não recebe esse sinal, e criá-lo aumentaria o payload —, a regra é uma só: a chave de cada escolha é `SHA-256(slug + "\n" + nome bruto do grupo)`, em hexadecimal, dentro de um único item `gatus:status-page-groups`, com o valor `c` ou `e`. Nada legível é gravado, para página nenhuma.

- O nome é o **bruto do payload**; o grupo sem nome é `""`. O rótulo "Other services" e a chave de renderização `__without-group__` são da tela e colidiriam com um grupo de verdade.
- O item é lido com validação de forma (objeto simples, chaves de 64 hexadecimais, valores `c`/`e`); qualquer outra coisa é descartada. Como as chaves são hashes, `__proto__` e afins não têm como aparecer.
- `crypto.subtle` só existe em contexto seguro (HTTPS ou localhost). Numa página servida por HTTP puro, ou com o armazenamento bloqueado, a página funciona sem lembrar.
- Só os grupos em que o visitante mexeu são gravados, e só escolhas sobre grupos operacionais (D2). O item é limitado a 500 entradas, descartando as mais antigas.
- **`crypto.subtle.digest` é assíncrono.** `StatusPage.vue` hoje publica o JSON assim que ele chega; aplicar as escolhas depois faria os grupos piscarem. As chaves de todos os grupos do payload são derivadas, e as escolhas lidas, **antes** de o payload ir para o estado reativo; depois de cada `await` o código confere que o slug ainda é o mesmo (a página zera o estado ao trocar de slug) e que o componente não foi desmontado, e descarta o resultado caso contrário.
- O descarte ao passar de 500 entradas é pela ordem de inserção do objeto JSON, as mais antigas primeiro; mexer de novo num grupo o reinsere no fim.
- As chaves de renderização da lista (`:key`) passam a ser o nome bruto do grupo, na página pública e na pré-visualização do formulário (`AdminStatusPageForm.vue`): o `__without-group__` de hoje colide com um grupo que tenha esse nome.
- A pré-visualização da administração não lê nem grava: ela mostra o padrão da página.

### D4 — `groups-collapsed` na página

O estado inicial é decisão de quem publica cada página. O campo vai em `pageconfig.Page`, passa pela validação e pela normalização das páginas gerenciadas e sai no payload como `groupsCollapsed`. O backup guarda a definição como YAML opaco e o carrega sem mudança de formato.

### D5 — Voltar de versão

Uma versão anterior **invalida** (não "põe em conflito": conflito é slug ocupado pelo YAML) uma página gerenciada gravada com `groups-collapsed`, por causa da decodificação estrita: a página sai do ar com erro na administração. Antes de voltar de versão é preciso tirar o campo das páginas gerenciadas, e um backup feito na versão nova não restaura essas páginas numa anterior. No arquivo de configuração é o contrário: a versão anterior ignora o campo em silêncio (conferido com `gatus config validate` da v6.0.1). As notas da release devem dizer as duas coisas.

### D6 — O limite de endpoints fica para outra change

A primeira versão desta proposta tornava o 200 configurável. Reprovada pelas duas revisões, com razão. O que elas mostraram, para quem for escrever a próxima:

- o corte está em `statuspage.Select` (`selection.go`), e `findShownEndpoint` reutiliza a seleção: o limite decide também o que responde 404 nos detalhes, gráficos, badges e streams de um endpoint. Mudar o número muda autorização;
- a carga das páginas gravadas chama `Parse` → `ValidateAndSetDefaults`, que usa o mesmo número: baixar o limite **invalidaria e despublicaria** páginas gravadas. É preciso separar um teto estrutural fixo (validação) do limite de exibição em vigor;
- o limite teria de ser capturado no snapshot publicado, para valer junto com a geração do cache;
- contagens da administração (`Item.Endpoints`, `Validation.Endpoints`), o texto fixo "Showing the first 200 services" e o aviso que hoje não chega à listagem (`Warning` só existe em `Validation`) entram no escopo;
- os tamanhos medidos (4,1 KB por endpoint, 0,65 KB com gzip, no servidor de validação) precisam de uma *fixture* reproduzível, e o limitador por IP protege as respostas 404, não os downloads válidos.

## Risks / Trade-offs

- **Problema escondido** → D2, e o cabeçalho recolhido sempre mostra estado e contagem.
- **Spec do payload contraditória** → o requisito vigente proibia qualquer data de expiração enquanto `certificate-expiration` exige `certificateExpiresAt`, que o código publica. Como o requisito é modificado aqui, a contradição é desfeita no mesmo lugar, sem mudar o comportamento.
- **Contrato da API** → campos novos quebram clientes estritos; está na proposta e vai nas notas. A lista de campos permitidos da spec e dos testes é atualizada junto.
- **Vazamento** → `groupsCollapsed` é um booleano da definição e o `summary` do grupo conta só endpoints já publicados naquele grupo: nada de chave, URL ou erro.
- **Rastro no navegador** → D3.
- **Desempenho** → com o limite atual, o pior caso é o de hoje (200 linhas, tudo expandido); recolher só melhora.

## Migration Plan

Campo opcional, padrão igual ao comportamento atual: nada a migrar na ida. A volta está em D5.

## Open Questions

- **`groups-collapsed: true` como padrão para páginas novas criadas na administração?** A proposta diz não.
- **Abrir a proposta do limite configurável (D6)?** Depende de o dono precisar de mais de 200 endpoints numa página.
