## MODIFIED Requirements

### Requirement: Tela de login do security.basic
Com `security.basic` configurado e sem `security.oidc`, o frontend MUST oferecer a rota `/login`, atendida pela SPA sem o cabeçalho do dashboard. A tela MUST mostrar:
- um cartão com o logo, o título de `ui.header`, os campos de usuário e de senha e o botão de entrar;
- o cartão centralizado na horizontal, com a borda superior a 15% da altura da janela;
- o tema (claro, escuro ou Bionexo) lido do mesmo cookie de tema das outras telas, com o mesmo seletor de tema das outras telas;
- o visual quadrado do fork e variantes `dark:`.

Com credenciais erradas, a tela MUST mostrar uma mensagem genérica que não indique se o usuário existe. Com o limite de falhas estourado, a tela MUST pedir para tentar mais tarde.

Depois de um login com sucesso, a tela MUST recarregar `GET /api/v1/config` antes de sair de `/login`, para o estado de autenticação da SPA refletir a nova sessão. Em seguida, MUST decodificar o `redirect` repetidamente, até ele parar de mudar (no máximo 3 vezes), e MUST levar a esse caminho somente quando:
- nenhuma decodificação tiver falhado (`%` malformado);
- o valor decodificado não tiver mais `%`;
- começar com um único `/`;
- não contiver `//`, `\`, esquema nem caracteres de controle;
- não apontar para `/login`.

Nos outros casos, MUST levar ao dashboard `/`. Sem `security.basic`, ou com `security.oidc`, a rota `/login` MUST levar a `/`.

#### Scenario: Visitante sem sessão abre a administração
- **WHEN** um navegador sem sessão abre `/admin`
- **THEN** a SPA mostra `/login?redirect=/admin` sem abrir a janela nativa de usuário e senha do navegador e sem o cabeçalho do dashboard
- **AND** o cartão de login fica a 15% do topo da janela, no tema do cookie de tema

#### Scenario: Senha errada
- **WHEN** o visitante envia a senha errada
- **THEN** a tela mostra "Invalid username or password" e continua em `/login`

#### Scenario: Login e redirecionamento
- **WHEN** o visitante envia as credenciais corretas em `/login?redirect=/admin/status-pages`
- **THEN** a SPA recarrega `GET /api/v1/config` e abre `/admin/status-pages`
- **AND** não volta para `/login`

#### Scenario: Redirecionamentos recusados
- **WHEN** o visitante faz login com `redirect` igual a `//site-malicioso.exemplo`, `/%2F%2Fsite-malicioso.exemplo`, `/%252F%252Fsite-malicioso.exemplo`, `/\site-malicioso.exemplo`, `https://site-malicioso.exemplo`, `/%ZZ` ou `/login`
- **THEN** a SPA abre o dashboard `/` em todos os casos
