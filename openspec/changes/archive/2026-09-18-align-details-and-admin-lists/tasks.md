## 0. Data do certificado no payload público

- [x] 0.1 `statuspage/certificate.go`: devolver, junto dos dias, o instante do vencimento do mesmo resultado (`timestamp + certificateExpiration`), em UTC.
- [x] 0.2 `statuspage/payload.go`: campo `certificateExpiresAt` (RFC 3339, `omitempty`) preenchido junto de `certificateExpiresInDays`, nos grupos, nos destaques e na API pública de detalhes.
- [x] 0.3 Testes Go: mesmo resultado para os dois campos, omissão sem certificado, push mais novo sem certificado não apaga a data, e a API pública de detalhes devolvendo os dois.

## 1. Telas de detalhes

- [x] 1.1 Cabeçalho comum conforme D1: título `text-2xl font-semibold tracking-tight break-words` nas duas, grupo à esquerda na linha abaixo do nome (com `Updated …` à direita só na pública), certificado com dias **e data** nas duas e as mesmas cores, `StatusBadge` à direita do título e o controle de voltar no mesmo formato. O host continua só no dashboard, e as linhas das listas públicas continuam só com os dias.
- [x] 1.2 Prop `compact` no `EndpointCard`, `default: false`, usada só pela tela de detalhes: sem o cabeçalho interno (nome, grupo, host, badge), barras de 20 px e sem `hover:scale`. O tooltip continua o do `App.vue`, que já não ocupa espaço e traz condições e erros. A grade do `Home.vue` não passa a prop e não muda.
- [x] 1.3 Rótulos de tempo nas pontas das barras nas duas telas de detalhes; no `EndpointRow`, presos ao modo `show-header` falso, para a lista da status page continuar dentro do teto de 72 px por linha.
- [x] 1.4 `data-testid="details-badges"` e `data-testid="details-health"` nos dois blocos, nas duas telas, para o teste conseguir comparar a ordem.
- [x] 1.5 `npm run lint`. Não há teste de unidade para estas telas: o runner só cobre `src/utils`, e a verificação é toda de ponta a ponta.

## 2. Listas da administração

- [x] 2.1 `table-layout: fixed` nas três tabelas, com `truncate` e `title` nos campos que crescem: nome e URL nos endpoints, slug e título nas status pages, nome nas chaves de push.
- [x] 2.2 Aviso de conflito com o YAML fora da linha do nome, numa segunda linha da própria célula, para não somar largura.
- [x] 2.3 Colunas por largura: todas a partir de `lg`; sem `Interval` e sem `Source` entre `md` e `lg` (e as equivalentes nas outras duas listas).
- [x] 2.4 Cartões abaixo de `md` nas três listas (`admin-card-*`, `status-page-card-*`, `push-key-card-*`), com todos os campos, inclusive o intervalo, e com as ações reusando os identificadores de hoje (`admin-open-*`, `admin-toggle-*`, `admin-remove-*`, `status-page-edit-*`, `push-key-revoke-*`).
- [x] 2.5 Conferir que `admin-table`, `status-pages-table` e `push-keys-table` continuam sendo os identificadores das respectivas tabelas, e que o `overflow-x-auto` do `AdminListLayout` continua só como rede de segurança.

## 3. Testes e entrega

- [x] 3.1 `test/e2e/status-pages.sh`: criar um endpoint com nome de ~60 caracteres e URL de ~120 e, para as listas de endpoints e de status pages (a página `team` que o roteiro já cria pela web), conferir em 1100×800, 900×700, 700×800, 390×844 e 360×800 — sem transbordo no painel nem no documento, ações do item gerenciado pela web dentro da janela, tabela ausente e cartões presentes abaixo de `md`, e a URL truncada com `title` completo. Restaurar 1280×900 ao fim.
- [x] 3.2 `test/e2e/push.sh`: as mesmas conferências para a lista de chaves de push, medindo a linha `push-key-row-admin-akamai`, que tem a ação de revogar.
- [x] 3.3 `test/e2e/status-pages.sh`: comparar as duas telas de detalhes do endpoint `_panel` na mesma largura — `font-size` do `h1`, altura da barra igual a 20 px, grupo nas duas, host ausente na pública, nome ausente do texto visível do cartão do histórico, e a ordem dos blocos por um mapa explícito de identificadores ordenado por posição vertical.
- [x] 3.4 Rodar `status-pages.sh`, `push.sh`, `admin.sh` e `admin-backup.sh`, regerar os prints e conferir os dois temas.
- [x] 3.5 Conferência manual em 1280×900, 1100×800, 900×700 e 390×844: as duas telas de detalhes e as três listas, com nome e URL longos.
- [x] 3.6 Atualizar os prints versionados que mudam de forma: `docs/screenshots/endpoint-details.png`, `status-page-endpoint.png`, `admin-endpoints.png`, `admin-status-pages.png` e `admin-push-keys.png`, e a versão citada em `docs/screenshots/README.md`.
- [x] 3.7 `make frontend-build` com o `web/static` no commit, `make lint` e `openspec validate align-details-and-admin-lists --strict`.
- [ ] 3.8 Entrega: PR em `jniltinho/gatus` com CI verde, release da próxima versão da série com notas **em inglês**, imagem no Docker Hub, pacote `mariadb`, versões dos exemplos e PR de arquivamento.
