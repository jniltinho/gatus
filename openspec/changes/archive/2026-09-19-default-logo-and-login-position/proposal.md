## Why

Sem `ui.logo`, o cabeçalho do dashboard, o das páginas públicas e a tela de login saem sem logo nenhum, embora o binário já sirva o logo do projeto em `/logo-192x192.png`. Uma instalação nova parece inacabada até alguém descobrir a opção. O dono do projeto também pediu o cartão de login mais perto do topo: 10% da altura da janela, no lugar de 15%.

## What Changes

1. **Logo padrão:** sem `ui.logo`, as três telas mostram o logo embutido (`/logo-192x192.png`).
2. **`ui.logo: none`** (sem diferenciar maiúsculas, com espaços tolerados) esconde o logo. Um valor vazio deixa de significar "sem logo" e passa a significar "o padrão".
3. Um `ui.logo` com URL continua valendo como hoje.
4. **Cartão de login a 10%** da altura da janela.

**Mudança visível na atualização:** quem não definiu `ui.logo` passa a ver o logo do projeto no cabeçalho e no login. Para manter como estava, `ui.logo: none`. Vai nas notas da release.

## Capabilities

### New Capabilities
- `ui-branding`: o logo mostrado pelas telas e como `ui.logo` o define.

### Modified Capabilities
- `basic-login-page`: o cartão a 10% do topo, e o logo do cartão seguindo `ui.logo`.

## Impact

- **Código:** `internal/config/ui` (`defaultLogo`, `NoLogo`, normalização em `ValidateAndSetDefaults`), `web/app/src/views/LoginPage.vue`.
- **Testes:** `TestConfig_Logo`, `test/e2e/login.sh` (logo embutido no cartão, no cabeçalho do dashboard e no da página pública; cartão a 10%).
- **Docs:** `ui.logo` em `docs/README.md`, `config.yaml` de exemplo.
- Sem mudança de API: `window.config.logo` já existia; só muda o valor padrão.
