## Why

Hoje o nome e o grupo de um endpoint gerenciado pela administração não podem mudar depois da criação: a API responde 400 e a tela bloqueia os campos. Para reorganizar grupos (por exemplo, mover um site de `web` para `clientes`) ou corrigir um nome, é preciso remover o endpoint, perdendo uptime, resultados, eventos e alertas disparados, e criá-lo de novo. Com o seletor de grupo na criação já disponível, o próximo passo natural é permitir a troca também na edição, mantendo o histórico.

## What Changes

- A alteração de um endpoint gerenciado passa a aceitar `name` e `group` diferentes dos armazenados (`PUT /api/v1/admin/endpoints/{chave-atual}`).
  - Quando a chave (`grupo_nome`) muda, o sistema renomeia a chave numa única transação: a definição em `managed_endpoints` e a linha do endpoint em `endpoints`, da qual dependem resultados, eventos, uptime e alertas disparados. O histórico continua sob a chave nova.
  - Quando só o texto muda e a chave continua a mesma (por exemplo, `My API` para `my-api`), a definição e o nome e grupo exibidos são atualizados, sem troca de chave.
  - Vale para SQLite, PostgreSQL e MySQL/MariaDB.
- A chave nova MUST ser livre:
  - conflito com endpoint, endpoint externo, suite ou endpoint de suite do YAML, ou com outro endpoint gerenciado, continua respondendo 409;
  - chave nova que já tenha histórico armazenado (por exemplo, dados antigos ainda não limpos) também responde 409, sem mesclar nem apagar dados;
  - com `mysql`, o limite de 768 caracteres da chave continua valendo.
- O monitoramento da chave antiga para antes da troca e o da chave nova começa depois, sem resultados, eventos, métricas ou alertas registrados sob a chave antiga depois da resposta. O estado dos alertas disparados cuja configuração não mudou é preservado.
- As séries Prometheus e os caches de status da chave antiga são apagados.
- Status pages que selecionam o endpoint pela chave:
  - as status pages gerenciadas pela administração têm a chave trocada em `endpoints` e `featured` na mesma transação, com nova versão;
  - as status pages do arquivo de configuração não podem ser alteradas: a resposta e a tela avisam quais páginas deixam de mostrar o endpoint até o YAML ser corrigido. Páginas que selecionam pelo grupo antigo ou novo seguem a regra de grupo, como hoje.
- Um endpoint gerenciado em conflito com o YAML pode ser renomeado: a definição vai para a chave nova, que começa sem histórico, e o histórico da chave antiga continua com o endpoint do YAML.
- Na tela de edição, nome e grupo ficam editáveis (grupo com o mesmo seletor da criação). Antes de salvar, a tela avisa que a chave muda, que URLs de badges e da página de detalhes mudam e quais status pages do arquivo são afetadas.
- A auditoria registra a chave antiga e a nova.
- **BREAKING** (API da administração): `PUT` com nome ou grupo diferente deixa de responder 400 e passa a renomear. Integrações que dependiam do 400 para evitar renomeação precisam conferir a chave antes de enviar.
- **Fora do escopo:**
  - renomear endpoints do arquivo de configuração, endpoints externos ou suites;
  - redirecionar URLs antigas de badges, da página de detalhes ou séries Prometheus para a chave nova;
  - propagar a troca para outras instâncias que compartilham o mesmo PostgreSQL ou MySQL antes de reiniciarem ou recarregarem.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `admin-endpoint-management`: a regra "Chave imutável" é substituída pela renomeação de endpoint gerenciado (chave livre, histórico mantido, parada consistente, estado de alertas, status pages gerenciadas atualizadas e aviso das páginas do arquivo).
- `admin-web-ui`: na edição, nome e grupo deixam de ser somente leitura; o grupo usa o seletor de grupos existentes ou novo (também na criação) e a tela avisa a troca de chave antes de salvar.
- `admin-access-control`: a auditoria de uma renomeação registra a chave antiga e a nova.

## Impact

- **Go:**
  - `managedendpoint/service.go` (`update`: fim de `ErrKeyChanged`, parada da chave antiga, restauração do estado de alertas, rollback, métricas e auditoria) e `managedendpoint/registry.go` (troca de chave no estado em memória);
  - `storage/store/managed_endpoint.go` e `storage/store/sql/managed_endpoints.go`: atualização com chave antiga e nova, renomeando `endpoints` e as referências das status pages gerenciadas na mesma transação, com detecção de chave ocupada;
  - `statuspage/`: recarga das status pages gerenciadas alteradas e cálculo das páginas do arquivo afetadas;
  - `api/admin.go`: códigos de resposta, invalidação do cache de status e avisos na resposta.
- **Frontend:** `web/app/src/views/admin/AdminEndpointForm.vue` (campos editáveis, aviso de troca de chave, navegação para a chave nova), `web/app/src/utils/adminApi.js` e `web/static/`.
- **Banco:** nenhuma mudança de esquema; só `UPDATE` nas tabelas existentes nos três bancos.
- **Testes:** store SQL (SQLite, PostgreSQL, MySQL e MariaDB), `api/admin_test.go`, `main_managed_test.go` (histórico preservado depois de reinício e limpeza), watchdog e `test/e2e/admin.sh`.
- **Documentação:** `docs/admin-endpoints.md` (seções de uso, comportamento e API).
- **Externos:** URLs de badges, da página de detalhes e séries Prometheus da chave antiga deixam de existir depois da renomeação; dashboards e regras externas precisam da chave nova.
