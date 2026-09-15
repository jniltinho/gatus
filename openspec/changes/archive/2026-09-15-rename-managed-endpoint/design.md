## Context

Endpoints gerenciados pela administração (`add-admin-endpoint-management`) têm a chave `grupo_nome` derivada de `group` e `name` (`endpoint.Key()`). Hoje `managedendpoint.Service.update` compara o texto de `name` e `group` com o armazenado e responde `ErrKeyChanged` (400), e a tela bloqueia os dois campos na edição.

Estado atual relevante:

- **Definição:** `managed_endpoints` tem id próprio e `endpoint_key` único (TEXT em SQLite e PostgreSQL, `VARCHAR(768)` em MySQL). `UpdateManagedEndpoint` não troca a chave. Na carga, uma definição cuja chave calculada difere de `endpoint_key` é marcada inválida (`managedendpoint/registry.go`).
- **Histórico:** só a tabela `endpoints` guarda `endpoint_key`, `endpoint_name` e `endpoint_group`. Eventos, resultados, condições, uptime e alertas disparados apontam para `endpoint_id` com `ON DELETE CASCADE`, nos três bancos. As leituras de status usam o nome e o grupo dessa linha.
- **Limpeza:** `DeleteAllEndpointStatusesNotInKeys` roda na inicialização e em toda recarga do arquivo, preservando as chaves do YAML e as de `managed_endpoints`.
- **Execução:** `watchdog` registra execuções por chave; parar aguarda a verificação em andamento e descarta seu resultado. `update` para o monitoramento antes da transação (SQLite tem uma conexão só), restaura os alertas disparados consultando o store e inicia o novo monitoramento no `apply` da transação.
- **Status pages gerenciadas:** ficam em `managed_status_pages` (definição YAML por `slug`, com versão), numa camada própria (`statuspage`) que importa `managedendpoint`, e não o contrário.
- **Outros consumidores da chave:** séries Prometheus (label `key`), caches de status (API com 10 s e `writeThroughCache` do store SQL), rotas de badges e da página de detalhes, e `endpoints`/`featured` das status pages.

Restrições: o ciclo de recarga (`lifecycle`) bloqueia alterações; SQLite tem uma conexão só, então nenhuma consulta ao store pode acontecer dentro do `apply`; toda consulta usa placeholders `$N`.

## Goals / Non-Goals

**Goals:**

- Trocar nome e grupo de um endpoint gerenciado na edição, com a chave nova assumindo todo o histórico, de forma atômica nos três bancos.
- Manter as garantias atuais da alteração: parada consistente, estado dos alertas, sem afetar outros endpoints, rollback quando a transação falha.
- Não deixar dados que a limpeza da recarga apague por engano nem resultados gravados sob a chave antiga.
- Manter as status pages gerenciadas apontando para o endpoint e avisar sobre as do arquivo.

**Non-Goals:**

- Renomear endpoints do arquivo de configuração, externos ou suites.
- Redirecionar URLs antigas de badges, detalhes ou séries Prometheus.
- Mesclar históricos de duas chaves.
- Propagar a troca a outras instâncias antes de reiniciarem ou recarregarem.
- Mudança de esquema.

## Decisions

### D1. Renomear pelo próprio `PUT /api/v1/admin/endpoints/{chave-atual}`

A chave atual continua no caminho e a definição enviada traz o nome e o grupo novos; a resposta traz a chave nova e o `ETag`. `If-Match` continua obrigatório.

- **Alternativa:** rota própria (`POST /endpoints/{chave}/rename`). Rejeitada: o formulário salva a definição inteira de uma vez, e uma rota separada exigiria duas escritas (renomear e alterar), com duas versões e um estado intermediário.
- **Alternativa:** manter o 400 e exigir remover e recriar. É o problema que motivou a mudança.

### D2. Mover o histórico atualizando a linha de `endpoints`

Na mesma transação da definição: `UPDATE endpoints SET endpoint_key = $1, endpoint_name = $2, endpoint_group = $3 WHERE endpoint_key = $4`. Como tudo aponta para `endpoint_id`, uma linha move o histórico inteiro, sem cópia. Se o endpoint ainda não tem resultados, a atualização não afeta linhas e está correto. Quando a chave não muda mas o texto muda, a mesma instrução atualiza só o nome e o grupo exibidos.

- **Alternativa:** copiar as linhas para uma chave nova e apagar as antigas. Rejeitada: cara em históricos grandes e sujeita a deadlocks no MySQL.
- **Alternativa:** começar sem histórico e apagar o antigo. Rejeitada pela decisão de manter o histórico.
- **Alternativa:** renomear o histórico e a definição em transações separadas. Rejeitada: se a segunda falhar, a limpeza da recarga apaga o histórico, que ficaria sob uma chave sem definição.

### D3. Chave nova ocupada responde 409, sem mesclar

