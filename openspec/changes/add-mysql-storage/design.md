## Context

- **Storage atual.** O store SQL (`storage/store/sql/`) atende SQLite (modernc, Go puro) e PostgreSQL (`lib/pq`) com um único `Store`.
  - O tipo do YAML vai direto para `sql.Open` como nome do driver (`sql.go:83`).
  - Os ramos de dialeto se resumem a `driver == "sqlite"`: PRAGMAs, `SetMaxOpenConns(1)` e a escolha do esquema. `managed_*.go` cai no DDL do PostgreSQL para qualquer tipo que não seja SQLite.
- **SQL do upstream.** As consultas usam `$N` e várias construções que o MySQL/MariaDB não aceita:
  - `RETURNING` em 4 inserções;
  - `ON CONFLICT ... DO UPDATE` em 3 upserts;
  - `NOT IN (SELECT ... LIMIT $2)` sobre a própria tabela em 3 exclusões (erros 1235 e 1093);
  - `CREATE INDEX IF NOT EXISTS` e `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, que existem no MariaDB mas não no MySQL;
  - `TEXT` em chaves únicas (erro 1170);
  - `REFERENCES` na definição da coluna, que o MySQL 8.4 aceita e **ignora** (só o 9.6+ aplica);
  - a coluna `condition`, palavra reservada.
- **Erros dentro da transação.** `InsertEndpointResult` só registra no log as falhas de evento, limpeza e uptime e segue até o `Commit` (`sql.go:279-388`).
  - No PostgreSQL, qualquer erro aborta a transação e o `Commit` falha.
  - No InnoDB:
    - um deadlock (1213) desfaz a transação inteira, e os comandos seguintes rodam em autocommit;
    - um lock wait timeout (1205) desfaz só o comando;
    - uma chave duplicada desfaz só o comando.
  - Portar sem cuidado grava metade dos dados e devolve `nil`.
- **SQL do fork.** As tabelas `managed_endpoints` e `managed_status_pages` e as leituras em lote das status pages também usam `$N`.
  - Uma consulta repete placeholders (`$1` três vezes) e outra usa `$1` depois da lista de `IN`.
  - A detecção de chave duplicada compara textos de erro de SQLite e pq.
- **Upstream.** O pedido existe desde 2022 ([TwiN/gatus#283](https://github.com/TwiN/gatus/issues/283)). Uma implementação ([#1003](https://github.com/TwiN/gatus/pull/1003), +588/−113, sem testes) foi recusada: o mantenedor não quer mais tipos de storage.
  - Qualquer mudança nossa em `sql.go` vira conflito potencial a cada sincronização.
- **Testes.** Os helpers de teste do fork rodam em SQLite e, com `GATUS_TEST_POSTGRES_URL`, em PostgreSQL. `sql_test.go` (upstream) roda só em SQLite. O CI sobe um `postgres:17.11-alpine`.
- **Driver.** O `github.com/go-sql-driver/mysql` (v1.10.1, Go puro, MPL-2.0) é mantido para MySQL 8.0+ e MariaDB 10.11+ com Go 1.25+.

## Goals / Non-Goals

**Goals:**
- `storage.type: mysql` funcionando em MySQL 8.4+ e MariaDB 10.11+ com a mesma semântica do PostgreSQL, inclusive nos caminhos de erro, na administração e nas status pages.
- Mudanças mínimas e localizadas nos arquivos do upstream; o grosso em arquivos novos.
- Comportamento independente da configuração do servidor (fuso, `sql_mode`, collation padrão).
- Cobertura de testes nas versões mínimas e nas LTS mais novas, local e no CI.
- Documentação, exemplo de compose e pacote de deploy `mariadb`.

**Non-Goals:**
- Migração de dados entre storages.
- Garantia para MySQL anterior à 8.4, MariaDB anterior à 10.11, TiDB, PlanetScale/Vitess (sem chaves estrangeiras) e Amazon Aurora (deve funcionar por compatibilidade, sem teste).
- Réplicas de leitura ou separação de conexões de leitura e escrita.
- Mudar os esquemas de SQLite e PostgreSQL.

## Decisions

### D0. Decisões do dono
- **Versões mínimas:** MySQL 8.4 LTS e MariaDB 10.11 LTS, as mais antigas ainda com suporte dos fabricantes (MySQL 8.0 encerrou em 30/04/2026 e MariaDB 10.6 em 06/07/2026).
  - A MariaDB 10.11 também é a mínima que os mantenedores do driver suportam.
  - O código só usa recursos presentes nessas versões (`ROW_NUMBER()`, chaves estrangeiras InnoDB, `DATETIME(6)`).
- **CI:**
  - job principal com as mínimas (`mysql:8.4.11` e `mariadb:10.11.19`);
  - job de storage com as LTS mais novas (`mysql:9.7.2` e `mariadb:12.3.3`).
  - O exemplo e o pacote de deploy usam `mariadb:11.4.13` (LTS até 2029), validada pelos E2E do marco 4.
- **Servidor antigo:** o Gatus não bloqueia versões anteriores. Na inicialização, lê `SELECT VERSION()` e registra um aviso quando o servidor está abaixo das mínimas, sem falhar.
  - **Alternativa:** recusar a inicialização. Rejeitada: impediria ambientes que funcionam na prática (por exemplo, MySQL 8.0 ainda em uso) sem ganho de segurança.
- **Tipo:** um único `mysql` para os dois bancos, sem alias `mariadb`; DSN em `storage.path`.
- **Migração de dados:** fora do escopo.
- **Deploy:** pacote `mariadb` no padrão dos pacotes `sqlite` e `postgres`.

### D1. DSN e parâmetros fixos da conexão
- A validação usa `mysql.ParseDSN`. O erro devolvido é genérico ("storage.path is not a valid MySQL DSN") e nunca inclui a DSN, porque o erro do driver pode ecoar trechos dela.
- O store monta a conexão com `mysql.NewConnector(cfg)` a partir do `*mysql.Config` interpretado, sobrepondo o que vier na DSN:
  - **Horários:** `ParseTime=true` e `Loc=time.UTC`. É o `Loc` que decide a conversão de `DATETIME`: o driver converte os argumentos `time.Time` para `Loc` e interpreta as leituras em `Loc`. Uma DSN com `loc=America/Sao_Paulo` ou `parseTime=false` gravaria horários deslocados ou quebraria a leitura.
  - **`time_zone='+00:00'` por sessão:** `DATETIME` não sofre conversão de `time_zone`, mas o valor fixo evita surpresa com `NOW()` e funções de data em consultas futuras.
  - **Texto:** `utf8mb4` com collation `utf8mb4_bin` na conexão. A collation padrão do driver é `utf8mb4_general_ci`, e as colunas indexadas também declaram `utf8mb4_bin` (D5).
  - **`ClientFoundRows=true`:** `RowsAffected` passa a contar linhas encontradas, como no PostgreSQL. A escrita otimista (`WHERE version = ?`, exige 1 linha) não vira falso conflito quando nada muda.
  - **`sql_mode='ANSI_QUOTES,ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION'`:**
    - `ANSI_QUOTES` permite citar `"condition"` do mesmo jeito nos três bancos;
    - `ONLY_FULL_GROUP_BY` mantém a recusa de agrupamentos que o PostgreSQL também recusaria, para a conformidade não esconder erro;
    - o modo estrito transforma truncamento em erro;
    - `NO_ENGINE_SUBSTITUTION` impede criar tabela fora do InnoDB silenciosamente.
    - `NO_ZERO_DATE` e `NO_ZERO_IN_DATE` ficam de fora, de propósito: além de obsoletos, rejeitariam o `time.Time` zero (enviado como `'0000-00-00'`), que SQLite e PostgreSQL gravam e leem de volta.
  - **`innodb_lock_wait_timeout=10` por sessão,** para uma espera de lock virar erro tratável (D6) em vez de travar o watchdog por 50 s.
  - As variáveis de sessão vão em `Config.Params` com o valor entre aspas simples (`"'+00:00'"`).
  - **`InterpolateParams=true`:** a interpolação do driver faz escape seguro com `utf8mb4` (só é proibida em BIG5, CP932, GB2312, GBK e SJIS, que não usamos) e evita prepare + execute + close em cada um dos 10–15 comandos de `InsertEndpointResult`. O benchmark do marco 2 confirmou o ganho: cada `InsertEndpointResult` levou ~6,4 ms contra ~8,3 ms no MySQL 8.4 e ~4,1 ms contra ~6,2 ms no MariaDB 10.11 (servidores locais, 300 gravações × 3). O `ErrSkip` continua tratado para os casos em que o driver recusa a interpolação (D2).
- TLS, timeouts, `allowCleartextPasswords` e demais parâmetros da DSN são preservados.
- **Pool:** `SetMaxOpenConns(25)`, `SetMaxIdleConns(25)`, `SetConnMaxLifetime(3 * time.Minute)` e `SetConnMaxIdleTime(1 * time.Minute)`.
  - O limite evita estourar o `max_connections` padrão (151) com `concurrency: 0`.
  - O tempo de vida fica abaixo dos 5 min que o driver recomenda.
- **Página InnoDB:** na inicialização, lê `@@innodb_page_size` e falha com erro claro abaixo de 16K, porque o limite de índice cai para 1536 ou 768 bytes e o esquema não caberia.
- **Log:** só host e banco (`cfg.Addr`, `cfg.DBName`), nunca usuário e senha.
- **Alternativa:** exigir que o usuário escreva `parseTime=true&loc=UTC...` na DSN. Rejeitada: é o erro mais comum de quem usa o driver, e o fuso errado corrompe dados sem erro visível.

### D2. Conector próprio: tradução de placeholders e transações com erro
- Um `driver.Connector` embrulha o conector do `go-sql-driver/mysql`. O `driver.Conn` devolvido traduz as consultas:
  - troca `$N` por `?`, na ordem de aparição;
  - monta a lista de argumentos repetindo e reordenando pelos ordinais (`$1 ... $1 ... $2` com `(a, b)` vira `? ... ? ... ?` com `(a, a, b)`).
- **Tokenizador:** ignora `$` dentro de `'literais'`, `"identificadores"`, `` `identificadores` `` e comentários `--`, `#` e `/* */`. Referência a argumento inexistente devolve erro sem executar. Traduções ficam num cache limitado por texto de consulta.
- **`ErrSkip`:** quando o conn interno devolve `driver.ErrSkip` em `ExecContext`/`QueryContext`, o wrapper **não** repassa o erro. O `database/sql` chamaria `PrepareContext` com o texto e os argumentos originais, e `NumInput()` do texto traduzido não bateria. O wrapper então prepara o SQL traduzido no conn interno e executa com os argumentos já reordenados:
  - em `Exec`, fecha o statement logo depois da execução;
  - em `Query`, mantém o statement aberto até `Rows.Close`, com um `Rows` que fecha o statement junto (fechar antes invalidaria o cursor);
  - em `PrepareContext`, embrulha o `driver.Stmt` com `NumInput() = -1` (o contrato do `database/sql` para não conferir o número de argumentos) e remapeia os argumentos em `StmtExecContext` e `StmtQueryContext`.
