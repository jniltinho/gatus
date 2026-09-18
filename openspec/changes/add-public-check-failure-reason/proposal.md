## Why

Numa status page com `show-messages`, a tabela de checagens mostra a mensagem dos envios de push e o `HTTP <código>` das verificações que responderam. Quando a verificação nem chega a responder — certificado que não bate com o host, DNS que não resolve, conexão recusada, tempo esgotado — a coluna Message fica **vazia** e o visitante vê só `Down`, sem saber se o serviço caiu, se a rede falhou ou se o certificado venceu.

O dashboard já mostra o erro inteiro, porque é uma tela autenticada. Na página pública o erro não pode ser publicado como veio: `dial tcp: lookup sso.exemplo.com on 127.0.0.11:53: no such host` entrega o resolvedor DNS interno da instalação, e `x509: certificate is valid for *.exemplo.com` entrega qual certificado o host serve. Esse requisito ("os erros das verificações ativas MUST NOT ser publicados") continua valendo.

Falta o meio-termo: dizer **o tipo** da falha, sem dizer nada sobre a infraestrutura.

## What Changes

- O payload de detalhes de uma página com `show-messages` passa a trazer, nos resultados de verificação que falharam sem mensagem e sem status HTTP, um motivo curto de um **conjunto fechado** de textos: `Certificate error`, `DNS error`, `Timeout`, `Connection failed` e `Check failed`.
- O motivo é escolhido a partir dos erros do resultado por marcadores que não ocorrem numa URL comum, e **nada do texto do erro é copiado**: o payload só pode conter uma das constantes acima.
- Resultado pendente, de push e da API externa de resultados **não** recebem motivo: pendente não é falha, e o texto que um sistema externo envia não é erro do Go para ser classificado.
- Falha sem erro registrado — o caso de TCP, UDP, SCTP e ICMP — passa a ser reconhecida pela conexão, para `Connection failed` não ser inalcançável justamente para eles.
- `HTTP <código>` passa a ser publicado só com código igual ou maior que 100, senão o banner SSH, que usa `1`, publica `HTTP 1`.
- A tabela de checagens da página pública passa a mostrar o motivo na coluna Message.
- Nada muda no dashboard, nas páginas sem `show-messages` e no payload da página (que continua sem mensagens).

## Impact

- Specs: `public-status-pages` (requisito "Mensagens opcionais nas status pages").
- Código: `statuspage/payload.go` (`publicMessage`), `common.ResultSummary` e as duas cargas do resumo (`storage/store/sql`, `storage/store/memory`), `api/external_endpoint.go` (origem dos resultados da API externa).
- Documentação: `docs/status-pages.md` e `docs/push-monitoring.md`; E2E em `test/e2e/status-pages.sh`.
- Frontend: nenhuma mudança — a tabela já mostra `message` quando vem preenchido.
