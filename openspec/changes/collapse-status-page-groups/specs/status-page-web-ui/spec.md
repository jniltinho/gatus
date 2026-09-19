## MODIFIED Requirements

### Requirement: Página pública de status
A rota `/status/:slug` do frontend MUST mostrar:
- logo e cabeçalho da configuração `ui`, título e descrição da página;
- uma faixa com o estado geral em texto e cor ("All systems operational", "Partial outage", "Major outage" ou "No data");
- na mesma faixa, a contagem dos endpoints da página por estado, com os que estão no ar e os que não estão sempre presentes (`12 up · 2 down`), e os pendentes e sem dados somente quando houver algum;
- uma seção por grupo cujo cabeçalho é um botão que recolhe e expande o grupo e mostra o nome, o estado do grupo e a contagem do `summary` do grupo, omitindo os estados com zero, com a seção sem grupo rotulada "Other services";
- para cada endpoint de um grupo expandido e para cada destaque: nome, indicador de estado, uptime de 24h, 7d e 30d ("—" quando indisponível) e barras dos últimos resultados, com tooltip de horário, sucesso e duração em milissegundos.

A contagem MUST vir do payload, e não ser somada no navegador, para continuar certa numa página truncada. Ela MUST ficar legível nas duas versões do tema e MUST NOT provocar rolagem horizontal a partir de 360 px.

**Densidade e alinhamento:** a linha de um endpoint de grupo MUST ocupar no máximo 72 px de altura em telas a partir de 640 px, sem a linha opcional da expiração do certificado, e as barras do histórico MUST ter 20 px de altura. Em telas a partir de 640 px os uptimes MUST ficar em colunas de largura fixa à direita, alinhadas entre os endpoints do mesmo grupo, com os valores visíveis, e os rótulos `24h`, `7d` e `30d` MUST aparecer uma vez no cabeçalho do grupo, escondidos nas linhas e fora do alcance dos leitores de tela. Abaixo de 640 px cada linha MUST mostrar rótulo e valor juntos, e a página MUST continuar sem rolagem horizontal a partir de 360 px.

**Tooltip da verificação:** o detalhe da barra ativa MUST aparecer sobre as barras, sem ocupar espaço no layout: mostrar ou esconder o detalhe MUST NOT mudar a altura da linha nem a posição das linhas seguintes. O tooltip MUST ficar dentro da largura da linha, qualquer que seja a barra ativa, MUST NOT capturar o ponteiro e MUST ser escondido dos leitores de tela, que recebem o mesmo texto pela região `aria-live`. Onde houver conteúdo acima das barras — cartão de destaque e página de detalhes — o tooltip MUST aparecer abaixo delas.

Com `truncated: true` no payload, a página MUST mostrar o aviso "Showing the first 200 services". A página MUST NOT mostrar anúncios, `ui.buttons`, links sociais, o link Admin nem "Powered by". A descrição MUST ser exibida como texto puro. A página MUST definir `document.title` com o título da página, seguir o visual quadrado (exceto indicadores circulares), ter variantes para o tema escuro e funcionar em telas a partir de 360 px de largura.

#### Scenario: Página com falha parcial
- **WHEN** um visitante abre `/status/infra` e um endpoint do grupo `core` está fora
- **THEN** a faixa mostra "Degradação parcial"
- **AND** o grupo `core` e o endpoint aparecem com estado de falha

#### Scenario: Contagem na faixa
- **WHEN** a página `infra` tem 12 endpoints no ar e 2 fora
- **THEN** a faixa mostra `12 up` e `2 down`
- **AND** não mostra contagem de pendentes nem de sem dados

#### Scenario: Contagem com pendentes
- **WHEN** a página `jobs` tem 3 endpoints no ar, 1 pendente e nenhum fora
- **THEN** a faixa mostra `3 up`, `0 down` e `1 pending`

#### Scenario: Grupo recolhido
- **WHEN** o grupo `sites`, operacional, está recolhido
- **THEN** o cabeçalho mostra "sites", "Operational" e a contagem do grupo
- **AND** nenhuma linha de endpoint de `sites` existe no documento, e os destaques continuam visíveis

