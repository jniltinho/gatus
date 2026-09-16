## Why

Tudo o que é cadastrado pela administração web fica só no banco:
- endpoints gerenciados, ativos e Push, com os seus grupos;
- status pages;
- chaves globais de push.

Hoje não há como:
- guardar uma cópia desses cadastros;
- levá-los para outra instalação (por exemplo, de SQLite para MariaDB, ou de um servidor de teste para produção);
- recuperá-los depois de um erro, sem copiar o banco inteiro.

O dono do fork pediu backup e restore desses cadastros na própria administração.

## What Changes

- **Backup na administração:**
  - nova aba **Backup** (`/admin/backup`), que baixa um arquivo com os endpoints gerenciados (os grupos vêm dentro das definições), as status pages gerenciadas e as chaves globais de push criadas pela web;
  - as chaves vão só com o hash e a dica, e os scripts que usam a chave continuam funcionando depois do restore;
  - o arquivo não leva histórico, cadastros do arquivo de configuração nem valores padrão.
- **Senha opcional escolhida na hora:**
  - com senha, o arquivo é cifrado (AES-256-GCM com chave derivada por Argon2id, com limite de derivações simultâneas e de tentativas por IP, separado do login);
  - sem senha, o arquivo é JSON legível, e a tela avisa que ele contém segredos (tokens de push, senhas e headers dos endpoints, webhooks de alertas).
- **Restore com prévia e mesclagem:**
  - **Prévia:** o administrador envia o arquivo (e a senha, se houver) e vê, por item, o que será criado, atualizado, mantido igual ou ignorado, com o motivo (conflito com o arquivo de configuração, definição inválida, token repetido, segredo mascarado, chave de push em uso). A prévia simula o estado acumulado dos itens, avisa quantos endpoints começarão a ser monitorados e com alertas, e não altera nada.
  - **Aplicação:** o administrador escolhe se sobrescreve os cadastros que já existem, e opcionalmente restaura os endpoints desabilitados, e aplica. Nada é apagado, e um fingerprint garante que nada mudou desde a prévia.
  - **Validação:** cada item passa pelas mesmas validações e efeitos da criação e da alteração pela web (monitoramento, publicação, auditoria).
  - **Resultado:** o resumo mostra o que foi feito em cada item.
- **API:**
  - `POST /api/v1/admin/backup`;
  - `POST /api/v1/admin/restore/preview`;
  - `POST /api/v1/admin/restore`;
  - todas com as proteções atuais da administração, só em JSON, e o arquivo limitado a 2 MiB (1.000 endpoints, 200 páginas e 500 chaves), dentro do limite de 4 MiB do servidor.
- **Documentação e E2E.**

## Capabilities

### New Capabilities

- `admin-backup-restore`: formato e cifragem do arquivo, backup, prévia e aplicação do restore dos endpoints, status pages e chaves de push gerenciados, aba Backup da administração.

### Modified Capabilities

- `admin-endpoint-management`: as rotas de restore aceitam corpos acima de 256 KB.
- `push-monitoring`: chaves globais restauradas mantêm hash e dica do arquivo, sem colidir com chaves nem tokens de endpoints.
- `admin-access-control`: as rotas de backup e restore exigem `application/json`.

## Impact

- **Backend:**
  - pacote novo do fork `adminbackup`: formato, cifragem, montagem do backup, plano e aplicação do restore;
  - `managedendpoint`, `statuspage` e `pushkey` ganham funções para montar o backup e aplicar itens restaurados com as regras e os locks que já existem (`lifecycle.TryBeginChange`, publicação depois do commit);
  - `pushkey` ganha a criação a partir de hash e dica;
  - `security` ganha um limitador de falhas independente;
  - rotas em `api/`, com a rota SPA `/admin/backup`.
- **Dependência:** `golang.org/x/crypto/argon2`, do módulo `golang.org/x/crypto`, que já é dependência.
- **Frontend:** `views/admin/AdminBackup.vue`, rota e aba nova, `utils/adminBackup.js` com testes.
- **Segurança:** o arquivo sem senha tem segredos em texto claro. A tela e a documentação avisam, e a senha é recomendada. Senha, tokens e definições nunca vão para o log.
- **Compatibilidade:** o arquivo tem formato e versão próprios; versões futuras continuam lendo a versão 1. O upstream não tem esse recurso.
