## MODIFIED Requirements

### Requirement: Página pública de status
A rota `/status/:slug` do frontend MUST mostrar:
- logo e cabeçalho da configuração `ui`, título e descrição da página;
- uma faixa com o estado geral em texto e cor ("All systems operational", "Partial outage", "Major outage" ou "No data");
- uma seção por grupo com o estado do grupo, com a seção sem grupo rotulada "Other services";
- para cada endpoint: nome, indicador de estado, uptime de 24h, 7d e 30d ("—" quando indisponível) e barras dos últimos resultados, com tooltip de horário, sucesso e duração em milissegundos.

**Densidade e alinhamento:** a linha de um endpoint de grupo MUST ocupar no máximo 72 px de altura em telas a partir de 640 px, sem a linha opcional da expiração do certificado, e as barras do histórico MUST ter 20 px de altura. Em telas a partir de 640 px os uptimes MUST ficar em colunas de largura fixa à direita, alinhadas entre os endpoints do mesmo grupo, com os valores visíveis, e os rótulos `24h`, `7d` e `30d` MUST aparecer uma vez no cabeçalho do grupo, escondidos nas linhas e fora do alcance dos leitores de tela. Abaixo de 640 px cada linha MUST mostrar rótulo e valor juntos, e a página MUST continuar sem rolagem horizontal a partir de 360 px.

**Tooltip da verificação:** o detalhe da barra ativa MUST aparecer sobre as barras, sem ocupar espaço no layout: mostrar ou esconder o detalhe MUST NOT mudar a altura da linha nem a posição das linhas seguintes. O tooltip MUST ficar dentro da largura da linha, qualquer que seja a barra ativa, MUST NOT capturar o ponteiro e MUST ser escondido dos leitores de tela, que recebem o mesmo texto pela região `aria-live`. Onde houver conteúdo acima das barras — cartão de destaque e página de detalhes — o tooltip MUST aparecer abaixo delas.

Com `truncated: true` no payload, a página MUST mostrar o aviso "Showing the first 200 services". A página MUST NOT mostrar anúncios, `ui.buttons`, links sociais, o link Admin nem "Powered by". A descrição MUST ser exibida como texto puro. A página MUST definir `document.title` com o título da página, seguir o visual quadrado (exceto indicadores circulares), ter variantes para o tema escuro e funcionar em telas a partir de 360 px de largura.

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

#### Scenario: Linha enxuta
- **WHEN** um visitante abre `/status/services` numa janela de 1280×900
- **THEN** a linha de um endpoint de grupo, sem aviso de certificado, tem no máximo 72 px de altura

#### Scenario: Uptimes alinhados
- **WHEN** a página mostra dois endpoints no mesmo grupo numa janela de 1280×900
- **THEN** as colunas de 24h, 7d e 30d dos dois começam na mesma posição horizontal
- **AND** os rótulos dos períodos aparecem só no cabeçalho do grupo

#### Scenario: Tooltip não empurra a lista
- **WHEN** o visitante passa o ponteiro sobre a última barra do histórico de um endpoint
- **THEN** o detalhe com horário, resultado e duração aparece
- **AND** a altura da linha e a posição da linha seguinte continuam as mesmas
- **AND** o tooltip cabe na largura da linha e a página não ganha rolagem horizontal

#### Scenario: Região aria-live sem verificação ativa
- **WHEN** a página acaba de carregar e nenhuma barra está ativa
- **THEN** cada endpoint tem a região `aria-live` do detalhe, vazia
- **AND** nenhum tooltip está visível

### Requirement: Acessibilidade da página pública
A página pública MUST:
- oferecer para cada endpoint um resumo textual acessível a leitores de tela com nome, estado, verificações com sucesso e uptime de 24h;
- marcar as barras de histórico com `aria-hidden`;
- permitir abrir o tooltip por teclado e por toque;
- manter na página a região `aria-live` que anuncia o detalhe da verificação, mesmo quando não há verificação ativa, com o tooltip visível escondido dos leitores de tela para não repetir o texto;
- usar `role="status"` só na faixa de estado geral, com o contador "Atualizado há X" fora de regiões `aria-live`;
- respeitar `prefers-reduced-motion`.

#### Scenario: Leitor de tela
- **WHEN** um leitor de tela percorre a linha do endpoint `api`
- **THEN** ele anuncia um texto como "api: no ar, 48 de 50 verificações com sucesso, uptime 24h 99,9%"
- **AND** as barras individuais não são anunciadas

#### Scenario: Tooltip por teclado
- **WHEN** o visitante navega com Tab até o histórico de um endpoint e aciona um resultado
- **THEN** o tooltip com horário, sucesso e duração aparece

### Requirement: Testes ponta a ponta das status pages
O roteiro `test/e2e/status-pages.sh` MUST subir o Gatus compilado localmente com SQLite temporário, `security.basic` e `admin.enabled` e usar o `agent-browser` para:
- criar uma página pelas telas, pré-visualizar e publicar;
- abrir `/status/<slug>` numa sessão sem credenciais e conferir pela lista de requisições que nenhuma foi a `/api/v1/config` e nenhuma recebeu 401;
- abrir `/status/a%2Fb` e `/status/nao-existe` na mesma sessão e conferir "Página não encontrada" sem 401;
- conferir 404 da API para página desabilitada;
- simular OIDC sem sessão respondendo `/api/v1/config` com `{"oidc":true,"authenticated":false}`, conferir que a página pública não busca a configuração nem mostra a tela de login e, como controle, que `/` mostra "Login with OIDC";
- capturar a página pública nos temas claro e escuro, também com 390 px de largura;
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
