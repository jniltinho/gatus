## Why

O formulário de status page (`/admin/status-pages/new` e `/admin/status-pages/:slug/edit`) é a única tela da administração que ainda rola a página inteira: as seções General, Groups e Endpoints ficam empilhadas, a lista de endpoints tem a própria rolagem dentro dessa página, e a validação e a pré-visualização crescem abaixo dos botões. O resultado é uma barra de rolagem muito longa, com rolagens aninhadas, e os botões Validate, Preview e Save saem da vista logo no começo da edição. As listas, a tela de Backup e os diálogos já seguem o padrão de preencher a janela e rolar só por dentro.

No cartão **Response Time Trend**, os períodos 3h, 6h, 24h e 1w desenham três linhas (média, mínimo e máximo) sem nada que diga o que é cada uma: a paridade com o Uptime Kuma trouxe o gráfico sem legenda, e quem olha não tem como saber qual linha é qual.

## What Changes

- O formulário de status page passa a preencher a janela em telas médias e grandes, nos seus quatro estados (carga, somente leitura, criação e edição), com o cabeçalho e a barra de ações fixos e a rolagem só dentro do conteúdo: General e Groups numa coluna, Endpoints na outra, cada uma com a própria rolagem, e o YAML da página somente leitura com a própria área de rolagem. Em telas pequenas o comportamento continua o de hoje, com a rolagem da página.
- A lista de endpoints deixa de ter altura fixa em telas médias e grandes e passa a ocupar a altura da coluna, acabando com a rolagem dentro da rolagem; o cabeçalho fixo da lista e a busca continuam.
- A pré-visualização sai de uma seção no fim da página e passa a abrir no diálogo padrão da administração, com rolagem própria, Esc e foco tratados pelo diálogo. Numa página ainda não salva, o botão continua validando e avisando que é preciso salvar antes.
- Mensagens de sucesso e o resumo da validação passam a ser toasts, um por ação, com o toast da criação sobrevivendo à navegação para a tela de edição; os avisos da validação aparecem numa faixa dentro do conteúdo quando existem, e os avisos que precisam ficar (somente leitura, definição salva inválida, falha ao carregar e conflito de versão com o botão de recarregar) ficam fora da área que rola.
- O cartão Response Time Trend ganha uma legenda abaixo do gráfico: em Recent, o tempo de resposta; em 3h, 6h, 24h e 1w, média, mínimo e máximo; e os itens Down e Pending quando o período mostrar essas colunas. Como as três linhas são quase o mesmo verde, o mínimo passa a ser tracejado e o máximo pontilhado, e a amostra de cada item da legenda repete o traço da sua linha.
- Os testes E2E de `test/e2e/status-pages.sh` e de `test/e2e/push.sh` passam a conferir o formulário sem rolagem nem conteúdo cortado, a pré-visualização no diálogo, o toast da validação e a legenda do gráfico.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `status-page-web-ui`: o formulário de administração passa a ter layout que preenche a janela, barra de ações sempre visível, pré-visualização em diálogo e mensagens em toasts.
- `response-time-chart`: o cartão passa a ter legenda das séries, no lugar do "sem legenda" herdado do Uptime Kuma.

## Impact

- `web/app/src/views/admin/AdminStatusPageForm.vue`: estrutura do template, pré-visualização em `AdminDialog`, uso de `utils/toast.js`.
- `web/app/src/router/index.js`: `meta.adminList` nas rotas `/admin/status-pages/new` e `/admin/status-pages/:slug/edit`, para o casco de altura total do `App.vue`.
- `web/app/src/components/ResponseTimeChart.vue` e `web/app/src/utils/responseTimeChart.js`: legenda das séries, as cores cheias das amostras e o traço das linhas de mínimo e máximo.
- `test/e2e/status-pages.sh` (inclusive mais endpoints na configuração do roteiro, para a lista transbordar) e `test/e2e/push.sh`; `web/static` versionado no repositório precisa ser regerado.
- Sem mudança no backend, na API, no banco nem na configuração.
