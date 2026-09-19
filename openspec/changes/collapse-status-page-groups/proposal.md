## Why

Uma página de status com muitos endpoints vira uma rolagem longa: quem abre para saber "está tudo bem com o grupo X?" precisa passar por dezenas de linhas de barras. A página pública não deixa recolher os grupos.

## What Changes

1. **Grupos recolhíveis na página pública.** O cabeçalho de cada grupo passa a ser um botão que recolhe e expande as linhas do grupo. Recolhido, o cabeçalho continua mostrando o estado agregado e ganha a contagem dos endpoints do grupo por estado, para que nada importante fique escondido. Os destaques (`Featured`) não são recolhíveis.
2. **Um grupo com problema é sempre aberto.** A precedência é contínua, reavaliada a cada payload: grupo não operacional → aberto; senão, a escolha lembrada do visitante; senão, o padrão da página.
3. **Estado inicial definido por página.** Campo novo e opcional na definição, `groups-collapsed` (padrão `false`, o comportamento de hoje).
4. **Escolha do visitante lembrada no navegador**, por página e por grupo, sem gravar o nome de nenhum grupo (D3).
5. **Payload público:** `groupsCollapsed` na página e `summary` em cada grupo, com os mesmos cinco campos do `summary` da página.
6. **Formulário da administração** com a opção "Start with the groups collapsed", e a pré-visualização respeitando-a sem tocar nas escolhas guardadas da página pública.

**Mudança de contrato, pequena mas real:** o payload ganha campos. Um cliente que decodifica a resposta rejeitando campos desconhecidos — como os testes do próprio projeto — precisa conhecê-los. Nenhum campo é removido nem muda de sentido.

**Fora desta change: o limite de 200 endpoints por página.** A primeira versão desta proposta o tornava configurável, e a revisão mostrou que isso não é uma mudança de exibição: o corte acontece em `statuspage.Select`, que também decide quais endpoints a página deixa acessar nos detalhes, gráficos, badges e streams, e a validação das páginas gravadas usa o mesmo número. Merece uma proposta própria (D6).

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `public-status-pages`: a lista de campos permitidos no payload público ganha `groupsCollapsed` e o `summary` de cada grupo; a definição de uma página ganha `groups-collapsed`; a página pública ganha grupos recolhíveis.

## Impact

- **Configuração:** `groups-collapsed` por página, no YAML e nas páginas gerenciadas. O backup guarda a definição inteira e carrega o campo sem mudança de formato.
- **API pública:** `GET /api/v1/status-pages/{slug}` ganha `groupsCollapsed` e `groups[].summary`.
- **Código:** `internal/config/statuspage`, `internal/statuspage` (payload, serviço, opções e pré-visualização da administração), `internal/api` (comentários e contrato), `web/app/src/views/public/StatusPage.vue`, `components/public/` e o formulário em `views/admin/`.
- **Atualização da página:** continua por *polling* de 60 s sobre um payload com cache de 30 s. Esta change não acrescenta tempo real à listagem (D2).
- **Voltar de versão:** uma versão anterior recusa uma página gerenciada gravada com o campo novo (D5).
- **Documentação:** `docs/status-pages.md`, screenshots, `AGENTS.fork.md`.
