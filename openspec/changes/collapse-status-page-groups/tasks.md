## 1. Definição e payload

- [ ] 1.1 `groups-collapsed` em `internal/config/statuspage` e na normalização das páginas gerenciadas (`internal/statuspage/definition.go`), com testes de tipo errado, ausência e ida e volta.
- [ ] 1.2 `groupsCollapsed` no payload e `summary` em cada grupo (`internal/statuspage/payload.go`), calculado na mesma passada do estado agregado, com os cinco campos; testes com pending, sem resultados, up+unknown, página truncada e destaque fora do ar fora da contagem do grupo.
- [ ] 1.3 Lista de campos permitidos dos testes do payload (`allowedPayload`, `allowedGroup`) e contrato HTTP (`internal/api/contract_test.go`) com os campos novos; comentários godoc dos campos e dos manipuladores.
- [ ] 1.4 Backup e restore de uma página com o campo (teste em `internal/adminbackup`, sem mudança de formato).

## 2. Página pública

- [ ] 2.1 Cabeçalho do grupo como `<button>` (`aria-expanded`, `aria-controls`, foco visível nos dois temas), grupo recolhido sem renderizar as linhas, contagem do grupo no cabeçalho omitindo zeros, aviso de truncamento sempre visível.
- [ ] 2.2 Precedência da D2 aplicada a cada payload (carga, *polling* de 60 s e volta da aba), com o recolhimento durante incidente valendo só até o próximo payload.
- [ ] 2.3 Armazenamento da D3: hash SHA-256 de slug e nome bruto, validação de forma na leitura, teto de 500 entradas, e funcionamento sem `crypto.subtle` ou sem armazenamento.

## 3. Administração

- [ ] 3.1 Opção "Start with the groups collapsed" no formulário, na validação e na pré-visualização, que não lê nem grava as escolhas da página pública.

## 4. Verificação

- [ ] 4.1 `make frontend-build` antes de qualquer teste com o binário.
- [ ] 4.2 `test/e2e/status-pages.sh`: recolher e expandir por clique, Enter e Espaço; lembrar ao recarregar; páginas diferentes não se afetam; `groups-collapsed: true` com um grupo `degraded` aberto; grupo recolhido que passa a falhar é aberto no payload seguinte e volta a fechar ao se recuperar (forçando a busca pela volta da aba, sem esperar 60 s); recolher durante incidente não é lembrado; grupo sem nome e grupo chamado "Other services"; página com login sem nome de grupo no armazenamento; armazenamento bloqueado; tema escuro.
- [ ] 4.3 `go test ./... -race`, `make lint`, contrato HTTP e as demais suítes E2E.

## 5. Entrega

- [ ] 5.1 `docs/status-pages.md` (campo, comportamento, atraso de até 90 s, D5), screenshots (`docs/screenshots/capture.sh`) e `AGENTS.fork.md`.
- [ ] 5.2 PR com CI verde; release com notas em inglês (campos novos no payload e o aviso de volta de versão da D5), `test/e2e/upgrade.sh` antes da tag, imagem, `mariadb/`, exemplos; arquivar a change.
