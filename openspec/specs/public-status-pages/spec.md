# public-status-pages Specification

## Purpose
TBD - created by archiving change add-public-status-pages. Update Purpose after archive.
## Requirements
### Requirement: Seção de configuração status-pages
O arquivo de configuração MUST aceitar a seção opcional `status-pages` com:
- `enabled` (booleano, padrão `true`);
- `trusted-proxies` (lista de IPs ou CIDRs, padrão vazia);
- `rate-limit` (inteiro não negativo, padrão `120`, `0` desliga o limite);
- `pages` (lista de páginas).

Cada página MUST aceitar `slug`, `title`, `description`, `groups`, `endpoints`, `show-certificate-expiration` (booleano, padrão `false`), `show-messages` (booleano, padrão `false`) e `enabled` (padrão `true` no YAML). Com `enabled: false` na seção, nenhuma página MUST ser publicada, e as rotas públicas MUST continuar respondendo como para uma página inexistente, sem `WWW-Authenticate`, com ou sem `security`.

#### Scenario: Página definida no YAML
- **WHEN** a configuração tem `status-pages.pages` com a página `infra`, título `Infraestrutura` e `groups: [core]`
- **THEN** a configuração é válida
- **AND** `GET /api/v1/status-pages/infra` responde 200

#### Scenario: Seção desligada
- **WHEN** a configuração tem `security.basic`, `status-pages.enabled: false` e a página `infra`
- **THEN** `GET /api/v1/status-pages/infra` responde o 404 idêntico sem `WWW-Authenticate`
- **AND** `GET /status/infra` responde 200 com o HTML da SPA

#### Scenario: Sem a seção
- **WHEN** a configuração não tem a seção `status-pages`
- **THEN** a configuração é válida
- **AND** `GET /api/v1/status-pages/qualquer` responde 404

#### Scenario: Página com a expiração do certificado
- **WHEN** a página `infra` do YAML tem `show-certificate-expiration: true`
- **THEN** a configuração é válida
- **AND** os endpoints da página com certificado têm `certificateExpiresInDays` no payload

#### Scenario: Página com mensagens
- **WHEN** a página `jobs` do YAML tem `show-messages: true`
- **THEN** a configuração é válida
- **AND** o payload de detalhes dos endpoints de `jobs` tem `page.showMessages: true`

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

### Requirement: Seleção dos endpoints da página
Uma página MUST incluir os endpoints publicáveis cujo `group`, depois de remover espaços das pontas, seja igual (diferenciando maiúsculas) a um item de `groups`, e os endpoints publicáveis cuja chave esteja em `endpoints`. São publicáveis:
- os endpoints habilitados do YAML;
- os external-endpoints habilitados;
- os endpoints gerenciados válidos, habilitados e sem conflito.

Suites, endpoints de suites, endpoints de instâncias `remote` e endpoints desabilitados MUST NOT ser incluídos. A seleção MUST ser recalculada a cada montagem, sem serializar objetos em monitoramento. Um endpoint incluído só pela chave MUST aparecer com o nome real do seu grupo.

#### Scenario: Endpoint novo no grupo
- **WHEN** a página `infra` seleciona o grupo `core` e um administrador cria o endpoint gerenciado `core/cache`
- **THEN** a próxima montagem da página inclui `cache` no grupo `core`

#### Scenario: Endpoint por chave
- **WHEN** a página seleciona `endpoints: [database_postgres]`
- **THEN** o endpoint `postgres` aparece na seção `database`, mesmo sem o grupo `database` em `groups`

#### Scenario: Endpoint desabilitado
- **WHEN** o endpoint `core/legacy` tem `enabled: false`
- **THEN** ele não aparece na página que seleciona o grupo `core`

#### Scenario: Suite no mesmo grupo
- **WHEN** existe uma suite com `group: core`
- **THEN** a suite e os endpoints dela não aparecem na página que seleciona o grupo `core`

#### Scenario: External endpoint
- **WHEN** um external-endpoint habilitado tem `group: core`
- **THEN** ele aparece na página que seleciona o grupo `core`

#### Scenario: Gerenciado em conflito
- **WHEN** um endpoint gerenciado está em conflito de chave com o YAML
- **THEN** só o endpoint do YAML aparece na página

### Requirement: Ordem de exibição
As seções MUST seguir a ordem de `groups`, seguidas, em ordem alfabética, dos grupos presentes só por `endpoints` e, por último, de uma seção com `name` vazio para os endpoints sem grupo. Dentro de cada seção, os endpoints MUST ser ordenados por nome sem diferenciar maiúsculas.

