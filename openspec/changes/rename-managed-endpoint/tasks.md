## 1. Store

- [x] 1.1 `common.ErrEndpointKeyInUse` e `common.ManagedStatusPageUpdate` (slug, definição, versão esperada, autor); método `RenameManagedEndpoint` na interface `ManagedEndpointStore`, com o contrato documentado (D4)
- [x] 1.2 Implementar `RenameManagedEndpoint` em `storage/store/sql/managed_endpoints.go`, numa transação `inTransaction`:
  - conferência de versão;
  - chave nova livre em `managed_endpoints` e `endpoints`;
  - troca de `endpoint_key`, definição e versão;
  - `UPDATE endpoints` quando `moveHistory` (inclusive nome e grupo com a mesma chave);
  - atualizações das status pages com conferência de versão;
  - `apply`, commit e limpeza do `writeThroughCache` das duas chaves.
- [x] 1.3 Testes em `storage/store/sql/managed_endpoints_test.go` com SQLite, PostgreSQL, MySQL e MariaDB:
  - resultados, eventos, uptime e alertas disparados sob a chave nova;
  - mesma chave com texto diferente;
  - chave nova ocupada em `managed_endpoints` e em `endpoints` devolve `ErrEndpointKeyInUse` sem mudanças;
  - versão desatualizada;
  - rollback com `apply` com erro e com página em versão desatualizada;
  - `moveHistory = false`.
- [x] 1.4 Teste de conformidade que compara MySQL, MariaDB e PostgreSQL com SQLite depois de uma renomeação

## 2. managedendpoint

- [x] 2.1 Remover `ErrKeyChanged` e, em `update`, usar `RenameManagedEndpoint` quando o texto de `name` ou `group` mudar
- [x] 2.2 Trocar o estado em memória numa publicação só, removendo a chave antiga e adicionando a nova
- [x] 2.3 Restaurar os alertas disparados pela chave antiga com uma cópia rasa do endpoint novo e copiar os contadores de volta (D6)
- [x] 2.4 Interface `KeyRenameParticipant`, com registro e plano (atualizações das páginas, publicar, descartar, páginas do arquivo afetadas). Chamar na ordem de D5, com rollback: parar a chave nova, reiniciar o endpoint anterior e descartar o plano.
- [x] 2.5 Depois do commit:
  - apagar as séries Prometheus da chave antiga;
  - registrar a auditoria com a chave antiga e a nova, sem segredos;
  - incluir `affectedConfigStatusPages` no `Detail`.
- [x] 2.6 Gerenciado em conflito com o YAML: `moveHistory = false`, sem restaurar alertas nem alterar status pages (D8)

## 3. statuspage

- [x] 3.1 Implementar `PrepareKeyRename` segurando o mutex de `statuspage` e registrar o participante na inicialização. Ele deve:
  - reescrever `endpoints` e `featured` das páginas gerenciadas sem duplicar entradas;
  - serializar com `yaml.Marshal`;
  - listar as páginas do arquivo afetadas;
  - publicar ou descartar.
- [x] 3.2 Invalidar o `publicCache` das páginas alteradas depois de publicar
- [x] 3.3 A exposição (`GET /api/v1/admin/status-pages/exposure`) passa a considerar `featured`
- [x] 3.4 Testes:
  - página gerenciada com a chave em `endpoints` e em `featured`;
  - chave nova já presente na lista, sem duplicar;
  - página do arquivo listada e não alterada;
  - descarte do plano no erro;
  - `statuspage` sem adquirir `statesMutex`.

## 4. API

- [x] 4.1 `api/admin.go`:
  - `ErrEndpointKeyInUse` e página alterada por outra instância respondem 409;
  - invalidar `endpoint-status-*` na alteração;
  - `affectedConfigStatusPages` na resposta.
- [x] 4.2 `api/admin_test.go`: trocar o subteste do 400 por renomeação e ajustar as versões dos subtestes seguintes.
  - Na renomeação: 200, chave nova e `ETag`; statuses com o histórico; chave antiga ausente; séries Prometheus da antiga apagadas.
  - 409 para chave do YAML, de outro gerenciado e com histórico armazenado.
  - 412 com versão desatualizada.
  - Renomear de volta.
- [x] 4.3 Testes de renomeação durante verificação lenta (nada gravado sob a chave antiga depois da resposta) e de alerta disparado preservado (sem novo disparo, resolve com a chave de resolução original)
- [x] 4.4 `main_managed_test.go`: histórico da chave nova preservado depois de `initializeStorage` (reinício e recarga)
- [x] 4.5 Teste de concorrência com `-race`: renomear enquanto uma status page gerenciada é salva, sem deadlock

## 5. Frontend

- [x] 5.1 `AdminEndpointForm.vue`: nome e grupo editáveis em endpoints gerenciados, com o seletor de grupo carregado também na edição e o grupo atual selecionado
- [x] 5.2 Aviso de troca de chave: chave antiga e nova, mudança das URLs de badges e da página de detalhes, e status pages do arquivo afetadas (exposição com a chave antiga, origem `config`, motivo `key`)
- [x] 5.3 Depois de salvar, mostrar `affectedConfigStatusPages` e voltar para a lista; `describeAdminError` com texto para o 409 de chave ocupada
- [x] 5.4 Lint e `make frontend-build` (`web/static/`)

## 6. E2E e documentação

- [x] 6.1 `test/e2e/admin.sh`: trocar o passo "nome bloqueado" pela troca de grupo. Conferir o aviso, a chave nova na lista e o histórico preservado, e ajustar os passos seguintes à chave nova.
- [x] 6.2 `test/e2e/status-pages.sh`: renomear um endpoint em destaque numa status page gerenciada e conferir a página pública
- [x] 6.3 `docs/admin-endpoints.md`: uso, comportamento (renomeação, histórico, URLs, status pages, várias instâncias), API e códigos de resposta
- [x] 6.4 `AGENTS.fork.md`: referência à change e às regras de renomeação

## 7. Verificação e entrega

- [x] 7.1 `make lint test` e testes de `storage/store/sql` com `GATUS_TEST_POSTGRES_URL`, `GATUS_TEST_MYSQL_URL` e `GATUS_TEST_MARIADB_URL`
- [x] 7.2 `openspec validate rename-managed-endpoint --strict`
- [ ] 7.3 PR no `jniltinho/gatus` com CI verde e merge
