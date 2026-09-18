## Context

- **Definição da página** (`config/statuspage.Page`): slug, título, descrição, seleção de endpoints, `show-certificate-expiration`, `show-messages` e `enabled`. Toda definição, venha de JSON ou de YAML, é decodificada **pelo decodificador YAML** com `KnownFields(true)` (`statuspage/definition.go`), e a persistência grava `yaml.Marshal(page)` da mesma struct (`statuspage/service.go`). Um campo de senha em claro na struct seria gravado no banco.
- **Rotas públicas** (`api/status_page.go`): `GET /status/:slug` e `/status/*` servem a SPA; sob `/api/v1/status-pages/:slug` estão a página, os detalhes do endpoint, o canal de eventos e o gráfico; um `All` de captura devolve 404 idêntico para o resto.
- **Badges e números por chave** (`api/api.go`): `/api/v1/endpoints/:key/health/badge.svg`, `/uptimes/:duration[/badge.svg]` e `/response-times/:duration[/badge.svg|/chart.svg|/history]` estão no router **não protegido e sem slug** — são públicos no Gatus original e a documentação do fork já diz isso. **A página pública de detalhes usa exatamente essas rotas** (`badgeURL` em `StatusPageEndpoint.vue`).
- **A SPA pública busca com `credentials: 'omit'`** em `StatusPage.vue`, `StatusPageEndpoint.vue` e `ResponseTimeChart.vue`.
- **`security.basic_auth.go`**: `checkCredentials` compara usuário em tempo constante e senha com bcrypt, sempre os dois; `credentialFingerprint` usa prefixo de comprimento; `compareHashAndPassword` é substituível nos testes; `isBrowserRequest` **omite** `WWW-Authenticate` de propósito, para o navegador não abrir a caixa nativa na tela de login.
- **`security.NewFailureLimiter`** trabalha com `netip.Addr` (prefixo /32 ou /64): **não** aceita chave composta.
- **`statuspage.ClientIP(remoteIP, forwardedFor, trustedProxies)`** resolve o IP do cliente; as rotas de status page não passam pelo middleware de IP do dashboard.
- **Mascaramento**: só existe para endpoints (`managedendpoint.MaskSecrets` e `RestoreMaskedSecrets`). Status page **não mascara nada** hoje: o detalhe devolve a definição e o YAML crus, e a validação devolve a definição normalizada.
- **Backup**: `adminbackup.Build(author, password)` cifra o arquivo opcionalmente, mas **não** tem opção de "incluir segredos" — as definições vão completas, e os tokens de push também.
- **Cache público**: por slug, revisão e geração, dentro dos montadores, depois do `Lookup`; e o requisito publicado exige **uma única** captura de slug, revisão, geração e definição por requisição.

## Goals / Non-Goals

**Goals:**

- Uma página pode exigir usuário e senha próprios, e aí **tudo o que aquela página serve** exige a credencial: HTML, API, detalhes, eventos, gráfico e os badges que a própria página mostra.
- Guardar só o hash bcrypt; a senha nunca é persistida, devolvida, registrada em log nem exportada.
- Nada muda para as páginas que não usam a opção.
- Uma página muito acessada não vira custo de CPU, e uma página não derruba as outras.

**Non-Goals:**

- Vários usuários, papéis ou grupos por página.
- Tela de login própria e sessão por cookie para o visitante: a escolha é a caixa do navegador (HTTP Basic).
- Esconder que o slug existe: o 401 revela isso, e é a mecânica do Basic.
- Proteger as rotas globais por chave (`/api/v1/endpoints/:key/...`), que são públicas no Gatus original e sustentam o dashboard.
- Criar a opção "incluir segredos" no backup, que não existe hoje.

## Decisions

### D1 — A credencial na definição

```go
// Auth requires a username and a password to view the page. Without it, the page stays public (fork).
Auth *PageAuth `yaml:"auth,omitempty" json:"auth,omitempty"`

type PageAuth struct {
    Username                        string `yaml:"username" json:"username"`
    PasswordBcryptHashBase64Encoded string `yaml:"password-bcrypt-base64" json:"password-bcrypt-base64"`
}
```

