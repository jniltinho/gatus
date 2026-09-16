## Why

Depois de um push, a página de detalhes do endpoint só mostra o resultado novo na próxima atualização automática: 5 minutos por padrão no dashboard e 60 segundos na página pública. E o gráfico **Response Time Trend** nem acompanha essas atualizações, porque só busca os dados ao abrir a página ou ao trocar o período. O dono do fork quer ver o push, inclusive o Pending em amarelo, no gráfico e na página no momento em que ele chega, em tempo real, com Server-Sent Events (SSE).

## What Changes

- **Aviso em tempo real de resultado novo:** um canal SSE por endpoint avisa o navegador quando o endpoint grava um resultado, seja verificação, push ou heartbeat:
  - `GET /api/v1/endpoints/{key}/events`, protegida como as outras rotas de status;
  - `GET /api/v1/status-pages/{slug}/endpoints/{key}/events`, pública, só para endpoints mostrados por uma página publicada.
- **Aviso sem dados:** o evento só diz que há resultado novo, com uma sequência para reconectar sem perder avisos. A página busca os dados pelas rotas que já existem, então nada novo é publicado e a sanitização das status pages não muda.
- **Detalhes do endpoint, no dashboard e na página pública:** ao receber o aviso, a página atualiza barras, painel de números, gráfico (linha e faixas vermelhas e amarelas) e tabela de checks. O gráfico passa a recarregar a cada atualização da página, inclusive nas automáticas.
- **Limites e robustez:**
  - número máximo de conexões abertas no total e por IP;
  - `ping` periódico, reconexão automática com `Last-Event-ID` e tempo máximo por conexão;
  - sem compressão nem buffer de proxy (`X-Accel-Buffering: no`);
  - fechamento das conexões antes de parar o servidor, para a recarga da configuração não ficar presa.
- **Página pública:** a resposta de detalhes em cache é renovada assim que chega um resultado novo do endpoint, sem esperar os 30 segundos.
- **Fallback:** sem SSE (limite atingido, proxy sem suporte, outra instância), as páginas continuam com a atualização periódica atual.
- **Documentação:** rotas, limites, configuração do nginx e várias instâncias.

## Capabilities

### New Capabilities

- `realtime-endpoint-updates`: canal SSE de resultados novos por endpoint (protegido e público), limites, ciclo de vida e atualização em tempo real das páginas de detalhes.

### Modified Capabilities

- `status-page-highlights`: a API pública de detalhes renova o cache quando o endpoint grava um resultado novo.
- `public-status-pages`: exceção do limite de conexões para os canais de eventos e cabeçalhos das respostas de eventos.
- `basic-login-page`: `Accept: text/event-stream` também evita o `WWW-Authenticate` num 401.

## Impact

- **Backend:**
  - novo pacote do fork para distribuir os avisos (sem goroutine por endpoint);
  - avisos publicados em `watchdog` depois de gravar cada resultado;
  - rotas SSE em `api/`;
  - `compress` sem as rotas de eventos;
  - `controller` com `HeaderReceived` do fasthttp, para dar às rotas de eventos um tempo de escrita maior que os 15 segundos padrão;
  - `main.go` fecha as conexões antes de `controller.Shutdown`;
  - chave do cache de detalhes em `statuspage/public.go`.
- **Frontend:** utilitário de `EventSource` com pausa em aba oculta, reconexão e fallback; `EndpointDetails.vue`, `StatusPageEndpoint.vue` e `ResponseTimeChart.vue`.
- **Infra:** o nginx precisa de `proxy_buffering off` (ou do `X-Accel-Buffering: no` enviado pelo Gatus) e de `proxy_read_timeout` maior que o intervalo do `ping`. Isso fica documentado.
- **Fora do escopo:** a lista do dashboard (`/`), as páginas de status com todos os endpoints e as suites.
- **Várias instâncias:** o aviso só sai da instância que gravou o resultado. As outras seguem com a atualização periódica.
