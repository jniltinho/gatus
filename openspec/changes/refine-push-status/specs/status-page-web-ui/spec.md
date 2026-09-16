## MODIFIED Requirements

### Requirement: Telas de administração de status pages
Com a administração habilitada e autorizada, o frontend MUST oferecer:
- as rotas `/admin/status-pages`, `/admin/status-pages/new` e `/admin/status-pages/:slug/edit`;
- abas para alternar entre endpoints e status pages.

A lista MUST mostrar slug, título, origem, estado (publicada, desabilitada, em conflito ou inválida) e número de endpoints, com as ações abrir (link em nova aba com `rel="noopener"`), copiar link, habilitar ou desabilitar, editar e remover (com confirmação). A lista MUST mostrar avisos quando a publicação estiver desligada no YAML, quando as páginas gerenciadas estiverem indisponíveis e quando houver aviso de limite compartilhado atrás de proxy.

O formulário MUST ter:
- slug (somente leitura na edição), título, descrição e `enabled`;
- a opção "Show certificate expiration" (`show-certificate-expiration`), desmarcada por padrão, com explicação curta;
- a opção "Show messages" (`show-messages`), desmarcada por padrão, ao lado da anterior, com a explicação de que as mensagens dos envios e o status HTTP ficam públicos e os erros das verificações não;
- seleção de grupos e de endpoints a partir de `/options`, com busca nos endpoints;
- avisos da validação e pré-visualização do payload público.

Uma página nova MUST começar desabilitada. As páginas do YAML MUST aparecer sem ações de alteração. Um 412 MUST levar a recarregar a página e avisar o administrador sem perder o que foi digitado.

#### Scenario: Criar e publicar
- **WHEN** um administrador cria a página `clientes` selecionando o grupo `core`, confere a pré-visualização e habilita a página
- **THEN** a lista mostra `clientes` como publicada
- **AND** o link copiado aponta para `/status/clientes`

#### Scenario: Página do YAML
- **WHEN** a lista mostra a página `infra` definida no YAML
- **THEN** a linha não tem as ações editar, habilitar, desabilitar e remover

#### Scenario: Edição concorrente
- **WHEN** outro administrador altera a página enquanto o formulário está aberto e o primeiro salva
- **THEN** a tela avisa que a página mudou e mantém o conteúdo digitado

#### Scenario: Publicação desligada
- **WHEN** a configuração tem `status-pages.enabled: false`
- **THEN** a lista mostra o aviso "Status pages desligadas no arquivo de configuração"

#### Scenario: Ligar a expiração do certificado
- **WHEN** um administrador marca "Show certificate expiration" na página `clientes` e salva
- **THEN** a definição salva tem `show-certificate-expiration: true`
- **AND** a página pública mostra os dias até o vencimento abaixo do nome dos endpoints com certificado

#### Scenario: Ligar as mensagens
- **WHEN** um administrador marca "Show messages" na página `jobs` e salva
- **THEN** a definição salva tem `show-messages: true`
- **AND** a página pública de detalhes dos endpoints de `jobs` mostra a tabela de verificações com Message e Origin
