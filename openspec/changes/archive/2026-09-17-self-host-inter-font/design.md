## Context

- `web/app/tailwind.config.js` define `sans: ['Inter', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'sans-serif']` e `mono: ['Consolas', 'Monaco', '"Courier New"', 'monospace']`. Nenhum `@font-face`, nenhum arquivo de fonte no repositório e nenhum link externo: conferido em `web/app/public/index.html`, `web/app/src/index.css` e no `web/static` gerado.
- O binário embute `web/static` (`web/static.go`, `//go:embed static`) e o Fiber serve tudo em `/` (`api/api.go`, `fiberfs`). Um arquivo em `web/app/public/fonts/` é copiado pelo build para `web/static/fonts/` e passa a ser servido em `/fonts/...` sem mudança no backend.
- `web/static` tem hoje 836 KB; o binário compilado, 35 MB.
- A Inter do Google Fonts na versão `v20` do CSS, pedindo `wght@100..900`, entrega arquivos variáveis por subconjunto. Medidos: latin **48 KB**, latin-ext **84 KB** (os demais subconjuntos — cirílico, grego, vietnamita — somam mais de 200 KB e não são usados pela interface, que é em inglês).
- `ui.custom-css` é um `<link>` carregado **antes** do `app.css`. O `@layer` do Tailwind é diretiva de compilação, não camada nativa: o CSS gerado não tem nenhum `@layer` (conferido com `grep -c "@layer" web/static/css/app.css`, que devolve 0). Ou seja, com a mesma especificidade quem vem depois vence, e quem vence é o `app.css`. Regras do usuário empatadas precisam de `!important` — é o que a documentação já diz. A justificativa de camadas escrita na change da barra de rolagem estava errada; o resultado dela continua valendo, mas por ordem de arquivo, não por layer.
- Nenhum lugar da interface usa `font-variant-numeric` hoje.

## Goals / Non-Goals

**Goals:**

- A mesma tipografia para todo mundo, sem depender do que está instalado na máquina de quem abre.
- Nenhuma requisição para fora do próprio Gatus, inclusive nas páginas públicas.
- Números que não mudam de largura quando o valor muda, e colunas de números alinhadas.
- Degradar para as fontes do sistema se o arquivo não carregar.

**Non-Goals:**

- Trocar a fonte monoespaçada (chaves, slugs, YAML), que continua sendo a do sistema.
- Oferecer escolha de fonte por configuração; quem quiser outra usa `ui.custom-css`.
- Empacotar todos os subconjuntos da Inter.
- Mudar tamanhos, pesos ou espaçamentos já definidos.

## Decisions

### D1 — Quais arquivos entram e de onde vêm

Dois arquivos variáveis, pesos 100–900, recortados pelo Google Fonts a partir da Inter oficial (`rsms/inter`, licença OFL 1.1), com nomes que carregam a versão:

| Arquivo | Subconjunto | Tamanho |
|---|---|---|
| `inter-4-1-latin.woff2` | latin | 48 KB |
| `inter-4-1-latin-ext.woff2` | latin-ext | 84 KB |

**Onde ficam:** os `woff2` vão para `web/app/src/assets/fonts/` e são referenciados por caminho **relativo** no `index.css`. Um `url('/fonts/…')` absoluto quebraria o build: o css-loader trata caminho iniciado por `/` como pedido de módulo e resolve a partir de `web/app`, onde o arquivo não existe. Pelo pipeline de assets do vue-cli, com `filenameHashing: false` no `vue.config.js`, os arquivos são emitidos em `web/static/fonts/<nome>.woff2` e servidos em `/fonts/<nome>.woff2` — a mesma URL que o `preload` usa. De brinde, o build falha se o arquivo sumir.

A licença (`OFL.txt`) vai para `web/app/public/fonts/`, que é copiado tal e qual: ela precisa ser distribuída junto e servida, e cai na mesma pasta dos `woff2` no `web/static`. O procedimento de atualização (URLs exatas do `fonts.gstatic.com`, `User-Agent` usado na chamada ao `css2`, `unicode-range` copiados e os SHA-256 dos arquivos **como foram commitados**) vai para o `AGENTS.fork.md`, e **não** para dentro de `public/`: qualquer arquivo ali é servido publicamente, e o middleware estático está com `Browse: true`, então `/fonts/` lista o diretório.

**Nomes com a versão** são a estratégia de cache: o middleware não emite `Cache-Control`, então o navegador usa cache heurístico por `Last-Modified`; trocar o nome no dia da atualização garante que ninguém fique preso na fonte velha, sem mexer no middleware.

**Por que só latin e latin-ext:** a interface e a documentação do fork são em inglês, e os nomes de endpoint dos usuários são majoritariamente latinos. Um nome em cirílico ou grego continua legível: cai na fonte do sistema, que é o comportamento de hoje para tudo.

### D2 — `@font-face` e carregamento

Em `index.css`, antes das camadas do Tailwind:

