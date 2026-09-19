## ADDED Requirements

### Requirement: Logo das telas
O cabeçalho do dashboard, o cabeçalho das páginas públicas de status e o cartão da tela de login (do `security.basic` e do OIDC) MUST mostrar o mesmo logo, definido por `ui.logo`:
- sem `ui.logo`, ou com o valor vazio ou só de espaços, o logo MUST ser o embutido no binário, servido em `/logo-192x192.png` sem autenticação;
- com `ui.logo: none`, sem diferenciar maiúsculas de minúsculas e ignorando espaços nas pontas, nenhum logo MUST ser mostrado;
- com qualquer outro valor, o logo MUST ser a URL informada.

A regra MUST ser resolvida no servidor, de modo que `window.config.logo` chegue ao frontend já com o caminho do logo ou vazio.

#### Scenario: Instalação sem ui.logo
- **WHEN** a configuração não define `ui.logo`
- **THEN** o cabeçalho do dashboard, o de uma página pública e o cartão de login mostram a imagem `/logo-192x192.png`

#### Scenario: Logo escondido
- **WHEN** a configuração tem `ui.logo: None`
- **THEN** nenhuma das três telas mostra logo

#### Scenario: Logo próprio
- **WHEN** a configuração tem `ui.logo: "https://example.org/logo.png"`
- **THEN** as três telas mostram essa imagem
