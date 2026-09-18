## MODIFIED Requirements

### Requirement: Expiração do certificado nas status pages
Com `show-certificate-expiration: true` na página, a página pública MUST mostrar, abaixo da linha do nome de cada endpoint (nos grupos e nos destaques) e abaixo do título da página pública de detalhes do endpoint, uma linha pequena com os dias até o vencimento, a partir de `certificateExpiresInDays`. As faixas de cor MUST ser as mesmas do dashboard. Na página de detalhes, a linha MUST mostrar também a data do vencimento, a partir de `certificateExpiresAt`, no mesmo formato do dashboard; nas linhas das listas e nos destaques, a linha MUST mostrar somente os dias, para caber na altura da linha. Sem a opção, ou sem `certificateExpiresInDays` no endpoint, a linha MUST NOT aparecer.

#### Scenario: Página com a opção ligada
- **WHEN** a página `services` tem `show-certificate-expiration: true` e o endpoint `site` tem certificado que vence em 73 dias
- **THEN** a página `/status/services` mostra "Certificate expires in 73 days" abaixo do nome `site`
- **AND** `/status/services/endpoints/<chave-de-site>` mostra a mesma linha abaixo do título

#### Scenario: Página sem a opção
- **WHEN** a página não tem `show-certificate-expiration`
- **THEN** nenhuma linha de certificado aparece e o payload não tem `certificateExpiresInDays`

#### Scenario: Data na página pública de detalhes
- **WHEN** a página `services` mostra a expiração e o visitante abre os detalhes de um endpoint cujo certificado vence em 73 dias
- **THEN** a linha mostra "Certificate expires in 73 days" com a data do vencimento
- **AND** a linha do mesmo endpoint na lista da status page mostra só os dias


### Requirement: Dias até o vencimento no payload público
Com a opção ligada na página, cada endpoint do payload público (`groups[].endpoints[]`, `featured[]` e a API pública de detalhes) MUST ter `certificateExpiresInDays`, calculado como `floor((timestamp + certificateExpiration − instante da montagem) / 24 horas)` a partir do resultado mais recente com certificado entre os resultados publicados, e `certificateExpiresAt`, o instante do vencimento (`timestamp + certificateExpiration`) do **mesmo** resultado, em UTC e no formato RFC 3339. Os dois valores MUST vir do mesmo resultado, o valor em dias MUST ser negativo depois do vencimento e ambos MUST ser omitidos quando nenhum resultado publicado tiver certificado. Nenhum outro dado do certificado MUST ser publicado. O cálculo MUST usar os dados já lidos do storage na montagem, sem consulta de rede.

#### Scenario: Vencido
- **WHEN** o certificado venceu há 36 horas no instante da montagem
- **THEN** `certificateExpiresInDays` é -2

#### Scenario: Sem resultado com certificado
- **WHEN** a página tem a opção ligada e o endpoint só tem resultados sem certificado
- **THEN** o endpoint não tem `certificateExpiresInDays`

#### Scenario: Data e dias do mesmo resultado
- **WHEN** o endpoint tem um resultado com certificado às 10:00 de 1º de janeiro, com 73 dias de validade restantes, e um push mais novo sem certificado
- **THEN** o payload publica `certificateExpiresAt` como o vencimento daquele resultado com certificado
- **AND** `certificateExpiresInDays` é calculado a partir do mesmo instante

