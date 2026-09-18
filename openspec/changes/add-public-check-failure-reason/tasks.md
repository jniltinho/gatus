## 1. Motivo no payload

- [x] 1.1 Classificação em `statuspage/payload.go` conforme D1: cinco constantes, varredura **por categoria sobre todos os erros** na ordem certificado → DNS → timeout → conexão → `Check failed`, marcadores distintivos (sem `certificate`, `dns`, `timeout` e `dial ` soltos, `eof` em minúsculas, `context canceled` fora de `Timeout`), só para resultado sem sucesso, **não pendente**, sem mensagem, sem heartbeat e sem status HTTP publicável.
- [x] 1.2 `HTTP <código>` só com `HTTPStatus >= 100` conforme D3, para o banner SSH (`1`) e o código de saída de comando SSH não virarem `HTTP 1`.
- [x] 1.3 `Connected` em `common.ResultSummary` e nas duas cargas (`storage/store/sql/endpoint_summary_batch.go` e `storage/store/memory/endpoint_uptime_batch.go`) conforme D2, sem publicá-lo; sem erros e sem conexão → `Connection failed`, sem erros e com conexão → `Check failed`.
- [x] 1.4 Origem de push nos resultados da API externa (`api/external_endpoint.go`) conforme D4, para o texto do operador não ser classificado como erro de rede.

## 2. Testes Go

- [x] 2.1 Um caso por motivo com as redações reais e **compostas** do Go, incluindo `Get "https://x": dial tcp: lookup x on 127.0.0.11:53: no such host` e `tls: failed to verify certificate: x509: certificate is valid for *.exemplo.com, not tag.exemplo.com`, verificando a constante e a ausência de host, IP, porta e qualquer trecho do erro.
- [x] 2.2 Falsos positivos: URL com `certificate`, `dns` ou `timeout` no host ou no caminho numa conexão recusada continua `Connection failed`.
- [x] 2.3 Atualizar `statuspage/payload_pending_test.go:TestBuildEndpointDetailsPayload_Messages`, que hoje exige mensagem vazia no `connection refused`: passa a esperar `Connection failed`, mantendo as buscas por `10.0.0.5` e `dial tcp`.
- [x] 2.4 Casos que **não** recebem motivo: resultado pendente, push com mensagem, heartbeat, resultado com sucesso sem status HTTP, resultado da API externa com `error=`, e página sem `show-messages`.
- [x] 2.5 `TestBuildPayload_Allowlist` com um resultado falho e com erros no fixture, provando que o payload da página (não o de detalhes) continua sem `message` e sem `origin`.
- [x] 2.6 Falha sem erro nenhum: TCP sem conexão dá `Connection failed`; condição falha com conexão dá `Check failed`. Banner SSH com status `1` e `connection refused` dá `Connection failed`, sem `HTTP 1`.
- [x] 2.7 Página com `auth` e `show-messages`: o payload autenticado é o mesmo sanitizado, com a constante e sem o texto do erro.

## 3. Contagem de endpoints na faixa

- [x] 3.1 `summary` no payload da página (`statuspage/payload.go`), com `total`, `up`, `down`, `pending` e `unknown`, contando os destaques e as seções do payload, e ausente no payload de detalhes.
- [x] 3.2 Faixa (`web/app/src/components/public/StatusSummary.vue`) mostrando `<n> up` e `<n> down` sempre, e pendentes e sem dados só quando houver, legível nos dois temas e sem rolagem horizontal a partir de 360 px.
- [x] 3.3 Testes Go do `summary`, inclusive numa página truncada, e atualização do struct de `TestBuildPayload_Allowlist` para o campo novo.
- [x] 3.4 E2E: a faixa da página `services` mostra a contagem com o endpoint `offline` fora.

## 4. Documentação e entrega

- [x] 4.1 `docs/status-pages.md` (a frase sobre o que é publicado e a seção de `show-messages`) e `docs/push-monitoring.md`: tabela dos cinco motivos, a nota de que são constantes estáveis em inglês, e que o erro completo continua só no dashboard, inclusive para páginas com `auth`.
- [x] 4.2 E2E em `test/e2e/status-pages.sh`: estender a verificação do endpoint `offline` (`http://127.0.0.1:1/`) na página `messages`, exigindo `"message":"Connection failed"` e mantendo a asserção que prova a ausência de `127.0.0.1`, `connection refused` e `dial tcp` no payload.
- [ ] 4.3 `make lint`, `go test ./...`, `openspec validate add-public-check-failure-reason --strict` e entrega na próxima versão da série.
