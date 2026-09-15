## Context

Estado atual:

- **Captura:** toda verificação HTTPS (`config/endpoint/endpoint.go`, `response.TLS.PeerCertificates[0]`), `tls://` e `starttls://` preenche `endpoint.Result.CertificateExpiration` com `time.Until(certificate.NotAfter)`, sem depender de condição. Resultados de push, falhas de conexão e endpoints sem TLS ficam com `0`.
- **Gravação:** o valor vai para `endpoint_results.certificate_expiration` em SQLite, PostgreSQL e MySQL/MariaDB e é lido de volta com os resultados. O store em memória guarda o objeto.
- **API protegida:** o campo tem `json:"-"`, então `/api/v1/endpoints/{key}/statuses` não o devolve e o dashboard não o mostra.
- **Status pages:** o payload público usa `common.ResultSummary` (`Timestamp`, `Success`, `Duration`), lido por `GetEndpointSummaries` somente com as colunas publicáveis. Os testes de lista permitida decodificam o JSON com `DisallowUnknownFields`.
- **Definições das páginas:** `config/statuspage.Page` é decodificada com o decoder YAML e `KnownFields(true)`, inclusive quando a administração envia JSON, e é gravada com `yaml.Marshal`. O detalhe da administração devolve a definição com as tags JSON.

Referência do Uptime Kuma 2.5.4:

- **Página do monitor:** "Cert Exp." com a data e os dias restantes.
- **Status page:** a opção "Show Certificate Expiry" mostra abaixo do nome uma etiqueta "Cert Exp.: N days", verde acima de 7 dias e vermelha a partir de 7 dias.

O dono do fork pediu que a informação fique discreta.

## Goals / Non-Goals

**Goals:**
- Mostrar os dias até o vencimento do certificado no dashboard e, com opção por página, nas status pages públicas.
- Reaproveitar o dado já gravado, sem migração e sem consulta de rede na montagem das páginas.
- Manter o payload público na lista permitida, com o campo novo só quando a página pede.
- Visual discreto, com cor só perto do vencimento.

**Non-Goals:**
- Alertas de vencimento (já existe a condição `[CERTIFICATE_EXPIRATION]`) e uma opção de formulário para gerar essa condição.
- Expiração de domínio (`[DOMAIN_EXPIRATION]` só é consultado quando uma condição o usa).
- Data exata do vencimento nas páginas públicas e detalhes do certificado (emissor, cadeia).

## Decisions

### D1. Fonte: último resultado com certificado

O vencimento é `timestamp + certificateExpiration` do resultado mais recente cuja expiração não seja zero, entre os resultados já carregados. Resultados de push e falhas de conexão têm zero e são ignorados, então um push não apaga a informação de um endpoint ativo.

**Alternativas consideradas:**
- **Usar só o último resultado:** rejeitada, porque um push ou uma falha de rede esconderia a informação.
- **Guardar a data do vencimento em uma coluna ou tabela nova:** rejeitada, porque o dado já existe e a data se deriva do instante da verificação.
- **Consultar o certificado na montagem da página pública:** rejeitada por fazer conexões de rede numa rota pública e sem autenticação.

### D2. API protegida com o campo do resultado

`endpoint.Result.CertificateExpiration` passa de `json:"-"` para `json:"certificateExpiration,omitempty"`, uma duração em nanossegundos como `duration`. O dashboard calcula o vencimento como `timestamp + certificateExpiration`.

**Alternativas consideradas:**
- **Campo derivado `certificateExpiresAt` só no fork:** rejeitada, porque exigiria um tipo de saída próprio para o status do endpoint, que hoje serializa `endpoint.Status` diretamente.
- **Rota nova só para o certificado:** rejeitada, porque os resultados já chegam na mesma chamada.

### D3. Payload público com dias inteiros e opção por página

- **Campo da página:** `Page` ganha `ShowCertificateExpiration bool` com `yaml:"show-certificate-expiration,omitempty"` e `json:"show-certificate-expiration,omitempty"`. A mesma chave nas duas tags evita dois nomes, porque as definições gerenciadas passam pelo decoder YAML e o detalhe é devolvido em JSON.
- **Campo do payload:** `EndpointPayload` ganha `CertificateExpiresInDays *int` com `json:"certificateExpiresInDays,omitempty"`, preenchido só com a opção ligada e com um resultado com certificado.
- **Cálculo:** `floor((vencimento − now) / 24h)`, negativo depois do vencimento, com o `now` da montagem.
- **Onde vale:** grupos, destaques e página de detalhes, pois todos usam `buildEndpointPayload`.

