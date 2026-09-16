## MODIFIED Requirements

### Requirement: Limite de requisições por IP
A API pública MUST limitar cada IP de cliente a `status-pages.rate-limit` respostas 404 por minuto, numa janela deslizante, contando os 404 da rota específica e dos caminhos fora do padrão. Requisições a uma página publicada (servida do cache, montada ou respondida com 503) MUST NOT contar nem ser bloqueadas, mesmo com o limite do IP esgotado. A exceção são os canais de eventos dos endpoints de uma página publicada (`/api/v1/status-pages/{slug}/endpoints/{key}/events`), que MUST respeitar o limite de conexões abertas por IP e no total dos canais de eventos, com 429, sem contar no limite de 404. Ao exceder, a API MUST responder 429 com `Retry-After`, `Cache-Control: no-store` e `{"error":"too many requests"}`.

O IP do cliente MUST ser o IP da conexão. Quando esse IP estiver em `trusted-proxies`, MUST ser o primeiro IP fora de `trusted-proxies` ao percorrer da direita para a esquerda todas as linhas de `X-Forwarded-For`, na ordem de chegada; com header ausente, entrada inválida, mais de 20 entradas ou linha acima de 1 KB, MUST ser o IP da conexão. Entradas com porta (`IP:porta`, `[v6]:porta`) MUST ser aceitas. O IP da conexão e as entradas MUST ser normalizados (IPv4 mapeado em IPv6 vira IPv4) antes da comparação com `trusted-proxies`. Endereços IPv6 MUST ser agregados por /64 na chave do limite. O comportamento de `c.IP()` no restante da aplicação MUST NOT mudar.

O limitador MUST manter no máximo 50 000 chaves, descartando as mais antigas, MUST NOT criar goroutines e MUST ser reaproveitado entre ciclos de recarga. Na primeira requisição de cada ciclo vinda de IP fora de `trusted-proxies` que seja privado (RFC 1918, `100.64.0.0/10`, `fc00::/7`), loopback ou link-local e traga `X-Forwarded-For`, o sistema MUST registrar um único aviso de limite compartilhado citando o IP; depois de uma recarga, o aviso MUST poder aparecer de novo.

#### Scenario: Limite excedido
- **WHEN** `rate-limit` é 120 e o mesmo IP recebe a 121ª resposta 404 no mesmo minuto
- **THEN** a API responde 429 com `Retry-After`
- **AND** a resposta não inclui cabeçalhos `X-RateLimit-*`

#### Scenario: Página publicada nunca é limitada
- **WHEN** `rate-limit` é 10, o mesmo IP já recebeu 10 respostas 404 no minuto e em seguida pede 500 vezes uma página publicada, inclusive depois de o cache da página expirar
- **THEN** todas as respostas da página publicada são 200

#### Scenario: Limite desligado
- **WHEN** `rate-limit` é 0
- **THEN** 500 requisições 404 do mesmo IP no mesmo minuto respondem 404

#### Scenario: Header forjado por cliente direto
- **WHEN** `trusted-proxies` está vazio e um cliente envia `X-Forwarded-For` diferente a cada requisição
- **THEN** todas as requisições contam para o IP da conexão

#### Scenario: Atrás do nginx no Docker
- **WHEN** `trusted-proxies` contém `172.30.0.1/32`, a conexão vem de `172.30.0.1` e o nginx repassa `X-Forwarded-For: 203.0.113.9, 198.51.100.7`
- **THEN** a requisição conta para o IP `198.51.100.7`

#### Scenario: Proxy com endereço IPv4 mapeado
- **WHEN** `trusted-proxies` contém `172.30.0.1/32`, a conexão aparece como `::ffff:172.30.0.1` e traz `X-Forwarded-For: 198.51.100.7`
- **THEN** a requisição conta para o IP `198.51.100.7`

#### Scenario: Header em duas linhas
- **WHEN** a conexão vem de um proxy confiável e chegam as linhas `X-Forwarded-For: 203.0.113.9` e `X-Forwarded-For: 198.51.100.7`
- **THEN** a requisição conta para o IP `198.51.100.7`

#### Scenario: Header gigante
- **WHEN** a conexão vem de um proxy confiável e `X-Forwarded-For` tem 10 000 entradas
- **THEN** a requisição conta para o IP da conexão

#### Scenario: Proxy não configurado
- **WHEN** `trusted-proxies` está vazio e chegam várias requisições de `172.30.0.1` com `X-Forwarded-For`
- **THEN** o log registra uma única vez, no ciclo, o aviso de limite compartilhado citando `172.30.0.1`
- **AND** depois de uma recarga da configuração, a próxima requisição nas mesmas condições registra o aviso de novo

#### Scenario: Recargas sucessivas
- **WHEN** a configuração é recarregada 20 vezes
- **THEN** o número de goroutines do processo não cresce por causa do limitador

#### Scenario: Canal de eventos acima do limite de conexões
- **WHEN** o IP de um visitante já tem 10 canais de eventos abertos e abre mais um para um endpoint de página publicada
- **THEN** a resposta é 429 com `Retry-After: 30`, `Cache-Control: no-store` e `{"error":"too many requests"}`
- **AND** o limite de respostas 404 do IP não muda

### Requirement: Cabeçalhos das rotas públicas
As respostas das rotas públicas MUST incluir `X-Robots-Tag: noindex, nofollow`, `X-Content-Type-Options: nosniff` e `Referrer-Policy: strict-origin-when-cross-origin`. As respostas da API pública MUST incluir `Vary: Accept-Encoding`, com ou sem compressão. A API MUST responder `Cache-Control: no-cache` com 200 e `Cache-Control: no-store` com 404, 429 e 503, para que nenhum cache HTTP mantenha no ar uma página desabilitada. Os canais de eventos MUST responder 200 com `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-store, no-transform` e `X-Accel-Buffering: no`, sem compressão, e 429 e 503 com `Cache-Control: no-store` e os corpos JSON das demais respostas da API pública. A rota HTML MUST responder `Cache-Control: no-cache`. A API pública MUST NOT enviar cabeçalhos CORS.

#### Scenario: Página publicada
- **WHEN** chega `GET /api/v1/status-pages/infra` para uma página publicada
- **THEN** a resposta inclui `X-Robots-Tag: noindex, nofollow`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `Vary: Accept-Encoding` e `Cache-Control: no-cache`

#### Scenario: Rota HTML
- **WHEN** chega `GET /status/infra`
- **THEN** a resposta inclui `X-Robots-Tag: noindex, nofollow` e `Cache-Control: no-cache`

#### Scenario: Canal de eventos
- **WHEN** chega `GET /api/v1/status-pages/infra/endpoints/core_api/events` com `Accept-Encoding: br`
- **THEN** a resposta é 200 sem compressão, com `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-store, no-transform`, `X-Accel-Buffering: no` e os cabeçalhos `X-Robots-Tag`, `X-Content-Type-Options` e `Referrer-Policy`
