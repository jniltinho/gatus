## MODIFIED Requirements

### Requirement: Mensagens opcionais nas status pages
Cada status page, do arquivo ou gerenciada, MUST aceitar `show-messages`, booleana e desligada por padrão. O formulário da administração MUST ter a opção "Show messages" junto de "Show certificate expiration". Somente com a opção ligada, o payload de detalhes do endpoint MUST incluir em cada resultado:
- `message`: a mensagem do resultado (envio ou heartbeat); senão o texto de heartbeat de resultados antigos, reconhecido pelo prefixo `heartbeat: no update received within ` nos erros; senão `HTTP <código>` de uma verificação ativa, somente com código igual ou maior que 100; senão, num resultado sem sucesso e **não pendente**, o motivo da falha; senão ausente;
- `origin`: `push` nos resultados de envio, inclusive nos recebidos pela API externa de resultados.

O motivo da falha MUST ser um destes textos, e nenhum outro: `Certificate error`, `DNS error`, `Timeout`, `Connection failed` e `Check failed`. A categoria MUST ser a primeira dessa ordem que casar com **qualquer um** dos erros do resultado, por marcadores que não ocorram numa URL comum; sem erros, o motivo MUST ser `Connection failed` quando o resultado não chegou a conectar e `Check failed` quando conectou. O texto do erro MUST NOT ser copiado, recortado nem reescrito no payload, e os cinco textos MUST ser estáveis e não localizados.

Os erros das verificações ativas MUST NOT ser publicados, com ou sem `auth` na página. O payload da página MUST NOT conter mensagens nem origem, com ou sem a opção. Mudar a opção MUST publicar uma nova revisão da página.

#### Scenario: Página com mensagens
- **WHEN** a página `jobs` tem `show-messages: true` e o endpoint Push `jobs/backup` recebeu `status=down&msg=Disco cheio`
- **THEN** o resultado no payload de detalhes tem `message` `Disco cheio` e `origin` `push`

#### Scenario: Conexão recusada
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/api` falhou com o erro `Get "https://api.exemplo.com/certificate-status": dial tcp 10.0.0.5:443: connect: connection refused`
- **THEN** o resultado no payload de detalhes tem `message` `Connection failed`
- **AND** o payload de detalhes não contém `10.0.0.5`, `dial tcp` nem `certificate-status`

#### Scenario: Certificado que não bate com o host
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/site` falhou com o erro `Get "https://tag.exemplo.com": tls: failed to verify certificate: x509: certificate is valid for *.exemplo.com, not tag.exemplo.com`
- **THEN** o resultado no payload de detalhes tem `message` `Certificate error`
- **AND** o payload de detalhes não contém `x509` nem `*.exemplo.com`

#### Scenario: Nome que não resolve
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/sso` falhou com o erro `Get "https://sso.exemplo.com": dial tcp: lookup sso.exemplo.com on 127.0.0.11:53: no such host`
- **THEN** o resultado no payload de detalhes tem `message` `DNS error`
- **AND** o payload de detalhes não contém `127.0.0.11` nem `lookup`

#### Scenario: Tempo esgotado
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/lento` falhou com o erro `context deadline exceeded (Client.Timeout exceeded while awaiting headers)`
- **THEN** o resultado no payload de detalhes tem `message` `Timeout`

#### Scenario: Falha sem erro registrado
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação do endpoint TCP `core/banco` falhou sem conectar e sem erro registrado
- **THEN** o resultado no payload de detalhes tem `message` `Connection failed`

#### Scenario: Condição que falhou com o serviço respondendo
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/dns` conectou, não registrou erro e falhou apenas a condição
- **THEN** o resultado no payload de detalhes tem `message` `Check failed`

#### Scenario: Verificação SSH sem status HTTP
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/bastion` falhou com status `1` e o erro `dial tcp 10.0.0.9:22: connect: connection refused`
- **THEN** o resultado no payload de detalhes tem `message` `Connection failed`
- **AND** o payload de detalhes não contém `HTTP 1` nem `10.0.0.9`

#### Scenario: Verificação ativa com status HTTP
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação de `core/site` recebeu HTTP 200
- **THEN** o resultado no payload de detalhes tem `message` `HTTP 200`

#### Scenario: Resultado bem-sucedido sem status HTTP
- **WHEN** a página `infra` tem `show-messages: true` e a última verificação do endpoint ICMP `core/gateway` teve sucesso sem status HTTP
- **THEN** o resultado no payload de detalhes não tem `message`

#### Scenario: Resultado pendente
- **WHEN** a página `jobs` tem `show-messages: true` e o último resultado de `jobs/backup` é pendente sem mensagem
- **THEN** o resultado no payload de detalhes não tem `message`

#### Scenario: Resultado da API externa de resultados
- **WHEN** a página `jobs` tem `show-messages: true` e `jobs/fila` recebeu `POST /api/v1/endpoints/jobs_fila/external?success=false&error=timeout na fila de pagamento`
- **THEN** o resultado no payload de detalhes tem `origin` `push` e não tem `message`
- **AND** o payload de detalhes não contém `fila de pagamento`

#### Scenario: Falha de heartbeat pública
- **WHEN** a página `jobs` tem `show-messages: true` e o endpoint Push `jobs/backup` ficou um intervalo de 1 minuto sem envio
- **THEN** o resultado no payload de detalhes tem `message` `heartbeat: no update received within 1m0s`

#### Scenario: Página com login próprio
- **WHEN** a página `clientes` tem `auth` e `show-messages: true` e a última verificação de `core/api` falhou com `connect: connection refused`
- **THEN** o payload de detalhes autenticado tem `message` `Connection failed` e não contém o texto do erro

#### Scenario: Página sem a opção
- **WHEN** a página `jobs` não tem `show-messages`
- **THEN** o payload de detalhes não tem `message` nem `origin`

#### Scenario: Payload da página
- **WHEN** a página `infra` tem `show-messages: true` e um dos resultados falhou com erro
- **THEN** o payload da página não tem `message` nem `origin` em nenhum resultado
