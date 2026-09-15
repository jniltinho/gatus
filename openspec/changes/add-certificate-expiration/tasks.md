## 1. Backend: expiração no resumo, no payload e na API protegida

- [x] 1.1 `config/endpoint/result.go`: `CertificateExpiration` com `json:"certificateExpiration,omitempty"`. Teste de que um resultado HTTPS serializado tem o campo e um resultado sem certificado não tem.
- [x] 1.2 `storage/store/common`: `ResultSummary.CertificateExpiration`. Ler a coluna `certificate_expiration` em `storage/store/sql/endpoint_summary_batch.go` e copiar o campo em `storage/store/memory/endpoint_uptime_batch.go`, com testes nos stores memory, SQLite, PostgreSQL e MySQL/MariaDB (conformidade).
- [x] 1.3 `config/statuspage.Page`: campo `ShowCertificateExpiration` (`show-certificate-expiration` nas tags YAML e JSON), aceito no YAML e nas definições gerenciadas (decodificação estrita), com teste de definição com e sem o campo.
- [x] 1.4 `statuspage/payload.go`: `EndpointPayload.CertificateExpiresInDays` (`*int`, `omitempty`), preenchido só com a opção da página, a partir do resultado mais recente com certificado, com `floor` dos dias e valor negativo depois do vencimento. Vale para grupos, destaques e página de detalhes.
- [x] 1.5 Testes do payload:
  - opção desligada sem o campo;
  - 73 dias;
  - vencido há 36 horas (-2);
  - push depois da verificação;
  - sem resultado com certificado;
  - listas permitidas (`payload_test.go`) com o campo opcional, ainda recusando qualquer outro.

## 2. Frontend: dashboard, status page e formulário

- [x] 2.1 `web/app/src/utils/certificate.js`: dias a partir de `timestamp + certificateExpiration`, texto ("Certificate expires in N days", "Certificate expires today", "Certificate expired N days ago") e classe discreta (cor secundária acima de 14 dias, âmbar de 8 a 14, vermelho com 7 ou menos), com variantes `dark:`.
- [x] 2.2 `views/EndpointDetails.vue`: linha pequena abaixo do nome e do grupo com os dias e a data, a partir do resultado mais recente com certificado da primeira página; sem linha quando não houver certificado.
- [x] 2.3 `components/public/EndpointRow.vue` (grupos e destaques) e `views/public/StatusPageEndpoint.vue`: linha pequena com os dias quando o endpoint tiver `certificateExpiresInDays`, sem data.
- [x] 2.4 `views/admin/AdminStatusPageForm.vue`: caixa "Show certificate expiration" na seção General, abaixo de Published, com explicação curta, enviando `show-certificate-expiration` e carregando o valor salvo.
- [x] 2.5 Lint e `make frontend-build`.

## 3. Documentação e E2E

- [x] 3.1 `docs/status-pages.md`:
  - campo `show-certificate-expiration` na tabela e no exemplo;
  - o que a página mostra;
  - campo `certificateExpiresInDays` na API pública.
- [x] 3.2 `docs/push-monitoring.md` (seção Dashboard: linha do certificado na página de detalhes do endpoint) e nota do fork em `docs/README.md` na condição `[CERTIFICATE_EXPIRATION]`.
- [x] 3.3 E2E com agent-browser:
  - endpoint HTTPS local com certificado autoassinado e `client.insecure`;
  - linha discreta no dashboard;
  - opção ligada no formulário e linha na status page;
  - página sem a opção sem a linha;
  - prints claro e escuro em `dist/prints/`.
- [x] 3.4 `go test ./... -race` com PostgreSQL, MySQL e MariaDB, `make lint` e `openspec validate add-certificate-expiration --strict`.
- [ ] 3.5 PR no `jniltinho/gatus` com CI verde e merge, release com imagem no Docker Hub e arquivamento da change.