A validação em memória (`ConfigKeyOrigin` e as chaves gerenciadas) já rejeita chaves do YAML e de outros gerenciados. Dentro da transação, antes de escrever, o store confere também `managed_endpoints` e `endpoints` com a chave nova (outra instância pode ter criado a chave, e pode haver histórico ainda não limpo, por exemplo de endpoint removido do YAML antes da recarga). Se existir, devolve um erro novo `common.ErrEndpointKeyInUse`, que a API mapeia para 409. A restrição `UNIQUE(endpoint_name, endpoint_group)` de SQLite e PostgreSQL não precisa de tratamento à parte: nome e grupo iguais geram a mesma chave, já conferida.

- **Alternativa:** mesclar os históricos. Rejeitada: uptime e eventos sobrepostos ficariam inconsistentes.
- **Alternativa:** apagar os dados da chave nova. Rejeitada: apagaria dados sem aviso.

### D4. Método novo no store: `RenameManagedEndpoint`

`ManagedEndpointStore` ganha:

```go
RenameManagedEndpoint(managedEndpoint *common.ManagedEndpoint, expectedVersion int64,
    rename *common.ManagedEndpointRename, apply func() error) error
```

`common.ManagedEndpointRename` agrupa a chave antiga, o nome e o grupo novos, `MoveHistory` e as atualizações das status pages (`[]*common.ManagedStatusPageUpdate`), para não espalhar parâmetros posicionais.

Numa transação `inTransaction`, ele:

1. confere a versão;
2. confere se a chave nova está livre (D3);
3. troca `endpoint_key`, `definition` e a versão em `managed_endpoints`;
4. atualiza `endpoints` quando `moveHistory`;
5. aplica cada `ManagedStatusPageUpdate` (slug, definição, versão esperada, autor) com conferência de versão;
6. chama `apply` e confirma.

Depois do commit, apaga do `writeThroughCache` os padrões da chave antiga e da nova. `UpdateManagedEndpoint` continua para alterações sem troca de texto de nome e grupo.

O SQL é portável nos três bancos (só `UPDATE` e `SELECT` com `$N`), sem entrada nova em `dialect_mysql.go`. A troca de `endpoint_key` em `VARCHAR(768)` com `utf8mb4_bin` segue a mesma unicidade do PostgreSQL.

- **Alternativa:** mudar a assinatura de `UpdateManagedEndpoint`. Rejeitada: espalha parâmetros de renomeação por todas as alterações e pela habilitação.
- **Alternativa:** expor a transação (`*sql.Tx`) para as camadas de cima. Rejeitada: quebra o contrato de store e a regra de não consultar o store dentro do `apply`.

### D5. Ordem da renomeação em `managedendpoint.Service.update`

Com a mudança de ciclo em andamento e `statesMutex`:

1. Validar a definição (`prepare`), que já deixa a chave atual fora das chaves ocupadas.
2. Parar o monitoramento da chave antiga, como hoje.
3. Restaurar os alertas disparados consultando pela chave antiga (D6).
4. Pedir às status pages as atualizações de referência (D7).
5. Chamar `RenameManagedEndpoint` com `apply` iniciando o monitoramento da chave nova.
6. Em erro: parar a chave nova, se iniciou, reiniciar o endpoint anterior e descartar as atualizações das páginas.
7. Depois do commit:
   - trocar o estado em memória numa publicação só (remove a antiga e adiciona a nova; hoje `putState` só adiciona);
   - publicar as status pages alteradas;
   - apagar as séries Prometheus da chave antiga;
   - invalidar o cache `endpoint-status-*` da API;
   - registrar a auditoria com as duas chaves.

Como a parada aguarda a verificação em andamento e descarta seu resultado, e o objeto do endpoint antigo não é mais usado depois dela, nenhuma escrita chega com a chave antiga depois do commit. Uma escrita da chave nova só acontece depois do commit, porque a primeira gravação espera a conexão (SQLite) ou a linha renomeada (PostgreSQL e MySQL, em `READ COMMITTED`).

- **Alternativa:** renomear sem parar e trocar o endpoint em execução. Rejeitada: uma verificação em andamento gravaria sob a chave antiga e recriaria a linha em `endpoints`.

### D6. Estado dos alertas restaurado pela chave antiga, antes da transação

`RestorePersistedTriggeredAlerts` usa `ep.Key()`. Na renomeação, é chamado com uma cópia rasa do endpoint novo com o nome e o grupo antigos, compartilhando os ponteiros de `Alerts`, e os contadores são copiados de volta para o endpoint novo.

Os alertas disparados com configuração alterada são apagados pela chave antiga e os demais são restaurados; as linhas vão para a chave nova junto com `endpoint_id` na transação. É o mesmo momento e a mesma garantia de hoje, que também apaga antes da transação.

- **Alternativa:** restaurar depois do commit pela chave nova. Rejeitada: o monitoramento já começou no `apply` e a primeira execução poderia disparar de novo o mesmo incidente.
- **Alternativa:** copiar o estado do objeto em memória do endpoint anterior. Rejeitada como mecanismo único: um endpoint inválido ou em conflito não tem objeto em execução, e o store é a fonte usada no início e na recarga.

### D7. Status pages atualizadas por um participante registrado por `statuspage`

`managedendpoint` não pode importar `statuspage`, que já importa `managedendpoint`. `managedendpoint` expõe uma interface, e `statuspage` a registra na inicialização:

