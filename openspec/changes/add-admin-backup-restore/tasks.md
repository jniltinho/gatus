## 1. Formato e cifragem

- [ ] 1.1 `adminbackup/format.go`:
  - tipos da versão 1;
  - decodificação estrita (chaves desconhecidas, listas, duplicados, política de versão);
  - limites de 2 MiB e de itens;
  - `gatusVersion` por `debug.ReadBuildInfo`;
  - testes e fuzz de `Decode`.
- [ ] 1.2 `adminbackup/crypto.go`:
  - envelope com Argon2id fixo (t=2, 19456 KiB, p=1) e AES-256-GCM;
  - cabeçalho canônico como AAD, com vetor de teste fixo;
  - decodificação estrita (salt 16, nonce 12, base64 padrão, nomes fixos);
  - erro único;
  - semáforo de 2 derivações com 429;
  - testes com envelope alterado, parâmetros forjados e fuzz de `Decrypt`.
- [ ] 1.3 `security`: construtor exportado de limitador de falhas independente com a janela como parâmetro (restore com 15 minutos, login com 1 minuto), e testes de que um não afeta o outro.

## 2. Backup e restore no backend

- [ ] 2.1 `Build`:
  - JSON montado sob `TryBeginChange`, e cifragem depois de soltar o lock, com o semáforo compartilhado;
  - registros disponíveis (503);
  - definições completas e 422 acima dos limites;
  - testes com Push, desabilitado, em conflito, página e chave.
- [ ] 2.2 Serviços:
  - `managedendpoint`:
    - `ValidateRestore` chamando `Prepare` com `RestoreContext` (sem `prepare(raw, key)` nem geração de token);
    - `HasMaskedSecret` com todos os lugares de `MaskSecrets` (URL e folhas de `provider-override` inclusive);
    - `RestoreCreate`, `IsManagedUnavailable`, `StoredPushTokens` e `WithPushTokensLocked`;
  - `statuspage`: `ValidateRestore` com conflito de slug explícito (inclusive em atualização) e `SelectionWarningsWith`;
  - `pushkey`: `Restore`, `IsNameInUse` e `IsHashInUse`, com locks `pushkey.mutex` → `statesMutex`.
- [ ] 2.3 `Plan`:
  - tokens em uso de todas as origens (inclusive gerenciados em conflito ou inválidos e endpoints do backup) antes das chaves;
  - simulação acumulada na ordem chaves → endpoints → páginas;
  - definição normalizada guardada por item e usada na aplicação;
  - todas as ações e motivos (YAML, validação, chave divergente, tipo, segredo mascarado, Push sem token, nome/grupo, chaves de push e hashes de tokens);
  - `disableEndpoints`;
  - normalização de endpoints e páginas;
  - avisos;
  - fingerprint por struct com ordem fixa de tipos, hash dos bytes do texto claro e vetor de teste;
  - testes de determinismo e de cada cenário da spec.
- [ ] 2.4 `Apply`:
  - fingerprint (409) e registros (503);
  - serviços com versão;
  - `ErrCycleInProgress` pula os itens restantes;
  - avisos recalculados;
  - auditoria sem segredos (teste capturando o log).
- [ ] 2.5 Rotas `POST /api/v1/admin/backup`, `/restore/preview` e `/restore`:
  - `application/json` obrigatório (415);
  - rotas de restore num grupo com CSRF e autorização, sem o limite de 256 KB, com 3,5 MiB;
  - limpeza do cache `endpoint-status-*` no handler depois de `Apply`;
  - middleware de IP do cliente e limitador próprio;
  - cabeçalhos do download;
  - `ExposeHeaders` no CORS de desenvolvimento;
  - rota SPA `/admin/backup`.
- [ ] 2.6 Testes de API e integração:
  - 401, 403 CSRF, 415, 400 (formato, senha, parâmetros), 409, 413 real do Fiber, 422, 429, 503 (recarga antes e durante, e endpoints gerenciados indisponíveis);
  - restore de um backup cifrado de 2 MiB passando pelas rotas (sem 413 do middleware);
  - segredos mascarados em header, URL e `provider-override`;
  - chave com o token de um endpoint gerenciado em conflito;
  - página gerenciada cujo slug passou a existir no YAML;
  - `disableEndpoints` aplicado como na prévia;
  - ciclo backup → restore numa instalação nova (SQLite e, com as variáveis de teste, PostgreSQL e MySQL/MariaDB);
  - restore repetido `unchanged`;
  - push com a chave restaurada;
  - `-race`.

## 3. Frontend

- [ ] 3.1 Utilitários, com testes em `node --test`:
  - `utils/adminBackup.js`: formato, nome, bytes da senha, resumo, filtros, contagens, limite de `2.7 * 1024 * 1024` bytes e mensagens de 413, 422 e 429;
  - `adminApi.download`.
- [ ] 3.2 `views/admin/AdminBackup.vue`, rota `/admin/backup` em `router/index.js` (`meta.admin`) e `AdminTabs` com a aba Backup:
  - download com senha opcional e aviso;
  - restore com arquivo, senha, opções, prévia (avisos, resumo, tabela, filtro), confirmação e resultados;
  - invalidação da prévia e bloqueios;
  - mensagens de 413, 422 e 429;
  - temas claro e escuro.
- [ ] 3.3 Lint, `npm run test:unit` e `make frontend-build`.

## 4. Documentação, E2E e entrega

- [ ] 4.1 Documentação:
  - `docs/admin-endpoints.md` (seção Backup and restore): conteúdo, limites, senha e segredos, prévia, mesclagem, monitoramento e alertas no destino, "restore as disabled", recarga no meio, timeout do nginx e `client_max_body_size 4m`, várias instâncias e API com `curl`;
  - `AGENTS.fork.md`.
- [ ] 4.2 E2E `test/e2e/admin-backup.sh`:
  - cria endpoint, página e chave;
  - baixa o backup com e sem senha;
  - restaura numa instância nova com um YAML em conflito (prévia com `skip`, confirmação, resultados e push com a chave restaurada);
  - repete o restore com `unchanged`;
  - testa senha errada e prévia invalidada;
  - prints em claro e escuro.
- [ ] 4.3 `go test ./... -race` com PostgreSQL, MySQL e MariaDB, `make lint` e `openspec validate add-admin-backup-restore --strict`.
- [ ] 4.4 Entrega:
  - PR no `jniltinho/gatus` com CI verde e merge;
  - release `v5.36.0-fork.18` com imagem no Docker Hub;
  - pacote `mariadb`;
  - arquivamento da change.