- **Interfaces repassadas ao conn interno,** com asserções `var _ driver.X = (*mysqlConn)(nil)`: `ConnBeginTx`, `ConnPrepareContext`, `QueryerContext`, `ExecerContext`, `Pinger`, `NamedValueChecker`, `SessionResetter` e `Validator`. O `ResetSession` do driver faz a checagem de conexão viva; sem ele, conexões mortas voltariam do pool.
- **Transação envenenada (paridade com o PostgreSQL):** a `driver.Tx` do wrapper marca a transação quando qualquer comando dentro dela devolve erro.
  - Depois disso, os comandos seguintes falham sem executar e o `Commit` faz rollback e devolve o primeiro erro, encadeado com `%w` junto de `errTransactionAborted`, para `errors.As` encontrar o `*mysql.MySQLError` original (1062, 1213 ou 1205).
  - Erros que não vêm do driver, como `sql.ErrNoRows` de um `Scan`, não passam pelo conector e não envenenam a transação, como no PostgreSQL.
  - É exatamente o que o PostgreSQL faz, e fecha o caminho de gravação parcial após deadlock descrito no Context.
- **Alternativas:**
  - (a) Reescrever as consultas do upstream com `?`: `lib/pq` não aceita `?`, o diff seria enorme e conflitaria em toda sincronização.
  - (b) `Rebind` explícito em cada chamada (estilo `sqlx`): muda todas as chamadas do upstream.
  - (c) Envenenar a transação no próprio store: exigiria checar erro em cada comando do upstream.
  - O conector (escolhido) concentra as duas regras num lugar testável, sem mudar chamadas.

