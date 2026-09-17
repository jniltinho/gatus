## Context

- **Estilos globais:** `web/app/src/index.css` tem as camadas do Tailwind, as variáveis do tema em `:root` e `:root.dark` (inclusive `--radius: 0`), um bloco `@layer base` com `* { @apply border-border }` e algumas regras soltas (`html { height: 100% }`, `body { min-height: 100vh }`). Não há regra de barra de rolagem.
- **Convenção das cores:** as variáveis guardam a tripla HSL sem a função (`--muted-foreground: 215.4 16.3% 46.9%`), e o Tailwind consome como `hsl(var(--…))`. `hsl(var(--…))` funciona igual em CSS comum.
- **Tema:** a classe `dark` no `<html>`, trocada por `utils/theme.js`, escuro por padrão (`ui-theme`).
- **Áreas com rolagem própria** (grep de `overflow-auto|overflow-x-auto|overflow-y-auto`), dez no total:
  - `components/StepDetailsModal.vue` (três áreas, num modal próprio, fora do `AdminDialog`);
  - `components/admin/AdminListLayout.vue` (painel das listas);
  - `components/admin/AdminDialog.vue` (corpo dos diálogos);
  - `components/RecentChecksTable.vue` (rolagem horizontal, também na página pública);
  - `views/admin/AdminBackup.vue` (os dois cartões);
  - `views/admin/AdminStatusPageForm.vue` (lista de seleção) e `views/admin/AdminEndpointForm.vue` (`<pre>` de 40 px de altura com rolagem horizontal).
  Nas páginas públicas, a única área própria é a tabela de checks; o resto rola com a página.
- **Navegadores:**
  - `scrollbar-width` e `scrollbar-color` são herdadas e valem no Firefox 64+, Chrome/Edge 121+ e Safari 18.2+;
  - `::-webkit-scrollbar` e companhia valem no Safari e no Chrome/Edge (qualquer versão), mas **são ignoradas quando `scrollbar-width` ou `scrollbar-color` estão definidas** no mesmo elemento;
  - `scrollbar-width: thin` deixa a largura a cargo do navegador (por volta de 11 px no Chromium e no Firefox), sem controle de hover nem de forma: no Windows 11 e no macOS o polegar sai arredondado;
  - as regras `::-webkit-scrollbar` dão controle total (largura, forma, hover), mas no macOS e no iOS **transformam a barra sobreposta em barra clássica**, que ocupa espaço.
- **Estilo do fork:** `ui-square-style` exige raio zero na interface.
- **`ui.custom-css`:** o `<link>` do CSS do usuário vem **antes** do `app.css` no `index.html`, então regras de barra escritas por quem usa o fork passam a precisar de `!important`.
- **E2E:** só o Chrome (agent-browser). Os roteiros `push.sh` e `admin-backup.sh` têm o helper `set_theme` (cookie), enquanto `admin.sh` e `login.sh` não têm.

## Goals / Non-Goals

**Goals:**
- Barra fina, quadrada e nas cores do tema no Chrome, no Edge e no Safari, com a mesma largura nos três.
- Firefox o mais próximo possível do mesmo resultado, com o que a plataforma permite.
- Valer na página e em todas as áreas com rolagem própria, na vertical e na horizontal, sem marcar componente por componente.
- Contraste mínimo de 3:1 do polegar sobre o fundo de menor contraste de cada tema.

**Non-Goals:**
- Reimplementar a barra em JavaScript.
- Reservar espaço para a barra (`scrollbar-gutter`).
- Esconder a barra.
- Igualar o Firefox ao pixel: lá a largura e a forma são do navegador.

## Decisions

### D1. Variáveis do tema
Em `index.css`, junto das outras, na convenção da tripla HSL:

| Variável | Onde | Valor |
|---|---|---|
| `--scrollbar-size` | `:root` | `10px` |
| `--scrollbar-thumb` | `:root` | `215.4 16.3% 46.9%` (o `--muted-foreground` claro, `#64748B`) |
| `--scrollbar-thumb-hover` | `:root` | `215.4 16.3% 32%` |
| `--scrollbar-thumb` | `:root.dark` | `215 20.2% 65.1%` (o `--muted-foreground` escuro, `#94A3B8`) |
| `--scrollbar-thumb-hover` | `:root.dark` | `210 40% 88%` |

