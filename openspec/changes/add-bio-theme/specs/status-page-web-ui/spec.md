## MODIFIED Requirements

### Requirement: Página pública de status
A rota `/status/:slug` do frontend MUST mostrar:
- logo e cabeçalho da configuração `ui`, título e descrição da página;
- uma faixa com o estado geral em texto e cor ("All systems operational", "Partial outage", "Major outage" ou "No data");
- na mesma faixa, a contagem dos endpoints da página por estado, com os que estão no ar e os que não estão sempre presentes (`12 up · 2 down`), e os pendentes e sem dados somente quando houver algum;
- uma seção por grupo cujo cabeçalho é um botão que recolhe e expande o grupo e mostra o nome, o estado do grupo e a contagem do `summary` do grupo, omitindo os estados com zero, com a seção sem grupo rotulada "Other services";
- para cada endpoint de um grupo expandido e para cada destaque: nome, indicador de estado, uptime de 24h, 7d e 30d ("—" quando indisponível) e barras dos últimos resultados, com tooltip de horário, sucesso e duração em milissegundos.

A contagem MUST vir do payload, e não ser somada no navegador, para continuar certa numa página truncada. Ela MUST ficar legível em todos os temas e MUST NOT provocar rolagem horizontal a partir de 360 px.

**Densidade e alinhamento:** a linha de um endpoint de grupo MUST ocupar no máximo 72 px de altura em telas a partir de 640 px, sem a linha opcional da expiração do certificado, e as barras do histórico MUST ter 20 px de altura. Em telas a partir de 640 px os uptimes MUST ficar em colunas de largura fixa à direita, alinhadas entre os endpoints do mesmo grupo, com os valores visíveis, e os rótulos `24h`, `7d` e `30d` MUST aparecer uma vez no cabeçalho do grupo, escondidos nas linhas e fora do alcance dos leitores de tela. Abaixo de 640 px cada linha MUST mostrar rótulo e valor juntos, e a página MUST continuar sem rolagem horizontal a partir de 360 px.

**Tooltip da verificação:** o detalhe da barra ativa MUST aparecer sobre as barras, sem ocupar espaço no layout: mostrar ou esconder o detalhe MUST NOT mudar a altura da linha nem a posição das linhas seguintes. O tooltip MUST ficar dentro da largura da linha, qualquer que seja a barra ativa, MUST NOT capturar o ponteiro e MUST ser escondido dos leitores de tela, que recebem o mesmo texto pela região `aria-live`. Onde houver conteúdo acima das barras — cartão de destaque e página de detalhes — o tooltip MUST aparecer abaixo delas.

Com `truncated: true` no payload, a página MUST mostrar o aviso "Showing the first 200 services". A página MUST NOT mostrar anúncios, `ui.buttons`, links sociais, o link Admin nem "Powered by". A descrição MUST ser exibida como texto puro. A página MUST definir `document.title` com o título da página, seguir o visual quadrado (exceto indicadores circulares), ter variantes para o tema escuro, as cores do tema Bio e funcionar em telas a partir de 360 px de largura.

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

### Requirement: Testes ponta a ponta das status pages
O roteiro `test/e2e/status-pages.sh` MUST subir o Gatus compilado localmente com SQLite temporário, `security.basic` e `admin.enabled` e usar o `agent-browser` para:
- criar uma página pelas telas, pré-visualizar e publicar;
- abrir `/status/<slug>` numa sessão sem credenciais e conferir pela lista de requisições que nenhuma foi a `/api/v1/config` e nenhuma recebeu 401;
- abrir `/status/a%2Fb` e `/status/nao-existe` na mesma sessão e conferir "Página não encontrada" sem 401;
- conferir 404 da API para página desabilitada;
- simular OIDC sem sessão respondendo `/api/v1/config` com `{"oidc":true,"authenticated":false}`, conferir que a página pública não busca a configuração nem mostra a tela de login e, como controle, que `/` mostra "Login with OIDC";
- capturar a página pública nos temas claro, escuro e Bio, também com 390 px de largura, e com 360 px no tema Bio com o seletor de tema aberto;
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