#### Scenario: Ordem das seções
- **WHEN** a página tem `groups: [web, core]` e `endpoints: [database_postgres, _ping]`, sendo `ping` um endpoint sem grupo
- **THEN** as seções aparecem na ordem `web`, `core`, `database` e a seção com `name` vazio

### Requirement: Acesso público sem autenticação
`GET /status/:slug`, qualquer caminho sob `/status/`, `GET /api/v1/status-pages/:slug` e qualquer outro caminho ou método sob `/api/v1/status-pages` MUST ser atendidos sem credenciais nem sessão, com qualquer configuração de `security` e de `status-pages.enabled`. Nenhuma dessas respostas MUST ser 401 nem incluir `WWW-Authenticate`, e elas MUST ser iguais para requisições anônimas e autenticadas. As rotas de status já protegidas MUST continuar exigindo autenticação.

A rota HTML `/status/*` MUST responder sempre 200 com o HTML da SPA, para `GET` e `HEAD`, sem revelar se a página existe.

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

### Requirement: Payload público sanitizado
A resposta de `GET /api/v1/status-pages/:slug` MUST conter apenas:
- `slug`, `title`, `description`, `status`, `updatedAt`, `truncated`, `featured` e `groups`;
- em cada grupo, `name`, `status` e `endpoints`;
- em cada destaque, os campos de endpoint e `group`;
- em cada endpoint, `name`, `status`, `uptime` (`24h`, `7d`, `30d`), `responseTime` (`24h`, `7d`, `30d`) e `results`, e `certificateExpiresInDays` somente quando a página tem `show-certificate-expiration: true` e o endpoint tem resultado publicado com certificado;
- em cada resultado, `timestamp`, `success` e `durationMs`, e `pending: true` somente quando o resultado é Pending.

MUST NOT conter nenhum outro campo, nem os valores de chave, URL, hostname, IP, porta, código HTTP, código DNS, erros, mensagens, condições, eventos, datas de expiração, alertas, `extra-labels` ou origem do endpoint. `certificateExpiresInDays` MUST ser um número inteiro de dias, sem data. `updatedAt` MUST ser o instante da montagem, no relógio do servidor.

Os resultados MUST ser os últimos `min(50, storage.maximum-number-of-results)`, do mais antigo para o mais recente. O uptime MUST ser `null` num período sem execuções, com qualquer tipo de storage. Uma página com mais de 200 endpoints MUST devolver os 200 primeiros na ordem de exibição, com `truncated: true`. As telas públicas MUST mostrar os resultados Pending em amarelo, com o rótulo "Pending".

#### Scenario: Resultado com dados sensíveis
- **WHEN** o último resultado do endpoint `core/api` tem hostname `10.0.0.5`, código HTTP 500, erro `dial tcp 10.0.0.5:443` e condições resolvidas
- **THEN** o JSON da página é decodificado sem erro por structs que só conhecem os campos permitidos, rejeitando campos desconhecidos
- **AND** não contém os textos `10.0.0.5` nem `dial tcp`

#### Scenario: Resultado Pending com mensagem
- **WHEN** o último resultado do endpoint Push `jobs/backup` é Pending com a mensagem `Aguardando disco`
- **THEN** o resultado no JSON da página tem `success: false` e `pending: true`
- **AND** o JSON da página não contém `Aguardando disco`, mesmo com `show-messages: true`

#### Scenario: Endpoint sem histórico de 30 dias
- **WHEN** o endpoint começou a ser monitorado há 2 horas
- **THEN** `uptime.24h`, `uptime.7d` e `uptime.30d` não são `null` e são calculados sobre as execuções existentes

#### Scenario: Endpoint sem execuções
- **WHEN** um endpoint selecionado ainda não tem nenhum registro no storage, com storage `memory`, `sqlite` ou `postgres`
- **THEN** ele aparece com `status: unknown`, `results` vazio e uptime `null` nos três períodos
- **AND** a resposta é 200

#### Scenario: Página grande
- **WHEN** a página seleciona 250 endpoints
- **THEN** a resposta tem 200 endpoints e `truncated: true`

#### Scenario: Expiração do certificado só com a opção
- **WHEN** o endpoint `core/site` tem resultados HTTPS com certificado e a página não tem `show-certificate-expiration`
- **THEN** o JSON da página não tem `certificateExpiresInDays` e é decodificado pelas structs dos campos permitidos
- **AND** com a opção ligada, o JSON tem `certificateExpiresInDays` inteiro para `site` e continua sem datas de expiração

### Requirement: Estados agregados
O estado de um endpoint MUST ser:
- `up` quando o último resultado teve sucesso;
- `pending` quando o último resultado é Pending;
- `down` quando o último resultado falhou sem ser Pending;
- `unknown` sem resultados.