```go
type KeyRenameParticipant interface {
    PrepareKeyRename(oldKey, newKey, author string) (*KeyRenamePlan, error)
}
```

O plano traz as atualizações das status pages gerenciadas e o que publicar depois do commit ou descartar no erro. Ele traz também a lista das status pages do arquivo que selecionam a chave antiga por `endpoints` ou `featured`.

- `PrepareKeyRename` segura o mutex de `statuspage` até publicar ou descartar.
- Reescreve cada página com `Parse`, troca a chave nos dois campos sem duplicar entradas e serializa com `yaml.Marshal`, como o salvamento das páginas.
- **Ordem de locks:** ciclo de alteração, `managedendpoint.statesMutex`, mutex de `statuspage`, transação. `statuspage` não pode adquirir `statesMutex`: ela só lê o snapshot atômico de `managedendpoint`, o que deve ser conferido nos testes.

A versão incrementada de uma página gerenciada faz um editor aberto dessa página receber 412, o comportamento já documentado para edição concorrente. Uma página alterada por outra instância entre a leitura e a escrita faz a renomeação inteira responder 409, sem mudar nada.

- **Alternativa:** orquestrar na API, que conhece os dois serviços. Rejeitada: a API teria de montar a transação e repetir a lógica de rollback dos dois serviços.
- **Alternativa:** atualizar as páginas depois do commit, fora da transação. Rejeitada: uma falha deixaria a página pública sem o endpoint em silêncio.
- **Alternativa:** bloquear a renomeação de endpoint selecionado pela chave. Rejeitada: obrigaria editar cada página antes e depois.

### D8. Endpoint gerenciado em conflito com o YAML

O histórico da chave pertence ao endpoint do arquivo. A renomeação passa `moveHistory = false`, não restaura alertas e não mexe em status pages: as referências à chave antiga continuam valendo para o endpoint do arquivo. A chave nova começa sem histórico. Endpoints gerenciados inválidos, que não estão em conflito, movem o histórico normalmente.

### D9. Frontend

- Na edição de um gerenciado, nome e grupo ficam editáveis. O grupo usa o mesmo seletor da criação (grupos existentes, "No group", "New group…"), agora carregado também na edição, com o grupo atual selecionado.
- Quando a chave calculada no navegador (`endpointKey` de `utils/statusPage.js`) difere da atual, um aviso mostra a chave antiga e a nova, informa que URLs de badges e da página de detalhes mudam e lista as status pages do arquivo que selecionam a chave antiga.
  - A lista vem de `GET /api/v1/admin/status-pages/exposure` com a chave antiga, filtrando origem `config` e motivo `key`.
  - A exposição passa a considerar também `featured`.
- A resposta do `PUT` traz `affectedConfigStatusPages` (slug e título), exibida depois de salvar.
- Depois de salvar, a navegação continua para a lista. Um 409 mostra a mensagem do servidor, e `describeAdminError` ganha texto para chave ocupada.
- O e2e troca o passo "nome bloqueado" por renomeação com histórico preservado.

## Risks / Trade-offs

- **[Badges, links e dashboards externos com a chave antiga deixam de funcionar]** → Aviso na tela antes de salvar e seção na documentação. Redirecionamento fica fora do escopo.
- **[Outra instância com o mesmo banco ainda monitora a chave antiga e recria sua linha em `endpoints`]** → Mesmo risco já documentado para remoção. A recarga ou o reinício dessa instância carrega a chave nova, e a limpeza apaga a linha antiga; a documentação de várias instâncias passa a citar a renomeação.
- **[Histórico grande no MySQL e lock da linha de `endpoints` durante a transação]** → Só uma linha é atualizada; as tabelas filhas não mudam porque apontam para `endpoint_id`. A gravação de resultados da chave antiga já foi parada antes.
- **[Alertas com configuração alterada são apagados antes da transação, e uma falha posterior não os recupera]** → Mesmo comportamento de hoje na alteração. O estado dos alertas sem mudança é preservado.
- **[Página gerenciada aberta em outro navegador recebe 412 depois da renomeação]** → Comportamento existente de edição concorrente, com recarga da versão atual.
- **[Deadlock entre os mutex de `managedendpoint` e `statuspage`]** → Ordem única de locks (D7) e teste que renomeia enquanto uma status page é salva.
- **[Cliente de API que dependia do 400]** → Marcado como BREAKING na proposta e nas notas da release.

## Migration Plan

- Sem mudança de esquema: a atualização é só trocar o binário ou a imagem.
- **Rollback:** voltar para a versão anterior é seguro. A definição renomeada tem `endpoint_key` coerente com nome e grupo, e o histórico está sob a chave nova. A versão anterior carrega normalmente, só volta a bloquear novas renomeações.
- Documentar em `docs/admin-endpoints.md` e nas notas da release a mudança de URLs de badges e da página de detalhes.

## Open Questions

- Nenhuma que bloqueie a implementação.
- Se o uso mostrar necessidade, uma change futura pode guardar a chave antiga para redirecionar badges por um período.
