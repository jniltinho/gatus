# Status pages públicas

> Funcionalidade exclusiva do fork [jniltinho/gatus](https://github.com/jniltinho/gatus). No Gatus original, o pedido de
> páginas de status públicas foi fechado como *not planned* ([TwiN/gatus#1311](https://github.com/TwiN/gatus/issues/1311)).

Uma status page é uma página pública, aberta **sem login** em `/status/<slug>`, que mostra só os grupos e endpoints
escolhidos, como os *Status Pages* do Uptime Kuma. O dashboard, a administração e a API de status continuam protegidos
por `security.basic` ou `security.oidc`.

Cada endpoint aparece com nome, estado atual, barras das últimas verificações e uptime de 24 horas, 7 dias e 30 dias.

## Configuração

```yaml
status-pages:
  enabled: true                          # padrão true; false despublica todas as páginas
  trusted-proxies: ["172.30.0.1/32"]     # veja "Atrás de proxy reverso"
  rate-limit: 120                        # respostas 404 por minuto por IP; 0 desliga
  pages:
    - slug: servicos
      title: "Serviços"
      description: "Sites e APIs externos"
      groups: [sites, apis]
    - slug: infraestrutura
      title: "Infraestrutura"
      groups: [dns]
      endpoints: [core_banco-de-dados]  # chaves de endpoints avulsos (grupo_nome)
      enabled: false                     # páginas do arquivo são publicadas por padrão
```

| Campo | Regra |
|-------|-------|
| `slug` | 1 a 64 caracteres: letras minúsculas, números e hífens, sem hífen no início ou no fim. Único. `new`, `options`, `preview`, `validate` e `exposure` são reservados. |
| `title` | Obrigatório, até 100 caracteres. |
| `description` | Opcional, até 1000 caracteres, texto puro (sem markdown ou HTML). |
| `groups` | Até 50 grupos. Todos os endpoints habilitados do grupo aparecem, inclusive os criados depois. |
| `endpoints` | Até 200 chaves no formato `grupo_nome` (a mesma chave dos badges). |
| `enabled` | Páginas do arquivo: padrão `true`. Páginas cadastradas pela web: padrão `false`. |

A página precisa selecionar ao menos um grupo ou um endpoint. Um grupo ou chave que ainda não existe não invalida a
página: a carga registra um aviso no log e a administração mostra o aviso na validação.

O `config.yaml` padrão do fork (imagem Docker e tarballs das releases) já traz as páginas `/status/servicos` e
`/status/infraestrutura` com endpoints de exemplo.

## Páginas cadastradas pela web

Com a [administração](admin-endpoints.md) habilitada, a aba **Status pages** em `/admin/status-pages` permite criar,
editar, pré-visualizar, publicar, despublicar e remover páginas. Elas ficam na tabela `managed_status_pages` do mesmo
banco dos endpoints (SQLite ou PostgreSQL).

- Uma página nova nasce **desabilitada**, também pela API: confira a pré-visualização e marque **Publicada**.
- As páginas do arquivo de configuração aparecem na lista só para consulta.
- Se o arquivo de configuração passar a usar o slug de uma página cadastrada pela web, o arquivo prevalece: a página da
  web fica marcada como em conflito e não é publicada até o arquivo deixar de usar o slug.
- Desligar `admin.enabled` **não** despublica as páginas cadastradas pela web (só some a administração). Para tirar
  todas as páginas do ar, use `status-pages.enabled: false`.
- O formulário de endpoints avisa em quais status pages o endpoint vai aparecer, pelo grupo ou pela chave.

## O que a página mostra

- Seções na ordem de `groups`, depois os grupos alcançados só por `endpoints` (em ordem alfabética) e, por último,
  **Outros serviços** com os endpoints sem grupo. Dentro de cada seção, endpoints em ordem de nome.
- Endpoints do arquivo, external endpoints e endpoints cadastrados pela web, desde que habilitados. Suites e instâncias
  `remote` ficam de fora.
- No máximo 200 endpoints por página; acima disso, a página avisa que mostra só os primeiros.
- Os últimos 50 resultados de cada endpoint (ou `storage.maximum-number-of-results`, se for menor).

Estados:

| Estado | Endpoint | Grupo e página |
|--------|----------|----------------|
| Operacional / No ar | último resultado com sucesso | todos os endpoints com resultado estão no ar |
| Degradação parcial | — | há endpoints no ar e fora do ar |
| Indisponível / Fora do ar | último resultado com falha | todos os endpoints com resultado estão fora do ar |
| Sem dados | nenhum resultado ainda | nenhum endpoint tem resultado |

O uptime aparece como "—" quando não houve verificação no período. Com SQLite e PostgreSQL, o histórico com mais de
48 horas é agregado por dia, então as bordas de 7 e 30 dias são aproximadas, como nos badges.

## API pública

`GET /api/v1/status-pages/<slug>` responde sem autenticação:

```json
{
  "slug": "servicos", "title": "Serviços", "description": "Sites e APIs externos",
  "status": "degraded", "updatedAt": "2026-09-14T19:30:00Z", "truncated": false,
  "groups": [{
    "name": "apis", "status": "degraded",
    "endpoints": [{
      "name": "github-api", "status": "up",
      "uptime": {"24h": 0.9993, "7d": 0.998, "30d": null},
      "results": [{"timestamp": "2026-09-14T19:29:00Z", "success": true, "durationMs": 123}]
    }]
  }]
}
```

**Nunca são publicados:** chave, URL, hostname, IP, porta, código HTTP, erros, condições, eventos, expiração de
certificado ou domínio, alertas e `extra-labels`.

- Página inexistente, desabilitada, em conflito, slug inválido ou qualquer outro caminho em `/api/v1/status-pages`
  respondem o mesmo `404 {"error":"status page not found"}`, com os mesmos cabeçalhos.
- `503 {"error":"status page temporarily unavailable"}` quando o banco não pôde ser lido; o erro fica só no log.
- A rota HTML `/status/<qualquer-coisa>` responde sempre 200 com a aplicação, que consulta a API e mostra
  "Página não encontrada" quando for o caso.
- Cabeçalhos: `X-Robots-Tag: noindex, nofollow`, `X-Content-Type-Options: nosniff`,
  `Referrer-Policy: strict-origin-when-cross-origin`, `Cache-Control: no-cache` (200) e `no-store` (404, 429 e 503).
- A página pode ser embutida em iframe (por exemplo, numa TV de monitoramento). A API pública não envia CORS.

## Cache e custo

- Cada página é montada no máximo uma vez a cada 30 segundos, por mais visitantes que tenha: as requisições simultâneas
  compartilham a mesma montagem. Uma alteração pela administração vale na próxima requisição.
- A montagem lê o banco numa transação com três consultas, qualquer que seja o número de endpoints, e no máximo 4
  páginas são montadas ao mesmo tempo.
- Se o banco falhar, o erro fica em cache por 5 segundos para não sobrecarregá-lo.
- A página no navegador se atualiza a cada 60 segundos e pausa quando a aba fica oculta.

## Limite de requisições

- Só as respostas **404** contam no limite (`rate-limit` por minuto por IP; IPv6 agregado por /64). Uma página
  publicada nunca é bloqueada: o custo dela já é limitado pelo cache.
- Ao exceder o limite: `429 {"error":"too many requests"}` com `Retry-After`.

## Atrás de proxy reverso

Sem configuração, o IP do visitante é o IP da conexão. Atrás de um nginx, todos os visitantes chegariam com o mesmo IP
e dividiriam o mesmo limite. Informe em `trusted-proxies` de onde o proxy conecta no Gatus: o `X-Forwarded-For` só é
lido dessas conexões, da direita para a esquerda, e o primeiro IP que não é um proxy confiável identifica o visitante.

Quando uma conexão de IP privado ou local não confiável traz `X-Forwarded-For`, o Gatus registra um aviso no log (uma
vez por carga) com o IP a colocar em `trusted-proxies`, e a lista de status pages da administração mostra o mesmo aviso.

### Docker com nginx no host

Com a porta publicada só em `127.0.0.1`, o Gatus vê as conexões vindas do **gateway da rede do Docker**, não de
`127.0.0.1`. Fixe a sub-rede da rede do compose para o gateway ter um IP conhecido:

```yaml
services:
  gatus:
    image: jniltinho/gatus:v5.36.0-fork.2
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - ./config:/config:ro
      - ./data:/data
    networks:
      - gatus

networks:
  gatus:
    ipam:
      config:
        - subnet: 172.30.0.0/24
```

```yaml
status-pages:
  trusted-proxies: ["172.30.0.1/32"]
```

O vhost do nginx precisa enviar o `X-Forwarded-For`:

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

Alternativas: `network_mode: host` com `web.address: 127.0.0.1`, ou o binário direto no host; nos dois casos, use
`trusted-proxies: ["127.0.0.1/32", "::1/128"]`.

## Segurança

- **O slug não é controle de acesso.** Qualquer pessoa com o endereço vê a página; não publique nomes de serviços que
  não podem ser vistos por terceiros.
- Os nomes dos endpoints e dos grupos ficam públicos. Como a seleção por grupo inclui endpoints novos automaticamente,
  confira o aviso de exposição no formulário de endpoints.
- Os badges, uptimes e tempos de resposta por chave (`/api/v1/endpoints/<chave>/...`) já são públicos no Gatus original
  e continuam assim.

## API de administração

Rotas em `/api/v1/admin`, com as mesmas exigências da [administração de endpoints](admin-endpoints.md#api)
(autenticação, permissão, proteção CSRF, corpo de até 256 KB em JSON ou YAML):

| Método e rota | Função |
|---------------|--------|
| `GET /status-pages` | Lista as páginas, com `publicationEnabled`, `managedUnavailable` e `sharedRateLimitWarning` |
| `GET /status-pages/options` | Grupos e endpoints que podem ser selecionados |
| `GET /status-pages/exposure?group=<g>&key=<k>` | Páginas em que um endpoint apareceria |
| `POST /status-pages/validate` | Valida uma definição e devolve avisos (`?slug=` para validar uma alteração) |
| `POST /status-pages` | Cria (201 com `ETag`) |
| `GET /status-pages/<slug>` | Obtém (com `ETag`) |
| `PUT /status-pages/<slug>` | Altera (exige `If-Match`) |
| `POST /status-pages/<slug>/enable` e `/disable` | Publica ou despublica (exige `If-Match`) |
| `DELETE /status-pages/<slug>` | Remove (exige `If-Match`) |
| `GET /status-pages/<slug>/preview` | Payload público de qualquer página, inclusive desabilitada, sem cache |

Erros: 400 (definição inválida, slug reservado ou alterado), 404, 409 (slug em uso ou página do arquivo), 412 (versão
desatualizada), 428 (sem `If-Match`), 501 (storage sem suporte) e 503 (partida ou recarga em andamento).

## Várias instâncias com o mesmo PostgreSQL

Uma página cadastrada pela web numa instância só aparece nas outras depois que elas recarregam a configuração ou
reiniciam. Atrás de um balanceador, isso faz o visitante alternar entre a página e "Página não encontrada" até todas as
instâncias recarregarem.

## Voltar para o Gatus original

A seção `status-pages` e a tabela `managed_status_pages` são ignoradas pelo Gatus original, e as páginas deixam de
existir. Faça backup do banco antes de trocar de versão.

## Testes ponta a ponta

```bash
test/e2e/status-pages.sh
```

Sobe o Gatus com SQLite temporário, basic auth e administração, percorre a página pública sem credenciais (temas claro
e escuro, 390 px, página não encontrada, OIDC simulado) e as telas de administração, com capturas em
`dist/prints/status-pages/`.
