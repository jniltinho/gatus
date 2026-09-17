## Why

A página pública gasta muita altura por serviço. Cada endpoint de grupo ocupa cerca de 100 px: a linha do nome, um espaço de 8 px, as barras de 24 px e ainda uma linha de 16 px reservada só para o detalhe da verificação que aparece ao passar o mouse — reservada mesmo quando não há nada para mostrar, porque senão a lista pularia. Somando o respiro de 32 px entre grupos e o `py-8` da página, uma página com poucos serviços já exige rolagem, e a leitura fica solta em vez de organizada.

Os números de uptime também ficam desalinhados: cada linha imprime "24h 100% 7d 100% 30d 100%" logo depois do nome, com a largura variando conforme o valor, então não dá para comparar dois serviços na vertical.

## What Changes

- A linha de cada endpoint fica em 66 px em vez de 100 px: menos respiro vertical, barras de 20 px e a linha reservada do detalhe deixa de existir.
- O detalhe da verificação (horário, resultado e duração) passa a aparecer num tooltip flutuante sobre as barras, que não ocupa espaço no layout e não empurra a lista. O texto para leitores de tela continua o mesmo.
- Os uptimes de 24h, 7d e 30d passam a ficar em colunas de largura fixa à direita, alinhadas entre todos os endpoints do grupo, com os rótulos dos períodos aparecendo uma vez no cabeçalho do grupo em vez de se repetirem em cada linha.
- O respiro entre grupos, o cabeçalho da página e os cartões dos endpoints em destaque encolhem na mesma proporção.
- O celular continua com o mesmo conteúdo: abaixo de 640 px cada linha mostra rótulo e valor juntos, como hoje, sem rolagem horizontal.
- De quebra, o requisito publicado da página pública é corrigido para o que o produto realmente faz: os rótulos de estado são em inglês ("Partial outage" e companhia) e o layout público **não** troca o idioma do documento para `pt-BR`. O texto da spec ficou para trás quando a interface passou a ser em inglês, e o roteiro E2E já reprova o contrário.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `status-page-web-ui`: a página pública passa a descrever o espaçamento enxuto, as colunas de uptime alinhadas e o detalhe da verificação em tooltip flutuante, sem linha reservada.

## Impact

- `web/app/src/components/public/EndpointRow.vue`: espaçamento, altura das barras, colunas de uptime e o tooltip do detalhe (o componente também é usado na página pública de detalhes, com `show-header` falso).
- `web/app/src/views/public/StatusPage.vue`: respiro do cabeçalho, das seções e dos cartões em destaque, e os rótulos dos períodos no cabeçalho do grupo.
- `test/e2e/status-pages.sh`: medidas da altura da linha, do alinhamento das colunas e do tooltip; `web/static` versionado precisa ser regerado.
- Sem mudança no backend, na API, no banco nem na configuração. O dashboard e as telas de administração não mudam.
