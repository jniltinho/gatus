## ADDED Requirements

### Requirement: Canal SSE de resultados novos
O sistema MUST oferecer um canal Server-Sent Events por endpoint que avise, em até 1 segundo, cada resultado gravado para ele: verificação, push, heartbeat ou resultado da API upstream de external endpoints.

**Rotas:**
- `GET /api/v1/endpoints/{key}/events` MUST exigir a mesma autenticação das rotas de status e MUST responder 404 para uma chave sem registro;
- `GET /api/v1/status-pages/{slug}/endpoints/{key}/events` MUST ser pública e MUST responder o 404 idêntico das status pages, contando no limitador e sem ler o storage, quando a página não estiver publicada ou não mostrar a chave.

**Resposta:**
- cabeçalhos `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-store` e `X-Accel-Buffering: no`;
- sem compressão;
- começar com `retry: 3000`.

**Aviso:** cada resultado MUST gerar um evento `result` com `id` igual ao instante do resultado em milissegundos e `data` só com `{"timestamp": "<RFC 3339>"}`. O evento MUST NOT conter mensagem, erros, status, duração nem nenhum outro dado do resultado. Avisos acumulados para o mesmo cliente MAY ser juntados num só.

**Conexão:**
- quando o `Last-Event-ID` recebido for menor que o instante do último resultado conhecido do endpoint, a conexão MUST começar com um aviso;
- MUST enviar um comentário a cada 15 segundos;
- MUST durar no máximo 5 minutos, sem ser cortada pelo prazo de escrita padrão do servidor.

#### Scenario: Push aparece na hora
- **WHEN** um navegador está conectado a `/api/v1/endpoints/jobs_backup/events` e o endpoint recebe `status=pending`
- **THEN** o navegador recebe em até 1 segundo um evento `result` com o instante do resultado
- **AND** o evento não contém a mensagem do push

#### Scenario: Reconexão depois de um resultado perdido
- **WHEN** um cliente reconecta com `Last-Event-ID` anterior ao último resultado do endpoint
- **THEN** a conexão começa com um evento `result`

#### Scenario: Conexão longa
- **WHEN** um cliente fica conectado por 3 minutos sem resultados novos
- **THEN** a conexão continua aberta, com comentários a cada 15 segundos

#### Scenario: Endpoint fora da página pública
- **WHEN** um visitante pede `/api/v1/status-pages/infra/endpoints/database_pg/events` e `database_pg` não está na página `infra`
- **THEN** a resposta é o 404 idêntico das status pages

#### Scenario: Sem autenticação
- **WHEN** um navegador sem sessão pede `/api/v1/endpoints/jobs_backup/events` com `security.basic`
- **THEN** a resposta é 401 sem `WWW-Authenticate`

### Requirement: Limites e ciclo de vida das conexões de eventos
O sistema MUST limitar as conexões de eventos abertas a 500 no total e a 10 por IP de cliente, calculado com `status-pages.trusted-proxies`, somando as duas rotas. Acima do limite, MUST responder 429 com `Retry-After: 30`. Uma conexão encerrada pelo cliente, por erro de escrita ou pelo tempo máximo MUST liberar a vaga.

Antes de parar o servidor, numa recarga da configuração ou no desligamento, o sistema MUST encerrar as conexões de eventos, e a parada MUST NOT esperar o tempo máximo das conexões. Novas conexões MUST ser recusadas com 503 durante a parada. Remover ou renomear um endpoint, ou tirá-lo da configuração numa recarga, MUST descartar o último instante conhecido da chave.

#### Scenario: Limite por IP
- **WHEN** um IP já tem 10 conexões de eventos abertas e abre a 11ª
- **THEN** a resposta é 429 com `Retry-After: 30`

#### Scenario: Recarga com conexões abertas
- **WHEN** há 20 conexões de eventos abertas e o arquivo de configuração é recarregado
- **THEN** as conexões são encerradas e a recarga não espera 5 minutos

### Requirement: Detalhes do endpoint em tempo real
A página de detalhes do endpoint no dashboard e a página pública de detalhes MUST abrir o canal de eventos do endpoint e, a cada aviso, atualizar sem indicador de carregamento as barras, o painel de números, o gráfico **Response Time Trend** (linha, faixas vermelhas e faixas amarelas) e a tabela de checks.

- **Aba oculta:** a página MUST fechar o canal quando a aba ficar oculta e, ao voltar, MUST reabri-lo e atualizar uma vez.
- **Erro:** se o canal falhar (limite, proxy sem suporte ou erro de conexão), a página MUST continuar com a atualização periódica atual.
- **Gráfico:** o gráfico MUST recarregar a linha a cada atualização dos dados da página, inclusive as periódicas, sem apagar o gráfico durante a busca.

#### Scenario: Pending no gráfico na hora
- **WHEN** o administrador está nos detalhes de `jobs_backup` e um script envia `status=pending`
- **THEN** em até 2 segundos, a barra, a tabela e o gráfico mostram o Pending em amarelo, sem recarregar a página

#### Scenario: Página pública em tempo real
- **WHEN** um visitante está em `/status/jobs/endpoints/jobs_backup` e o endpoint recebe um push
- **THEN** a página mostra o resultado novo em até 2 segundos, mesmo com a resposta de detalhes ainda em cache

#### Scenario: Canal indisponível
- **WHEN** a abertura do canal responde 429
- **THEN** a página continua mostrando os dados e se atualiza no intervalo periódico
