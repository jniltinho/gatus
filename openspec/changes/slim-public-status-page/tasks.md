## 1. Layout

- [x] 1.1 Ritmo vertical de `views/public/StatusPage.vue` conforme a tabela de D1: contêiner `py-6`, cabeçalho `mb-4` com título `text-2xl sm:text-3xl`, seções `mt-6`, cabeçalho do grupo `pb-1.5`, grade dos destaques `gap-3`, título "Featured" em `mb-2`, avisos em `mt-2`/`mt-6`.
- [x] 1.2 `components/public/EndpointRow.vue`: linha `py-2`, barras `h-5` com `mt-1.5`, cartão do destaque `p-3` com tabela `mt-2` e células `py-0.5`.
- [x] 1.3 Colunas de uptime conforme D2: um só markup com `<dt>`/`<dd>`, `flex` abaixo de `sm` e `sm:grid sm:grid-cols-3` com filhos `sm:w-16` a partir dali, rótulos `sm:hidden` nas linhas e grade igual (também `aria-hidden`) no cabeçalho do grupo, que passa a ter dois filhos no `justify-between` com o nome truncado.
- [x] 1.4 Detalhe em tooltip conforme D3: contêiner das barras `relative`, tooltip absoluto `aria-hidden` com `pointer-events-none`, ancorado pela esquerda ou pela direita conforme a metade da barra ativa e com `max-width` fechando a outra ponta, acima nas linhas de grupo e abaixo no destaque e na página de detalhes, mantendo `data-testid="status-endpoint-detail"`; região `sr-only` com `aria-live` e `aria-atomic` sempre presente, vazia sem barra ativa.
- [x] 1.5 `npm run lint` e `npm run test:unit`.

## 2. Testes e entrega

- [x] 2.1 Ampliar `test/e2e/status-pages.sh` conforme D5, medindo em `/status/messages` (dois endpoints no mesmo grupo): altura entre 56 e 72 px com `Math.round`, colunas com largura e texto antes de comparar os `left`, `<dt>` escondidos nas linhas e visíveis no cabeçalho do grupo, tooltip da última barra sem empurrar nem vazar e com `pointer-events: none`, região `aria-live` vazia antes de qualquer hover, ponteiro retirado depois do teste, e 390 px e 360 px sem rolagem horizontal com o rótulo visível na linha.
- [x] 2.2 Rodar `test/e2e/status-pages.sh` e `test/e2e/push.sh` e conferir os prints (página pública e página de detalhes, que usa o mesmo componente).
- [x] 2.3 Conferência manual em 1280×900, 800×600, 390×844 e 360×800, nos temas claro e escuro, incluindo um endpoint com aviso de certificado, um nome longo e a página pública de detalhes (tooltip abaixo das barras, título do cartão sem cobertura).
- [x] 2.4 `make frontend-build` com o `web/static` no commit, `make lint` e `openspec validate slim-public-status-page --strict`.
- [ ] 2.5 Entrega: PR em `jniltinho/gatus` com CI verde, release da próxima versão da série com notas em pt-BR, imagem no Docker Hub, pacote `mariadb` e versões dos exemplos, e PR de arquivamento.
