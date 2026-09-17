## 1. Formulário de status page

- [ ] 1.1 Marcar `meta.adminList: true` nas rotas `AdminStatusPageNew` e `AdminStatusPageEdit` em `web/app/src/router/index.js`, ajustando o comentário do grupo.
- [ ] 1.2 Reestruturar `AdminStatusPageForm.vue` conforme D1: casco `flex flex-col … md:min-h-0 md:flex-1` com `max-w-7xl`, cabeçalho e avisos que ficam em `shrink-0`, e um corpo por estado — spinner centrado, YAML somente leitura com `md:min-h-0 md:flex-1 md:overflow-auto`, e grade de duas colunas `md:grid-cols-2 md:grid-rows-1` com rolagem própria em cada uma — mais a barra de ações `shrink-0` no rodapé.
- [ ] 1.3 Lista de endpoints ocupando a coluna: `max-h-80 overflow-y-auto md:max-h-none md:min-h-0 md:flex-1`, com a busca fora da área que rola e o cabeçalho continuando `sticky top-0` **dentro** de `status-page-endpoint-list`; grupos trocando `sm:grid-cols-2 lg:grid-cols-3` por `lg:grid-cols-2` e a linha do slug empilhando os botões até `lg`.
- [ ] 1.4 Pré-visualização em `AdminDialog` conforme D2, com `:open`/`@close` (não existe `update:open`), `size="xl"`, `testid="status-page-preview"` e `:return-focus` no botão Preview, mantendo `status-page-preview-featured`, removendo `status-page-preview-toggle`, `previewExpanded` e os ícones sem uso, e mantendo o aviso da página não salva.
- [ ] 1.5 Mensagens conforme D3: toast de sucesso ao salvar emitido no `watch` do slug depois do `load()`, um único toast por ação, faixa `status-page-validation` só com avisos e rolando até ela, e estado próprio para o conflito de versão, que só some ao recarregar a versão ou salvar com sucesso.
- [ ] 1.6 `npm run lint` e `npm run test:unit` no `web/app`.

## 2. Legenda do Response Time Trend

- [ ] 2.1 Acrescentar `CHART_LEGEND_COLORS` e `chartLegend(period, summary)` em `web/app/src/utils/responseTimeChart.js` conforme D4, com o estilo de traço por série.
- [ ] 2.2 Desenhar a legenda em `ResponseTimeChart.vue` abaixo do gráfico e fora do contêiner de altura fixa (que continua com `data-testid="response-time-chart"` e todos os `data-*`), com `role="list"`, `aria-label`, `data-series` por item, amostras `aria-hidden` (traço para as linhas, quadrado com borda de contraste para as colunas) e `v-if="!loading && !error && payload"`; `borderDash` no mínimo e no máximo.
- [ ] 2.3 Testes de unidade de `chartLegend`: Recent sem colunas, Recent só com Down, Recent com Pending, agregado com Down e Pending, agregado sem ponto de linha e a ordem dos itens.

## 3. Testes e entrega

- [ ] 3.1 Ampliar `test/e2e/status-pages.sh` conforme D5: ~25 endpoints extras na configuração do roteiro, `layout_ok` e `not_covered` copiados do `admin-backup.sh`, rolagem da lista com `clientHeight > 320`, Esc fechando o diálogo antes dos cliques seguintes, `scrollintoview` nos cliques que agora ficam dentro das colunas, toast de informação da validação e legenda por `data-series`; no `push.sh`, a legenda com `pending` no dashboard e a altura do gráfico.
- [ ] 3.2 Rodar `test/e2e/status-pages.sh` e `test/e2e/push.sh` e conferir os prints nos temas claro e escuro.
- [ ] 3.3 Conferência manual em 1280×900, 800×600 e 390×844: formulário novo, edição de página do YAML (somente leitura), conflito de versão e cartão do gráfico no dashboard e na página pública.
- [ ] 3.4 `npm run build` no `web/app` com o `web/static` versionado, `gofmt`/lint e `openspec validate refine-status-page-form --strict`.
- [ ] 3.5 Entrega: PR em `jniltinho/gatus` com CI verde, release na próxima versão livre da série (`v5.36.0-fork.22` se a barra de rolagem fina sair antes) com notas em pt-BR, imagem no Docker Hub, pacote `mariadb` e versões dos exemplos atualizadas, e PR de arquivamento da change.
