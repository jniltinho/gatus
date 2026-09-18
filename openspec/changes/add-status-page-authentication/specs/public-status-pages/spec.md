## MODIFIED Requirements

### Requirement: Acesso público sem autenticação
`GET /status/:slug`, qualquer caminho sob `/status/`, `GET /api/v1/status-pages/:slug` e qualquer outro caminho ou método sob `/api/v1/status-pages` MUST ser atendidos sem credenciais nem sessão, com qualquer configuração de `security` e de `status-pages.enabled`, **exceto as rotas de uma página que exija login** (requisito "Páginas com login próprio"). Fora essa exceção, nenhuma dessas respostas MUST ser 401 nem incluir `WWW-Authenticate`, e elas MUST ser iguais para requisições anônimas e autenticadas. As rotas de status já protegidas MUST continuar exigindo autenticação.

O `security` da instalação MUST NOT valer para as rotas públicas: a credencial ou a sessão da administração MUST NOT abrir uma página que exige login, e a credencial de uma página MUST NOT abrir nenhuma rota protegida nem outra página.

A rota HTML `/status/*` MUST responder sempre 200 com o HTML da SPA, para `GET` e `HEAD`, sem revelar se a página existe — **exceto** o caminho de uma página que exige login e os caminhos sob ele, que respondem 401 em `GET` e em `HEAD`.

#### Scenario: Basic auth configurado
- **WHEN** a configuração usa `security.basic` e uma requisição sem credenciais pede `GET /api/v1/status-pages/infra`
- **THEN** a API responde 200 sem `WWW-Authenticate`
- **AND** `GET /api/v1/endpoints/statuses` sem credenciais continua respondendo 401

#### Scenario: OIDC sem sessão
- **WHEN** a configuração usa `security.oidc` e uma requisição sem cookie de sessão pede `GET /status/infra` e `GET /api/v1/status-pages/infra`
- **THEN** as duas respondem 200 sem redirecionar para o provedor OIDC

#### Scenario: Caminhos fora do padrão da API
- **WHEN** a configuração usa `security.basic` e chegam, sem credenciais, `GET /api/v1/status-pages/`, `GET /api/v1/status-pages/a/b`, `GET /api/v1/status-pages/infra/extra` e `POST /api/v1/status-pages/infra`
- **THEN** todas respondem o 404 idêntico, sem `WWW-Authenticate`

#### Scenario: Rota HTML de página inexistente
- **WHEN** chega `HEAD /status/nao-existe` e `GET /status/a%2Fb`
- **THEN** as duas respondem 200 com os mesmos cabeçalhos de `GET /status/infra`

### Requirement: Validação das páginas
A validação do arquivo de configuração MUST ser estrutural e MUST recusar:
- `slug` fora de `^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`;
- `slug` reservado (`options`, `validate`, `new`, `preview`, `exposure`);
- `slug` repetido;
- `title` vazio ou com mais de 100 runas depois de remover espaços das pontas;
- `description` com mais de 1000 runas;
- página sem nenhum item em `groups` e em `endpoints`;
- mais de 50 grupos, grupo com mais de 200 runas ou mais de 200 chaves;
- itens repetidos;
- entradas de `trusted-proxies` que não sejam IP nem CIDR;
- `rate-limit` negativo.

Um grupo ou uma chave sem endpoint correspondente MUST NOT invalidar a página. O aviso correspondente MUST ser registrado na carga das páginas, depois da carga dos endpoints gerenciados, e MUST NOT ser registrado na validação do arquivo.

#### Scenario: Slug inválido
- **WHEN** uma página do YAML tem `slug: Infra_1`
- **THEN** a configuração é inválida com uma mensagem citando o slug

#### Scenario: Slug reservado
- **WHEN** uma página do YAML tem `slug: options`
- **THEN** a configuração é inválida

#### Scenario: Slug repetido
- **WHEN** duas páginas do YAML têm `slug: infra`
- **THEN** a configuração é inválida

#### Scenario: Página sem seleção
- **WHEN** uma página do YAML não tem `groups` nem `endpoints`
- **THEN** a configuração é inválida

#### Scenario: Título com acentos
- **WHEN** o título tem 100 caracteres acentuados (mais de 100 bytes)
- **THEN** a configuração é válida

