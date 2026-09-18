## 1. Definição, validação e segredo

- [x] 1.1 Campo `Auth *PageAuth` em `config/statuspage` conforme D1, com validação de usuário e de hash bcrypt em base64 (alfabeto URL), e aviso na carga quando houver `auth` sem `security` na instalação.
- [x] 1.2 Documento de submissão da administração com a senha em claro, convertido para a definição antes de persistir, conforme D2: hash com bcrypt custo 10, senha nunca gravada, nunca registrada em log e nunca em mensagem de erro; senha entre 8 e 72 bytes.
- [x] 1.3 Mascaramento do hash nas leituras da administração conforme D3 — detalhe, YAML, validação e pré-visualização — e o caminho de volta: `********` recebido significa "manter o hash guardado".
- [x] 1.4 Testes Go: validação (inclusive hash com `-`/`_` no base64), e busca por `password:` e pelo prefixo do bcrypt na definição gravada, no YAML do detalhe, na validação, na pré-visualização e no backup.

## 2. Proteção das rotas da página

- [x] 2.1 Middleware por rota conforme D4: uma única resolução do slug guardada em `c.Locals`, regras de página antes do desafio pelo mesmo caminho de erro de hoje, 401 com `WWW-Authenticate: Basic realm="<slug publicado>", charset="UTF-8"` e `Cache-Control: no-store`, e 404 por chave só **depois** do 401.
- [x] 2.2 Não reaproveitar `isBrowserRequest`: o cabeçalho do desafio vai em toda requisição. Reusar a comparação de credencial de `security` (usuário em tempo constante e bcrypt sempre nos dois) e o gancho de teste de comparação.
- [x] 2.3 Limitador com chave por página e IP conforme D7, com o IP vindo de `statuspage.ClientIP`, janela de 5 min, 10 falhas, teto de chaves e descarte no `Load`.
- [x] 2.4 Memória da verificação conforme D7: HMAC com chave de processo, prefixo de comprimento, impressão digital de usuário e hash, TTL absoluto de 5 min, teto de entradas e limpeza no `Load`; ordem limitador → memória → bcrypt.
- [x] 2.5 Teto de verificações bcrypt simultâneas, nos moldes do semáforo de montagem pública.
- [x] 2.6 `Cache-Control: private, no-store` nos 200 das páginas protegidas e `private, no-cache, no-store, no-transform` no canal de eventos delas.
- [x] 2.7 Rotas de badge sob a página conforme D5, reusando os geradores atuais depois de conferir que a chave pertence à página.
- [x] 2.8 Testes Go conforme D10.2 a D10.7.

## 3. Frontend

- [ ] 3.1 `credentials: 'same-origin'` nas buscas das rotas públicas da página (`StatusPage.vue`, `StatusPageEndpoint.vue`, `ResponseTimeChart.vue`).
- [ ] 3.2 Badges da página de detalhes apontando para as rotas com slug.
- [ ] 3.3 Tratamento do 401 na atualização periódica: parar o ciclo e pedir recarga, sem reabrir a caixa de credencial em laço.
- [ ] 3.4 Formulário e lista conforme D9, com o booleano `requiresLogin` na listagem da API.
- [ ] 3.5 `npm run lint` e `npm run test:unit`.

## 4. Backup

- [x] 4.1 Mesclagem da credencial guardada no destino **antes** da validação no restore, conforme D8, com `skip` para página nova com hash mascarado.
- [x] 4.2 Testes Go da prévia e da aplicação do restore nos dois casos.

## 5. Testes e entrega

- [ ] 5.1 E2E em `test/e2e/status-pages.sh` conforme D10.11 a D10.15, provando o 401 com requisição direta (`curl -i`) e o 200 com `curl -u`, sem levar o navegador à caixa nativa de credencial.
- [ ] 5.2 Rodar `status-pages.sh`, `push.sh`, `admin.sh` e `admin-backup.sh`, com prints.
- [ ] 5.3 Conferência manual: navegador pedindo a credencial, credencial errada, página de detalhes com gráfico, eventos e badges dentro da página protegida, e a página pública de sempre intacta.
- [ ] 5.4 Documentação: seção em `docs/status-pages.md` (como ligar, o que passa a exigir credencial, que as rotas globais por chave continuam públicas, que não há logout no HTTP Basic, a relação com `trusted-proxies` e como fazer no YAML colando o hash), nota em `docs/README.md` e seção no `AGENTS.fork.md`.
- [ ] 5.5 `make frontend-build` com o `web/static` no commit, `make lint` e `openspec validate add-status-page-authentication --strict`.
- [ ] 5.6 Recapturar os prints de `docs/screenshots/` que mudarem e atualizar a versão citada.
- [ ] 5.7 Entrega: PR em `jniltinho/gatus` com CI verde, release da próxima versão da série com notas em inglês, imagem no Docker Hub, pacote `mariadb`, versões dos exemplos e PR de arquivamento.