Mesmas chaves e mesma codificação do `security.basic` (base64 com alfabeto URL, que é o que o gerador documentado do fork produz). **Validação:** usuário não vazio, hash que decodifique nesse alfabeto e cujo custo o bcrypt reconheça. Uma página inválida é recusada na administração e impede a carga quando vem do arquivo.

**`security` ausente:** uma página do arquivo com `auth` e `security` nulo protege a página enquanto `/api/v1/endpoints/statuses` continua aberto, publicando mais do que ela. A carga MUST avisar no log, e a documentação diz que `auth` não substitui o `security` da instalação.

### D2 — A senha em claro nunca encosta na definição persistida

A definição é persistida com `yaml.Marshal` da própria struct, então um campo de senha nela vazaria para o banco, para o YAML do detalhe e para o backup. Por isso a senha vive **só** no documento de submissão da administração:

- as rotas de administração aceitam, além da definição, `auth.password` em claro;
- o serviço converte submissão → definição **antes** de persistir: gera o hash com `bcrypt.GenerateFromPassword` (custo 10), preenche `password-bcrypt-base64`, e a senha em claro não existe em nenhuma struct persistida;
- **senha vazia numa página que já exige login** mantém o hash guardado; numa página que ainda não exige, a senha é obrigatória. Tirar `auth` apaga a credencial na mesma escrita;
- senha com menos de 8 caracteres ou mais de 72 bytes (limite do bcrypt, que ignora o excedente em silêncio) é recusada com 400;
- a senha MUST NOT aparecer em log, nem em mensagem de erro.

**Teste obrigatório:** procurar `password:` na definição gravada, no YAML do detalhe e no arquivo de backup, e falhar se aparecer.

### D3 — Mascaramento do hash nas leituras da administração (trabalho novo)

Não existe mascaramento de status page hoje; é preciso criar, no espelho do que `managedendpoint` faz:

- **detalhe** (`GET`): `definition.auth` vira `{ username, hasPassword: true }` e o YAML mostra `password-bcrypt-base64: '********'`;
- **validação e pré-visualização**: a definição normalizada devolvida também vai mascarada;
- **caminho de volta**: como o administrador pode salvar o YAML que leu, a submissão com `********` no lugar do hash MUST ser entendida como "manter o hash guardado", do jeito que `RestoreMaskedSecrets` já faz nos endpoints. Sem isso, salvar o YAML lido tranca a página com um hash inválido;
- o registro em memória que serve as rotas públicas continua com o hash verdadeiro.

### D4 — O middleware das rotas da página

Um middleware registrado **em cada rota da página** (nunca no grupo nem na rota de captura):

1. resolve o slug **uma única vez** com `Lookup` e guarda a página publicada em `c.Locals`; os manipuladores passam a usar essa captura, para a requisição não resolver duas vezes e não autorizar numa definição e servir outra;
2. aplica as **regras de página**: inexistente, não publicada, em conflito ou `status-pages.enabled: false` respondem o mesmo 404 de hoje, pelo mesmo caminho de erro que já existe (para manter cabeçalhos e limitador), **sem** `WWW-Authenticate`;
3. se a página não tiver `auth`, segue direto — nada muda;
4. com `auth`: exige `Authorization: Basic`. Sem credencial, ou errada, responde **401** com `WWW-Authenticate: Basic realm="<slug da página publicada>", charset="UTF-8"` e `Cache-Control: no-store`. O realm vem do slug **da definição publicada**, nunca do parâmetro da requisição;
5. o 404 por **chave** (endpoint que não está na página) MUST vir **depois** do 401: senão quem não tem credencial distingue 401 de 404 e enumera as chaves da página.

**Não reaproveitar `isBrowserRequest`:** ele omite `WWW-Authenticate` quando a requisição parece de navegador, que é exatamente o contrário do que esta tela precisa. O 401 da página manda o cabeçalho sempre.

**Ordem fixa:** regras de página → limitador (`Blocked`) → memória de verificação → bcrypt → registro da falha (`Failure`).

