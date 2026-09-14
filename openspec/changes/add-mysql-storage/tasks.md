## 1. Marco 1 — Conector, DSN, esquema e CI

- [x] 1.1 Registrar as decisões do dono (D0): tipo único `mysql`, versões mínimas MySQL 8.4 e MariaDB 10.11, migração fora do escopo e pacote de deploy `mariadb`
- [x] 1.2 Adicionar `github.com/go-sql-driver/mysql` v1.10.1 com `go get` e `go mod tidy`; conferir licença MPL-2.0 e ausência de CGO
- [x] 1.3 `storage/type.go` com `TypeMySQL` e `storage/config.go` exigindo `path` e validando a DSN com `mysql.ParseDSN`, com erro sem a DSN; testes de DSN válida, ausente e malformada sem senha no erro
- [x] 1.4 `placeholders.go`: tradutor `$N` → `?` (tokenizador com literais, identificadores e comentários, repetição e reordenação de argumentos, erro em argumento inexistente, cache limitado); testes unitários
- [x] 1.5 `mysql_connector.go`: conector, conn, tx e stmt do wrapper com tradução, tratamento de `driver.ErrSkip`, stmt com `NumInput() = -1`, repasse de `ConnBeginTx`, `ConnPrepareContext`, `QueryerContext`, `ExecerContext`, `Pinger`, `NamedValueChecker`, `SessionResetter` e `Validator` com asserções, e transação envenenada; testes de `ErrSkip` com `InterpolateParams` ligado e desligado, statement preparado, transação envenenada e reconexão após `KILL`
- [x] 1.6 Conexão com os parâmetros de D1 (`ParseTime`, `Loc`, `time_zone`, `utf8mb4_bin`, `ClientFoundRows`, `sql_mode`, `innodb_lock_wait_timeout`, `InterpolateParams`), pool (25 conexões, 3 min de vida, 1 min ocioso), log só com host e banco, aviso de versão abaixo das mínimas e erro com `innodb_page_size` menor que 16K; `store.Initialize` e `sql.NewStore` com `mysql`; teste de sobrescrita de `loc` e `parseTime` e de variáveis de sessão
- [x] 1.7 Constante `driverMySQL` nos desvios de `NewStore`, `createSchema` e `managed_*.go`, sem mudar o comportamento de SQLite e PostgreSQL
- [x] 1.8 `specific_mysql.go` (D4 e D5): InnoDB, `utf8mb4_bin`, `BIGINT AUTO_INCREMENT`, `DATETIME(6)`, `MEDIUMTEXT` nos textos livres, índices dentro do `CREATE TABLE`, chaves estrangeiras no nível da tabela com cascata, sem `UNIQUE(name, group)`, tabelas do fork incluídas e ramo `mysql` em `managed_*.go`; testes de criação, reinício idempotente e `information_schema.REFERENTIAL_CONSTRAINTS`
- [x] 1.9 Helper de testes com `GATUS_TEST_MYSQL_URL` e `GATUS_TEST_MARIADB_URL` e banco próprio por teste (`CREATE DATABASE` e remoção no `t.Cleanup`), em `mysql_connector_test.go`; a junção com SQLite e PostgreSQL nos helpers do fork fica na tarefa 3.6
- [x] 1.10 `.github/workflows/ci.yml`: serviços `mysql:8.4.11` (porta 3306) e `mariadb:10.11.19` (porta 3307) com healthchecks e variáveis no job principal, e job `storage-latest` com `mysql:9.7.2` e `mariadb:12.3.3`
- [ ] 1.11 `make lint`, `go test ./... -race` com PostgreSQL, MySQL e MariaDB locais; pull request no `jniltinho/gatus`, CI verde e merge

## 2. Marco 2 — Paridade do store do upstream

