## Why

O fork só grava em SQLite ou PostgreSQL. Onde já existe um servidor MariaDB ou MySQL (hospedagens, infraestrutura legada, clusters Galera), é preciso manter um PostgreSQL só para o Gatus ou aceitar o SQLite, que não serve para várias instâncias. O pedido existe no upstream desde 2022 ([TwiN/gatus#283](https://github.com/TwiN/gatus/issues/283)), e uma implementação ([TwiN/gatus#1003](https://github.com/TwiN/gatus/pull/1003)) foi fechada sem merge em 2025: o mantenedor decidiu não suportar mais tipos de storage. Como fork, com a administração pela web e as status pages já dependendo de storage SQL, podemos oferecer esse terceiro banco.

## What Changes

- Novo `storage.type: mysql`, que serve para **MySQL 8.4+** e **MariaDB 10.11+** (mesmo driver e mesmo protocolo), com a DSN do `go-sql-driver/mysql` em `storage.path`, por exemplo `gatus:senha@tcp(mariadb:3306)/gatus`.
  - As mínimas são as LTS mais antigas ainda com suporte dos fabricantes; um servidor abaixo delas gera aviso no log.
  - A DSN é validada na carga da configuração, e a senha nunca aparece no log.
  - O Gatus fixa na conexão o que o código precisa, independentemente da configuração do servidor: horários em UTC, `utf8mb4`, contagem de linhas encontradas, um `sql_mode` conhecido, espera de lock curta e um pool de até 25 conexões.
- Mesmo comportamento do PostgreSQL:
  - resultados, eventos, uptime, alertas disparados, suites, `storage.caching` e os limites de resultados e eventos;
  - endpoints gerenciados pela administração e status pages gerenciadas (versão, `If-Match`, conflito de chave e slug);
  - leituras em lote das status pages.
- Esquema criado automaticamente e idempotente a cada início: InnoDB, `utf8mb4` e colunas indexadas com tamanho e collation binária, para a unicidade se comportar como no PostgreSQL.
- Camada de dialeto no store SQL, com as consultas do upstream quase intactas:
  - conector próprio que traduz os placeholders `$N` para `?` e invalida a transação no primeiro erro, como o PostgreSQL faz;
  - SQL próprio só para o que não é portável: `RETURNING`, `ON CONFLICT`, exclusões com `LIMIT` em subconsulta, chaves estrangeiras, criação de índices e detecção de chave duplicada;
  - retry único da gravação de resultados de endpoint e de suite em deadlock, espera de lock esgotada ou conflito no `COMMIT` (Galera).
- Com `mysql`, chaves de endpoint e de suite acima do limite indexável (768 caracteres) são rejeitadas na validação, com mensagem clara, em vez de falhar na gravação.
- `admin.enabled` passa a aceitar `mysql`, e o aviso de várias instâncias do PostgreSQL vale também para MySQL.
- Testes do store SQL também em MySQL e MariaDB quando `GATUS_TEST_MYSQL_URL` e `GATUS_TEST_MARIADB_URL` estiverem definidos, cada teste num banco próprio.
  - O CI sobe os dois bancos desde o primeiro marco: as versões mínimas no job principal e as LTS mais novas (MySQL 9.7 e MariaDB 12.3) num job de storage.
- Documentação (`docs/storage-mysql.md` e seção de storage do `README.md`), exemplo `.examples/docker-compose-mariadb-storage/` e pacote de deploy `mariadb`.
  - O pacote fica fora do repositório, no padrão dos pacotes `sqlite` e `postgres`, com a imagem `jniltinho/gatus` e tags fixas.
- **Fora do escopo:**
  - migração de dados de SQLite ou PostgreSQL para MySQL (o banco novo começa vazio);
  - garantia para MySQL anterior à 8.4 e MariaDB anterior à 10.11.
- Entrega em 4 marcos, cada um num pull request no `jniltinho/gatus`, com release ao final.

Nenhuma mudança é **BREAKING**: SQLite, PostgreSQL e memória continuam iguais.

## Capabilities

### New Capabilities
- `mysql-storage`:
  - configuração `storage.type: mysql` e validação da DSN;
  - parâmetros fixos de conexão e esquema automático;
  - paridade com o PostgreSQL no monitoramento, na administração e nas status pages;
  - limites de chave, horários em UTC e tradução de placeholders;
  - testes em MySQL e MariaDB.

### Modified Capabilities
Nenhuma. `openspec/specs/` está vazio. As regras que hoje citam só SQLite e PostgreSQL estão nas changes `add-admin-endpoint-management` e `add-public-status-pages`, ainda não arquivadas, e a ampliação para `mysql` fica descrita na capability nova.

## Impact

- **Go (arquivos do upstream, mudanças pontuais):**
  - `storage/type.go`: `TypeMySQL`;
  - `storage/config.go`: validação da DSN;
  - `storage/store/store.go`: inicialização e log sem senha;
  - `storage/store/sql/sql.go`: pontos de dialeto nas consultas não portáveis e escolha do esquema;
  - `storage/store/sql/specific_*.go`: só a escolha do esquema.
- **Go (arquivos novos ou do fork):**
  - `storage/store/sql/dialect.go`, `dialect_mysql.go`, `mysql_connector.go` (conector com tradução e transação envenenada), `placeholders.go` e `specific_mysql.go`;
  - ramo `mysql` em `managed_endpoints.go` e `managed_status_pages.go`;
  - `isUniqueViolation` com o erro 1062;
  - validação do limite de chave em `config/` e na administração;
  - `config/config_admin.go` (tipo aceito e aviso de várias instâncias).
- **Dependência nova:** `github.com/go-sql-driver/mysql` v1.10.1 (traz `filippo.io/edwards25519`), em Go puro e sem CGO, sob licença MPL-2.0, compatível com o uso como dependência de um projeto Apache-2.0.
- **Testes e CI:**
  - helpers de teste do store SQL parametrizados por banco, com banco próprio por teste;
  - testes novos de conformidade, do conector e de concorrência;
  - `.github/workflows/ci.yml` com `mysql:8.4.11` e `mariadb:10.11.19` no job principal e job `storage-latest` com `mysql:9.7.2` e `mariadb:12.3.3`.
- **Documentação e deploy:**
  - `docs/storage-mysql.md`, `README.md`, `docs/admin-endpoints.md` e `docs/status-pages.md`;
  - `.examples/docker-compose-mariadb-storage/`;
  - `AGENTS.fork.md` e `openspec/config.yaml` (contexto);
  - pacote `mariadb` em `/home/jnsilva/Projetos/gatus/mariadb`.
- **Banco:** esquema próprio em MySQL/MariaDB. Nenhuma mudança nos esquemas SQLite e PostgreSQL.
- **Upstream:** as consultas novas do upstream precisam ser conferidas nos três bancos a cada sincronização; o passo entra no `AGENTS.fork.md`.
