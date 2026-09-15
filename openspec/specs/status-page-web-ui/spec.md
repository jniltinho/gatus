# status-page-web-ui Specification

## Purpose
TBD - created by archiving change add-public-status-pages. Update Purpose after archive.
## Requirements
### Requirement: Página pública de status
A rota `/status/:slug` do frontend MUST mostrar:
- logo e cabeçalho da configuração `ui`, título e descrição da página;
- uma faixa com o estado geral em texto e cor ("Todos os sistemas operacionais", "Degradação parcial", "Indisponível" ou "Sem dados");
- uma seção por grupo com o estado do grupo, com a seção sem grupo rotulada "Outros serviços";
- para cada endpoint: nome, indicador de estado, uptime de 24h, 7d e 30d ("—" quando indisponível) e barras dos últimos resultados, com tooltip de horário, sucesso e duração em milissegundos.

Com `truncated: true` no payload, a página MUST mostrar o aviso "Mostrando os primeiros 200 serviços". A página MUST NOT mostrar anúncios, `ui.buttons`, links sociais, o link Admin nem "Powered by". A descrição MUST ser exibida como texto puro. A página MUST definir `document.title` com o título da página, seguir o visual quadrado (exceto indicadores circulares), ter variantes para o tema escuro e funcionar em telas a partir de 360 px de largura.

#### Scenario: Página com falha parcial
- **WHEN** um visitante abre `/status/infra` e um endpoint do grupo `core` está fora
- **THEN** a faixa mostra "Degradação parcial"
- **AND** o grupo `core` e o endpoint aparecem com estado de falha

#### Scenario: Página truncada
- **WHEN** a resposta da API tem `truncated: true`
- **THEN** a página mostra "Mostrando os primeiros 200 serviços"

#### Scenario: Descrição com HTML
- **WHEN** a descrição da página é `<img src=x onerror=alert(1)>`
- **THEN** o texto aparece literalmente e nenhum script é executado

#### Scenario: Tela de celular
- **WHEN** a página é aberta com 390 px de largura
- **THEN** o conteúdo cabe na largura sem rolagem horizontal e mostra até 25 barras por endpoint

### Requirement: Acessibilidade da página pública
A página pública MUST:
- oferecer para cada endpoint um resumo textual acessível a leitores de tela com nome, estado, verificações com sucesso e uptime de 24h;
- marcar as barras de histórico com `aria-hidden`;
- permitir abrir o tooltip por teclado e por toque;
- usar `role="status"` só na faixa de estado geral, com o contador "Atualizado há X" fora de regiões `aria-live`;
- definir o idioma do documento como `pt-BR` enquanto estiver no layout público;
- respeitar `prefers-reduced-motion`.

#### Scenario: Leitor de tela
- **WHEN** um leitor de tela percorre a linha do endpoint `api`
- **THEN** ele anuncia um texto como "api: no ar, 48 de 50 verificações com sucesso, uptime 24h 99,9%"
- **AND** as barras individuais não são anunciadas

#### Scenario: Tooltip por teclado
- **WHEN** o visitante navega com Tab até o histórico de um endpoint e aciona um resultado
- **THEN** o tooltip com horário, sucesso e duração aparece

### Requirement: Layout público sem login
As rotas públicas (`/status/:slug` e o catch-all `/status/*`) MUST ter `meta.public` e MUST ser renderizadas num layout próprio que reage à rota atual, inclusive em navegação dentro da SPA. No layout público, o frontend MUST NOT:
- mostrar a tela de login do OIDC;
- mostrar os botões de `ui.buttons`, o link Admin, os anúncios ou as configurações de atualização do dashboard;
- buscar `/api/v1/config`.

A configuração MUST ser buscada só quando o visitante entrar numa rota não pública, e só então o intervalo de atualização da configuração MUST começar. A view pública MUST validar o slug com a regex das páginas antes de chamar a API, MUST codificar o slug com `encodeURIComponent`, MUST chamar apenas `/api/v1/status-pages/:slug` e MUST NOT navegar para rotas que exigem autenticação. Um caminho sob `/status/` que não seja um slug válido MUST mostrar "Página não encontrada" sem chamar a API.