O estado de um grupo e o da página MUST ser calculados sobre os endpoints conhecidos (ignorando `unknown`):
- `operational` se todos estiverem `up`;
- `down` se todos estiverem `down`;
- `degraded` em qualquer outra combinação, inclusive quando todos estiverem `pending`;
- `unknown` se nenhum for conhecido.

#### Scenario: Todos no ar
- **WHEN** todos os endpoints da página estão `up` e um está `unknown`
- **THEN** a página tem `status: operational`

#### Scenario: Falha parcial
- **WHEN** um endpoint do grupo `core` está `down` e os outros estão `up`
- **THEN** o grupo `core` e a página têm `status: degraded`

#### Scenario: Tudo fora
- **WHEN** todos os endpoints conhecidos da página estão `down`
- **THEN** a página tem `status: down`

#### Scenario: Endpoint Pending
- **WHEN** o grupo `jobs` tem um endpoint `pending` e os outros `up`
- **THEN** o endpoint tem `status: pending` e o grupo tem `status: degraded`

#### Scenario: Só Pending
- **WHEN** o único endpoint conhecido da página está `pending`
- **THEN** a página tem `status: degraded`

### Requirement: Páginas não publicadas indistinguíveis
Para slug inexistente, slug inválido, slug vazio, caminho com mais segmentos, página desabilitada, página gerenciada em conflito ou inválida, e para qualquer página com `status-pages.enabled: false`, a API MUST responder 404 com o corpo `{"error":"status page not found"}`, sem consultar o storage. Status, corpo e os cabeçalhos `Content-Type`, `Cache-Control`, `X-Robots-Tag`, `X-Content-Type-Options`, `Referrer-Policy` e `Vary` MUST ser iguais em todos esses casos, com `GET` e `HEAD`; `Date` não entra na comparação. Nenhuma resposta pública MUST incluir cabeçalhos `X-RateLimit-*`.

#### Scenario: Inexistente e desabilitada
- **WHEN** a página `interna` está desabilitada e a página `nao-existe` não existe
- **THEN** `GET /api/v1/status-pages/interna` e `GET /api/v1/status-pages/nao-existe` têm o mesmo status, corpo e cabeçalhos comparados

#### Scenario: Slug malformado
- **WHEN** chega `GET /api/v1/status-pages/..%2Fadmin`
- **THEN** a API responde 404 com o mesmo corpo e cabeçalhos de uma página inexistente
- **AND** o storage não é consultado

### Requirement: Cabeçalhos das rotas públicas
As respostas das rotas públicas MUST incluir `X-Robots-Tag: noindex, nofollow`, `X-Content-Type-Options: nosniff` e `Referrer-Policy: strict-origin-when-cross-origin`. As respostas da API pública MUST incluir `Vary: Accept-Encoding`, com ou sem compressão. A API MUST responder `Cache-Control: no-cache` com 200 e `Cache-Control: no-store` com 404, 429 e 503, para que nenhum cache HTTP mantenha no ar uma página desabilitada. A rota HTML MUST responder `Cache-Control: no-cache`. A API pública MUST NOT enviar cabeçalhos CORS.

#### Scenario: Página publicada
- **WHEN** chega `GET /api/v1/status-pages/infra` para uma página publicada
- **THEN** a resposta inclui `X-Robots-Tag: noindex, nofollow`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `Vary: Accept-Encoding` e `Cache-Control: no-cache`

#### Scenario: Rota HTML
- **WHEN** chega `GET /status/infra`
- **THEN** a resposta inclui `X-Robots-Tag: noindex, nofollow` e `Cache-Control: no-cache`

### Requirement: Cache e montagem única
Cada requisição MUST capturar uma única vez o slug, a revisão em memória, a geração do ciclo e a definição da página, e a montagem MUST usar só a definição capturada. A resposta de cada revisão de página MUST ser mantida em cache próprio por até 30 s, com chave formada por slug, revisão e geração; a revisão MUST mudar a cada publicação e a cada carga, e MUST NOT ser a versão do banco. Requisições simultâneas à mesma revisão MUST disparar no máximo uma montagem, síncrona na goroutine da requisição, e no máximo 4 montagens públicas MUST rodar ao mesmo tempo. A vaga MUST ser pedida só pela montagem que efetivamente monta, depois da deduplicação, de modo que requisições da mesma revisão ocupem uma única vaga. Uma montagem que esperar mais de 5 s por vaga MUST responder 503 a todas as requisições que aguardavam por ela, sem guardar a resposta em cache. O leitor do storage MUST ser obtido a cada montagem, e nenhum leitor MUST ser reaproveitado entre recargas.

Com storage SQL, os resultados e o uptime dos endpoints da página MUST ser lidos numa única transação de leitura, com uma consulta para os resultados e uma para o uptime.

