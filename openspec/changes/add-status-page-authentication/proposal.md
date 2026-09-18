## Why

Hoje uma status page é pública ou não existe: qualquer pessoa com o endereço `/status/<slug>` vê tudo o que ela publica. Quem quer mostrar o estado dos serviços **só para os clientes daquele contrato**, ou só para o time interno, não tem saída dentro do produto — precisa pôr um proxy na frente, o que quebra a página de detalhes, o canal de eventos em tempo real e o gráfico, todos servidos pelo próprio Gatus.

A administração já tem tudo de que essa proteção precisa: verificação de credencial em tempo constante com bcrypt, limite de tentativas por IP e mascaramento de segredos no backup e na visualização em YAML. Falta apenas oferecer isso por página.

## What Changes

- O formulário de criação e edição de status page ganha a opção **"Require login to view this page"**, com usuário e senha. Marcada, a página passa a exigir credencial; desmarcada, continua pública como hoje.
- A senha é digitada no formulário e o Gatus guarda **apenas o hash bcrypt**: a senha não é gravada, não volta em nenhuma resposta da API e não aparece no YAML da página. Na edição, o campo de senha vem vazio e só troca a senha se for preenchido.
- O visitante de uma página protegida é desafiado pelo navegador (autenticação HTTP Basic): `/status/<slug>`, a página de detalhes, a API pública da página, o canal de eventos e o gráfico passam a exigir a credencial daquela página. `curl -u` funciona do mesmo jeito.
- A página de detalhes hoje carrega os badges pelas rotas globais por chave, que são públicas no Gatus original e sustentam o dashboard. A change cria badges **da página**, atrás da mesma credencial, e a tela passa a usá-los; as rotas globais continuam públicas, e a documentação passa a dizer que proteger uma página esconde o conjunto que ela publica, não cada número de um endpoint cuja chave alguém já conheça.
- As buscas da página pública passam a enviar a credencial da mesma origem — hoje elas são feitas com `credentials: 'omit'`, e sem esse ajuste o visitante autenticaria e veria a página vazia.
- Tentativas erradas entram no mesmo limite por IP que a administração já usa, e a verificação bem-sucedida vale por alguns minutos sem repetir o bcrypt, para uma página muito acessada não virar custo de CPU.
- Páginas definidas no arquivo de configuração também podem exigir login, colando um hash bcrypt, como o `security.basic` já faz.
- As leituras da administração (detalhe, YAML, validação e pré-visualização) passam a mascarar o hash, e reenviar o documento mascarado mantém a credencial guardada — mascaramento que hoje só existe para endpoints.
- A lista da administração marca as páginas protegidas. O backup continua exportando as definições completas, como já faz com os tokens de push; o que muda é o restore, que passa a manter a credencial do destino quando o arquivo vier com o hash mascarado.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `public-status-pages`: a definição da página ganha a credencial, com validação; e as rotas públicas de uma página protegida passam a exigir autenticação, o que muda o requisito que hoje proíbe 401 nessas rotas.
- `status-page-management`: a API de administração aceita usuário e senha na criação e na alteração, guarda só o hash e nunca devolve a credencial.
- `status-page-web-ui`: o formulário ganha a opção com usuário e senha, e a lista marca as páginas protegidas.
- `admin-backup-restore`: o restore passa a mesclar a credencial guardada no destino antes de validar, quando o arquivo trouxer o hash mascarado.

## Impact

- `config/statuspage`: campo novo na definição e validação.
- `statuspage/` e `api/status_page.go`: verificação da credencial nas rotas públicas da página, com uma única resolução do slug por requisição, cabeçalho `WWW-Authenticate` sempre presente, limite de tentativas por página e memória curta da verificação.
- `api/badge.go` e `api/status_page.go`: rotas de badge sob a página, atrás da mesma credencial.
- `web/app/src/views/public/StatusPage.vue`, `StatusPageEndpoint.vue` e `components/ResponseTimeChart.vue`: envio de credencial da mesma origem, badges da página e tratamento do 401 na atualização periódica.
- `security/limiter.go`: limitador com chave por página e IP, que hoje só aceita IP.
- `managedendpoint`/`statuspage` da administração: aceitar a senha, gerar o hash, preservar o hash quando a senha vier vazia e mascarar na leitura.
- `web/app/src/views/admin/AdminStatusPageForm.vue` e `AdminStatusPages.vue`: campos e marcação na lista.
- `adminbackup/`: a credencial como segredo.
- `test/e2e/status-pages.sh` e testes Go; `web/static` versionado precisa ser regerado.
- `docs/status-pages.md` e `docs/README.md`: a opção nova e o comportamento das rotas.