- **Contraste medido** (WCAG 1.4.11, mínimo 3:1):
  - claro `#64748B` sobre o branco da página: 4,76:1; sobre `bg-gray-50` (`#F9FAFB`): 4,55:1; sobre `bg-muted` (`#F1F5F9`), o pior fundo claro com rolagem: 4,34:1;
  - escuro `#94A3B8` sobre o fundo (`#020817`): 7,8:1; sobre `dark:bg-gray-800` (`#1F2937`): 5,73:1.
- **Largura de 10 px:** fina, mas acima dos 8 px iniciais, para o polegar continuar clicável (WCAG 2.5.8 deixa de isentar quando o autor muda o tamanho) e para ficar perto do `thin` do Firefox (≈11 px), o que aproxima os navegadores.
- O trilho é transparente, para combinar com o fundo de cada área.

### D2. Regras de estilo
As regras ficam **fora de `@layer base`**, ao lado de `html { height: 100% }`: o `custom.css` de `ui.custom-css` é um `<link>` sem layer e venceria qualquer regra dentro de um layer, seja qual for a ordem. Fora de layer, quem já estiliza a barra pelo `ui.custom-css` passa a precisar de `!important`, como diz a documentação. As variáveis continuam junto das outras, em `@layer base`.

```css
/* Fork: thin scrollbar in the colors of the theme, only where the pointer is fine: on touch screens the scrollbar of
   the system is kept */
@media not all and (pointer: coarse) {
  /* Chrome, Edge and Safari: full control of the size, the shape and the hover */
  *::-webkit-scrollbar {
    width: var(--scrollbar-size);
    height: var(--scrollbar-size);
  }

  *::-webkit-scrollbar-track,
  *::-webkit-scrollbar-corner {
    background: transparent;
  }

  *::-webkit-scrollbar-button {
    display: none;
    width: 0;
    height: 0;
  }

  *::-webkit-scrollbar-thumb {
    background-color: hsl(var(--scrollbar-thumb));
    border-radius: 0;
  }

  *::-webkit-scrollbar-thumb:hover {
    background-color: hsl(var(--scrollbar-thumb-hover));
  }

  /* Firefox, which has no ::-webkit-scrollbar: the standard properties are inherited from the root */
  @supports not selector(::-webkit-scrollbar) {
    html {
      scrollbar-width: thin;
      scrollbar-color: hsl(var(--scrollbar-thumb)) transparent;
    }
  }

  @media (forced-colors: active) {
    *::-webkit-scrollbar-thumb {
      background-color: ButtonBorder;
    }
  }
}

@media (forced-colors: active) {
  html {
    scrollbar-color: auto;
  }
}

@media (prefers-contrast: more) {
  :root {
    --scrollbar-thumb: 222.2 47.4% 20%;
    --scrollbar-thumb-hover: 222.2 47.4% 11%;
  }

  :root.dark {
    --scrollbar-thumb: 210 40% 96%;
    --scrollbar-thumb-hover: 0 0% 100%;
  }
}
```

- **Por que separar os caminhos:** definir `scrollbar-width`/`scrollbar-color` fora do `@supports` faria o Chromium ignorar as regras `::-webkit-scrollbar`, e a barra sairia com a largura, a forma e o hover do navegador, que foi o erro da primeira versão.
- **Herança:** as propriedades padrão são herdadas, então basta o `html`. O `*` fica só nos pseudo-elementos do WebKit, que não herdam.
- **Telas de toque:** tudo fica dentro de `@media not all and (pointer: coarse)`, inclusive o caminho do Firefox e o `forced-colors` do WebKit, porque a regra webkit sozinha já troca a barra sobreposta pela clássica. Só `html { scrollbar-color: auto }` fica global, porque não muda o tipo de barra.
- **Por que `not all and (pointer: coarse)` e não `(pointer: fine)`:** um navegador sem nenhum dispositivo apontador responde `pointer: none` — é o caso do Chromium headless, onde os testes medem a barra, e de TVs e quiosques. Com `(pointer: fine)` esses ambientes ficariam sem o estilo e o E2E não teria o que medir. A negação de `coarse` mantém o que interessa: o celular e o tablet continuam com a barra do sistema. A forma `not all and (...)` é a sintaxe aceita em qualquer navegador.
- **Setas:** `display: none` mais `width/height: 0`, porque em alguns Chromium do Windows a seta continua ocupando espaço só com `display: none`.
- **Mais contraste:** `prefers-contrast: more` redefine o repouso **e** o realce, para o hover continuar mais forte que o repouso.
- **Se o Chromium reconhecer `selector(::-webkit-scrollbar)`:** o caminho do Firefox vira código morto, o que a conferência manual verifica (`CSS.supports('selector(::-webkit-scrollbar)')`). Se um dia o Firefox passar a reconhecer, o isolamento troca para uma condição só dele (por exemplo `@supports (-moz-appearance: none)`), sem mudar o resto.