### D3. Dialeto para o que não é portável
- Os desvios usam o campo `driver` que o `Store` já tem, comparado com a constante `driverMySQL`, como o upstream faz com `"sqlite"`. O SQL específico do MySQL fica em `dialect_mysql.go`.
  - **Alternativa:** um tipo `dialect` com métodos por banco. Adiada: com cerca de 10 desvios, a comparação direta muda menos linhas do upstream; o tipo vale a pena se os desvios crescerem.
- **`RETURNING <id>`** (4 inserções): emulado no conector, sem mudar as inserções do upstream. Uma consulta `INSERT ... RETURNING <coluna>` é executada como `INSERT` com `Exec`, e o `LastInsertId()` volta como a única linha da consulta.
  - É confiável porque a conexão não é compartilhada durante o comando e as inserções são de uma linha com chave `AUTO_INCREMENT`.
  - O MariaDB tem `RETURNING` nativo desde a 10.5, mas a emulação vale para os dois bancos, para o comportamento ser o mesmo.
  - **Alternativa:** desvio explícito em cada uma das 4 funções, que mudaria mais linhas do upstream.
- **`ON CONFLICT ... DO UPDATE`** (alertas disparados, `endpoint_uptimes` e consolidação diária): `INSERT ... ON DUPLICATE KEY UPDATE`, reproduzindo a expressão de cada coluna do upstream. As três formas são diferentes:
  - **alertas disparados** (`sql.go:497-499`) substituem: `resolve_key = VALUES(resolve_key)` e `number_of_successes_in_a_row = VALUES(number_of_successes_in_a_row)`;
  - **uptime horário** (`sql.go:686-689`) acumula: `total_executions = total_executions + VALUES(total_executions)`, e o mesmo para as execuções com sucesso e o tempo de resposta;
  - **consolidação diária** (`sql.go:1128-1131`) substitui: `total_executions = VALUES(total_executions)` e demais colunas. Somar dobraria o uptime quando o conflito ocorresse.
  - `VALUES()` é obsoleta no MySQL, mas segue aceita no 8.4 e no 9.7. O MariaDB não documenta o alias de linha (`AS new`), por isso ele não é usado.
  - Cada upsert consome um valor de `AUTO_INCREMENT` (lacunas inofensivas) e pega next-key locks, cobertos pelo teste de concorrência.
