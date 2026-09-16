## MODIFIED Requirements

### Requirement: Aba Backup da administração
A administração MUST ter a aba **Backup** em `/admin/backup`, depois de "Push keys", servida também ao abrir a URL diretamente.

**Layout:**
- em telas médias e grandes, a aba MUST ocupar a altura da janela como as listas da administração, sem rolagem da página e sem cortar conteúdo;
- Download e Restore MUST ficar em dois cartões lado a lado, cada um com cabeçalho, corpo com rolagem própria e rodapé com os botões, e os rodapés MUST continuar visíveis em janelas baixas;
- no celular, a página MUST poder rolar.

**Download:**
- MUST mostrar as quantidades;
- MUST oferecer a cifragem com senha e confirmação, desligada por padrão, validando o tamanho em bytes;
- sem senha, MUST avisar no cartão que o arquivo contém segredos em texto claro.

**Restore:**
- **Arquivo:** MUST aceitar qualquer arquivo, detectando o formato pelo conteúdo, e pedir a senha quando ele estiver cifrado. Um arquivo recusado MUST deixar a explicação visível junto do campo enquanto ele estiver selecionado;
- **Opções:** MUST ter "Overwrite existing items" e "Restore endpoints as disabled";
- **Prévia:** MUST abrir num diálogo com os avisos, o resumo, o filtro por ação e a tabela, e com Cancel e Restore no rodapé. Fechar o diálogo por Cancel, pelo botão de fechar ou por Esc MUST descartar a prévia;
- **Prévia em andamento:** trocar ou reler o arquivo, a senha ou as opções enquanto a prévia é calculada MUST descartar a resposta, sem abrir o diálogo;
- **Confirmação:** Restore MUST pedir confirmação com as quantidades de criações e atualizações do plano, e durante a aplicação a prévia MUST NOT poder ser fechada;
- **Resultado:** depois de aplicar, MUST abrir um diálogo com o resultado por item;
- **Senha errada:** além da mensagem, o campo de senha MUST ficar marcado como inválido, com o erro associado a ele, até ser editado.

**Mensagens:**
- o download concluído, o resumo do restore e os erros de download, das quantidades, da leitura do arquivo, da prévia e do restore MUST aparecer em toasts legíveis;
- os erros com instrução (409, 413, 422 e 429) MUST ficar até serem dispensados;
- os avisos de texto claro, de senha e de monitoramento MUST continuar no cartão ou no diálogo.

#### Scenario: Download cifrado pela tela
- **WHEN** o administrador marca "Encrypt with a password", digita a senha duas vezes e clica em Download
- **THEN** o navegador baixa `gatus-backup-<data>.enc.json`
- **AND** um toast de sucesso informa o nome do arquivo

#### Scenario: Restore pela tela
- **WHEN** o administrador escolhe um backup, clica em Preview, confere a tabela no diálogo e confirma
- **THEN** o diálogo de resultados mostra o resultado de cada item, um toast resume o restore e as listas mostram os itens restaurados

#### Scenario: Sem rolagem nem corte
- **WHEN** o administrador abre a aba Backup em janelas de 1280×900, 1280×720 ou 1024×600, com a cifragem ligada, com um arquivo cifrado selecionado ou com a prévia aberta
- **THEN** a página não rola, não há rolagem horizontal e os botões Download, Preview e os do rodapé do diálogo ficam dentro da janela

#### Scenario: Prévia fechada
- **WHEN** o administrador fecha a prévia com Esc
- **THEN** a prévia é descartada e um restore exige uma nova prévia

#### Scenario: Opção alterada durante a prévia
- **WHEN** o administrador clica em Preview e marca "Overwrite existing items" antes da resposta
- **THEN** o diálogo da prévia não abre

#### Scenario: Senha errada na tela
- **WHEN** a prévia de um backup cifrado é pedida com a senha errada
- **THEN** um toast de erro mostra "Invalid password or corrupted file.", nenhum diálogo abre e o campo de senha fica marcado como inválido com essa mensagem associada

#### Scenario: Arquivo que não é backup
- **WHEN** o administrador escolhe um arquivo JSON que não é backup
- **THEN** um toast de erro aparece e o status do arquivo continua explicando que ele não é um backup

#### Scenario: Backup cifrado no limite
- **WHEN** o administrador escolhe o envelope cifrado de um backup de 2 MiB
- **THEN** a tela aceita o arquivo e pede a senha

#### Scenario: URL aberta diretamente
- **WHEN** o administrador recarrega `/admin/backup`
- **THEN** a aba Backup é mostrada
