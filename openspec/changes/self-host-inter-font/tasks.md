## 1. Fonte

- [x] 1.1 Baixar os dois `woff2` variáveis da Inter (latin e latin-ext, `wght@100..900`) para `web/app/src/assets/fonts/`, como `inter-4-1-latin.woff2` e `inter-4-1-latin-ext.woff2`, e registrar no `AGENTS.fork.md` as URLs do `fonts.gstatic.com`, o `User-Agent` usado na chamada ao `css2`, os `unicode-range` copiados e os SHA-256 **dos arquivos commitados**.
- [x] 1.2 Acrescentar `web/app/public/fonts/OFL.txt` (a licença é servida junto dos arquivos). Nada de README dentro de `public/`: o conteúdo é servido e `/fonts/` lista o diretório.
- [x] 1.3 Blocos `@font-face` em `index.css` conforme D2, com caminho **relativo** (`./assets/fonts/…`), `font-display: swap` e `unicode-range`; conferir no `web/static` gerado que os arquivos saíram em `fonts/` e que o CSS aponta para `/fonts/…`.
- [x] 1.4 `preload` do subconjunto latino em `web/app/public/index.html`, com `as="font" type="font/woff2" crossorigin`, antes do `custom.css`.
- [x] 1.5 Conferir a pilha `sans` do `tailwind.config.js` e ajustar o comentário; a reserva do sistema continua.

## 2. Algarismos tabulares

- [x] 2.1 `font-variant-numeric: tabular-nums` no `body` e `font-variant-numeric: inherit` em `button, input, optgroup, select, textarea`, dentro de `@layer base`, conforme D3.
- [x] 2.2 `Chart.defaults.font.family` a partir da família computada do `body` no registro do Chart.js.
- [x] 2.3 Conferir na tela os campos de formulário, as tabelas, os painéis de números, o gráfico e os sinais `✓`, `✕` e `✗`, que ficam fora do subconjunto e caem na fonte do sistema.

## 3. Testes e entrega

- [x] 3.1 Testes Go conforme D4: `TestEmbed` exigindo os dois `woff2` e o `OFL.txt`, e um teste em `api` pedindo `/fonts/inter-4-1-latin.woff2` e conferindo `200`, `Content-Type: font/woff2` e a assinatura `wOF2`.
- [x] 3.2 Ampliar `test/e2e/status-pages.sh` conforme D4: `FontFace` da Inter com `status === 'loaded'` depois de `document.fonts.ready`, entrada de `/fonts/inter-4-1-latin.woff2` em `performance.getEntriesByType('resource')` com `decodedBodySize > 0` e nenhuma entrada do Google, e o par de controle de algarismos tabulares num único `eval` devolvendo `true`/`false`; na primeira captura de requisições do roteiro.
- [x] 3.3 Rodar `test/e2e/status-pages.sh`, `test/e2e/push.sh` e `test/e2e/admin-backup.sh` **depois** da troca de fonte e regerar todos os prints, que foram tirados com fonte de sistema.
- [ ] 3.4 Conferência manual: página pública, dashboard e administração em 1280×900 e 390×844; e uma carga com `network route "/fonts/**" --abort`, conferindo que a página continua legível e que a família computada cai na reserva.
- [x] 3.5 Documentação: seção em `AGENTS.fork.md` (origem, procedimento de atualização, subconjuntos, por que o nome carrega a versão) e nota em `docs/README.md` com os exemplos de `ui.custom-css` — família sem `!important`, algarismos tabulares com `!important`.
- [x] 3.6 `npm run lint`, `npm run test:unit`, `make frontend-build` com `web/static` no commit, `make lint` e `openspec validate self-host-inter-font --strict`.
- [ ] 3.7 Entrega: PR em `jniltinho/gatus` com CI verde, release da próxima versão da série com notas em pt-BR, imagem no Docker Hub, pacote `mariadb`, versões dos exemplos e PR de arquivamento.