```css
@font-face {
  font-family: 'Inter';
  font-style: normal;
  font-weight: 100 900;
  font-display: swap;
  src: url('./assets/fonts/inter-4-1-latin.woff2') format('woff2');
  unicode-range: U+0000-00FF, U+0131, U+0152-0153, U+02BB-02BC, U+02C6, U+02DA, U+02DC, U+0304, U+0308, U+0329,
    U+2000-206F, U+20AC, U+2122, U+2191, U+2193, U+2212, U+2215, U+FEFF, U+FFFD;
}

@font-face {
  font-family: 'Inter';
  font-style: normal;
  font-weight: 100 900;
  font-display: swap;
  src: url('./assets/fonts/inter-4-1-latin-ext.woff2') format('woff2');
  unicode-range: U+0100-02BA, U+02BD-02C5, U+02C7-02CC, U+02CE-02D7, U+02DD-02FF, U+0304, U+0308, U+0329,
    U+1D00-1DBF, U+1E00-1E9F, U+1EF2-1EFF, U+2020, U+20A0-20AB, U+20AD-20C0, U+2113, U+2C60-2C7F, U+A720-A7FF;
}
```

- **Caminho relativo ao `index.css`, não à página:** o css-loader reescreve a referência para a URL final do asset (`/fonts/…`, absoluta, porque `publicPath` é `/`), então nada quebra em `/status/<slug>/endpoints/<key>`.
- **`font-display: swap`:** o texto aparece imediatamente com a fonte do sistema e troca quando a Inter chega. Numa rede local isso é imperceptível; num link ruim, é melhor que texto invisível.
- **`preload` no `index.html`** só do subconjunto latino: `<link rel="preload" href="/fonts/inter-4-1-latin.woff2" as="font" type="font/woff2" crossorigin />`, ao lado dos outros `<link>` manuais e **antes** do `custom.css`, para o download começar cedo. O `crossorigin` é obrigatório mesmo na mesma origem, senão o navegador baixa o arquivo duas vezes. O latin-ext fica sem preload. O minificador do template Go não mexe em aspas de atributo (o `index.html` gerado mantém `href="/apple-touch-icon.png"`); o que quebrou no passado foi aspa dentro de uma ação de template, e este `link` não tem nenhuma.
- **Pilha:** `sans` passa a ser `['Inter', 'system-ui', …]` — a mesma de hoje, só que agora a primeira existe de fato. Nada muda se o arquivo faltar.
- Os blocos ficam fora de `@layer` porque `@font-face` não participa de cascata. Trocar a família pelo `ui.custom-css` continua funcionando **sem** `!important`, mas por especificidade e não por camada: o Tailwind põe a pilha no `html`, e um `body { font-family: … }` do usuário vence para o body e tudo que herda dele.

### D3 — Algarismos tabulares

Duas regras em `@layer base`:

```css
body {
  font-variant-numeric: tabular-nums;
}

/* The user agent resets font-variant-numeric on form controls through the font shorthand */
button,
input,
optgroup,
select,
textarea {
  font-variant-numeric: inherit;
}
```

A segunda existe porque o preflight do Tailwind herda família, tamanho e peso nesses elementos, mas não `font-variant-numeric`: sem ela, os campos da administração (intervalos, limiares, portas, busca, YAML) ficariam com algarismos proporcionais ao lado de texto tabular.

- **Por que global e não classe por classe:** quase todo número visível no Gatus é dado que se atualiza sozinho (uptime, tempo de resposta, contadores, datas, versões). Marcar caso a caso deixaria buracos justamente nas telas novas. O texto corrido do produto é curto — títulos, descrições e mensagens — e a Inter tabular continua legível neles.
- **Onde isso aparece:** colunas de uptime da página pública, badges e painéis de tempo de resposta, tabelas de verificações, linha do tempo de eventos, contadores da administração e os números da aba Backup.
- **Gráfico:** o Chart.js desenha em `canvas` com a pilha padrão dele (`Helvetica Neue`, Arial…), que hoje coincide com o resto da tela porque tudo cai em fonte de sistema. Depois desta change o gráfico seria o único bloco fora da Inter, então o registro do Chart.js passa a definir `Chart.defaults.font.family` a partir da família computada do `body`. Algarismos tabulares não se aplicam a texto desenhado em canvas, e isso fica registrado.
- **Fora do alcance:** os badges SVG gerados pelo backend (`api/badge.go`), cujas larguras são calculadas no servidor com outra pilha de fontes — mexer ali quebraria o desenho.
- **Como desligar:** `body { font-variant-numeric: normal !important }` no `ui.custom-css`. O `!important` é necessário porque o `app.css` é carregado depois do `custom.css` e a especificidade é a mesma; a documentação passa a trazer esse exemplo.

### D4 — Testes

**No backend**, onde está a prova de que o arquivo é mesmo entregue:

1. `web/static_test.go` (`TestEmbed`) passa a exigir `fonts/inter-4-1-latin.woff2`, `fonts/inter-4-1-latin-ext.woff2` e `fonts/OFL.txt` no `embed.FS`. Sem isso, um `web/static` commitado sem as fontes passa no `go test`;
2. um teste em `api` MUST pedir `/fonts/inter-4-1-latin.woff2` e exigir `200`, `Content-Type: font/woff2` e os quatro primeiros bytes `wOF2`. É o que fecha o buraco de o SPA engolir a rota ou de o arquivo faltar.

**No E2E** (`test/e2e/status-pages.sh`, sessão pública sem credenciais):

3. **Fonte carregada de verdade:** depois de `await document.fonts.ready`, `Array.from(document.fonts).some((f) => f.family === 'Inter' && f.status === 'loaded')` MUST ser verdadeiro. `document.fonts.check('16px Inter')` **não** serve como prova: sem nenhum `@font-face`, a família resolve para fonte de sistema e ele devolve verdadeiro; e a família computada do `body` já começa por `Inter` hoje, sem fonte nenhuma entregue;
4. **Veio do próprio servidor:** `performance.getEntriesByType('resource')` MUST ter uma entrada de `/fonts/inter-4-1-latin.woff2` com `decodedBodySize > 0`, e nenhuma entrada de `fonts.googleapis.com` ou `fonts.gstatic.com`. A lista do `network requests` não serve sozinha: ela registra o pedido mesmo quando a resposta é 404, e a captura precisa ser a primeira do roteiro, porque as seguintes vêm depois de `--clear` e do cache;
5. **Algarismos tabulares de verdade:** um único `eval` que espera `document.fonts.ready`, insere no corpo da página dois `span` fora da tela herdando o estilo, um com `font-variant-numeric: normal` e outro com `tabular-nums`, cada um com `1111111111` e `9999999999`, compara `offsetWidth` e devolve `true`/`false`: no tabular as duas larguras MUST ser iguais, e o par de controle mostra que o recurso existe mesmo. Comparar só `111` com `999` não prova nada, porque quase toda fonte de sistema já tem dígitos de largura fixa; comparar `getBoundingClientRect` em ponto flutuante no bash também não funciona;
6. **Reserva:** com `network route "/fonts/**" --abort` (o agent-browser 0.37.1 não devolve 404, só aborta ou responde 200), a página MUST continuar legível e a família computada MUST cair na reserva do sistema. O console registra o aborto, então o critério não é console limpo.

No `push.sh` e nos demais roteiros nada muda, mas **todos os prints precisam ser regerados**: os antigos foram tirados com fonte de sistema.

## Risks / Trade-offs

- **Binário 132 KB maior** (0,4% dos 35 MB). É o preço de não depender de CDN nem do que está instalado na máquina.
- **Atualizar a fonte é manual:** quando a Inter mudar de versão, alguém precisa baixar de novo e conferir o `unicode-range`. O `README.md` da pasta guarda o comando e os hashes.
- **Algarismos tabulares em texto corrido** deixam números levemente mais largos e espaçados em frases. É o compromisso de um painel onde o número importa mais que a prosa; `ui.custom-css` desliga.
- **Licença OFL** obriga distribuir o texto da licença junto e não vender a fonte isolada — nada disso restringe o Gatus, que é distribuído como binário com os arquivos embutidos.
- **A fonte muda as larguras, não as alturas:** o preflight fixa `line-height: 1.5` e o projeto usa `leading-*` explícito, então a altura das linhas não depende da fonte. O que muda é largura: truncamento de nomes, quebra de linha e o espaço dos números, que os algarismos tabulares alargam um pouco. As medidas exatas dos roteiros são de altura ou comparam dois elementos com a mesma fonte, então nenhuma delas quebra — mas os prints todos precisam ser regerados.
- **Ordem em relação à change `slim-public-status-page`:** os 66 px por endpoint são conta daquela change, que ainda não foi implementada. Esta change entra primeiro, e lá as medidas já são feitas com a Inter aplicada; o contrário obrigaria a refazer medidas e prints duas vezes.
- **Sinais fora do subconjunto:** a interface usa `✓`, `✕` e `✗` (U+2713, U+2715, U+2717), que estão no bloco Dingbats e ficam fora de latin e latin-ext. Eles passam a ser desenhados pela fonte do sistema no meio de texto Inter. A conferência manual olha se destoa; se destoar, a saída é trocar por ícones `lucide`, que a interface já usa.
- **Sem itálico próprio:** só o `@font-face` normal é entregue, então os poucos trechos em itálico (anúncios antigos) usam o itálico sintético do navegador. Aceito.
- **Sem `Cache-Control`:** o middleware estático não emite cabeçalho de cache; o navegador usa heurística. Por isso o nome do arquivo carrega a versão da fonte.
- **`preload` continua baixando 48 KB** mesmo para quem trocou a família pelo `ui.custom-css`. É o custo de um `link` estático no HTML.