### D3. macOS, iOS e Android
- **Trade-off assumido:** no macOS e no Safari do iPad, as regras `::-webkit-scrollbar` trocam a barra sobreposta por uma barra clássica, sempre visível, que ocupa espaço. É o preço de ter a mesma barra nos navegadores, que é o pedido, e fica registrado em Risks e na documentação.
- **Celular:** as regras ficam dentro de `@media not all and (pointer: coarse)`, então em telas de toque (Android e iPhone) a barra do sistema continua como é hoje, sem ocupar espaço.
- **O `pointer` olha o ponteiro principal:** um notebook Windows com tela de toque, um 2-em-1 em modo notebook e um iPad com teclado e trackpad não entram como ponteiro grosso e recebem a barra clássica. É o esperado: usar `any-pointer: coarse` desligaria o estilo em qualquer desktop com tela de toque, o oposto do pedido.
- **Sem ponteiro:** um ambiente que responde `pointer: none`, como o Chromium headless dos testes, recebe o mesmo estilo do desktop.

### D4. Testes
- **E2E** em `test/e2e/push.sh` (tem o helper `set_theme`, a lista da administração e a tabela de checks):
  - **Sessão com barras:** a sessão do roteiro sobe com `--hide-scrollbars false`, porque o agent-browser 0.37.1 esconde as barras nativas por padrão no Chromium headless. Sem isso, toda medida daria zero e o teste passaria sem barra nenhuma;
  - **Forçar a rolagem:** o painel da lista e a tabela de checks só têm barra quando o conteúdo transborda. O passo reduz a janela até `scrollHeight > clientHeight` no painel e `scrollWidth > clientWidth` na tabela, mede, e depois volta a janela para 1280×900, para não afetar o resto do roteiro;
  - **Espessura:** `offsetWidth - clientWidth` na vertical e `offsetHeight - clientHeight` na horizontal MUST ficar entre 9 e 10 px, o valor exato do `--scrollbar-size`. Um valor de 11 px indicaria que o Chromium caiu no caminho padrão, que foi o erro da primeira versão, e a faixa larga de antes (1–12) esconderia isso;
  - **Ponteiro:** o passo confere `matchMedia('(pointer: coarse)').matches === false` antes de medir, porque as regras estão nesse recorte. O Chromium headless responde `pointer: none`, então conferir `(pointer: fine)` faria o passo falhar sempre;
  - **Cor por tema:** `getComputedStyle(document.documentElement).getPropertyValue('--scrollbar-thumb').trim()` (o valor vem com espaço à esquerda) MUST ser `215.4 16.3% 46.9%` no claro e `215 20.2% 65.1%` no escuro;
  - **Prints:** lista com rolagem nos dois temas, na sessão com as barras visíveis.
- **Conferência manual** (registrada no PR): Firefox (barra fina e na cor do tema, e `CSS.supports('selector(::-webkit-scrollbar)') === false`), telas estreitas e, se houver acesso, Safari.
- **Sem teste unitário:** é só CSS.

## Risks / Trade-offs

- **macOS e iPad perdem a barra sobreposta** nas áreas com rolagem: passa a ser clássica e ocupa espaço. Assumido para ter a mesma barra nos navegadores de desktop.
- **Firefox não fica idêntico:** largura e forma são do navegador (`thin`), sem hover próprio. Fica fino e na cor do tema.
- **Alvo de clique de 10 px** é menor que os 24 px do WCAG 2.5.8. A rolagem continua disponível por roda, teclado e toque, e a barra do sistema seria de largura parecida em muitos ambientes.
- **`ui.custom-css`** de quem já estilizava a barra passa a precisar de `!important`.
- **Regra com `*`** nos pseudo-elementos vale para qualquer área de rolagem, inclusive as futuras. É o objetivo.

## Migration Plan

Só CSS, com o build de `web/static` versionado. Voltar à versão anterior traz a barra padrão do navegador.

## Open Questions

Nenhuma. As duas decisões em aberto na primeira versão foram fechadas: o caminho padrão fica só para o Firefox (D2), e a troca da barra sobreposta no macOS é aceita, com o celular preservado por `pointer: fine` (D3).
