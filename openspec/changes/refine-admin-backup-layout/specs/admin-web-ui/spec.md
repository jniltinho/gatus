## ADDED Requirements

### Requirement: Diálogos e toasts da administração
As telas da administração MUST usar um diálogo padrão, com o visual do diálogo de confirmação, e MUST poder mostrar mensagens em toasts.

**Diálogos:**
- **Estrutura:** título, descrição opcional, botão de fechar, corpo com rolagem própria, rodapé com os botões à direita e altura máxima da janela menos uma margem;
- **Acessibilidade:** o painel MUST ter `role="dialog"`, `aria-modal`, nome pelo título e descrição associada quando existir;
- **Empilhamento:** diálogos abertos depois MUST ficar visualmente por cima dos anteriores;
- **Foco:** ao abrir, o foco MUST ir para o diálogo (no diálogo de confirmação, para Cancel) e MUST ficar preso no diálogo do topo, inclusive depois de um clique fora dele. Quando um diálogo fecha e outro continua aberto, o foco MUST ir para o diálogo que ficou no topo. Quando o último diálogo fecha, o foco MUST voltar a um elemento habilitado da página;
- **Fechar:** Esc e o botão de fechar MUST agir só no diálogo do topo e MUST NOT agir enquanto ele indicar uma operação em andamento; clicar fora MUST NOT fechar;
- **Rolagem:** a rolagem da página por trás MUST ficar bloqueada enquanto houver diálogo aberto;
- **Confirmação:** o diálogo de confirmação MUST usar esse diálogo, mantendo os botões Cancel e de confirmação e seus identificadores de teste, e fechar por Esc ou pelo botão de fechar MUST equivaler a Cancel.

**Toasts:**
- **Tipos e visual:** sucesso, informação, aviso e erro, com título opcional, ícone e cor por tipo, botão de dispensar e variantes do tema escuro;
- **Tempo:** MUST desaparecer sozinhos depois de 5 segundos (sucesso e informação), 8 segundos (aviso) ou 10 segundos (erro), salvo quando criados para ficar até serem dispensados. O tempo MUST pausar com o ponteiro ou o foco sobre os toasts e enquanto houver diálogo aberto. No máximo 4 ao mesmo tempo;
- **Posição:** centralizados, no topo em telas médias e grandes e abaixo do cabeçalho do app no celular, acima dos diálogos, sem cobrir botões de ação (inclusive os do cabeçalho) e sem bloquear cliques fora dos próprios toasts;
- **Anúncio:** erros MUST ser anunciados por uma região de alerta e os demais por uma região de status, ambas presentes antes das mensagens, e os toasts visíveis MUST aparecer em ordem cronológica;
- **Onde aparecem:** só com o app autenticado visível, nunca nas páginas públicas nem na tela de login, e as mensagens MUST ser descartadas ao trocar de tela.

#### Scenario: Esc no diálogo de confirmação
- **WHEN** o administrador abre a confirmação de remoção de um endpoint e aperta Esc
- **THEN** a confirmação fecha como Cancel e o endpoint continua existindo

#### Scenario: Foco preso
- **WHEN** um diálogo está aberto e o administrador aperta Tab repetidamente
- **THEN** o foco circula só pelos elementos do diálogo

#### Scenario: Foco depois da confirmação cancelada
- **WHEN** a prévia do restore está aberta, o administrador abre a confirmação e clica em Cancel
- **THEN** o foco fica dentro da prévia

#### Scenario: Toast não cobre a ação
- **WHEN** há um toast visível em janelas de 1280×900, 800×600 ou 390×844 e o administrador clica em Download, em Preview, num botão do rodapé de um diálogo ou, no celular, no menu do cabeçalho
- **THEN** o clique chega ao botão

#### Scenario: Toast dispensado
- **WHEN** o administrador clica no botão de dispensar de um toast de erro
- **THEN** o toast desaparece antes do tempo

#### Scenario: Troca de tela
- **WHEN** há um toast de erro na aba Backup e o administrador abre a aba Endpoints
- **THEN** o toast não aparece na aba Endpoints
