## ADDED Requirements

### Requirement: Cabeçalho e histórico iguais nas duas telas de detalhes
A tela de detalhes do endpoint do dashboard (`/endpoints/<key>`) e a pública (`/status/<slug>/endpoints/<key>`) MUST ter o mesmo cabeçalho e a mesma apresentação do histórico de verificações. A ordem dos blocos já é fixada pelo requisito "Destaques e página de detalhes nas telas" de `status-page-highlights`, e a altura das barras e o tooltip que não ocupa espaço no layout pelo requisito "Página pública de status" de `status-page-web-ui`; este requisito cuida do que falta.

**Cabeçalho:** o nome do endpoint MUST ser o título, com o mesmo tamanho e peso nas duas telas e quebra por palavra; o grupo MUST aparecer na linha abaixo do título nas duas; a expiração do certificado, quando a tela a mostrar, MUST usar as mesmas cores por faixa; e o indicador de estado MUST ficar à direita do título nas duas. Cada tela MUST ter um controle de voltar no mesmo lugar e no mesmo formato, com o destino de cada uma.

**Histórico:** o cartão do histórico MUST mostrar as barras dos últimos resultados com 20 px de altura nas duas telas, com os rótulos de tempo do resultado mais antigo e do mais recente nas pontas, e MUST NOT repetir visualmente o nome, o grupo, o host nem o estado do endpoint, que já estão no cabeçalho da página. O resumo textual para leitores de tela continua como está, com o nome do endpoint.

**Diferenças previstas**, que MUST continuar existindo:
- só o dashboard mostra o host, porque o payload público não publica endereço;
- só o dashboard tem os botões de atualizar e de alternar entre média e mínimo-máximo, a paginação da tabela e o tooltip com as condições e os erros da verificação;
- só a tela pública mostra o horário da última atualização, o link de volta para a status page e a tabela de verificações sanitizada, com mensagens apenas quando a página permitir;
- a quantidade de barras pode ser diferente entre as duas.

#### Scenario: Mesmo cabeçalho
- **WHEN** o mesmo endpoint é aberto nas duas telas, na mesma largura de janela
- **THEN** o título tem o mesmo tamanho nas duas
- **AND** as duas mostram o grupo na linha abaixo do nome

#### Scenario: Nada de host na tela pública
- **WHEN** um visitante abre a tela pública de detalhes de um endpoint
- **THEN** o cabeçalho não mostra o host
- **AND** a tela do dashboard do mesmo endpoint mostra o host

#### Scenario: Mesmo histórico
- **WHEN** o mesmo endpoint é aberto nas duas telas
- **THEN** as barras têm 20 px de altura nas duas
- **AND** nenhuma das duas mostra o nome do endpoint em texto visível dentro do cartão do histórico

#### Scenario: Ações só do dashboard
- **WHEN** um visitante abre a tela pública de detalhes
- **THEN** não há botão de atualizar nem de alternar entre média e mínimo-máximo