- **Exclusão dos excedentes** (resultados, eventos e resultados de suite) com corte exato no identificador, mantendo os N mais recentes na mesma ordem do upstream (por id):

  ```sql
  -- texto do dialeto, com os mesmos argumentos do upstream: $1 = endpoint_id, $2 = N
  DELETE FROM endpoint_results
  WHERE endpoint_id = $1 AND endpoint_result_id <= (
    SELECT id FROM (
      SELECT endpoint_result_id AS id FROM endpoint_results
      WHERE endpoint_id = $1 ORDER BY endpoint_result_id DESC LIMIT 1 OFFSET $2
    ) AS corte
  )
  ```

  - O conector traduz o texto para `?` com os argumentos `(endpoint_id, endpoint_id, N)` e recusa `?` escrito direto, por isso o dialeto usa `$N` como o resto do store.
  - O `OFFSET` recebe o mesmo N do upstream (`maximumNumberOfResults`, `maximumNumberOfEvents` ou o limite de resultados de suite). A linha na posição N (base zero) é a primeira a sair, então ficam exatamente N.
  - A tabela derivada com `LIMIT` é materializada, o que evita o erro 1093.
  - Sem linha no `OFFSET`, a comparação com `NULL` não apaga nada.
  - A subconsulta de um `DELETE` faz leitura com lock, o que entra no teste de concorrência.
  - **Alternativa rejeitada:** `DELETE ... ORDER BY ... LIMIT` sobre os mais antigos, que exigiria contar antes (duas consultas e corrida com inserções).
- **Coluna `condition`:** as 3 consultas do upstream passam a citar `"condition"`. Funciona em SQLite e PostgreSQL e, com `ANSI_QUOTES`, no MySQL. Nenhuma outra coluna do esquema é reservada no MySQL 8.4 (`errors`, `status`, `timestamp` e `definition` não são).
  - **Alternativa rejeitada:** renomear a coluna só no MySQL, o que faria o texto das consultas divergir por banco.
- **Chave duplicada:** `isUniqueViolation` usa `errors.As(err, &*mysql.MySQLError)` com `Number == 1062`, mantendo as comparações de SQLite e pq. Como o conector envenena a transação (D2), o fluxo que devolve 409 continua fazendo rollback, como no PostgreSQL.

### D4. Esquema MySQL em `specific_mysql.go`
- Todas as tabelas com `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin` e `CREATE TABLE IF NOT EXISTS`, inclusive `managed_endpoints` e `managed_status_pages`. `managed_*.go` ganham o ramo `mysql` em vez de cair no DDL do PostgreSQL.
- **Tipos:**
  - `BIGINT AUTO_INCREMENT PRIMARY KEY` no lugar de `BIGSERIAL`;
  - `DATETIME(6)` para horários (microssegundos; `TIMESTAMP` tem o limite de 2038 e conversão de fuso);
  - `BOOLEAN` (`TINYINT(1)`, lido como `bool` pelo driver);
  - `BIGINT` para os instantes em milissegundos das tabelas do fork.
- **Textos:** toda coluna `TEXT` do PostgreSQL que não entra em índice vira `MEDIUMTEXT` (até 16 MB). O `TEXT` do MySQL guarda só 65.535 bytes, e `definition`, `errors` e outros textos livres truncariam ou falhariam onde o PostgreSQL aceita.
- **Chaves estrangeiras no nível da tabela:** `CONSTRAINT fk_<tabela>_<coluna> FOREIGN KEY (col) REFERENCES t(id) ON DELETE CASCADE`. `REFERENCES` na definição da coluna é aceito e ignorado pelo MySQL 8.4, e sem cascata `Clear` e a remoção de endpoints deixariam linhas órfãs.
- **Índices:** declarados dentro do `CREATE TABLE`, evitando `CREATE INDEX IF NOT EXISTS`, que o MySQL não tem. Colunas acrescentadas no futuro por migração aditiva conferem `information_schema.COLUMNS` antes do `ALTER TABLE`.
- **Hot reload:** o ciclo de recarga reabre o store e roda a criação do esquema de novo, então tudo precisa ser idempotente e sem `SET GLOBAL`.

