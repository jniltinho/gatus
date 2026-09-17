## MODIFIED Requirements

### Requirement: Lista de endpoints
A tela `/admin` MUST listar os endpoints ativos e Push, do arquivo de configuração (inclusive external endpoints) e gerenciados pela web. Cada item MUST expor nome, grupo, tipo (`PUSH` para endpoints Push), URL (com credenciais mascaradas e vazia para Push), intervalo (de heartbeat para Push), estado habilitado e origem (Web ou YAML), na tabela ou no cartão, conforme a largura da janela.

**Sem rolagem horizontal:** a lista MUST caber na largura disponível em qualquer janela a partir de 360 px, sem barra de rolagem horizontal, por mais longos que sejam o nome e a URL, e as ações de cada item MUST estar alcançáveis sem rolar para o lado. Para isso:

- a largura da tabela MUST NOT depender do conteúdo: os campos que podem crescer — nome e URL — MUST ser truncados, com o valor completo disponível ao repousar o ponteiro, e o aviso de conflito com o YAML MUST NOT somar largura à célula do nome;
- em janelas a partir de 1024 px a tabela MUST mostrar todas as colunas; entre 768 px e 1024 px o intervalo e a origem MUST sair da tabela;
- abaixo de 768 px a lista MUST deixar de ser tabela e virar um cartão por endpoint, com **todos** os campos, inclusive o intervalo, e com as mesmas ações, pelos mesmos identificadores de teste das ações da tabela.

As listas de status pages e de chaves de push MUST seguir as mesmas regras, com os campos de cada uma. A tela MUST permitir buscar por nome, grupo ou URL, MUST destacar endpoints em conflito ou com erro de validação e MUST permitir habilitar e desabilitar endpoints de origem Web. Endpoints de origem YAML MUST ser apenas visualizáveis.

#### Scenario: Busca
- **WHEN** o administrador digita `core` na busca
- **THEN** a lista mostra apenas endpoints cujo nome, grupo ou URL contém `core`

#### Scenario: Endpoint do YAML
- **WHEN** o administrador abre um endpoint de origem YAML
- **THEN** a tela mostra a definição em YAML com segredos mascarados, sem ações de salvar, habilitar, desabilitar ou remover

#### Scenario: External endpoint do YAML
- **WHEN** o arquivo de configuração define o external endpoint `jobs_backup`
- **THEN** a lista mostra `jobs_backup` com tipo `PUSH` e origem YAML, somente para visualização

#### Scenario: Endpoint em conflito
- **WHEN** existe um endpoint gerenciado marcado como em conflito
- **THEN** a lista mostra esse endpoint com um aviso de conflito com o YAML

#### Scenario: Janela estreita sem rolagem lateral
- **WHEN** o administrador abre `/admin` numa janela de 900 px de largura, com um endpoint de nome e URL longos
- **THEN** nem a lista nem a página têm rolagem horizontal
- **AND** as ações do endpoint gerenciado pela web ficam dentro da janela
- **AND** a URL aparece truncada, com o endereço completo ao repousar o ponteiro

#### Scenario: Lista em cartões no celular
- **WHEN** a lista é aberta com 390 px de largura
- **THEN** cada endpoint aparece como um cartão, sem tabela, com o intervalo entre os campos
- **AND** as ações continuam disponíveis pelos mesmos identificadores da tabela
- **AND** a página não tem rolagem horizontal