### D5 — Badges da página protegida

As rotas por chave (`/api/v1/endpoints/:key/...`) são públicas no Gatus original, sustentam o dashboard e **não** serão fechadas. Mas a página pública de detalhes usa justamente elas, então hoje quem viu uma página protegida ficaria com as chaves e leria uptime, tempo de resposta e saúde para sempre, sem credencial.

Então a change cria as rotas equivalentes **sob a página**, atrás do mesmo middleware:

```
GET /api/v1/status-pages/:slug/endpoints/:key/health/badge.svg
GET /api/v1/status-pages/:slug/endpoints/:key/response-times/:duration/badge.svg
```

Elas reaproveitam os geradores de badge existentes depois de conferir, como as demais rotas da página, que a chave pertence à página. A página pública passa a apontar os `<img>` para elas. As rotas globais continuam públicas, e a documentação passa a dizer, com todas as letras, que elas continuam abertas e que quem quiser esconder esses números também do dashboard precisa do `security` da instalação.

### D6 — A SPA precisa mandar a credencial

`StatusPage.vue`, `StatusPageEndpoint.vue` e `ResponseTimeChart.vue` buscam com `credentials: 'omit'`, então, depois do 401 da navegação, o navegador **não** mandaria a credencial nas chamadas da API e a página abriria vazia. Nas rotas públicas da página, as buscas passam a usar `credentials: 'same-origin'`.

Isso faz o navegador enviar também o cookie de sessão do `security`, se houver: o middleware da página **ignora** sessão, OIDC e o `security` da instalação, e só aceita a credencial daquela página — está escrito no requisito.

**401 com a aba aberta:** a página recarrega a cada 60 s; se a credencial for trocada nesse meio tempo, a resposta 401 MUST parar o ciclo de atualização e mostrar que é preciso recarregar e entrar de novo, em vez de repetir a caixa de credencial em laço.

### D7 — Limite de tentativas e memória da verificação

- **Limitador:** `security.NewFailureLimiter` só aceita IP. Esta change acrescenta um limitador de chave textual (ou um por página, com descarte no `Load`), com chave `slug + "\x00" + prefixo do IP do cliente` (/32 ou /64), janela de 5 minutos, 10 falhas e teto de chaves. O IP vem de `statuspage.ClientIP`, respeitando `status-pages.trusted-proxies` — sem proxies confiáveis configurados, todos os visitantes contam como um só, e a documentação amarra `auth` a `trusted-proxies`.
- **Memória da verificação bem-sucedida:** chave `HMAC-SHA-256(chave aleatória do processo, comprimento(usuário) ∥ usuário ∥ senha ∥ impressão digital da credencial da página)` — com prefixo de comprimento, porque `usuário:senha` concatenado é ambíguo, e com HMAC de chave de processo para não guardar um resumo da senha quebrável fora do processo. A impressão digital cobre **usuário e hash**. TTL absoluto de 5 minutos, sem renovação no uso, com teto de entradas, descarte do menos usado e limpeza no `Load`, como o cache público já faz. Só sucessos entram.
- **Teto de bcrypt simultâneos** na verificação de página, nos moldes do semáforo de montagem pública, para uma enxurrada de senhas erradas não consumir a CPU toda.

### D8 — Backup e restore

O backup do fork não tem "incluir segredos": ele exporta as definições completas, e os tokens de push também. Então **o hash vai no backup como os demais segredos já vão**, sem máscara, e o requisito publicado continua valendo sem exceção.

O que precisa de regra é o caminho contrário: um arquivo **montado a partir das leituras da administração** traz `********`. Na prévia e na aplicação do restore, a mesclagem com a definição já existente no destino MUST acontecer **antes** da validação (hoje `ValidateRestore` valida primeiro e recusaria o `********`): página existente mantém o hash do destino; página nova com hash mascarado é `skip`, com o motivo de segredo mascarado que já existe.

### D9 — Administração