### Requirement: Telas de administração de status pages
Com a administração habilitada e autorizada, o frontend MUST oferecer:
- as rotas `/admin/status-pages`, `/admin/status-pages/new` e `/admin/status-pages/:slug/edit`;
- abas para alternar entre endpoints e status pages.

A lista MUST mostrar slug, título, origem, estado (publicada, desabilitada, em conflito ou inválida), se a página exige login e número de endpoints, com as ações abrir (link em nova aba com `rel="noopener"`), copiar link, habilitar ou desabilitar, editar e remover (com confirmação). A lista MUST mostrar avisos quando a publicação estiver desligada no YAML, quando as páginas gerenciadas estiverem indisponíveis e quando houver aviso de limite compartilhado atrás de proxy.

O formulário MUST ter:
- slug (somente leitura na edição), título, descrição e `enabled`;
- a opção "Show certificate expiration" (`show-certificate-expiration`), desmarcada por padrão, com explicação curta;
- a opção "Show messages" (`show-messages`), desmarcada por padrão, ao lado da anterior, com a explicação de que as mensagens dos envios e o status HTTP ficam públicos e os erros das verificações não;
- a opção "Start with the groups collapsed" (`groups-collapsed`), desmarcada por padrão, com a explicação de que um grupo com problema aparece sempre aberto;
- seleção de grupos e de endpoints a partir de `/options`, com busca nos endpoints;
- avisos da validação e pré-visualização do payload público;
- a opção "Require login to view this page", desmarcada por padrão, com usuário e senha mostrados apenas quando ela estiver marcada. Na edição de uma página que já exige login, o campo da senha MUST vir vazio, com a explicação de que deixá-lo vazio mantém a senha atual. A tela MUST NOT mostrar a senha nem o hash em nenhum momento, e o texto da opção "Published" MUST dizer que uma página protegida é visível **com login** no endereço público.

**Layout do formulário:** em telas médias e grandes a tela MUST caber na janela, sem rolagem nem conteúdo cortado, em todos os seus estados — carga, somente leitura, criação e edição. O cabeçalho com o título e o botão Back e a barra com as ações disponíveis na tela (Preview sempre; Validate e Save fora do modo somente leitura) MUST ficar sempre visíveis, e a rolagem MUST acontecer dentro do conteúdo:

- na edição e na criação, em duas colunas — General e Groups numa, Endpoints na outra, cada uma com a própria rolagem. A lista de endpoints MUST ocupar a altura da coluna, sem altura máxima fixa e sem rolagem dentro de outra rolagem, mantendo o cabeçalho fixo da lista, a busca e o filtro "Only selected";
- no modo somente leitura, o YAML MUST ter a própria rolagem, com o aviso de que a página só pode ser vista fora dessa área;
- na carga, o indicador MUST ficar centrado na área disponível.

Em telas pequenas as seções MUST continuar empilhadas, com a rolagem da página e a lista de endpoints com altura máxima, como hoje.

**Mensagens e pré-visualização:** o sucesso ao salvar e o resumo da validação MUST aparecer em toasts, um por ação, e o toast do sucesso MUST sobreviver à navegação da criação para a edição. Os avisos da validação MUST aparecer numa faixa dentro do conteúdo, só quando existirem, e a coluna MUST rolar até ela. Os avisos que valem enquanto a tela estiver aberta — página somente leitura do YAML, definição salva inválida, falha ao carregar e conflito de versão com o botão de recarregar — MUST ficar fora da área que rola, e o conflito de versão MUST NOT ser apagado por outras ações da tela, só por recarregar a versão atual ou por um save bem-sucedido. Na edição de uma página já salva, a pré-visualização MUST abrir no diálogo padrão da administração, com a própria rolagem, e fechar por Esc ou pelo botão de fechar, devolvendo o foco ao botão que a abriu; numa página ainda não salva, o botão MUST validar e avisar que a página precisa ser salva para ser pré-visualizada.

Uma página nova MUST começar desabilitada. As páginas do YAML MUST aparecer sem ações de alteração. Um 412 MUST levar a recarregar a página e avisar o administrador sem perder o que foi digitado.

