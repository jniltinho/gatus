## 1. Definição e payload

- [ ] 1.1 `groups-collapsed` em `internal/config/statuspage` e na normalização das páginas gerenciadas (`internal/statuspage/definition.go`), com testes de tipo errado, ausência e ida e volta.
- [ ] 1.2 `groupsCollapsed` no payload e `summary` em cada grupo (`internal/statuspage/payload.go`), calculado na mesma passada do estado agregado, com os cinco campos; testes com pending, sem resultados, up+unknown, página truncada e destaque fora do ar fora da contagem do grupo.
- [ ] 1.3 Lista de campos permitidos dos testes do payload (`allowedPayload`, `allowedGroup`) e contrato HTTP (`internal/api/contract_test.go`) com os campos novos; comentários godoc dos campos e dos manipuladores.
- [ ] 1.4 Backup e restore de uma página com o campo (teste em `internal/adminbackup`, sem mudança de formato).

## 2. Página pública

- [ ] 2.1 Cabeçalho do grupo como `<button>` (`aria-expanded`, `aria-controls`, foco visível nos dois temas), grupo recolhido sem renderizar as linhas, contagem do grupo no cabeçalho omitindo zeros, aviso de truncamento sempre visível.
- [ ] 2.1a Texto acessível do botão do cabeçalho com nome, estado e contagem do grupo; resumos e região `aria-live` das linhas voltando ao expandir.
- [ ] 2.2 Precedência da D2 aplicada a cada payload (carga, *polling* de 60 s e volta da aba), com o recolhimento durante incidente valendo só até o próximo payload.
- [ ] 2.3 Escolha da visita em memória, valendo até fechar a página com ou sem armazenamento (D2).
- [ ] 2.4 Chaves de renderização pelo nome bruto do grupo na página pública e na pré-visualização do formulário, no lugar de `__without-group__`.
- [ ] 2.5 Armazenamento da D3: hash SHA-256 de slug e nome bruto, validação de forma na leitura, teto de 500 entradas com descarte das mais antigas, chaves e escolhas resolvidas antes de publicar o payload, conferência do `requestGeneration` e da desmontagem depois de cada `await`, e funcionamento sem `crypto.subtle` ou sem armazenamento; testes de unidade do módulo (dados inválidos, 501ª entrada, derivação atrasada, troca de slug, e duas buscas sobrepostas do mesmo slug com a segunda respondendo 401 e 404).

## 3. Administração

- [ ] 3.1 Opção "Start with the groups collapsed" no formulário, na validação e na pré-visualização, que não lê nem grava as escolhas da página pública.

## 4. Verificação

- [ ] 4.1 `make frontend-build` antes de qualquer teste com o binário.
- [ ] 4.2 `test/e2e/status-pages.sh`: recolher e expandir por clique, Enter e Espaço; lembrar ao recarregar; páginas diferentes não se afetam; `groups-collapsed: true` com um grupo `degraded` aberto; grupo recolhido que passa a falhar é aberto no payload seguinte e volta a fechar ao se recuperar (forçando a busca pela volta da aba, sem esperar 60 s); recolher durante incidente não é lembrado; grupo `unknown` aberto e fechando ao receber dados; sem piscar na primeira exibição; grupo sem nome, grupo chamado "Other services" e grupo chamado `__without-group__`; pré-visualização com escolha pública oposta ao padrão; página com login sem nome de grupo no armazenamento; armazenamento bloqueado; todos os grupos recolhidos numa página sem destaques e reexpansão pelo teclado, com o tooltip abrindo; tema escuro.
- [ ] 4.3 `go test ./... -race`, `make lint`, contrato HTTP e as demais suítes E2E.

## 5. Entrega

- [ ] 5.1 `docs/status-pages.md` (campo, comportamento, atraso de até 90 s, D5), screenshots (`docs/screenshots/capture.sh`) e `AGENTS.fork.md`.
- [ ] 5.2 PR com CI verde; release com notas em inglês (campos novos no payload e o aviso de volta de versão da D5), `test/e2e/upgrade.sh` antes da tag, imagem, `mariadb/`, exemplos; arquivar a change.
