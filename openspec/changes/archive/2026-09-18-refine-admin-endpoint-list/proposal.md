## Why

A lista de endpoints da administração ficou desalinhada depois que as colunas passaram a ter largura fixa. O caso mais visível é o selo `+ push`, que marca os endpoints ativos que também recebem push: ele não cabe na coluna Type e quebra em duas linhas, com o `+` numa e `push` na outra. A linha inteira cresce para acomodar isso — medido em 1000 px de largura: **53 px de altura**, contra 49 px de uma linha normal —, então uma única linha com selo desalinha a tabela toda. O aviso "Conflicts with YAML" tem o mesmo efeito, pelo mesmo motivo: ele ocupa uma segunda linha dentro da célula do nome.

Fora esse defeito, a lista gasta mais espaço do que precisa para o que mostra: cada linha tem 49 px de altura, quase toda ela vinda dos botões de texto de 36 px, que ainda ocupam 240 px de largura por linha — quase um quinto da largura em 1280 px. Numa tela que existe para varrer muitos endpoints de relance, isso significa menos endpoints visíveis e mais movimento de olho.

## What Changes

- O selo `+ push` deixa de quebrar linha: passa a ser um selo compacto que cabe na coluna, e a coluna Type ganha a largura de que precisa.
- O aviso de conflito com o YAML e o de definição inválida deixam de ocupar uma segunda linha: viram um ícone ao lado do nome, com a explicação na dica do ponteiro.
- A tabela fica mais enxuta: texto menor, respiro menor nas células e altura de linha de 49 px para cerca de 29 px, sem perder nenhuma informação.
- As colunas são reproporcionadas para o conteúdo real de cada uma, e o alinhamento fica consistente — texto à esquerda, números e ações à direita.
- As ações viram ícones com dica ao repousar o ponteiro e nome acessível para leitores de tela: editar ou ver, habilitar ou desabilitar, remover. A coluna de ações encolhe de 240 px para cerca de 100 px, e a largura recuperada vai para as colunas de conteúdo.
- As listas de status pages e de chaves de push recebem o mesmo tratamento, porque são a mesma tabela com outros campos e ficariam destoando — na de status pages o ganho é maior ainda, porque ela tem cinco ações por linha.
- Os cartões das telas estreitas seguem o mesmo padrão de ícones, e todos os identificadores de teste das ações continuam os mesmos.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `admin-web-ui`: a lista de endpoints passa a ter a densidade descrita, o selo de push sem quebra de linha e as ações como ícones com nome acessível, valendo também para as outras duas listas.

## Impact

- `web/app/src/views/admin/AdminEndpoints.vue`, `AdminStatusPages.vue` e `AdminPushKeys.vue`: densidade, colunas, selos e ações.
- Possivelmente um componente pequeno para o botão de ação com ícone, para as três listas não repetirem o mesmo bloco.
- `test/e2e/status-pages.sh`, `test/e2e/push.sh` e `test/e2e/admin.sh`: conferências de altura de linha e das ações por ícone; `web/static` versionado precisa ser regerado.
- `docs/screenshots/`: os três prints das listas são recapturados.
- Sem mudança no backend, na API, no banco nem na configuração.