### D5. Colunas indexadas, unicidade e limite de 768 caracteres
- O InnoDB com página de 16K e formato `DYNAMIC` (padrão) indexa até 3072 bytes; em `utf8mb4` (até 4 bytes por caractere) cabem 768 caracteres.
- **Chaves:** `endpoint_key` e `suite_key` como `VARCHAR(768)` únicas; `slug` de status page como `VARCHAR(64)`; `configuration_checksum` como `VARCHAR(64)` (hex do SHA-256).
- **Sem `UNIQUE(endpoint_name, endpoint_group)` no MySQL:** a chave única já é mais restritiva. Nomes e grupos iguais produzem a mesma chave (`key.ConvertGroupAndNameToKey` é determinística), então nenhum par duplicado passa pela unicidade da chave. Nome e grupo ficam `MEDIUMTEXT`.
- **`utf8mb4_bin` nas colunas.** A collation padrão do MySQL 8.4 (`utf8mb4_0900_ai_ci`) considera `API` igual a `api` e `é` igual a `e`.
  - `utf8mb4_bin` existe nos dois bancos; `utf8mb4_0900_bin` é só do MySQL.
  - Ela é PAD SPACE (`'a '` igual a `'a'`), sem efeito prático, porque a sanitização da chave troca espaços e o slug só aceita `[a-z0-9-]`.
  - A ordenação binária pode diferir de um PostgreSQL com collation linguística. A conformidade não compara ordem entre bancos com chaves de pontuação.
- **Limite de 768 caracteres na validação,** só com `storage.type: mysql`, contado em caracteres (`utf8.RuneCountInString`, não `len`):
  - endpoints e external-endpoints (`ValidateEndpointsConfig`);
  - suites e as chaves dos endpoints de suite (grupo da suite + nome);
  - a validação da administração.
  - Sem isso, o erro só apareceria na primeira gravação, dentro do watchdog.
- **Alternativa:** coluna gerada com `SHA2(chave)` indexada, sem limite. Rejeitada por agora: mais esquema, e chaves de 768 caracteres não existem na prática; fica registrada se surgir demanda.

### D6. Transações, concorrência e retry
- As inserções concorrentes do watchdog (padrão de 3 simultâneas) geram deadlock (1213) com o `REPEATABLE READ` padrão do InnoDB. Os gap e next-key locks da limpeza de excedentes, dos upserts de uptime e das inserções de endpoints diferentes se cruzam.
  - No marco 3, com a carga de vários pacotes de teste ao mesmo tempo, o MariaDB deu deadlock duas vezes seguidas na mesma gravação.
- **Isolamento `READ COMMITTED` em toda conexão,** o padrão do PostgreSQL (`SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED` ao conectar, sintaxe aceita nos dois bancos). Paridade com o PostgreSQL:
  - o InnoDB deixa de pegar gap locks nas leituras com lock e nos `UPDATE`/`DELETE`, e os deadlocks entre endpoints diferentes somem;
  - as leituras em lote das status pages ficam como no PostgreSQL, sem snapshot único entre as consultas da mesma transação;
  - a escrita otimista das tabelas gerenciadas continua segura com `ClientFoundRows` (D1).
- **Retry:**
  - Só no dialeto MySQL, `InsertEndpointResult` e `InsertSuiteResult` repetem **a chamada inteira**, até 3 tentativas, quando o erro final é deadlock (1213), lock wait timeout (1205) ou falha de certificação no `COMMIT` (1213 no Galera).
  - A espera entre tentativas cresce e tem uma parte aleatória (20 ms × tentativa + até 20 ms), para as transações que travaram entre si não tentarem de novo ao mesmo tempo.
  - Como a transação envenenada faz o `Commit` devolver o erro (D2), o retry sempre enxerga a falha.
  - Nenhum outro erro é repetido.
- **Teste de deadlock forçado:** duas transações travando linhas em ordem inversa, conferindo que nada foi gravado pela metade e que o retry grava tudo.
- **Alternativas:**
  - Manter `REPEATABLE READ` só com o retry único: rejeitada depois do teste de concorrência sob carga, que perdeu uma gravação.
  - Definir o isolamento por `BeginTx` em cada transação: exige mudar as chamadas do upstream e custa um comando a mais por transação.
  - Variável de sessão `transaction_isolation` na DSN: só existe no MariaDB 11.1+, e o comando `SET SESSION TRANSACTION ISOLATION LEVEL` funciona nos dois bancos.

### D7. Administração, status pages e configuração
- `config/config_admin.go`:
  - aceita `sqlite`, `postgres` e `mysql`, com a mensagem de erro de tipo atualizada (hoje em `config_admin.go:17`);
  - o aviso de várias instâncias com PostgreSQL passa a valer para `mysql`.
- As mensagens 501 de `statuspage/service.go` e `managedendpoint/service.go` não citam bancos e ficam como estão.
- As interfaces `ManagedEndpointStore`, `ManagedStatusPageStore`, `EndpointUptimeBatchReader` e `EndpointSummaryBatchReader` já são implementadas por `*sql.Store`: o MySQL as recebe sem código novo além do dialeto e do esquema.
- `endpoint_uptime_batch.go` e `endpoint_summary_batch.go`:
  - continuam com `$N`, traduzidos pelo conector;
  - `SUM(...)` devolve `DECIMAL`, lido pelo driver como texto e convertido para `int64`. O teste de igualdade com `GetUptimeByKey` cobre a conversão e o caso de conjunto vazio.

