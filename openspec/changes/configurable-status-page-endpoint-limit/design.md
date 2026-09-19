## Context

`pageconfig.MaximumEndpoints = 200` (`internal/config/statuspage/statuspage.go`) é usado em dois lugares:

- **validação:** `normalizeList(p.Endpoints, MaximumEndpoints, ...)` recusa uma definição com mais de 200 chaves escolhidas uma a uma. Essa validação roda para as páginas do arquivo, para as enviadas pela administração, para as **gravadas** — `newManagedState` chama `Parse`, que valida, e uma página inválida fica com erro e sai do ar — e para as restauradas de um backup;
- **seleção:** `statuspage.Select` (`selection.go`) mantém no máximo 200 endpoints, os destaques primeiro e depois as seções na ordem de exibição, e marca `Truncated`.

`Select` tem quatro chamadores: a montagem do payload (`public.go`, que também registra o aviso no log), `findShownEndpoint` — de onde saem `IsEndpointShown` e `IsEndpointShownOf`, que autorizam a página de detalhes, o gráfico (`response_time_chart.go`), o stream de eventos (`live_updates.go`) e os badges de uma página —, e duas contagens da administração em `service.go` (`Validation.Endpoints` e `Item.Endpoints`).

O registro das páginas é um snapshot imutável trocado a cada carga (`registry.go`), que já guarda um valor tirado da configuração (`maximumResults`). A chave do cache do payload é `slug|revisão|geração`, e a geração sobe a cada `Load`.

A página pública mostra "Showing the first 200 services" com `truncated: true`; o texto está fixo em `StatusPage.vue`, em `status-page-web-ui` e em `docs/status-pages.md`. `adminbackup.MaximumEndpoints = 1000` é outra coisa: o número de endpoints **de um backup**.

## Goals / Non-Goals

**Goals:** tornar o limite de exibição configurável sem enfraquecer a autorização por endpoint, sem tirar do ar páginas já gravadas, e sem mudar nada para quem não configurar.

**Non-Goals:** limite por página; paginar o payload ou carregar grupos sob demanda (é a resposta certa acima de 1.000, e outra mudança); mudar os limites de 50 grupos e 10 destaques; mudar o limitador por IP.

## Decisions

### D1 — Dois números

- `MaximumEndpointKeys = 1000`, constante: o teto de chaves que uma definição pode listar. É estrutural — limita o tamanho de uma definição — e não depende da configuração, porque `Parse` valida páginas gravadas sem ter a configuração à mão, e porque uma validação que dependesse de um valor ajustável voltaria a tirar páginas do ar quando ele baixasse.
- `maximum-endpoints-per-page`, de 1 a 1000, padrão **400** (decisão do dono; era 200): quantos endpoints uma página mostra. Com 4,1 KB por endpoint, o padrão novo leva uma página cheia a cerca de 1,6 MB, 260 KB com gzip. O máximo do intervalo é igual ao teto para que o campo `endpoints` de uma definição válida possa ser exibido inteiro por alguma configuração. Isso **não** vale para a página toda: os grupos selecionam um inventário sem limite, e `featured` é uma lista à parte (mil chaves mais dez destaques distintos já passam de 1000). O limite de 50 é de grupos escolhidos explicitamente, não das seções que resultam.

A constante do padrão passa a se chamar `DefaultMaximumEndpointsPerPage` (400), para não ser confundida com o teto.

### D2 — O limite viaja com a captura da página

`Lookup` devolve um `Published`: a definição da página, a revisão, a geração e `MaximumResults`, tirados **de uma só leitura** do snapshot. O limite entra aí (`Published.MaximumEndpoints`), e a regra passa a ser: todo caminho que decide o que uma página mostra trabalha sobre **uma** captura, e chama `Select` com o limite dela.

Hoje isso não é verdade, e a change precisa consertar antes de tornar o número ajustável:

- `findShownEndpoint` recebe só a definição (`*pageconfig.Page`), não a captura; passa a receber o `Published`;
- **o gráfico resolve a página duas vezes**: o manipulador confere `IsEndpointShownOf` sobre o `Published` já autenticado, e `PublicResponseTimeChart` chama `IsEndpointShown(slug, key)` e outro `Lookup(slug)`. Entre as duas resoluções a página, a credencial ou o limite podem ter mudado. A captura autenticada passa a ser levada até a montagem e o cache do gráfico, sem nova resolução;
- as contagens da administração (`Validation.Endpoints`, `Item.Endpoints`) e a pré-visualização leem o estado e a configuração em momentos separados (`service.go`); passam a ler o limite junto com o estado, do mesmo snapshot;
- o snapshot é recriado copiando os campos um a um nas republicações (`cloneCurrentSnapshot`, em `service.go`) e tem valores iniciais fora de `Load`: o campo novo entra nos três lugares, senão um criar, editar ou renomear página voltaria o limite ao padrão.