- **Formulário:** opção "Require login to view this page" na seção General, abaixo de "Published", com usuário e senha aparecendo só quando marcada; na edição a senha vem vazia com a explicação de que vazio mantém a atual; o texto de "Published" passa a dizer que a página protegida é visível **com login**.
- **Lista:** a API de listagem ganha um booleano por página (`requiresLogin`), e a lista mostra um cadeado com a dica "Requires login", além da contagem no rodapé.

### D10 — Testes

**Go:**

1. validação: `auth` sem usuário, hash fora do alfabeto, hash que não é bcrypt; e um hash cujo base64 contenha `-`/`_`;
2. rotas protegidas: HTML (`GET` **e** `HEAD`), HTML aninhado (`/status/<slug>/endpoints/<key>`), API da página, detalhes, eventos, gráfico e os **badges com slug** respondem 401 com `WWW-Authenticate` e `Cache-Control: no-store`; com credencial certa, 200 com `Cache-Control: private, no-store`;
3. o 401 vem **mesmo** com `Sec-Fetch-Site: same-origin` e `X-Requested-With` (prova que `isBrowserRequest` não foi reaproveitado);
4. credencial do `security.basic` da instalação e credencial de outra página não abrem a página;
5. página inexistente, desabilitada e caminho fora do padrão continuam no 404 idêntico, sem `WWW-Authenticate`; chave que não está na página responde 401 antes de qualquer 404 por chave;
6. limitador: 10 falhas na página A não bloqueiam a página B no mesmo IP; estourado o limite, 429 com `Retry-After` **sem** chamar a comparação de senha (contada pelo gancho de teste que já existe em `security`); credencial certa dentro da janela bloqueada continua 429;
7. memória: duas requisições seguidas com a mesma credencial fazem uma comparação bcrypt; trocar a senha da página faz a seguinte comparar de novo; `usuário="a"`/`senha="b:c"` não acerta a entrada de `usuário="a:b"`/`senha="c"`;
8. administração: criar com senha guarda hash; o `GET` do detalhe, o YAML, a validação e a pré-visualização **não** contêm o prefixo do bcrypt nem a senha; salvar o YAML lido (com `********`) mantém o hash; senha vazia mantém; desmarcar apaga; senha curta e senha acima de 72 bytes são 400;
9. backup: o hash vai no arquivo; um arquivo com `********` restaura mantendo o hash do destino, e é `skip` para página nova;
10. `security` nulo com página do arquivo com `auth`: aviso no log.

**E2E** (`test/e2e/status-pages.sh`), sem levar o navegador à caixa nativa de credencial:

11. criar pela tela uma página protegida e publicá-la; conferir com `curl -i`, sem credencial, que HTML, API, detalhes, eventos, gráfico e badge com slug respondem 401 com `WWW-Authenticate` e `Cache-Control: no-store`;
12. com `curl -u`, as mesmas rotas respondem 200, e o payload é o de sempre;
13. a página pública que o roteiro já usa continua 200 sem credencial, e as asserções existentes de "nenhuma requisição recebeu 401" continuam valendo para ela;
14. o `GET` da administração da página protegida não traz o hash;
15. a lista mostra o cadeado.

## Risks / Trade-offs

- **O 401 revela que o slug existe.** Inerente ao Basic; quem não quiser revelar nada não publica a página.
- **Sem logout:** a credencial fica no navegador até fechá-lo.
- **Os números por chave continuam públicos** nas rotas globais do Gatus original: a página protegida esconde o conjunto (quem está na página, o estado geral, as mensagens), não cada métrica isolada de um endpoint cuja chave alguém já conheça. Está escrito no requisito e na documentação.
- **Senha em claro na requisição de administração**, entre o administrador autenticado e o Gatus, na mesma origem, com HTTPS na frente em produção.
- **Sem `trusted-proxies`, o limite é compartilhado:** 10 senhas erradas trancam a página para todos por 5 minutos. O fork já avisa disso para o limite das páginas públicas; a documentação passa a amarrar `auth` a `trusted-proxies`.
- **Mascaramento novo** em três leituras (detalhe, validação e pré-visualização) mais o caminho de volta: é a parte com mais risco de esquecer um lugar, e por isso o teste procura o prefixo do bcrypt em todas elas.