#### Scenario: Proxy confiável inválido
- **WHEN** `status-pages.trusted-proxies` contém `nginx.local`
- **THEN** a configuração é inválida

#### Scenario: Grupo ainda inexistente
- **WHEN** uma página tem `groups: [pagamentos]` e nenhum endpoint do YAML nem gerenciado usa esse grupo
- **THEN** a configuração é válida
- **AND** a carga das páginas registra no log um aviso citando a página e o grupo

#### Scenario: Grupo existente só em gerenciados
- **WHEN** uma página do YAML tem `groups: [clientes]` e só endpoints gerenciados usam esse grupo
- **THEN** nenhum aviso de grupo sem correspondência é registrado

**Credencial da página:** com `auth` na definição, o usuário MUST ser não vazio e a senha MUST ser um hash bcrypt válido, codificado em base64 com o alfabeto URL, como em `security.basic`. Uma definição com `auth` incompleto ou com hash inválido MUST ser recusada, com a mesma severidade das demais validações de página. Sem `auth`, a página continua pública. Uma página com `auth` numa instalação **sem** `security` MUST registrar um aviso na carga, porque as rotas por chave do dashboard continuam abertas e publicam mais do que a página protegida.

#### Scenario: Credencial incompleta
- **WHEN** uma página traz `auth` com o usuário vazio, ou com um hash que não é bcrypt
- **THEN** a definição é recusada com erro de validação

#### Scenario: Página com login sem security na instalação
- **WHEN** o arquivo de configuração define uma página com `auth` e não define `security`
- **THEN** a carga registra um aviso de que as rotas por chave continuam públicas

### Requirement: Cabeçalhos das rotas públicas
As respostas das rotas públicas MUST incluir `X-Robots-Tag: noindex, nofollow`, `X-Content-Type-Options: nosniff` e `Referrer-Policy: strict-origin-when-cross-origin`. As respostas da API pública MUST incluir `Vary: Accept-Encoding`, com ou sem compressão. A API MUST responder `Cache-Control: no-cache` com 200 e `Cache-Control: no-store` com 401, 404, 429 e 503, para que nenhum cache HTTP mantenha no ar uma página desabilitada. As respostas 200 das rotas de uma página que exige login MUST usar `Cache-Control: private, no-store` no lugar de `no-cache`, e o canal de eventos dessa página MUST usar `private, no-cache, no-store, no-transform`, para nenhum cache compartilhado guardar conteúdo restrito. Os canais de eventos MUST responder 200 com `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-store, no-transform` e `X-Accel-Buffering: no`, sem compressão, e 429 e 503 com `Cache-Control: no-store` e os corpos JSON das demais respostas da API pública. A rota HTML MUST responder `Cache-Control: no-cache`. A API pública MUST NOT enviar cabeçalhos CORS.

#### Scenario: Página publicada
- **WHEN** chega `GET /api/v1/status-pages/infra` para uma página publicada
- **THEN** a resposta inclui `X-Robots-Tag: noindex, nofollow`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `Vary: Accept-Encoding` e `Cache-Control: no-cache`

#### Scenario: Rota HTML
- **WHEN** chega `GET /status/infra`
- **THEN** a resposta inclui `X-Robots-Tag: noindex, nofollow` e `Cache-Control: no-cache`

#### Scenario: Canal de eventos
- **WHEN** chega `GET /api/v1/status-pages/infra/endpoints/core_api/events` com `Accept-Encoding: br`
- **THEN** a resposta é 200 sem compressão, com `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-store, no-transform`, `X-Accel-Buffering: no` e os cabeçalhos `X-Robots-Tag`, `X-Content-Type-Options` e `Referrer-Policy`

#### Scenario: Página com login
- **WHEN** chega `GET /api/v1/status-pages/clientes` com a credencial certa de uma página que exige login
- **THEN** a resposta é 200 com `Cache-Control: private, no-store`
- **AND** sem credencial a resposta é 401 com `Cache-Control: no-store`



## ADDED Requirements

### Requirement: Páginas com login próprio
Uma página MAY exigir usuário e senha para ser vista. Com `auth` na definição, as rotas daquela página MUST exigir autenticação HTTP Basic com a credencial **daquela página**:

