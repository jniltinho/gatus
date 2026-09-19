## 1. Implementação

- [x] 1.1 `defaultLogo = "/logo-192x192.png"` e `NoLogo = "none"` em `internal/config/ui`, com a normalização em `ValidateAndSetDefaults` e `TestConfig_Logo`.
- [x] 1.2 Cartão de login a 10% (`LoginPage.vue`) e `make frontend-build`.

## 2. Verificação

- [x] 2.1 `test/e2e/login.sh`: logo embutido no cartão, no cabeçalho do dashboard e no da página pública; cartão a 10%.
- [ ] 2.2 `go test ./... -race`, `make lint`, testes de unidade do frontend e as suítes E2E que abrem o cabeçalho.

## 3. Entrega

- [ ] 3.1 `ui.logo` em `docs/README.md` e no `config.yaml` de exemplo; telas de `docs/screenshots` refeitas se o logo aparecer nelas.
- [ ] 3.2 PR com CI verde; nota na release v6.3.0 (mudança visível e `ui.logo: none`); arquivar a change.
