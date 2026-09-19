## Context

`ui.Config.Logo` é uma string que vai para `window.config.logo` pelo template de `index.html`; o frontend mostra `<img>` quando ela não é vazia. O logo do projeto já é um arquivo estático embutido (`web/static/logo-192x192.png`), servido sem autenticação, como o favicon.

## Decisions

### D1 — O padrão é resolvido no servidor
`ValidateAndSetDefaults` troca o valor vazio por `/logo-192x192.png` e `none` por vazio. O frontend não muda de regra: continua mostrando o logo quando a string não é vazia. Assim as três telas (dashboard, página pública, login) seguem juntas, sem uma segunda cópia da regra em JavaScript.

### D2 — `none` como palavra, não um booleano novo
Uma opção `ui.show-logo` criaria duas opções para uma decisão. `none` não colide com uma URL válida de logo (um caminho relativo chamado `none` não é servido por nada). A comparação ignora maiúsculas e espaços nas pontas.

### D3 — O caminho é absoluto na raiz
`/logo-192x192.png` é o mesmo caminho do manifesto e do `apple-touch-icon`. Atrás de um proxy com subcaminho o Gatus já não funciona (os demais recursos também são absolutos), então o logo não piora nada.

### D4 — 10% com `pt-[10vh]`
Mesma técnica de hoje (`pt-[15vh]`), só o número. O E2E mede a posição da borda superior do cartão em relação à altura da janela.

## Risks / Trade-offs

- **Mudança visível sem ação de quem atualiza** → documentada nas notas da release, com a saída (`ui.logo: none`).
- **Logo do projeto numa instalação com `ui.header` próprio** → é o caso do servidor de validação do dono ("Bio Uptime"), que validou o resultado na pré-visualização.