#### Scenario: Criar e publicar
- **WHEN** um administrador cria a página `clientes` selecionando o grupo `core`, confere a pré-visualização e habilita a página
- **THEN** a lista mostra `clientes` como publicada
- **AND** o link copiado aponta para `/status/clientes`

#### Scenario: Página do YAML
- **WHEN** a lista mostra a página `infra` definida no YAML
- **THEN** a linha não tem as ações editar, habilitar, desabilitar e remover

#### Scenario: Edição concorrente
- **WHEN** outro administrador altera a página enquanto o formulário está aberto e o primeiro salva
- **THEN** a tela avisa que a página mudou e mantém o conteúdo digitado

#### Scenario: Publicação desligada
- **WHEN** a configuração tem `status-pages.enabled: false`
- **THEN** a lista mostra o aviso "Status pages desligadas no arquivo de configuração"

#### Scenario: Ligar a expiração do certificado
- **WHEN** um administrador marca "Show certificate expiration" na página `clientes` e salva
- **THEN** a definição salva tem `show-certificate-expiration: true`
- **AND** a página pública mostra os dias até o vencimento abaixo do nome dos endpoints com certificado

#### Scenario: Ligar as mensagens
- **WHEN** um administrador marca "Show messages" na página `jobs` e salva
- **THEN** a definição salva tem `show-messages: true`
- **AND** a página pública de detalhes dos endpoints de `jobs` mostra a tabela de verificações com Message e Origin

#### Scenario: Formulário sem rolagem da página
- **WHEN** um administrador abre, numa janela de 1280×900, a edição de uma página com mais endpoints do que cabem na coluna
- **THEN** a página não rola na vertical nem na horizontal
- **AND** os botões Back, Validate, Preview e Save ficam inteiros dentro da janela
- **AND** a lista de endpoints rola por dentro, com altura maior que a altura máxima usada no celular

#### Scenario: Pré-visualização em diálogo
- **WHEN** o administrador de uma página já salva clica em Preview e depois aperta Esc
- **THEN** a pré-visualização abre num diálogo e fecha com o Esc, sem perder o que foi digitado
- **AND** o foco volta para o botão Preview

#### Scenario: Pré-visualização de página não salva
- **WHEN** o administrador clica em Preview numa página que ainda não foi salva
- **THEN** nenhum diálogo abre e um toast avisa que a página precisa ser salva para ser pré-visualizada

#### Scenario: Validação em toast
- **WHEN** o administrador clica em Validate numa página sem avisos
- **THEN** um toast de informação diz quantos endpoints a página vai mostrar
- **AND** a faixa de avisos da validação não aparece

#### Scenario: Toast da criação sobrevive à navegação
- **WHEN** o administrador salva uma página nova e a tela passa para a edição dela
- **THEN** o toast de sucesso da criação aparece na tela de edição

#### Scenario: Conflito de versão não é apagado
- **WHEN** o save devolve 412, o administrador clica em Validate e depois volta ao formulário
- **THEN** o aviso do conflito e o botão de recarregar continuam na tela, com o que foi digitado

#### Scenario: YAML somente leitura rola por dentro
- **WHEN** o administrador abre uma página definida no YAML, com uma definição longa, numa janela de 1280×900
- **THEN** o YAML rola dentro da própria área
- **AND** o aviso de que a página só pode ser vista continua visível

#### Scenario: Página com login na lista e no formulário
- **WHEN** o administrador marca "Require login to view this page" na página `clientes`, preenche usuário e senha e salva
- **THEN** a lista marca `clientes` como página que exige login
- **AND** ao reabrir o formulário o usuário aparece preenchido e a senha vazia, com a explicação de que vazio mantém a senha atual

#### Scenario: Grupos recolhidos por padrão
- **WHEN** um administrador marca "Start with the groups collapsed" na página `clientes` e salva
- **THEN** a definição salva tem `groups-collapsed: true`
- **AND** a pré-visualização mostra os grupos operacionais recolhidos, sem ler nem gravar as escolhas guardadas da página pública
