## Why

Uma página de status mostra no máximo 200 endpoints. O número protege uma rota sem login — limita a carga no storage, o tamanho do payload e do DOM —, mas é uma constante no código (`MaximumEndpoints = 200` em `internal/config/statuspage`): uma instalação com um inventário maior não tem como ajustá-lo, e o dono perguntou por que o `config.yaml` não tem essa opção. Com os grupos recolhíveis (`v6.1.0`), uma página grande deixou de ser um problema de tela, o que torna razoável subir o limite.

## What Changes

1. **`status-pages.maximum-endpoints-per-page`**, inteiro de `1` a `1000`, padrão `200` (o comportamento de hoje). É global, porque o que ele protege é o servidor, não a página.
2. **Dois limites no lugar de um.** Hoje a mesma constante faz duas coisas: limita quantas chaves uma definição pode listar uma a uma (validação) e quantos endpoints a página mostra (exibição). Elas se separam:
   - **teto estrutural**, fixo em `1000` chaves, na validação das definições — do arquivo, gravadas, enviadas pela administração e restauradas de um backup;
   - **limite de exibição**, o configurável, aplicado na seleção dos endpoints da página.
3. **O limite de exibição continua decidindo o acesso.** A seleção (`statuspage.Select`) é a mesma que autoriza as rotas por endpoint de uma página — detalhes, gráfico, stream de eventos e badges: um endpoint além do corte responde 404 em todas, como hoje além dos 200. O limite é capturado no snapshot publicado, para que todas essas rotas e o payload enxerguem o mesmo valor.
4. **Baixar o limite não tira página do ar.** Uma página que seleciona mais endpoints do que o limite em vigor continua válida e publicada, truncada, e a administração avisa.
5. **Aviso dinâmico** na página pública: "Showing the first N services", com o N do payload, no lugar do "200" fixo.

**Mudança de comportamento para quem não configurar nada:** uma definição com 201 a 1000 chaves escolhidas uma a uma, hoje recusada, passa a ser aceita e exibida truncada em 200. Nenhuma página que funciona hoje muda.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `public-status-pages`: a opção nova na seção `status-pages`, o teto de chaves da validação, e o truncamento pelo limite em vigor.
- `status-page-web-ui`: o aviso de página truncada com o número do payload, e o aviso na administração.

## Impact

- **Configuração:** `status-pages.maximum-endpoints-per-page`.
- **Código:** `internal/config/statuspage` (campo, validação, constante do teto), `internal/statuspage` (`Select` recebendo o limite, snapshot, `public.go`, `service.go`: contagens e aviso), `web/app/src/views/public/StatusPage.vue`, listagem e formulário da administração.
- **API da administração:** os itens da listagem e a validação ganham a indicação de que a página excede o limite em vigor. O payload público não ganha campo: o N do aviso é o `summary.total` que já existe.
- **Desempenho:** medido no servidor de validação, com 50 resultados por endpoint: 4,1 KB por endpoint, 0,65 KB com gzip. Com 1.000 endpoints, cerca de 4,1 MB (650 KB comprimido) por payload, guardado em cache por 30 s. O limitador por IP protege as respostas 404, não os downloads válidos: o custo de banda de um limite alto é de quem o configura, e a documentação diz isso.
- **Coordenação:** a change `add-bio-theme`, ainda ativa, modifica o mesmo requisito "Página pública de status". Os blocos MODIFICADOS das duas saem de geradores a partir do texto vigente; a que for arquivada por último regenera o seu.
