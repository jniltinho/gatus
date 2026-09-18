## Context

- `statuspage/payload.go`, `publicMessage(result)`: devolve a mensagem do resultado; senão o erro de heartbeat (prefixo `heartbeat: no update received within `); senão `HTTP <código>` quando `HTTPStatus > 0`; senão **vazio**. É esse vazio que aparece como coluna Message em branco.
- O requisito publicado proíbe publicar os erros das verificações ativas, e `TestBuildPayload_Allowlist` decodifica o JSON público com `DisallowUnknownFields` para impedir campo novo por acidente.
- `common.ResultSummary` (o que o storage entrega para a montagem pública) tem `Timestamp`, `Success`, `Duration`, `CertificateExpiration`, `Pending`, `Message`, `Origin`, `HTTPStatus` e `Errors`. **Não tem `Connected` nem o tipo do endpoint.**
- Os erros vêm da biblioteca padrão do Go (`net/http`, `crypto/tls`, `net`), então são textos que embutem a URL e o host do operador: `Get "https://api.exemplo.com/certificate-status": dial tcp 10.0.0.5:443: connect: connection refused`.
- `client.CanCreateNetworkConnection` (TCP, UDP, SCTP) e `client.Ping` (ICMP) **não devolvem erro**: em falha o resultado só fica com `Connected: false` e a condição falha. Condição que falha não acrescenta erro.
- `client.CheckSSHBanner` devolve `1` como status em qualquer falha, e `ExecuteSSHCommand` devolve o código de saída do comando: hoje um SSH fora do ar publica `HTTP 1`.
- `api/external_endpoint.go` (a API externa do Gatus original) grava `error=<texto do operador>` em `Errors`, com `Message`, `Origin` e `HTTPStatus` vazios. `api/push.go` (o push do fork) grava `Message` (`OK` por padrão) e `Origin: push`.
- `RecentChecksTable.vue` com `sanitized` só lê `result.message`; sem `sanitized` (dashboard) cai para `errors.join('; ')`.

## Goals / Non-Goals

**Goals:**

- O visitante entender **por que** a verificação falhou, num nível que não ajuda a mapear a infraestrutura.
- Nada do texto do erro chegar ao payload: o que é publicado é uma constante escolhida pelo Gatus.

**Non-Goals:**

- Publicar o erro completo, nem atrás de opção **nem na página com `auth`**: a página com login próprio serve o mesmo payload sanitizado, e a credencial dela é de um parceiro ou cliente, não do operador. Quem precisa do texto usa o dashboard, protegido por `security`.
- Traduzir os motivos: os cinco textos são constantes estáveis da API pública, em inglês, como o resto da interface.
- Mudar o dashboard, que já mostra o erro inteiro.
- Suites: o pacote `statuspage` não as referencia, então não há impacto.

## Decisions

### D1 — Um conjunto fechado de motivos, escolhido por marcadores

`publicMessage` ganha um último passo, para um resultado **sem sucesso, não pendente, sem mensagem, sem heartbeat e sem status HTTP publicável**:

| Ordem | Motivo | Marcadores (comparados em minúsculas, em qualquer um dos erros) |
|-------|--------|------------------------------------------------------------------|
| 1 | `Certificate error` | `x509:`, `tls:`, `certificate is valid for`, `certificate has expired`, `certificate signed by`, `unknown authority` |
| 2 | `DNS error` | `no such host`, `server misbehaving`, `lookup `, `dns: ` |
| 3 | `Timeout` | `i/o timeout`, `context deadline exceeded`, `handshake timeout`, `timeout awaiting` |
| 4 | `Connection failed` | `connection refused`, `no route to host`, `network is unreachable`, `connection reset by peer`, `broken pipe`, `unexpected eof`, `: eof` |
| 5 | `Check failed` | qualquer outro erro |

**A varredura é por categoria sobre todos os erros do resultado**, na ordem da tabela: a primeira categoria que casar com qualquer erro ganha. Não é "o primeiro erro e depois as categorias", senão um `invalid condition: ...timeout...` antes de um `x509:` viraria `Timeout`.

