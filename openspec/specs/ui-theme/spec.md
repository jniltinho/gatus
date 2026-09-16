# ui-theme Specification

## Purpose
TBD - created by archiving change refine-admin-backup-layout. Update Purpose after archive.
## Requirements
### Requirement: Tema escuro por padrão
Sem uma escolha de tema válida salva pelo visitante, a interface web MUST usar o tema definido por `ui.dark-mode`, que é escuro por padrão, sem seguir a preferência de tema do sistema operacional. Isso vale para o dashboard, as páginas de detalhes, a administração, as páginas públicas e a tela de login.

**Entrega pelo servidor:**
- o HTML entregue MUST já ter o tema inicial, para a página não trocar de tema ao carregar;
- MUST informar o tema padrão configurado;
- MUST ter a cor de tema do navegador (`theme-color`) coerente com o tema.

**Escolha do visitante:**
- a escolha feita pelo botão de tema MUST continuar salva e MUST prevalecer sobre `ui.dark-mode` nas visitas seguintes;
- um valor salvo inválido MUST ser ignorado;
- com `ui.dark-mode: false` e sem escolha salva, a interface MUST usar o tema claro.

#### Scenario: Primeira visita com o sistema em modo claro
- **WHEN** um visitante sem escolha de tema salva, com o sistema operacional em modo claro, abre o dashboard ou a tela de login com a configuração padrão
- **THEN** o HTML entregue já tem o tema escuro e a interface continua escura depois de carregar

#### Scenario: Tema claro escolhido
- **WHEN** o visitante escolhe o tema claro e recarrega a página
- **THEN** a interface continua no tema claro

#### Scenario: Escolha salva inválida
- **WHEN** o cookie de tema do visitante tem um valor diferente de `dark` e `light`
- **THEN** a interface usa o tema de `ui.dark-mode`

#### Scenario: Claro por configuração
- **WHEN** a configuração tem `ui.dark-mode: false` e o visitante não tem escolha salva
- **THEN** a interface é exibida no tema claro