### D8. Testes
- **Helper comum de bancos de teste** (novo `sqltest_helpers_test.go` no pacote `sql`, que generaliza `managedEndpointTestStores`):
  - SQLite sempre;
  - PostgreSQL com `GATUS_TEST_POSTGRES_URL`;
  - MySQL com `GATUS_TEST_MYSQL_URL`;
  - MariaDB com `GATUS_TEST_MARIADB_URL`.
- **Isolamento:**
  - `go test ./...` roda pacotes em paralelo contra o mesmo servidor. Para MySQL e MariaDB, o helper cria um banco por teste (`CREATE DATABASE gatus_test_<pacote>_<aleatório>`) e o remove no `t.Cleanup`.
  - As URLs de teste usam um usuário com permissão para criar bancos (root no container de teste).
  - Os helpers de PostgreSQL continuam como hoje.
- **Suíte de conformidade** (`conformance_test.go`, arquivo novo; `sql_test.go` do upstream não é alterado), em todos os bancos disponíveis:
  - inserção e paginação, limites exatos, uptime horário e diário, médias;
  - alertas disparados, suites, remoção de endpoints com cascata, `Clear`;
  - reinício sobre esquema existente;
  - horário zero, concorrência de inserções com limite baixo e deadlock forçado.
- **Testes do conector:**
  - tradução (repetição, reordenação, literais, identificadores, comentários, argumento inexistente, listas `IN` dinâmicas);
  - caminho de `ErrSkip` com `InterpolateParams` ligado e desligado;
  - statements preparados;
  - transação envenenada;
  - reconexão após `KILL` da conexão no servidor.
- **Parâmetros fixos:** conferir por uma conexão do pool `@@session.time_zone`, `@@session.sql_mode` e `@@session.innodb_lock_wait_timeout`, e que uma DSN com `loc=America%2FSao_Paulo&parseTime=false` é sobrescrita. Nada de `SET GLOBAL`.
- **Validação:** DSN válida e inválida sem senha no erro, limite de 768 caracteres só com `mysql` e contado em caracteres, `admin.enabled` com `mysql`.
- **CI a partir do marco 1:**
  - job principal com serviços `mysql:8.4.11` (porta 3306, healthcheck `mysqladmin ping -h 127.0.0.1`) e `mariadb:10.11.19` (porta 3307, healthcheck `healthcheck.sh --connect --innodb_initialized`), além do PostgreSQL atual;
  - job `storage-latest` com `mysql:9.7.2` e `mariadb:12.3.3`, rodando os pacotes de storage, `statuspage`, `managedendpoint` e `config`;
  - tags de patch fixas, nunca `latest`.
- **Local:** containers `gatus-test-mysql` (8.4.11, porta 53306) e `gatus-test-mariadb` (10.11.19, porta 53307), documentados no `AGENTS.fork.md`.
- **E2E:** `test/e2e/admin.sh` e `test/e2e/status-pages.sh` aceitam `E2E_STORAGE_TYPE` e `E2E_STORAGE_PATH`, para rodar com MariaDB 11.4 antes da release.
- **Benchmark:** `InsertEndpointResult` com `InterpolateParams` ligado e desligado (D1).

### D9. Documentação, exemplo e deploy
- **`docs/storage-mysql.md`:**
  - DSN e parâmetros sobrepostos;
  - criação do banco e do usuário (`CREATE DATABASE gatus CHARACTER SET utf8mb4 COLLATE utf8mb4_bin`, com `GRANT` nesse banco);
  - versões suportadas, página InnoDB de 16K, limite de chave de 768 caracteres e limite de 25 conexões;
  - backup com `mysqldump`/`mariadb-dump`;
  - várias instâncias, ausência de migração a partir de SQLite e PostgreSQL, e volta ao Gatus original (que não lê MySQL).
- **README do fork:** seção de storage com o tipo novo.
- **Exemplo `.examples/docker-compose-mariadb-storage/`:** `jniltinho/gatus:<tag>` e `mariadb:11.4.13` fixos, com `healthcheck.sh --connect --innodb_initialized`.
- **Pacote `/home/jnsilva/Projetos/gatus/mariadb`:**
  - `docker-compose.yml`, `docker-compose.build.yml`, `.env` com `GATUS_VERSION` e `MARIADB_VERSION` fixos, `config/config.yaml`, `gatus.sh` e `Dockerfile`, no padrão dos pacotes existentes;
  - imagem `jniltinho/gatus`, porque a imagem do upstream não tem MySQL;
  - bind em `127.0.0.1`, dados em `./mariadb-data` e credenciais no `.env`.

### D10. Relação com o upstream
- **Arquivos do upstream alterados:**
  - `storage/type.go`, `storage/config.go` e `storage/store/store.go` (tipo, validação, inicialização);
  - `sql.go`: `dialectQuery` nos 3 upserts, desvio no início das 3 exclusões, `InsertEndpointResult` e `InsertSuiteResult` renomeadas para as versões sem retry (o retry fica em `dialect_mysql.go`) e `"condition"` citado;
  - `specific_*.go`: só a seleção do esquema.