- [ ] 2.1 Dialeto MySQL para as 4 inserções com `RETURNING` (`LastInsertId` na mesma transação)
- [ ] 2.2 Dialeto MySQL para os 3 upserts (`ON DUPLICATE KEY UPDATE` com `VALUES()` e as mesmas expressões do upstream): alertas disparados e consolidação diária substituem (`col = VALUES(col)`), uptime horário acumula (`col = col + VALUES(col)`)
- [ ] 2.3 Dialeto MySQL para as 3 exclusões de excedentes com corte `<=` por identificador em tabela derivada e o mesmo N do upstream
- [ ] 2.4 `"condition"` citado nas consultas do upstream (SQLite, PostgreSQL e MySQL com `ANSI_QUOTES`); conferir que nenhuma consulta usa aspas duplas para literal de texto
- [ ] 2.5 Retry único da chamada inteira de `InsertEndpointResult` e da inserção de resultado de suite em 1213 e 1205, inclusive no `COMMIT`, só no dialeto MySQL
- [ ] 2.6 `conformance_test.go` em todos os bancos disponíveis: inserção e paginação, limites exatos, uptime horário e diário, médias, alertas disparados, suites, remoção de endpoints com cascata, `Clear`, reinício e horário zero
- [ ] 2.7 Testes de concorrência de inserções com limite baixo e de deadlock forçado (nada gravado pela metade, retry grava tudo); decidir sobre `READ COMMITTED` pelo resultado
- [ ] 2.8 Benchmark de `InsertEndpointResult` com `InterpolateParams` ligado e desligado; manter o padrão de D1 ou ajustar
- [ ] 2.9 `make lint`, `go test ./... -race` com os quatro bancos; pull request, CI verde e merge

## 3. Marco 3 — Tabelas do fork, administração e status pages

- [ ] 3.1 Escrita otimista de `managed_endpoints` e `managed_status_pages` em MySQL e MariaDB (`ClientFoundRows`), inclusive atualização sem mudança de valores
- [ ] 3.2 `isUniqueViolation` com `*mysql.MySQLError` número 1062; testes de chave e slug duplicados (409) com rollback da transação
- [ ] 3.3 Leituras em lote (`GetUptimesByKeys` e `GetEndpointSummaries`) em MySQL e MariaDB comparadas com `GetUptimeByKey` e `GetEndpointStatusByKey`, inclusive conjunto vazio
- [ ] 3.4 Limite de 768 caracteres (`utf8.RuneCountInString`) com `storage.type: mysql` na validação de endpoints, external-endpoints, suites, endpoints de suite e da administração; testes com `mysql`, com chave multibyte e sem o limite nos outros tipos
- [ ] 3.5 `config/config_admin.go` aceitando `mysql`, com a mensagem de tipo atualizada e o aviso de várias instâncias; testes
- [ ] 3.6 Helpers de teste do fork (`managedEndpointTestStores` e similares) usando o helper comum com os quatro bancos
- [ ] 3.7 `make lint`, `go test ./... -race` com os quatro bancos; pull request, CI verde e merge

## 4. Marco 4 — Documentação, deploy e release

- [ ] 4.1 `test/e2e/admin.sh` e `test/e2e/status-pages.sh` com `E2E_STORAGE_TYPE` e `E2E_STORAGE_PATH`; rodar os dois com MariaDB 11.4.13 e com SQLite
- [ ] 4.2 `docs/storage-mysql.md` (DSN, parâmetros sobrepostos, criação de banco e usuário, versões, página de 16K, limite de chave, pool de 25 conexões, backup, várias instâncias, sem migração, volta ao upstream), seção de storage no `README.md`, `docs/admin-endpoints.md` e `docs/status-pages.md`
- [ ] 4.3 Exemplo `.examples/docker-compose-mariadb-storage/` com `jniltinho/gatus` e `mariadb:11.4.13` em tags fixas
- [ ] 4.4 `AGENTS.fork.md` (containers de teste, variáveis e passo de sincronização com o upstream) e contexto do `openspec/config.yaml`
- [ ] 4.5 Pacote de deploy `/home/jnsilva/Projetos/gatus/mariadb` (compose, compose de build, `.env` com versões fixas, config, `gatus.sh`, `Dockerfile`), bind em `127.0.0.1`; subir e conferir dashboard, admin e status page
- [ ] 4.6 `openspec validate add-mysql-storage --strict`, pull request, CI verde e merge; release pela skill `create-release` e atualização do servidor local de validação