O que **não** muda: o inventário de endpoints (`Endpoints()`) continua sendo lido na hora da seleção — o snapshot mais `managedendpoint.List()` —, como hoje. Essa leitura já é separada da captura da página e a change não a piora; capturá-la junto é outra mudança. O cache não precisa do limite na chave: mudar o limite é uma recarga, que sobe a geração.

### D3 — O corte continua sendo de acesso, rota por rota

Um endpoint além do corte não aparece na página e deixa de ser acessível pelas rotas por endpoint dela, exatamente como hoje além dos 200. As rotas não respondem todas do mesmo jeito, e a spec diz cada uma:

- **APIs por endpoint** (detalhes, gráfico, stream de eventos) e **badges**: 404, **depois** do que vem antes delas — numa página com login próprio, 401 sem a credencial, e 429 quando o limitador barra —, como para um endpoint que a página não seleciona. Os badges de uma página já passam por `IsEndpointShownOf`;
- **HTML** (`/status/<slug>/endpoints/<chave>`): continua respondendo 200 com a SPA, sem olhar a chave, como manda o requisito vigente de acesso público; é a tela que mostra "não encontrado" quando a API responde 404;
- os detalhes conferem a seleção **antes** do cache, então um payload de detalhes guardado não é servido para um endpoint que saiu do corte.

Consequências que a documentação e as notas dizem: **subir o limite publica mais endpoints**, inclusive nessas rotas; **baixar** faz endpoints visíveis passarem a 404 na próxima recarga, e a recarga completa já encerra os streams abertos, cuja reconexão é então recusada.

### D4 — Baixar o limite não invalida nada

Como a validação usa o teto fixo (D1), uma página com 300 chaves é válida com o limite em 200: ela é publicada, truncada, com `truncated: true`. A administração mostra o fato: `Item` ganha `truncated` (a página seleciona mais endpoints do que mostra), exibido na listagem, e a validação do formulário ganha um aviso de tipo próprio. O restore de um backup passa pela mesma validação estrutural, de modo que o plano e a aplicação concordam; o aviso novo não é traduzido como "não seleciona nada", que é o que o restore faz hoje com qualquer aviso.

### D5 — O aviso usa o número do payload

"Showing the first N services", com N igual a `summary.total`, que já conta exatamente os endpoints publicados de uma página truncada. Nenhum campo novo no payload público, e a lista de campos permitidos não muda.

### D6 — O teto de 1.000

4,1 KB por endpoint com 50 resultados (0,65 KB com gzip), medido no servidor de validação: 1.000 endpoints são cerca de 4,1 MB por payload, 650 KB comprimidos, e 50 mil barras no DOM com tudo expandido — por isso `groups-collapsed` é a recomendação para páginas grandes. A medição vira uma *fixture* reproduzível (tarefa 4.3); se 1.000 endpoints expandidos travarem o navegador de referência, o teto cai para o maior valor que passar, antes da release.

## Risks / Trade-offs

- **Autorização divergente durante a recarga** → D2.
- **Mais dados públicos por engano ao subir o limite** → D3, dito na documentação e nas notas.
- **Custo de banda, de storage e de memória** → o cache de 30 s e o `singleflight` reduzem as montagens, mas não são uma garantia de "uma por página a cada 30 s": o cache de payloads guarda até 1000 entradas **sem teto de memória**, expulsa por quantidade, e os detalhes criam uma variante por sequência de resultados. Com payloads de 4 MB, 1000 entradas seriam gigabytes. A tarefa 4.3 mede memória e concorrência com várias páginas e detalhes, não só uma página, e a change só sai com uma proteção definida a partir da medição — um orçamento em bytes para o cache, um teto menor, ou os dois. A banda é de quem configura; o limitador por IP não muda. **Medido na implementação** (`TestMeasureEndpointLimit`): ~3,9 KiB por endpoint, 3,9 MiB e ~20 ms por página de 1.000 endpoints. **Proteção adotada:** orçamento de 128 MiB no cache dos payloads, com expulsão LRU (`maximumPublicCacheMemory`, ao lado das 1000 entradas); acima dele o payload menos usado é remontado quando pedido, o que custa milissegundos. O teto de 1.000 fica.
- **Definições maiores aceitas** (até 1000 chaves) → o tamanho de uma definição continua limitado pelo corpo máximo da administração (256 KiB) e pelo comprimento máximo de cada chave.

## Migration Plan

Opção nova e opcional, com o valor de hoje como padrão. Voltar de versão: a versão anterior ignora a opção (o YAML é lido de forma tolerante) e volta a mostrar 200. Mas ela **recusa qualquer definição com mais de 200 chaves**: uma página gerenciada assim fica inválida e sai do ar, e uma página **do arquivo de configuração** assim torna a configuração inválida — o Gatus da versão anterior não inicia. Antes de voltar, reduzir as listas a 200 chaves no YAML e no banco. As notas da release dizem isso.

## Open Questions

- ~~O teto de 1.000 depende da medição da tarefa 4.3.~~ Resolvida: a medição sustenta o teto, com o orçamento em bytes do cache e a recomendação de `groups-collapsed` para páginas com muitas centenas de endpoints.
