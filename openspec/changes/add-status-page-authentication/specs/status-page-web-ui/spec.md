## MODIFIED Requirements

### Requirement: Telas de administração de status pages
Com a administração habilitada e autorizada, o frontend MUST oferecer:
- as rotas `/admin/status-pages`, `/admin/status-pages/new` e `/admin/status-pages/:slug/edit`;
- abas para alternar entre endpoints e status pages.

A lista MUST mostrar slug, título, origem, estado (publicada, desabilitada, em conflito ou inválida), se a página exige login e número de endpoints, com as ações abrir (link em nova aba com `rel="noopener"`), copiar link, habilitar ou desabilitar, editar e remover (com confirmação). A lista MUST mostrar avisos quando a publicação estiver desligada no YAML, quando as páginas gerenciadas estiverem indisponíveis e quando houver aviso de limite compartilhado atrás de proxy.

O formulário MUST ter:
- slug (somente leitura na edição), título, descrição e `enabled`;
- a opção "Show certificate expiration" (`show-certificate-expiration`), desmarcada por padrão, com explicação curta;
- a opção "Show messages" (`show-messages`), desmarcada por padrão, ao lado da anterior, com a explicação de que as mensagens dos envios e o status HTTP ficam públicos e os erros das verificações não;
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


### Requirement: Atualização e estados de erro da página pública
A página pública MUST buscar os dados de novo a cada 60 s, pausar a atualização com a aba oculta e retomar ao voltar. MUST mostrar há quanto tempo os dados foram atualizados, calculado a partir de `updatedAt` e nunca negativo. Com resposta 404, MUST mostrar "Página não encontrada". Com 429, 503, erro de rede ou resposta que não seja JSON:
- havendo dados exibidos, MUST mantê-los, avisar a falha e tentar de novo no ciclo seguinte;
- sem dados, MUST mostrar uma mensagem de indisponibilidade e tentar de novo no ciclo seguinte.

Com `Retry-After` maior que 60 s, a próxima tentativa MUST respeitá-lo.

#### Scenario: Página removida enquanto aberta
- **WHEN** a página está aberta e é desabilitada pela administração
- **THEN** depois da próxima atualização a view mostra "Página não encontrada"

#### Scenario: Limite de requisições
- **WHEN** a atualização recebe 429
- **THEN** os dados anteriores continuam na tela com um aviso de falha na atualização

#### Scenario: Proxy fora do ar na primeira carga
- **WHEN** a primeira busca recebe 502 com HTML do nginx
- **THEN** a view mostra a mensagem de indisponibilidade, sem erro de JavaScript, e tenta de novo no ciclo seguinte

**Página que exige login:** ao receber 401 numa atualização, a página pública MUST parar o ciclo de atualização e pedir que a pessoa recarregue e entre de novo, em vez de repetir a tentativa e reabrir a caixa de credencial do navegador em laço. As buscas das rotas públicas da página MUST enviar as credenciais da mesma origem, senão a página autenticada carregaria vazia.

#### Scenario: Credencial trocada com a aba aberta
- **WHEN** a credencial da página muda enquanto um visitante está com ela aberta e a atualização periódica recebe 401
- **THEN** a página para de se atualizar e avisa que é preciso recarregar e entrar de novo


### Requirement: Testes ponta a ponta das status pages
O roteiro `test/e2e/status-pages.sh` MUST subir o Gatus compilado localmente com SQLite temporário, `security.basic` e `admin.enabled` e usar o `agent-browser` para:
- criar uma página pelas telas, pré-visualizar e publicar;
- abrir `/status/<slug>` numa sessão sem credenciais e conferir pela lista de requisições que nenhuma foi a `/api/v1/config` e nenhuma recebeu 401;
- abrir `/status/a%2Fb` e `/status/nao-existe` na mesma sessão e conferir "Página não encontrada" sem 401;
- conferir 404 da API para página desabilitada;
- simular OIDC sem sessão respondendo `/api/v1/config` com `{"oidc":true,"authenticated":false}`, conferir que a página pública não busca a configuração nem mostra a tela de login e, como controle, que `/` mostra "Login with OIDC";
- capturar a página pública nos temas claro e escuro, também com 390 px de largura;
- criar pela tela uma página que exige login e conferir, com requisições diretas e sem credencial, que a rota HTML, a API da página, os detalhes, os eventos, o gráfico e o badge da página respondem 401 com `WWW-Authenticate` e `Cache-Control: no-store`, e que com a credencial certa respondem 200;
- conferir que o detalhe da página na administração não traz o hash da credencial e que a lista marca a página como protegida;
- conferir que a página pública sem login continua respondendo 200 sem credencial;
- conferir, em `/status/messages`, que a linha de um endpoint de grupo cabe no teto de altura, que as colunas de uptime dos dois endpoints do grupo têm valor visível e começam na mesma posição, e que os rótulos dos períodos aparecem só no cabeçalho do grupo;
- conferir que o tooltip da última barra aparece sem mudar a altura da linha nem a posição da linha seguinte, sem capturar o ponteiro, dentro da largura da linha e sem rolagem horizontal, e que a região `aria-live` do detalhe existe vazia antes de qualquer barra ficar ativa;
- conferir a ausência de rolagem horizontal em 390 px e em 360 px, com o rótulo do período visível na linha do endpoint;
- conferir, na edição de uma página com mais endpoints do que cabem na coluna, que a tela não rola, que Back, Validate, Preview e Save ficam dentro da janela e que a lista de endpoints rola por dentro;
- conferir a pré-visualização no diálogo, fechada por Esc sem perder o que foi digitado, antes de continuar a usar o formulário;
- conferir o toast de informação da validação e a legenda do gráfico da página pública, pelos valores de `data-series`.

As capturas MUST ficar em `dist/prints/status-pages/`, fora do git.

#### Scenario: Execução do roteiro
- **WHEN** um desenvolvedor executa `test/e2e/status-pages.sh` com `agent-browser` e Chrome instalados
- **THEN** todas as etapas passam
- **AND** as capturas ficam em `dist/prints/status-pages/`