#### Scenario: Muitos visitantes ao mesmo tempo
- **WHEN** 100 requisições simultâneas pedem a página `infra` com o cache vazio
- **THEN** a página é montada uma única vez, ocupando uma única vaga de montagem, e todas recebem a mesma resposta

#### Scenario: Alteração pela administração
- **WHEN** um administrador remove o grupo `web` da página `infra`
- **THEN** a próxima requisição à página não inclui o grupo `web`, mesmo antes de 30 s

#### Scenario: Escrita durante uma montagem
- **WHEN** uma montagem da página `infra` está bloqueada na leitura do storage e um administrador altera a página
- **THEN** a requisição seguinte dispara outra montagem com a definição nova, sem se juntar à montagem bloqueada

#### Scenario: Recarga do YAML
- **WHEN** a página `infra` é removida do YAML e a configuração é recarregada
- **THEN** `GET /api/v1/status-pages/infra` responde 404

#### Scenario: Semáforo esgotado
- **WHEN** 4 montagens de páginas diferentes estão bloqueadas por mais de 5 s e chega uma requisição de outra página com o cache vazio
- **THEN** a API responde 503 genérico
- **AND** a requisição seguinte a essa página tenta montar de novo

### Requirement: Limite de requisições por IP
A API pública MUST limitar cada IP de cliente a `status-pages.rate-limit` respostas 404 por minuto, numa janela deslizante, contando os 404 da rota específica e dos caminhos fora do padrão. Requisições a uma página publicada (servida do cache, montada ou respondida com 503) MUST NOT contar nem ser bloqueadas, mesmo com o limite do IP esgotado. Ao exceder, a API MUST responder 429 com `Retry-After`, `Cache-Control: no-store` e `{"error":"too many requests"}`.

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

### Requirement: Erros internos sem detalhes
Um endpoint selecionado sem registro no storage MUST ser tratado como `unknown`. Qualquer outra falha ao ler o storage durante a montagem MUST resultar em 503 com `{"error":"status page temporarily unavailable"}` e `Cache-Control: no-store`, sem o texto do erro na resposta. O erro MUST ser registrado no log com o slug. A falha MUST ficar em cache negativo por 5 s para a mesma revisão da página, e nenhum payload de revisão anterior MUST ser servido no lugar.

#### Scenario: Banco indisponível
- **WHEN** a leitura em lote falha com `connection refused` durante a montagem da página `infra`
- **THEN** a API responde 503 sem o texto `connection refused`
- **AND** o log registra o erro com o slug

#### Scenario: Sem stampede
- **WHEN** a leitura em lote falha e chegam 50 requisições sequenciais à página `infra` em 1 s
- **THEN** o storage é consultado no máximo uma vez

### Requirement: Mensagens opcionais nas status pages
Cada status page, do arquivo ou gerenciada, MUST aceitar `show-messages`, booleana e desligada por padrão. O formulário da administração MUST ter a opção "Show messages" junto de "Show certificate expiration". Somente com a opção ligada, o payload de detalhes do endpoint MUST incluir em cada resultado:
- `message`: a mensagem do resultado (envio ou heartbeat); senão o texto de heartbeat de resultados antigos, reconhecido pelo prefixo `heartbeat: no update received within ` nos erros; senão `HTTP <código>` de uma verificação ativa; senão ausente;
- `origin`: `push` nos resultados de envio.

Os erros das verificações ativas MUST NOT ser publicados. O payload da página MUST NOT conter mensagens nem origem, com ou sem a opção. Mudar a opção MUST publicar uma nova revisão da página.

#### Scenario: Página com mensagens
- **WHEN** a página `jobs` tem `show-messages: true` e o endpoint Push `jobs/backup` recebeu `status=down&msg=Disco cheio`
- **THEN** o resultado no payload de detalhes tem `message` `Disco cheio` e `origin` `push`

#### Scenario: Erro de verificação ativa
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/api` falhou com o erro `dial tcp 10.0.0.5:443` sem status HTTP
- **THEN** o payload de detalhes não contém `10.0.0.5` nem `dial tcp`

#### Scenario: Verificação ativa com status HTTP
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/site` recebeu HTTP 200
- **THEN** o resultado no payload de detalhes tem `message` `HTTP 200`

#### Scenario: Falha de heartbeat pública
- **WHEN** a página `jobs` tem `show-messages: true` e o endpoint Push `jobs/backup` ficou um intervalo de 1 minuto sem envio
- **THEN** o resultado no payload de detalhes tem `message` `heartbeat: no update received within 1m0s`

#### Scenario: Página sem a opção
- **WHEN** a página `jobs` não tem `show-messages`
- **THEN** o payload de detalhes não tem `message` nem `origin`

