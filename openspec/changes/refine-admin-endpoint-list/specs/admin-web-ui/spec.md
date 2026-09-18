## MODIFIED Requirements

### Requirement: Lista de endpoints
A tela `/admin` MUST listar os endpoints ativos e Push, do arquivo de configuração (inclusive external endpoints) e gerenciados pela web. Cada item MUST expor nome, grupo, tipo (`PUSH` para endpoints Push), URL (com credenciais mascaradas e vazia para Push), intervalo (de heartbeat para Push), estado habilitado e origem (Web ou YAML), na tabela ou no cartão, conforme a largura da janela.

**Sem rolagem horizontal:** a lista MUST caber na largura disponível em qualquer janela a partir de 360 px, sem barra de rolagem horizontal, por mais longos que sejam o nome e a URL, e as ações de cada item MUST estar alcançáveis sem rolar para o lado. Para isso:

- a largura da tabela MUST NOT depender do conteúdo: os campos que podem crescer — nome e URL — MUST ser truncados, com o valor completo disponível ao repousar o ponteiro, e o aviso de conflito com o YAML MUST NOT somar largura à célula do nome;
- em janelas a partir de 1024 px a tabela MUST mostrar todas as colunas; entre 768 px e 1024 px o intervalo e a origem MUST sair da tabela;
- abaixo de 768 px a lista MUST deixar de ser tabela e virar um cartão por endpoint, com **todos** os campos, inclusive o intervalo, e com as mesmas ações, pelos mesmos identificadores de teste das ações da tabela.

As listas de status pages e de chaves de push MUST seguir as mesmas regras, com os campos de cada uma.

**Densidade e ações:** na tabela, todas as linhas MUST ter a mesma altura — inclusive a de um endpoint que também recebe push e a de um endpoint em conflito ou com definição inválida — e essa altura MUST ser de no máximo 32 px. Os selos MUST NOT quebrar em mais de uma linha, e o aviso de conflito ou de definição inválida MUST caber na mesma linha do nome, com a explicação disponível ao repousar o ponteiro e um nome acessível equivalente.

As ações de cada item MUST ser alvos de ícone, sem texto visível, cada um com nome acessível que inclua a ação e o nome do item, dica ao repousar o ponteiro e o mesmo identificador de teste da ação equivalente de antes. Uma ação que leva a outra página MUST continuar sendo um link, com o destino em nova aba onde já era. Uma ação indisponível MUST continuar explicando o motivo ao repousar o ponteiro. A confirmação de remoção MUST continuar a mesma, e os cartões das telas estreitas MUST usar os mesmos ícones, com alvo maior por serem telas de toque. A tela MUST permitir buscar por nome, grupo ou URL, MUST destacar endpoints em conflito ou com erro de validação e MUST permitir habilitar e desabilitar endpoints de origem Web. Endpoints de origem YAML MUST ser apenas visualizáveis.

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

#### Scenario: Linhas com a mesma altura
- **WHEN** a lista, numa janela de 1000 px, mostra um endpoint que também recebe push e um endpoint em conflito com o YAML, junto de endpoints comuns
- **THEN** o selo de push e o aviso de conflito cabem na mesma linha de cada um
- **AND** todas as linhas da lista têm a mesma altura, de no máximo 32 px

#### Scenario: Ações por ícone
- **WHEN** o administrador olha a linha de um endpoint gerenciado pela web
- **THEN** as ações aparecem como ícones, sem texto visível
- **AND** cada ícone tem nome acessível com a ação e o nome do endpoint
- **AND** o ícone de remover abre a mesma confirmação de antes
- **AND** uma ação indisponível continua explicando o motivo ao repousar o ponteiro

