## Why

O Uptime Kuma mostra quando o certificado TLS de cada monitor vence: "Cert Exp." com a data e os dias restantes na página do monitor e, com a opção "Show Certificate Expiry" da status page, uma etiqueta com os dias abaixo do nome do monitor. O fork já captura a expiração do certificado em toda verificação HTTPS, `tls://` e `starttls://` e a grava em `endpoint_results.certificate_expiration`, mas nenhuma tela a mostra: a API esconde o campo e o payload público nem o lê. Quem migra do Kuma perde essa informação, que evita certificados vencidos sem precisar configurar condições.

## What Changes

- **Dashboard** (`/endpoints/<key>`): uma linha discreta abaixo do nome e do grupo, "Certificate expires in 73 days · Dec 1, 2026", em texto pequeno e na cor secundária. Ela ganha cor só perto do vencimento (âmbar com até 14 dias, vermelho com até 7 dias ou vencido). Endpoints sem certificado não mostram a linha.
- **API protegida de status** (`/api/v1/endpoints/{key}/statuses`): cada resultado com certificado passa a trazer `certificateExpiration`, a duração até o vencimento no momento da verificação.
- **Status pages públicas**: opção por página `show-certificate-expiration`, desligada por padrão, no YAML e na administração, como o "Show Certificate Expiry" do Kuma. Com a opção ligada:
  - o payload público de cada endpoint (grupos, destaques e página de detalhes) traz `certificateExpiresInDays`, o número inteiro de dias até o vencimento, calculado a partir do último resultado com certificado;
  - a página mostra abaixo do nome, também de forma discreta, "Certificate expires in 73 days".
- **Formulário de status page**: caixa "Show certificate expiration" na seção General.
- **Storage**: o resumo dos endpoints usado pelas status pages passa a ler `certificate_expiration`, que já existe nos quatro bancos, sem migração.
- Documentação em `docs/status-pages.md` e `docs/push-monitoring.md`, onde couber, e em `docs/README.md` (seção do fork).

Fora do escopo:
- alertas de vencimento: a condição `[CERTIFICATE_EXPIRATION] > 168h` já existe;
- expiração do domínio: só é consultada quando uma condição usa `[DOMAIN_EXPIRATION]`;
- mostrar a data exata do vencimento nas páginas públicas.

## Capabilities

### New Capabilities
- `certificate-expiration`: cálculo e exibição da expiração do certificado TLS no dashboard, na API protegida de status e nas status pages com a opção ligada, com o texto discreto e as faixas de cor.

### Modified Capabilities
- `public-status-pages`: a página aceita `show-certificate-expiration`, e o payload público sanitizado passa a permitir `certificateExpiresInDays` somente quando a página tem a opção ligada.
- `status-page-highlights`: a API pública de detalhes do endpoint passa a incluir `certificateExpiresInDays` quando a página tem a opção ligada.
- `status-page-web-ui`: o formulário de status page ganha a opção "Show certificate expiration".

## Impact

- **Backend**:
  - `config/endpoint/result.go`: o campo `CertificateExpiration` passa a ser serializado na API protegida;
  - `config/statuspage/statuspage.go`: novo campo da página;
  - `storage/store/common` (`ResultSummary`), `storage/store/sql/endpoint_summary_batch.go` e `storage/store/memory/endpoint_uptime_batch.go`: leitura do campo;
  - `statuspage/payload.go`: cálculo dos dias, com os testes de lista permitida atualizados.
- **Frontend**:
  - `views/EndpointDetails.vue` e `components/public/EndpointRow.vue`;
  - `views/public/StatusPageEndpoint.vue` e `views/admin/AdminStatusPageForm.vue`;
  - um utilitário comum de formatação e cores.
- **Compatibilidade**:
  - definições de status pages sem o campo continuam válidas;
  - uma versão anterior do fork recusa definições gerenciadas com o campo novo, que é desconhecido para ela;
  - o upstream não usa `certificateExpiration` no JSON.
- **Testes**: payload com e sem a opção, lista permitida, stores memory, SQLite, PostgreSQL e MySQL/MariaDB, e E2E das status pages e do dashboard.