Os marcadores são **distintivos de propósito**. `certificate`, `dns`, `timeout` e `dial ` soltos casariam com a URL do operador — um host `dns.corp.exemplo.com` ou um caminho `/certificate-status` numa conexão recusada viraria `DNS error` ou `Certificate error`, e a categoria publicada passaria a depender do nome do host que a página esconde. `EOF` é escrito em minúsculas porque a comparação é em minúsculas, e `context canceled` fica fora de `Timeout` porque no Gatus ele aparece em parada, recarga e cancelamento de suite, não em tempo esgotado.

**A função MUST NOT concatenar, recortar ou reescrever o texto do erro.** O valor devolvido é sempre uma das cinco constantes.

### D2 — Falha sem erro nenhum: `Connected` entra no resumo

TCP, UDP, SCTP e ICMP falham **sem erro registrado**: sem isto, o motivo mais citado desta change (`Connection failed`) seria inalcançável justamente para eles, e todo `tcp://` caído mostraria `Check failed`.

`common.ResultSummary` ganha `Connected`, preenchido pelas duas cargas (`storage/store/sql/endpoint_summary_batch.go` e `storage/store/memory/endpoint_uptime_batch.go`), e a regra passa a ser: **sem erros e sem conexão → `Connected failed`**, isto é `Connection failed`; sem erros e com conexão → `Check failed` (a condição falhou com o serviço respondendo).

`Connected` não é publicado: ele só entra na escolha do motivo, e o payload continua com os mesmos campos.

### D3 — `HTTP <código>` só com código de HTTP

Hoje `HTTPStatus > 0` publica `HTTP <código>` antes de qualquer motivo, e `client.CheckSSHBanner` usa `1` como status: um SSH recusado publica `HTTP 1`, e um comando SSH com saída 2 publica `HTTP 2`. A condição passa a ser `HTTPStatus >= 100`, que é o menor código HTTP que existe; abaixo disso o resultado segue para a classificação.

### D4 — Resultado pendente e resultado de push não são classificados

- **Pendente** tem `Success: false` mas não é falha: fica fora dos alertas e dos eventos, e aparece em amarelo. Um push pendente sem mensagem passaria a mostrar `Check failed` numa linha amarela. Pendente MUST NOT receber motivo.
- **Push** (`api/push.go`) sempre tem mensagem, e o heartbeat antigo é reconhecido pelo prefixo, então nenhum dos dois chega à classificação.
- **API externa do Gatus original** (`POST /api/v1/endpoints/:key/external?success=false&error=...`) é o caso que a versão anterior desta proposta errou: ela grava o texto do operador em `Errors`, sem mensagem e sem origem, então a classificação trataria `error=timeout na fila de pagamento` como um erro de rede do Go e publicaria `Timeout`. A change passa a marcar esses resultados com a **origem de push**, que é o que eles são — um sistema externo reportando o estado —, e com isso eles ficam fora da classificação e a coluna Message continua vazia para eles, como hoje. O texto do operador continua **não** publicado.

### D5 — O motivo vai no campo `message`, sem campo novo

- a tabela pública já mostra `message`, então o frontend não muda;
- `message` já está na allowlist do teste de sanitização: um campo novo quebraria o `DisallowUnknownFields` até o struct do teste mudar;
- `origin` continua sendo o que separa push de verificação: numa verificação ativa, `message` é `HTTP <código>` ou uma das cinco constantes, **nunca** o texto do erro.

Quem consome a API pública e tratava `message` vazio como "falha de rede" passa a ver uma das cinco strings; quem olha `origin === "push"` não muda.

## Risks / Trade-offs

- **O motivo já diz algo**: saber que caiu por certificado é mais do que só `Down`. É o preço de uma tabela útil, e é o mesmo que qualquer visitante descobre abrindo o site no navegador.
- **Classificação por texto**: as redações do Go podem mudar entre versões. O pior caso é cair em `Check failed` — nada vaza, e os testes cobrem as redações de hoje.
- **`ui.hide-errors`** zera os erros depois da avaliação: com ele, uma falha HTTP sem status vira `Check failed`.
- **DNS por rcode e perda de ICMP**: `NXDOMAIN` não gera erro e o resumo não tem o rcode, então cai em `Check failed`; ICMP sem resposta cai em `Connection failed` pela regra de `Connected`.
- **`Check failed` é vago**: é intencional, porque o que não foi reconhecido não pode ser descrito sem olhar o texto.

## Migration Plan

Nada a migrar: o campo já existe no payload, e páginas sem `show-messages` não mudam. `Connected` no resumo é leitura de uma coluna que já é gravada.