- **Arquivos novos:** `mysql.go` (configuração, pool e checagem do servidor), `mysql_connector.go` (conn, tx, stmt e rows do wrapper), `placeholders.go`, `dialect_mysql.go` e `specific_mysql.go`.
- O `AGENTS.fork.md` ganha o passo de sincronização: rodar a suíte de conformidade com MySQL e MariaDB e revisar consultas novas do upstream que usem `RETURNING`, `ON CONFLICT`, `LIMIT` em subconsulta, `REFERENCES` na coluna ou palavras reservadas.

## Risks / Trade-offs

- **[Consultas novas do upstream quebram no MySQL sem aviso]** → A suíte de conformidade roda no CI com MySQL e MariaDB (mínimas e LTS novas), e há um passo explícito no roteiro de sincronização.
- **[Tradutor ou wrapper do driver erra num caso de borda]** → Tokenizador pequeno com testes dedicados, tratamento explícito do `ErrSkip`, asserções de interfaces e teste de conexão derrubada. As consultas são do próprio código, não entrada de usuário.
- **[Gravação parcial após deadlock ou lock wait timeout]** → Transação envenenada no conector (D2) e retry único da chamada inteira (D6), com teste de deadlock forçado.
- **[`VALUES()` removida numa versão futura do MySQL]** → Ponto único no dialeto, com troca pelo alias de linha só para MySQL, detectado pela versão do servidor; o job `storage-latest` avisa cedo.
- **[Cascata desligada por DDL na sintaxe errada]** → Chaves estrangeiras no nível da tabela e teste que confere `information_schema.REFERENTIAL_CONSTRAINTS` e a remoção em cascata.
- **[Collation `utf8mb4_bin` torna ordenação e buscas sensíveis a maiúsculas]** → É o comportamento do PostgreSQL. O Gatus não faz busca textual no banco.
- **[Servidor abaixo das versões mínimas ou com página InnoDB menor que 16K]** → Aviso no log para a versão; erro claro na inicialização para a página.
- **[Pool de 25 conexões insuficiente com muitos endpoints simultâneos]** → Espera no pool, não erro; valor documentado e fácil de ajustar se surgir demanda.
- **[Dependência nova (MPL-2.0)]** → Driver padrão de fato do ecossistema, Go puro. MPL-2.0 é copyleft por arquivo e não afeta o código do Gatus.
- **[Bancos compartilhados com `sql_mode` ou fuso diferentes]** → Os parâmetros são fixados por sessão (D1) e não mudam a configuração global do servidor.

## Migration Plan

- **Instalação nova com MySQL/MariaDB:** criar banco e usuário conforme a documentação, configurar `storage.type: mysql` e `storage.path` e iniciar; o esquema é criado sozinho.
- **Instalação existente (SQLite ou PostgreSQL):** nada muda. Trocar para MySQL começa com banco vazio.
  - Endpoints e status pages do YAML voltam na carga.
  - Os gerenciados pela administração precisam ser recriados (ou exportados em YAML antes, pela própria administração).
- **Rollback:**
  - voltar `storage.type` ao valor anterior restaura o storage antigo, intacto;
  - voltar a uma versão do fork sem MySQL com `storage.type: mysql` falha na validação, sem tocar nos dados;
  - o Gatus original não lê o banco MySQL.
- **Entrega em 4 marcos,** cada um num PR no `jniltinho/gatus` com CI verde (os serviços de banco entram no CI já no marco 1):
  1. conector, DSN, esquema e CI;
  2. paridade do store do upstream e conformidade;
  3. tabelas do fork, administração e status pages;
  4. documentação, exemplo, E2E, pacote de deploy e release.

## Open Questions

- **Galera e réplicas.** Clusters MariaDB Galera devolvem 1213 no `COMMIT` em conflito de certificação, e o retry de D6 cobre esse caso nas inserções de resultado. Falta validar em cluster real se as escritas da administração (sem retry, respondem erro ao usuário) precisam de tratamento. Hoje ficam sem garantia e sem teste.
- **`READ COMMITTED`.** Resolvido no marco 3. O teste de concorrência (8 endpoints × 25 resultados com limite de 5) tinha passado no marco 2 com o `REPEATABLE READ` padrão, mas perdeu uma gravação no MariaDB com a carga de vários pacotes ao mesmo tempo. As conexões passam a usar `READ COMMITTED`, e o retry vai a até 3 tentativas (D6).

## Ajustes da revisão de QA

Revisão feita por um agente QA sênior Go sobre a primeira versão desta change, com checagem dos fatos de fabricantes em documentação oficial.