#### Scenario: OIDC sem sessão
- **WHEN** a configuração usa OIDC e um visitante sem sessão abre `/status/infra`
- **THEN** a página de status aparece sem a tela "Login with OIDC"
- **AND** nenhuma requisição é feita a `/api/v1/config`

#### Scenario: Da página pública para o dashboard
- **WHEN** a configuração usa OIDC, o visitante sem sessão está em `/status/infra` e navega para `/`
- **THEN** o frontend busca `/api/v1/config` nesse momento e mostra a tela "Login with OIDC"

#### Scenario: Basic auth
- **WHEN** a configuração usa basic auth e um visitante sem credenciais abre `/status/infra`
- **THEN** o navegador não pede usuário e senha
- **AND** nenhuma requisição recebe resposta 401

#### Scenario: Link malicioso com barra codificada
- **WHEN** a configuração usa basic auth e um visitante sem credenciais abre `/status/a%2Fb`
- **THEN** a view mostra "Página não encontrada"
- **AND** nenhuma requisição recebe resposta 401

#### Scenario: Troca de slug na mesma aba
- **WHEN** o visitante navega de `/status/infra` para `/status/clientes` dentro da SPA
- **THEN** os dados de `infra` somem, a view busca `clientes` e a atualização periódica passa a usar `clientes`

#### Scenario: Clique num endpoint
- **WHEN** o visitante clica no nome de um endpoint da página pública
- **THEN** a rota não muda para `/endpoints/:key`

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

### Requirement: Telas de administração de status pages
Com a administração habilitada e autorizada, o frontend MUST oferecer:
- as rotas `/admin/status-pages`, `/admin/status-pages/new` e `/admin/status-pages/:slug/edit`;
- abas para alternar entre endpoints e status pages.

A lista MUST mostrar slug, título, origem, estado (publicada, desabilitada, em conflito ou inválida) e número de endpoints, com as ações abrir (link em nova aba com `rel="noopener"`), copiar link, habilitar ou desabilitar, editar e remover (com confirmação). A lista MUST mostrar avisos quando a publicação estiver desligada no YAML, quando as páginas gerenciadas estiverem indisponíveis e quando houver aviso de limite compartilhado atrás de proxy.

O formulário MUST ter:
- slug (somente leitura na edição), título, descrição e `enabled`;
- seleção de grupos e de endpoints a partir de `/options`, com busca nos endpoints;
- avisos da validação e pré-visualização do payload público.

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

### Requirement: Aviso de exposição no formulário de endpoints
O formulário de endpoints da administração MUST consultar `/api/v1/admin/status-pages/exposure` com o grupo e a chave do endpoint ao abrir e quando o grupo ou o nome mudarem, e MUST mostrar em quais páginas públicas o endpoint vai aparecer, indicando as desabilitadas.

#### Scenario: Endpoint em grupo publicado
- **WHEN** a página publicada `infra` seleciona o grupo `core` e um administrador digita o grupo `core` num endpoint novo
- **THEN** o formulário mostra "Este endpoint aparecerá publicamente nas páginas: infra"

### Requirement: Testes ponta a ponta das status pages
O roteiro `test/e2e/status-pages.sh` MUST subir o Gatus compilado localmente com SQLite temporário, `security.basic` e `admin.enabled` e usar o `agent-browser` para:
- criar uma página pelas telas, pré-visualizar e publicar;
- abrir `/status/<slug>` numa sessão sem credenciais e conferir pela lista de requisições que nenhuma foi a `/api/v1/config` e nenhuma recebeu 401;
- abrir `/status/a%2Fb` e `/status/nao-existe` na mesma sessão e conferir "Página não encontrada" sem 401;
- conferir 404 da API para página desabilitada;
- simular OIDC sem sessão respondendo `/api/v1/config` com `{"oidc":true,"authenticated":false}`, conferir que a página pública não busca a configuração nem mostra a tela de login e, como controle, que `/` mostra "Login with OIDC";
- capturar a página pública nos temas claro e escuro, também com 390 px de largura.

As capturas MUST ficar em `dist/prints/status-pages/`, fora do git.

#### Scenario: Execução do roteiro
- **WHEN** um desenvolvedor executa `test/e2e/status-pages.sh` com `agent-browser` e Chrome instalados
- **THEN** todas as etapas passam
- **AND** as capturas ficam em `dist/prints/status-pages/`

