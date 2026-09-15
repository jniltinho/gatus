## ADDED Requirements

### Requirement: Expiração do certificado na API protegida de status
A API protegida de status (`/api/v1/endpoints/statuses` e `/api/v1/endpoints/{key}/statuses`) MUST devolver `certificateExpiration` em cada resultado que tenha certificado: a duração, em nanossegundos, entre o instante da verificação e o vencimento do certificado TLS apresentado pelo servidor. Resultados sem certificado MUST NOT ter o campo. O valor MUST ser o que a verificação já grava no storage, sem nova consulta de rede.

#### Scenario: Verificação HTTPS
- **WHEN** o endpoint `core_site` verifica `https://example.org` e o certificado vence em 73 dias
- **THEN** o resultado em `/api/v1/endpoints/core_site/statuses` tem `certificateExpiration` equivalente a cerca de 73 dias

#### Scenario: Endpoint sem TLS
- **WHEN** o endpoint verifica `http://example.org` ou recebe um push
- **THEN** o resultado não tem `certificateExpiration`

### Requirement: Expiração do certificado discreta no dashboard
A página de detalhes do endpoint no dashboard (`/endpoints/<key>`) MUST mostrar, abaixo do nome e do grupo, uma linha com os dias até o vencimento do certificado e a data do vencimento, calculados a partir do resultado mais recente com `certificateExpiration`. A linha MUST usar texto pequeno. Acima de 14 dias, MUST usar a cor secundária. De 8 a 14 dias, MUST usar âmbar. Com 7 dias ou menos, ou já vencido, MUST usar vermelho. Todas as cores MUST ter variantes `dark:`. Sem resultado com certificado, a linha MUST NOT aparecer. A página MUST NOT ganhar um cartão novo para essa informação.

#### Scenario: Certificado longe do vencimento
- **WHEN** o resultado mais recente com certificado indica vencimento em 73 dias
- **THEN** a página mostra "Certificate expires in 73 days" com a data, em texto pequeno na cor secundária

#### Scenario: Certificado perto do vencimento
- **WHEN** faltam 10 dias
- **THEN** a linha aparece em âmbar
- **AND** com 5 dias ou com o certificado vencido há 2 dias ("Certificate expired 2 days ago") a linha aparece em vermelho

#### Scenario: Push depois da verificação
- **WHEN** um endpoint ativo com push ligado recebe um push depois de uma verificação HTTPS
- **THEN** a linha continua mostrando os dias da última verificação com certificado

#### Scenario: Endpoint sem certificado
- **WHEN** o endpoint é TCP, ICMP ou HTTP sem TLS
- **THEN** a página não mostra a linha do certificado

### Requirement: Expiração do certificado nas status pages
Com `show-certificate-expiration: true` na página, a página pública MUST mostrar, abaixo da linha do nome de cada endpoint (nos grupos e nos destaques) e abaixo do título da página pública de detalhes do endpoint, uma linha pequena com os dias até o vencimento, a partir de `certificateExpiresInDays`. As faixas de cor MUST ser as mesmas do dashboard, e a linha MUST NOT mostrar a data. Sem a opção, ou sem `certificateExpiresInDays` no endpoint, a linha MUST NOT aparecer.

#### Scenario: Página com a opção ligada
- **WHEN** a página `services` tem `show-certificate-expiration: true` e o endpoint `site` tem certificado que vence em 73 dias
- **THEN** a página `/status/services` mostra "Certificate expires in 73 days" abaixo do nome `site`
- **AND** `/status/services/endpoints/<chave-de-site>` mostra a mesma linha abaixo do título

#### Scenario: Página sem a opção
- **WHEN** a página não tem `show-certificate-expiration`
- **THEN** nenhuma linha de certificado aparece e o payload não tem `certificateExpiresInDays`

### Requirement: Dias até o vencimento no payload público
Com a opção ligada na página, cada endpoint do payload público (`groups[].endpoints[]`, `featured[]` e a API pública de detalhes) MUST ter `certificateExpiresInDays`, calculado como `floor((timestamp + certificateExpiration − instante da montagem) / 24 horas)` a partir do resultado mais recente com certificado entre os resultados publicados. O valor MUST ser negativo depois do vencimento e MUST ser omitido quando nenhum resultado publicado tiver certificado. O cálculo MUST usar os dados já lidos do storage na montagem, sem consulta de rede.

#### Scenario: Vencido
- **WHEN** o certificado venceu há 36 horas no instante da montagem
- **THEN** `certificateExpiresInDays` é -2

#### Scenario: Sem resultado com certificado
- **WHEN** a página tem a opção ligada e o endpoint só tem resultados sem certificado
- **THEN** o endpoint não tem `certificateExpiresInDays`