| # | Severidade | Achado | Como ficou |
|---|------------|--------|------------|
| 1 | Alta | Deadlock desfaz a transação inteira no InnoDB e o `Commit` "passava", sem o retry perceber | Transação envenenada no conector (D2), retry da chamada inteira incluindo erro no `COMMIT` (D6), `innodb_lock_wait_timeout` curto (D1) e teste de deadlock forçado |
| 2 | Alta | `REFERENCES` na coluna é ignorado no MySQL 8.4, sem cascata | Chaves estrangeiras no nível da tabela (D4), cenário de cascata no spec e teste em `information_schema` |
| 3 | Alta | `ErrSkip` levava o `database/sql` a preparar o texto original; `ResetSession` não repassado | Tratamento explícito do `ErrSkip`, stmt embrulhado com `NumInput() = -1`, interfaces repassadas com asserções e teste de `KILL` (D2) |
| 4 | Média | `OFFSET` ambíguo na exclusão de excedentes | Corte com `<=` e o mesmo N do upstream, exato (D3) |
| 5 | Média | Justificativa do `time_zone` errada e cenário que exigia `SET GLOBAL` | `Loc` como fonte da conversão, `time_zone` mantido por segurança, cenário de sobrescrita de `loc` e `parseTime` (D1, D8) |
| 6 | Média | `sql_mode` removia `ONLY_FULL_GROUP_BY` e rejeitava o horário zero | `sql_mode` com `ONLY_FULL_GROUP_BY` e sem `NO_ZERO_*`, aspas documentadas, isolamento por `TxOptions` (D1, D6) |
| 7 | Média | LTS mais novas não consideradas | CI principal com as mínimas e job `storage-latest` com MySQL 9.7.2 e MariaDB 12.3.3, tags de patch fixas (D0, D8) |
| 8 | Média | Testes paralelos no mesmo banco e serviços de CI só no marco 4 | Banco por teste com remoção no `Cleanup`, portas e healthchecks definidos, serviços no CI desde o marco 1 (D8) |
| 9 | Média | `len` em vez de caracteres, chaves de endpoints de suite, página InnoDB e `TEXT` de 64 KB | `utf8.RuneCountInString`, validação das chaves de suite, checagem de `@@innodb_page_size`, `MEDIUMTEXT` (D1, D4, D5) |
| 10 | Baixa | `utf8mb4_bin` é PAD SPACE e ordena diferente | Registrado em D5; conformidade não compara ordem com chaves de pontuação |
| 11 | Baixa | Motivo para `InterpolateParams=false` não procedia | `InterpolateParams=true` com benchmark no marco 2 (D1) |
| 12 | Baixa | Tradução do upsert do uptime e efeitos colaterais | Expressões acumulativas explícitas, lacunas de `AUTO_INCREMENT` e locks no teste de concorrência (D3) |
| 13 | Baixa | Pool sem limite de conexões | `SetMaxOpenConns(25)` documentado (D1) |
| 14 | Baixa | Referências de arquivo erradas (501, nomes de arquivos, DDL gerenciado) | Corrigidas em D2, D4, D7 e D10 e na proposta |
| 15 | Baixa | Cenários faltando e requisito de testes impreciso | Cenários de cascata, transação envenenada, sobrescrita de parâmetros, reconexão e Galera no spec e nas Open Questions |

## Ajustes da validação com o grok

Validação independente dos ajustes acima com a CLI do grok, só leitura, conferindo o código do store e do `go-sql-driver/mysql` v1.10.1. Veredito: **aprovado com ajustes**.

- **Confirmado sem mudança:**
  - nenhum fluxo do upstream ou do fork depende de continuar a transação depois de um erro do driver, então a transação envenenada com retry da chamada inteira é correta e suficiente;
  - as interfaces repassadas cobrem o conn do driver;
  - o corte `<=` com `LIMIT 1 OFFSET N` mantém exatamente N nas três exclusões;
  - `ONLY_FULL_GROUP_BY` não quebra nenhuma consulta do store;
  - nenhuma consulta usa aspas duplas como literal.

| # | Severidade | Achado | Como ficou |
|---|------------|--------|------------|
| 1 | Média | A tarefa dizia que os 3 upserts são acumulativos; só o uptime horário é | D3 e a tarefa 2.2 listam as três formas (substituição nos alertas e na consolidação diária, soma no horário) |
| 2 | Média | O SQL de exclusão do D3 usava `?`, que o conector recusa | Exemplo com `$1`/`$2` e o texto traduzido descrito à parte |
| 3 | Média | "Executa e fecha o statement" invalidaria o cursor no `Query` com `ErrSkip` | D2 separa `Exec` (fecha logo) de `Query` (fecha no `Rows.Close`), como implementado em `rowsClosingStmt` |
| 4 | Baixa | Faltava cenário do horário zero | Cenário de ida e volta de `time.Time{}` no spec, coberto por `TestMySQLConnector_SessionAndTimes` |
| 5 | Baixa | A proposta não citava o retry do resultado de suite nem o 1213 no `COMMIT` | Proposta alinhada com D6 |
| 6 | Baixa | O `Commit` precisa encadear o erro original para o retry reconhecer 1213 e 1205 | D2 explicita o encadeamento com `%w`, como implementado |