- a rota HTML `/status/<slug>` e os caminhos sob ela, em `GET` e em `HEAD`;
- `GET /api/v1/status-pages/<slug>` e as rotas de detalhes, de eventos e do gráfico de tempo de resposta dos endpoints dela;
- as rotas de badge da página (`/api/v1/status-pages/<slug>/endpoints/<chave>/health/badge.svg` e `.../response-times/<período>/badge.svg`), que a página de detalhes MUST usar no lugar das rotas globais por chave.

As rotas globais por chave (`/api/v1/endpoints/<chave>/...`) MUST continuar públicas, como no Gatus original: proteger uma página esconde o conjunto que ela publica, não cada número de um endpoint cuja chave já seja conhecida, e a documentação MUST dizer isso.

Sem credencial, ou com credencial errada, a resposta MUST ser 401 com `WWW-Authenticate: Basic realm="<slug>", charset="UTF-8"`, com o slug vindo da definição publicada, e o cabeçalho MUST ser enviado em qualquer requisição, inclusive as que parecem de navegador. A verificação MUST comparar o usuário em tempo constante e a senha com bcrypt, sempre as duas. Sessão, OIDC e a credencial do `security` da instalação MUST ser ignorados por essa verificação.

A requisição MUST resolver a página uma única vez: a definição capturada na autorização MUST ser a mesma usada para montar a resposta. As regras **de página** — inexistente, não publicada, em conflito, `status-pages.enabled: false` e caminho fora do padrão da API — MUST continuar respondendo o mesmo 404 de hoje, sem `WWW-Authenticate`, **antes** do desafio; já o 404 de uma chave que não pertence à página MUST vir **depois** do 401, para que a resposta não diga quais endpoints a página tem.

Tentativas com credencial errada MUST entrar num limite por página e por IP do cliente, resolvido com `status-pages.trusted-proxies`: uma página bloqueada MUST NOT bloquear outra. Estourado o limite, a resposta MUST ser 429 com `Retry-After`, sem comparar a senha, inclusive para quem apresentar a credencial certa dentro da janela. Uma verificação bem-sucedida MUST poder ser memorizada por no máximo cinco minutos para não repetir o bcrypt, com a memória perdendo efeito quando o usuário ou o hash da página mudar, e sem que credenciais diferentes possam colidir na mesma entrada.

Uma página sem `auth` MUST continuar respondendo exatamente como hoje, sem 401 e sem `WWW-Authenticate`.

#### Scenario: Página protegida sem credencial
- **WHEN** chegam, sem credencial, `GET /status/clientes`, `HEAD /status/clientes`, `GET /status/clientes/endpoints/core_api`, `GET /api/v1/status-pages/clientes` e o badge da página
- **THEN** todas respondem 401 com `WWW-Authenticate: Basic` e `Cache-Control: no-store`

#### Scenario: Requisição que parece de navegador
- **WHEN** a requisição sem credencial traz `Sec-Fetch-Site: same-origin` e `X-Requested-With`
- **THEN** a resposta continua 401 **com** `WWW-Authenticate`

#### Scenario: Página protegida com a credencial certa
- **WHEN** a requisição traz a credencial da página
- **THEN** a resposta é 200 com o mesmo conteúdo de uma página pública equivalente e `Cache-Control: private, no-store`

#### Scenario: Credencial de outra origem não serve
- **WHEN** a requisição traz a credencial ou a sessão do `security` da instalação, ou a credencial de outra status page
- **THEN** a resposta é 401

#### Scenario: Chave que não está na página
- **WHEN** chega, sem credencial, o detalhe de uma chave que não pertence à página protegida
- **THEN** a resposta é 401, e não 404

#### Scenario: Página protegida e desabilitada
- **WHEN** a página com `auth` está desabilitada e chega `GET /api/v1/status-pages/clientes` sem credencial
- **THEN** a resposta é o 404 de sempre, sem `WWW-Authenticate`

#### Scenario: Excesso de tentativas numa página
- **WHEN** um mesmo cliente erra a senha da página `clientes` mais vezes que o limite dentro da janela
- **THEN** as requisições seguintes a `clientes` respondem 429 com `Retry-After`, sem comparar a senha, mesmo com a credencial certa
- **AND** a página `parceiros` continua aceitando a credencial dela no mesmo IP

#### Scenario: Página pública não muda
- **WHEN** chega `GET /api/v1/status-pages/infra` sem credencial, para uma página sem `auth`
- **THEN** a resposta é 200, sem `WWW-Authenticate`
