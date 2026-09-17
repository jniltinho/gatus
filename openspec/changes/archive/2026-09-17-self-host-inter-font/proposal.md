## Why

O `tailwind.config.js` do fork põe `Inter` como primeira fonte da pilha `sans`, mas a Inter nunca é entregue: não há `@font-face`, arquivo de fonte no repositório nem link para serviço externo. Na prática, só quem tem a Inter instalada no sistema vê a interface como ela foi desenhada; todo o resto cai em `system-ui`, Segoe UI ou Roboto, com métricas e altura de linha diferentes. O mesmo painel muda de aparência conforme a máquina de quem olha, e as medidas de layout do fork (a linha de 66 px da página pública, as colunas de uptime) foram calculadas com uma fonte que nem sempre está lá.

Além disso, os números da interface usam algarismos proporcionais: cada dígito tem largura própria, então um tempo de resposta que vai de 188 ms para 111 ms muda de largura, as colunas de uptime "dançam" a cada atualização e os valores não se alinham na vertical. Num painel de monitoramento, onde quase todo número é dado que muda sozinho, isso atrapalha a leitura.

## What Changes

- A Inter passa a ser hospedada pelo próprio Gatus: dois arquivos `woff2` variáveis (latin e latin-ext, pesos 100 a 900) entregues pelo pipeline de assets do frontend e servidos em `/fonts/` pelo binário, com a licença OFL junto. Nenhuma requisição sai para Google Fonts ou qualquer outro domínio — inclusive nas páginas públicas, que hoje não chamam nada de fora.
- `@font-face` com `font-display: swap` e `unicode-range` por subconjunto, para o navegador baixar só o que a página usa, e `preload` do subconjunto latino no HTML.
- Os números da interface passam a usar algarismos tabulares (`font-variant-numeric: tabular-nums`), inclusive dentro dos campos de formulário, que o navegador reseta: todos os dígitos passam a ter a mesma largura, valores que mudam sozinhos param de mexer no layout e colunas de números ficam alinhadas.
- O gráfico de tempo de resposta, que desenha os rótulos em `canvas` com a pilha de fontes padrão do Chart.js, passa a usar a mesma família do resto da interface.
- A pilha `sans` continua a mesma, com as fontes do sistema como reserva: se o arquivo não carregar, a interface fica como é hoje.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `ui-theme`: a tipografia passa a fazer parte do tema — a fonte é entregue pelo próprio serviço, sem depender do que está instalado na máquina, e os números usam algarismos tabulares.

## Impact

- `web/app/src/assets/fonts/`: dois arquivos `woff2` (48 KB e 84 KB) versionados no repositório e emitidos pelo build em `web/static/fonts/`; `web/app/public/fonts/OFL.txt` com a licença.
- `web/app/src/index.css`: blocos `@font-face` e as regras dos algarismos tabulares (corpo e campos de formulário).
- `web/app/src/components/ResponseTimeChart.vue`: família do Chart.js igual à do corpo.
- `web/static_test.go` e um teste em `api`: garantia de que os arquivos estão embutidos e são servidos com `200` e `font/woff2`.
- `web/app/public/index.html`: `preload` do subconjunto latino.
- `web/app/tailwind.config.js`: a pilha `sans` passa a citar a família entregue.
- O binário cresce cerca de 132 KB (o `web/static` é embutido), contra 35 MB de hoje.
- `docs/README.md` e `AGENTS.fork.md`: de onde vem a fonte, como atualizá-la e o que fazer para trocá-la por `ui.custom-css`.
- Sem mudança de comportamento no backend, na API, no banco ou na configuração — só testes novos do lado Go.
