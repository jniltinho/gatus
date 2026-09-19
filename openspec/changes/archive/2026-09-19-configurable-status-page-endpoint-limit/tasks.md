## 1. Configuração e validação

- [x] 1.1 `maximum-endpoints-per-page` em `internal/config/statuspage` (inteiro de 1 a 1000, padrão 400; 0, negativo, 1001 e valor não inteiro invalidam a configuração, também em `gatus config validate`), com testes.
- [x] 1.2 `MaximumEndpointKeys = 1000` na validação das chaves de uma definição, no lugar de `MaximumEndpoints`; testes com 200, 201, 1000 e 1001 chaves, para definição do arquivo, enviada, gravada e restaurada.

## 2. Seleção e acesso

- [x] 2.1 `MaximumEndpoints` no snapshot e em `Published`: em `Load`, nos valores iniciais, em `Lookup` e na cópia de `cloneCurrentSnapshot`; `Select` recebendo o limite; aviso do log com o limite em vigor. Teste de criar, editar e renomear página depois de carregar um limite diferente de 200.
- [x] 2.1a `findShownEndpoint`, `IsEndpointShownOf` e a montagem do payload sobre a mesma captura; `PublicResponseTimeChart` recebendo o `Published` autenticado, sem `IsEndpointShown(slug, …)` nem segundo `Lookup`; contagens da administração e pré-visualização lendo estado e limite do mesmo snapshot.
- [x] 2.2 Testes da seleção: padrão; limite maior que a seleção; limite 1 com vários destaques (um destaque exibido, `truncated`); mais de 50 seções alcançadas por chaves; a ordem dos destaques primeiro preservada.
- [x] 2.3 Testes de acesso acima do corte, para cada rota por endpoint de uma página (API de detalhes, gráfico, stream de eventos, badge de saúde e de tempo de resposta): 200 dentro do corte e 404 fora, com o limite no padrão, maior e menor; o HTML de detalhes responde 200 com a SPA nos dois casos; numa página com login, 401 sem credencial antes de qualquer 404, e 429 do limitador respeitado.
- [x] 2.4 Recarga da configuração com o limite alterado: o payload, as rotas por endpoint e as contagens mudam juntos; o payload e os detalhes guardados antes não são servidos para um endpoint que saiu do corte, mesmo sem mudança na sequência de resultados; um stream aberto é encerrado e a reconexão é recusada; teste concorrente de leituras do gráfico com troca de limite e de credencial.

## 3. Administração e página pública

- [x] 3.1 `truncated` em `Item` e aviso de tipo próprio na validação; listagem e formulário mostrando; `describeWarnings` do restore (`internal/adminbackup/restore.go`) com o texto certo para esse aviso, no lugar de "selects nothing"; pré-visualização truncada testada para página do arquivo, desabilitada e com login; comentários godoc e contrato HTTP.
- [x] 3.2 "Showing the first N services" com `summary.total`, na página pública e na pré-visualização.

## 4. Verificação

- [x] 4.1 `make frontend-build`; E2E em `status-pages.sh` ou suíte própria: página com mais endpoints que o limite, aviso com o número certo, endpoint fora do corte com 404 na página de detalhes, limite alterado com o Gatus no ar.
- [x] 4.2 `go test ./... -race`, `make lint`, testes de unidade do frontend, contrato HTTP e as demais suítes E2E.
- [x] 4.3 *Fixture* reproduzível de 1.000 endpoints com 50 resultados: tamanho do payload com e sem gzip, tempo de montagem, resposta do navegador com os grupos recolhidos e com tudo expandido, e **memória e concorrência com várias páginas grandes e seus detalhes** (o cache guarda até 1000 entradas sem teto de bytes). Definir e implementar a proteção que a medição pedir (orçamento em bytes do cache, teto menor, ou os dois) antes da release. **Medido:** ~3,9 KiB por endpoint (1.000 endpoints: 3,9 MiB, 27–37 KiB com gzip, montagem em ~20 ms); navegador com 1.000 linhas expandidas: ~70 mil nós DOM e 56 MiB, 275 nós e 6 MiB com os grupos recolhidos. **Proteção:** orçamento de 128 MiB no cache dos payloads (`maximumPublicCacheMemory`); o teto de 1000 fica.

## 5. Entrega

- [x] 5.1 `docs/status-pages.md` (a opção, os dois limites, o corte como acesso, o custo de um limite alto, a volta de versão), `config.yaml` de exemplo com a opção comentada, `AGENTS.fork.md`.
- [x] 5.2 Regerar os blocos MODIFICADOS se `add-bio-theme` tiver sido arquivada antes; PR com CI verde; release com notas em inglês, `test/e2e/upgrade.sh` antes da tag, imagem, `mariadb/`, exemplos; arquivar a change.
