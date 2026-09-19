## 1. Configuração e validação

- [ ] 1.1 `maximum-endpoints-per-page` em `internal/config/statuspage` (inteiro de 1 a 1000, padrão 200; 0, negativo, 1001 e valor não inteiro invalidam a configuração, também em `gatus config validate`), com testes.
- [ ] 1.2 `MaximumEndpointKeys = 1000` na validação das chaves de uma definição, no lugar de `MaximumEndpoints`; testes com 200, 201, 1000 e 1001 chaves, para definição do arquivo, enviada, gravada e restaurada.

## 2. Seleção e acesso

- [ ] 2.1 Limite no snapshot (`Load`), `Select` recebendo o limite, e os quatro chamadores lendo do snapshot que têm em mãos; aviso do log com o limite em vigor.
- [ ] 2.2 Testes da seleção: padrão; limite maior que a seleção; limite 1 com vários destaques (um destaque exibido, `truncated`); mais de 50 seções alcançadas por chaves; a ordem dos destaques primeiro preservada.
- [ ] 2.3 Testes de acesso acima do corte, para cada rota por endpoint de uma página (detalhes, gráfico, stream de eventos, badge de saúde e de tempo de resposta): 200 dentro do corte, 404 fora, com o limite no padrão, maior e menor.
- [ ] 2.4 Recarga da configuração com o limite alterado: o payload, as rotas por endpoint e as contagens mudam juntos; o cache antigo não é servido; teste de recarga concorrente com leituras.

## 3. Administração e página pública

- [ ] 3.1 `truncated` em `Item` e aviso de tipo próprio na validação; listagem e formulário mostrando; o restore não o traduz como "não seleciona nada"; comentários godoc e contrato HTTP.
- [ ] 3.2 "Showing the first N services" com `summary.total`, na página pública e na pré-visualização.

## 4. Verificação

- [ ] 4.1 `make frontend-build`; E2E em `status-pages.sh` ou suíte própria: página com mais endpoints que o limite, aviso com o número certo, endpoint fora do corte com 404 na página de detalhes, limite alterado com o Gatus no ar.
- [ ] 4.2 `go test ./... -race`, `make lint`, testes de unidade do frontend, contrato HTTP e as demais suítes E2E.
- [ ] 4.3 *Fixture* reproduzível de 1.000 endpoints com 50 resultados: tamanho do payload com e sem gzip, tempo de montagem, e resposta do navegador com os grupos recolhidos e com tudo expandido; ajustar o teto da D6 se for preciso.

## 5. Entrega

- [ ] 5.1 `docs/status-pages.md` (a opção, os dois limites, o corte como acesso, o custo de um limite alto, a volta de versão), tabela de `status-pages` em `docs/README.md`, `config.yaml` de exemplo com a opção comentada, `AGENTS.fork.md`.
- [ ] 5.2 Regerar os blocos MODIFICADOS se `add-bio-theme` tiver sido arquivada antes; PR com CI verde; release com notas em inglês, `test/e2e/upgrade.sh` antes da tag, imagem, `mariadb/`, exemplos; arquivar a change.
