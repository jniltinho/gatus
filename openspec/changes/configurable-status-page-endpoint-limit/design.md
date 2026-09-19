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
- `maximum-endpoints-per-page`, de 1 a 1000, padrão 200: quantos endpoints uma página mostra. Igual ao teto no máximo, para que toda definição válida possa ser exibida inteira por alguma configuração.

`MaximumEndpoints` continua existindo como o **padrão** do limite de exibição.

### D2 — O limite mora no snapshot

`Load` lê o limite da configuração e o guarda no snapshot, como já faz com `maximumResults`. `Select` passa a receber o limite, e os quatro chamadores o tiram do snapshot que já têm em mãos — nunca da configuração global nem de uma variável de pacote. Assim o payload, a autorização por endpoint e as contagens da administração não podem divergir entre si durante uma recarga, e o cache não precisa do número na chave: mudar o limite é uma recarga, que sobe a geração e invalida o que estava guardado.

### D3 — O corte continua sendo de acesso

Um endpoint além do corte não aparece na página e responde 404 nas rotas por endpoint dela, exatamente como hoje além dos 200. Não é efeito colateral: é o que impede que "não aparece na página" e "dá para consultar mesmo assim" sejam coisas diferentes. Consequências explícitas:

- **subir o limite publica mais endpoints**, inclusive nas rotas por endpoint: a documentação diz isso ao lado da opção;
- **baixar o limite** faz endpoints que eram visíveis passarem a 404 na próxima recarga; um stream de eventos aberto para um deles é encerrado pela recarga, como qualquer outro.

### D4 — Baixar o limite não invalida nada

Como a validação usa o teto fixo (D1), uma página com 300 chaves é válida com o limite em 200: ela é publicada, truncada, com `truncated: true`. A administração mostra o fato: `Item` ganha `truncated` (a página seleciona mais endpoints do que mostra), exibido na listagem, e a validação do formulário ganha um aviso de tipo próprio. O restore de um backup passa pela mesma validação estrutural, de modo que o plano e a aplicação concordam; o aviso novo não é traduzido como "não seleciona nada", que é o que o restore faz hoje com qualquer aviso.

### D5 — O aviso usa o número do payload

"Showing the first N services", com N igual a `summary.total`, que já conta exatamente os endpoints publicados de uma página truncada. Nenhum campo novo no payload público, e a lista de campos permitidos não muda.

### D6 — O teto de 1.000

4,1 KB por endpoint com 50 resultados (0,65 KB com gzip), medido no servidor de validação: 1.000 endpoints são cerca de 4,1 MB por payload, 650 KB comprimidos, e 50 mil barras no DOM com tudo expandido — por isso `groups-collapsed` é a recomendação para páginas grandes. A medição vira uma *fixture* reproduzível (tarefa 4.3); se 1.000 endpoints expandidos travarem o navegador de referência, o teto cai para o maior valor que passar, antes da release.

## Risks / Trade-offs

- **Autorização divergente durante a recarga** → D2.
- **Mais dados públicos por engano ao subir o limite** → D3, dito na documentação e nas notas.
- **Custo de banda e de storage** → cache de 30 s e `singleflight` já limitam o storage a uma montagem por página a cada 30 s; a banda é de quem configura. O limitador por IP não muda.
- **Definições maiores aceitas** (até 1000 chaves) → o tamanho de uma definição continua limitado pelo corpo máximo da administração (256 KiB) e pelo comprimento máximo de cada chave.

## Migration Plan

Opção nova e opcional, com o valor de hoje como padrão. Voltar de versão: a versão anterior ignora a opção (o YAML é lido de forma tolerante) e volta a mostrar 200; uma página **gerenciada** gravada com mais de 200 chaves fica inválida numa versão anterior e sai do ar — reduzir as chaves antes de voltar. As notas da release dizem isso.

## Open Questions

- **O teto de 1.000** depende da medição da tarefa 4.3.