**Alternativas consideradas:**
- **Publicar a data do vencimento:** rejeitada, porque os dias bastam, como no Kuma, e o spec atual proíbe datas de expiração no payload.
- **Ligar para todas as páginas sem opção:** rejeitada, porque o Kuma usa opção por página e a informação não deve aparecer sem que o administrador escolha.
- **Opção global em `status-pages`:** rejeitada, porque páginas diferentes têm públicos diferentes.

### D4. Resumo dos endpoints com a expiração

`common.ResultSummary` ganha `CertificateExpiration time.Duration`:

- `GetEndpointSummaries` do SQL passa a ler a coluna `certificate_expiration` na mesma consulta com `ROW_NUMBER`;
- o store em memória copia o campo.

**Alternativa considerada:** uma consulta separada só para a expiração, quando a opção estiver ligada. Rejeitada porque é uma coluna a mais na consulta existente, sem custo relevante, e evita uma segunda ida ao banco.

### D5. Texto e cores discretos

Um utilitário comum no frontend, `utils/certificate.js`, formata e escolhe a classe:

- **Texto:**
  - "Certificate expires in N days";
  - "Certificate expires today" (0 dias);
  - "Certificate expired N days ago";
  - no dashboard, a data vem junto: "· Dec 1, 2026".
- **Cores:** cor secundária (`text-muted-foreground`, `text-xs`) acima de 14 dias; âmbar de 8 a 14 dias; vermelho com 7 dias ou menos, ou vencido. Todas com variantes `dark:`.
- **Posição no dashboard:** uma linha abaixo do nome e do grupo, no cabeçalho, sem card novo.
- **Posição na status page:** uma linha pequena abaixo da linha do nome em `EndpointRow.vue` (grupos e destaques) e abaixo do título na página pública de detalhes.

**Alternativas consideradas:**
- **Card na grade de estatísticas do dashboard:** rejeitada, por pedido de discrição.
- **Etiqueta colorida sempre, como no Kuma:** rejeitada, porque chama atenção mesmo quando falta muito tempo.
- **Duas faixas (verde e vermelho):** rejeitada, porque a faixa âmbar avisa com antecedência sem alarmar.

### D6. Formulário da status page

A seção General do formulário ganha a caixa "Show certificate expiration", abaixo de Published, com a explicação "Shows below the name of each endpoint how many days are left until its TLS certificate expires". A caixa envia `show-certificate-expiration` só quando marcada, e a pré-visualização mostra os dias.

**Alternativa considerada:** deixar a opção só no YAML. Rejeitada, porque as páginas gerenciadas devem ter as mesmas opções pela web.

## Risks / Trade-offs

- **[Dias defasados até a próxima verificação]** → O vencimento é calculado a partir do instante da verificação, e o `now` da montagem, então os dias contam corretamente mesmo sem verificação nova. O cache das páginas é de poucos segundos.
- **[Endpoint com `client.insecure`]** → O certificado continua sendo lido do handshake. Um certificado inválido também mostra a data dele, sem indicar que é inválido (diferente do "Bad cert" do Kuma). Isso fica documentado.
- **[Informação a mais no payload público]** → Só com a opção da página. O teste de lista permitida cobre os dois casos, e o campo tem só dias, sem data, hostname ou emissor.
- **[Conflito com o upstream na tag JSON]** → É uma linha em `result.go`. O frontend do upstream ignora o campo.
- **[Rollback para versão anterior do fork]** → Uma versão anterior recusa definições gerenciadas com `show-certificate-expiration` por decodificação estrita e marca a página como inválida. Antes de voltar, é preciso desligar a opção nas páginas.

## Migration Plan

- Não há migração de dados: a coluna `certificate_expiration` já existe nos quatro bancos.
- Definições sem o campo continuam válidas, e a opção começa desligada.
- **Rollback:** desmarcar "Show certificate expiration" nas páginas gerenciadas antes de voltar para uma versão anterior. No YAML, remover o campo.

## Open Questions

- Nenhuma que bloqueie. As faixas de 14 e 7 dias podem virar configuração se o uso pedir.
