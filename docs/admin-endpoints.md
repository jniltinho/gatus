# Administração de endpoints pela web

> Funcionalidade exclusiva do fork [jniltinho/gatus](https://github.com/jniltinho/gatus). O Gatus original só aceita
> endpoints no arquivo de configuração ([TwiN/gatus#1345](https://github.com/TwiN/gatus/issues/1345)).

Com a administração habilitada, endpoints podem ser cadastrados, editados, desabilitados e removidos em `/admin`,
sem acesso ao servidor e sem reiniciar o Gatus. Os endpoints do arquivo de configuração continuam funcionando como
sempre e aparecem na administração apenas para consulta.

## Requisitos

- `security.basic` ou `security.oidc` configurado. Com OIDC, `admin.allowed-subjects` é obrigatório.
- `storage.type` igual a `sqlite` ou `postgres`: os endpoints cadastrados pela web ficam na tabela `managed_endpoints`.

## Configuração

```yaml
storage:
  type: sqlite
  path: /data/data.db

security:
  basic:
    username: admin
    # Senha em bcrypt codificada em base64 (veja abaixo)
    password-bcrypt-base64: "JDJhJDEwJHRiMnRFakxWazZLdXBzRERQazB1TE8vckRLY05Yb1hSdnoxWU0yQ1FaYXZRSW1McmladDYu"

admin:
  enabled: true
  # Obrigatório com security.oidc: subjects (claim sub) que podem administrar, sem diferenciar maiúsculas
  # allowed-subjects: ["ops@exemplo.com"]
  # Origens aceitas nas alterações; necessário atrás de proxy que publica outra porta
  # allowed-origins: ["https://status.exemplo.com:8443"]

# Sem nenhum endpoint no arquivo, o Gatus inicia normalmente quando admin.enabled é true
endpoints: []
```

Para gerar `password-bcrypt-base64`:

```bash
htpasswd -bnBC 10 "" 'sua-senha' | tr -d ':\n' | sed 's/$2y/$2a/' | base64 -w0 | tr '+/' '-_'
```

O Gatus decodifica esse valor com o alfabeto base64 de URL; o `tr '+/' '-_'` evita falhas com hashes que geram `+` ou
`/` no base64 padrão.

Com `security.basic`, o único usuário basic é o administrador. Com `security.oidc`, só os subjects de
`admin.allowed-subjects` administram; os demais usuários autenticados continuam vendo o dashboard.

## Uso

- **Lista (`/admin`):** busca por nome, grupo ou URL; mostra a origem (Web ou YAML), endpoints em conflito com o YAML e
  endpoints inválidos; permite habilitar, desabilitar e remover os endpoints cadastrados pela web.
- **Formulário (`/admin/endpoints/new` e `/admin/endpoints/<chave>/edit`):** modo formulário (nome, grupo, URL, método,
  intervalo, condições, headers, alertas e habilitado) e modo YAML, com as mesmas chaves de um item de `endpoints` do
  arquivo de configuração. Nome e grupo não mudam depois de criados.
- **Validar** confere a definição sem gravar. **Testar** executa uma verificação única, sem gravar resultado nem
  disparar alertas, e mostra cada condição. **Salvar** grava e aplica.
- **Remover** apaga a definição e todo o histórico do endpoint. Alertas disparados não são notificados aos provedores.

## Comportamento

- Criar, alterar, habilitar, desabilitar ou remover vale na hora, sem reiniciar o Gatus e sem reiniciar os demais
  endpoints. Uma verificação em andamento termina antes da mudança e o resultado dela é descartado.
- O histórico dos endpoints cadastrados pela web é mantido em reinícios e recargas do arquivo de configuração.
- Se o arquivo de configuração passar a definir a mesma chave (grupo e nome) de um endpoint cadastrado pela web, o do
  arquivo prevalece e o da web fica marcado como em conflito, sem ser apagado. Remover o da web, nesse caso, não afeta o
  do arquivo.
- Desligar `admin.enabled` apenas esconde a administração: os endpoints cadastrados pela web continuam monitorados.
- Durante a partida ou a recarga da configuração, alterações respondem 503 e nada é gravado.
- Com `skip-invalid-config-update: true`, um arquivo de configuração inválido não derruba mais o Gatus: a configuração
  anterior continua em uso até o arquivo ser corrigido.

## Restrições dos endpoints cadastrados pela web

- Variáveis de ambiente (`$VAR`, `${VAR}`) não são expandidas.
- Não são aceitos `client.identity-aware-proxy`, `client.tls.certificate-file`, `client.tls.private-key-file`, `store`
  e `always-run`, que usariam credenciais ou arquivos do servidor.
- `client.tunnel` precisa existir em `tunneling`.
- Alertas precisam de um provedor configurado em `alerting`; overrides inválidos são rejeitados.
- `extra-labels` só podem usar labels que já existam nos endpoints do arquivo de configuração (métricas Prometheus).
- A chave não pode ser igual à de um endpoint, external-endpoint, suite ou endpoint de suite do arquivo.

## Segredos

Toda resposta mascara com `********`: headers cujo nome contém `authorization`, `cookie`, `token`, `secret`, `password`
ou `key`; senha e parâmetros sensíveis da URL; `client.oauth2.client-secret`; `ssh.password`; `ssh.private-key`; e os
valores de `alerts[].provider-override`. Ao salvar, um valor igual à máscara mantém o valor armazenado. Os segredos
ficam em texto no banco, como ficariam no arquivo de configuração: proteja os backups.

## API

Todas as rotas exigem autenticação de administrador. Alterações exigem `Content-Type` `application/json` ou
`application/yaml` quando há corpo, e o `Origin` do navegador deve corresponder ao endereço do Gatus.

| Método e rota | Descrição |
|---------------|-----------|
| `GET /api/v1/admin/endpoints` | Lista endpoints do arquivo e cadastrados pela web |
| `GET /api/v1/admin/endpoints/{chave}` | Definição armazenada e efetiva (com `ETag`) |
| `POST /api/v1/admin/endpoints` | Cria (201) |
| `PUT /api/v1/admin/endpoints/{chave}` | Altera (exige `If-Match`) |
| `POST /api/v1/admin/endpoints/{chave}/enable` e `/disable` | Habilita ou desabilita (exige `If-Match`) |
| `DELETE /api/v1/admin/endpoints/{chave}` | Remove (exige `If-Match`) |
| `POST /api/v1/admin/endpoints/validate[?key=]` | Valida sem gravar |
| `POST /api/v1/admin/endpoints/test[?key=]` | Testa uma vez (no máximo 2 testes simultâneos) |
| `POST /api/v1/admin/endpoints/parse` | Converte YAML em documento JSON, sem validar |
| `GET /api/v1/admin/metadata` | Tipos de alerta, túneis e labels disponíveis |

Códigos: 400 definição inválida ou nome/grupo alterados; 404 inexistente; 409 chave em uso ou endpoint do arquivo de
configuração; 412 versão desatualizada; 413 corpo acima de 256 KB; 415 tipo de conteúdo inválido; 428 sem `If-Match`;
429 testes demais; 503 partida ou recarga em andamento.

```bash
curl -u admin:sua-senha -H 'Content-Type: application/yaml' \
  --data-binary $'name: site\ngroup: web\nurl: https://exemplo.com\nconditions: ["[STATUS] == 200"]\n' \
  http://127.0.0.1:8080/api/v1/admin/endpoints
```

## Atrás de proxy reverso

A origem esperada nas alterações é derivada do header `Host` e do esquema (`X-Forwarded-Proto`). Com nginx, repasse
esses headers:

```nginx
proxy_set_header Host $host;
proxy_set_header X-Forwarded-Proto $scheme;
```

Se o endereço público usar uma porta diferente da repassada no `Host`, liste-o em `admin.allowed-origins`.

## Várias instâncias com o mesmo PostgreSQL

Uma alteração feita numa instância só vale nas outras depois que elas reiniciarem ou recarregarem a configuração. Nesse
meio-tempo, uma instância que ainda monitora um endpoint removido pode recriar o histórico dele.

## Versões e imagens

- Releases do fork usam tags `v<versão-do-upstream>-fork.<N>` (ex.: `v5.36.0-fork.1`). Pela precedência do SemVer,
  essas tags ordenam abaixo da versão do upstream de mesma base em ferramentas como Renovate e `sort -V`.
- Imagem no Docker Hub: `jniltinho/gatus:<tag>` (`linux/amd64` e `linux/arm64`), publicada com
  `make docker-release VERSION=<versão sem o v>`. A tag `latest` não é publicada.

## Voltar para o Gatus original

A imagem original ignora a tabela `managed_endpoints`: os endpoints cadastrados pela web deixam de ser monitorados e o
histórico deles é apagado no primeiro start. Antes de voltar, faça backup do banco e garanta que o arquivo de
configuração tenha pelo menos um endpoint, senão a imagem original não inicia.

## Testes ponta a ponta

`test/e2e/admin.sh` sobe o Gatus local com SQLite temporário e percorre as telas com o
[agent-browser](https://github.com/vercel-labs/agent-browser), salvando capturas em `dist/prints/` (fora do git).
