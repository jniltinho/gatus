## Why

Uma página de status com muitos endpoints vira uma rolagem longa: quem abre para saber "está tudo bem com o grupo X?" precisa passar por dezenas de linhas de barras. O dashboard interno já deixa recolher os grupos; a página pública não. O dono também perguntou por que uma página só mostra 200 endpoints: o número protege uma rota sem login (carga no storage, tamanho do payload e do DOM), mas é uma constante no código, e uma instalação com um inventário grande não tem como ajustá-la.

As duas coisas se ligam: um grupo recolhido que não desenha suas linhas é o que torna viável mostrar mais endpoints sem travar o navegador de quem visita.

## What Changes

1. **Grupos recolhíveis na página pública.** O cabeçalho de cada grupo passa a ser um botão que recolhe e expande as linhas do grupo. Recolhido, o cabeçalho continua mostrando o estado agregado (`Operational`, `Partial outage`...) e ganha a contagem `N up · N down` do grupo, para que nada importante fique escondido. Os destaques (`Featured`) não são recolhíveis.
2. **Estado inicial definido por página.** Campo novo e opcional na definição da página, `groups-collapsed` (padrão `false`, o comportamento de hoje). Com `true`, os grupos começam recolhidos, **exceto os que não estão operacionais**, que começam abertos: um problema nunca nasce escondido. A escolha de quem visita é lembrada no navegador, por página e por grupo, e vale mais que o padrão da página, com a mesma exceção: um grupo que passa a ter problema é aberto de novo.
3. **Limite de endpoints por página configurável.** `status-pages.maximum-endpoints-per-page`, global, padrão `200` (o de hoje), de `1` a `1000`. A validação de uma página, a montagem do payload (`truncated`) e o aviso do log passam a usar esse valor. O limite de 10 destaques e o de 50 grupos não mudam.
4. **Formulário da administração** com a opção "Start with the groups collapsed", e a pré-visualização respeitando-a.

Sem mudança de contrato para quem já usa: sem `groups-collapsed` e sem `maximum-endpoints-per-page`, a página e o payload ficam como estão, mais o campo `groupsCollapsed: false` e as contagens por grupo no payload.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `public-status-pages`: grupos recolhíveis com estado inicial por página, contagem por grupo no payload, e limite de endpoints por página configurável.

## Impact

- **Configuração:** `status-pages.maximum-endpoints-per-page` (global) e `groups-collapsed` (por página, no YAML e nas páginas gerenciadas). O backup/restore da administração carrega o campo novo dentro da definição, sem mudança de formato.
- **API pública:** `GET /api/v1/status-pages/{slug}` ganha `groupsCollapsed` (booleano) e, em cada grupo, `summary` (`total`, `up`, `down`, `pending`, no mesmo formato do `summary` da página). Campos novos; nenhum é removido.
- **Código:** `internal/config/statuspage` (campo, limite, validação), `internal/statuspage` (payload, serviço, opções da administração), `internal/api` (só comentários dos manipuladores), `web/app/src/views/public/StatusPage.vue`, `components/public/` e o formulário `views/admin/`.
- **Desempenho:** com o limite no padrão, nada muda. Com 1.000 endpoints o payload mede cerca de 4,1 MB (650 KB com gzip, medido em 4,1 KB por endpoint com 50 resultados); por isso um grupo recolhido não renderiza suas linhas, e o design trata do que acontece quando tudo é expandido.
- **Documentação:** `docs/status-pages.md`, `docs/README.md` (tabela de `status-pages`), screenshots da página pública e do formulário.
- **Testes:** unidade (validação, payload, contagem por grupo, limite), contrato HTTP (payload novo), E2E `status-pages.sh` (recolher, lembrar, grupo com problema aberto, tema escuro, teclado).
