## 1. Configuração e payload

- [ ] 1.1 `groups-collapsed` em `internal/config/statuspage` (campo, validação de tipo, normalização das páginas gerenciadas), com testes.
- [ ] 1.2 `status-pages.maximum-endpoints-per-page` (padrão 200, 1 a 1000, inválido fora do intervalo), usado na validação das chaves e na montagem (`truncated`, aviso do log), com testes do padrão, de um limite maior, do intervalo e de uma página gravada acima do limite (D6).
- [ ] 1.3 `groupsCollapsed` e `summary` por grupo no payload público (`internal/statuspage/payload.go`), com os destaques fora da contagem do grupo (D5), e testes.
- [ ] 1.4 Aviso na administração para a página que excede o limite em vigor (`Warning`), com teste.
- [ ] 1.5 Comentários godoc dos campos e dos manipuladores afetados, e casos novos no contrato HTTP (`internal/api/contract_test.go`).

## 2. Página pública

- [ ] 2.1 Cabeçalho do grupo como botão (`aria-expanded`, `aria-controls`, foco visível), grupo recolhido sem renderizar as linhas (D1), contagem do grupo no cabeçalho, nos temas claro e escuro.
- [ ] 2.2 Estado inicial: `groupsCollapsed` com a exceção dos grupos não operacionais (D2).
- [ ] 2.3 Escolha do visitante em `localStorage` por página e por grupo, tolerante a armazenamento indisponível e a grupos que sumiram (D3).
- [ ] 2.4 Grupo recolhido pelo visitante que deixa de estar operacional é aberto, inclusive por SSE, e volta a fechar quando se recupera (D2).

## 3. Administração

- [ ] 3.1 Opção "Start with the groups collapsed" no formulário da página, na validação e na pré-visualização.
- [ ] 3.2 Backup e restore de uma página com o campo novo (teste de `adminbackup`, sem mudança de formato).

## 4. Verificação

- [ ] 4.1 `test/e2e/status-pages.sh`: recolher e expandir por clique e por teclado, lembrar ao recarregar, página com `groups-collapsed: true` e um grupo com falha aberto, grupo que falha enquanto está recolhido, tema escuro, `localStorage` bloqueado.
- [ ] 4.2 `go test ./... -race`, `make lint`, contrato HTTP e as demais suítes E2E.
- [ ] 4.3 Medição com 1.000 endpoints: tamanho do payload, tempo de montagem, e tempo de resposta do navegador com tudo recolhido e com tudo expandido; ajustar o teto da D6 se for preciso.

## 5. Entrega

- [ ] 5.1 `docs/status-pages.md`, tabela de `status-pages` em `docs/README.md`, screenshots (`docs/screenshots/capture.sh`) e `AGENTS.fork.md`.
- [ ] 5.2 PR com CI verde; release com notas em inglês (incluindo o aviso de reversão do Migration Plan), `test/e2e/upgrade.sh` antes da tag, imagem, `mariadb/`, exemplos; arquivar a change.
