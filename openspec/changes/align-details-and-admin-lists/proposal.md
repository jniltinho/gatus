## Why

**Detalhes do endpoint.** A mesma informação é mostrada em duas telas — `/endpoints/<key>` no dashboard e `/status/<slug>/endpoints/<key>` na página pública — que foram escritas em momentos diferentes e não se parecem. O título é `text-2xl` numa e `text-4xl` na outra; a linha abaixo do nome mostra grupo e host numa e "Atualizado há X" na outra; o cartão Recent Checks do dashboard repete nome, grupo e host dentro dele, com dois botões de ação, enquanto o público mostra só as barras; e as barras vêm de componentes diferentes, com alturas e comportamentos de tooltip diferentes desde que a página pública ficou mais enxuta. Quem administra e quem só acompanha veem a mesma coisa de dois jeitos.

**Lista de endpoints da administração.** A tabela de `/admin` tem oito colunas com largura de conteúdo e a coluna de URL limitada a 320 px, então a soma passa da largura disponível e aparece uma barra de rolagem horizontal. Medido na instalação de validação: sem transbordo até 900 px, 5 px em 800, 37 px em 768 e 165 px em 640 — e quanto mais longos os nomes e as URLs, mais cedo começa. Rolar a tabela para o lado para alcançar as ações é o pior jeito de usar a tela.

## What Changes

- As duas telas de detalhes passam a ter o mesmo cabeçalho: título do mesmo tamanho, a mesma linha de grupo e host, a mesma linha de expiração do certificado e o indicador de estado no mesmo lugar.
- O histórico de verificações passa a ser apresentado igual nas duas: barras com a mesma altura e as mesmas cores, os mesmos rótulos de tempo nas pontas e, nas duas, sem repetir nome, grupo, host e estado dentro do cartão. O conteúdo do tooltip continua diferente de propósito — o do dashboard mostra também as condições e os erros da verificação.
- A ordem dos blocos, que já é a mesma, passa a ser descrita no requisito: barras, painel de números, gráfico, tabela de verificações, badges, saúde atual e eventos.
- A página pública de detalhes passa a mostrar também a **data** de vencimento do certificado, ao lado dos dias, como o dashboard: o payload público ganha a data para isso, calculada do mesmo resultado de onde já saem os dias e publicada só quando a página liga `show-certificate-expiration`.
- Ficam registradas as diferenças que **devem** continuar existindo: só o dashboard mostra o host, porque o payload público não publica endereço; o dashboard tem os botões de atualizar e de alternar entre média e mínimo-máximo e a paginação da tabela; a página pública tem o link de volta para a status page, o horário da última atualização e só mostra mensagens quando a página permite.
- A lista de endpoints da administração deixa de ter rolagem horizontal em qualquer largura: a largura da tabela deixa de depender do conteúdo, o nome e a URL passam a ser truncados com o valor completo no `title`, o aviso de conflito sai de dentro da linha do nome, as colunas menos importantes somem entre 768 e 1024 px, e abaixo disso cada endpoint vira um cartão com todos os campos e as mesmas ações.
- As listas de status pages e de chaves de push, que têm a mesma estrutura de tabela, recebem o mesmo tratamento, para a administração não ficar com três comportamentos diferentes.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `endpoint-details-summary`: ganha o requisito da apresentação comum das duas telas de detalhes, com as diferenças que continuam existindo. O requisito de `status-page-highlights` que já manda a página pública seguir o layout do dashboard continua valendo e não muda.
- `certificate-expiration`: o payload público passa a publicar a data de vencimento junto dos dias, e a página pública de detalhes passa a mostrá-la. As linhas das listas públicas continuam só com os dias.
- `admin-web-ui`: a lista de endpoints passa a caber na largura disponível, sem rolagem horizontal, com as colunas e o cartão do celular descritos, e as outras duas listas seguem a mesma regra.

## Impact

- `web/app/src/views/EndpointDetails.vue` e `web/app/src/views/public/StatusPageEndpoint.vue`: cabeçalho e cartão do histórico.
- `web/app/src/components/EndpointCard.vue` e `web/app/src/components/public/EndpointRow.vue`: a apresentação das barras nas telas de detalhes.
- `web/app/src/views/admin/AdminEndpoints.vue`, `AdminStatusPages.vue` e `AdminPushKeys.vue`: colunas responsivas e cartões no celular.
- `test/e2e/status-pages.sh` e `test/e2e/push.sh`: conferências de largura nas três listas e comparação das duas telas de detalhes; `web/static` versionado precisa ser regerado.
- `docs/screenshots/`: cinco prints mudam de forma e são recapturados.
- `statuspage/payload.go` e `statuspage/certificate.go`: a data de vencimento no payload público, com os testes correspondentes. Sem mudança no banco nem na configuração.
