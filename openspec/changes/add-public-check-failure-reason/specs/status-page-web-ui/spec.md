## MODIFIED Requirements

### Requirement: Página pública de status
A rota `/status/:slug` do frontend MUST mostrar:
- logo e cabeçalho da configuração `ui`, título e descrição da página;
- uma faixa com o estado geral em texto e cor ("All systems operational", "Partial outage", "Major outage" ou "No data");
- na mesma faixa, a contagem dos endpoints da página por estado, com os que estão no ar e os que não estão sempre presentes (`12 up · 2 down`), e os pendentes e sem dados somente quando houver algum;
- uma seção por grupo com o estado do grupo, com a seção sem grupo rotulada "Other services";
- para cada endpoint: nome, indicador de estado, uptime de 24h, 7d e 30d ("—" quando indisponível) e barras dos últimos resultados, com tooltip de horário, sucesso e duração em milissegundos.

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
