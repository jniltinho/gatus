## 1. Densidade e colunas

- [ ] 1.1 Tabela das três listas em `text-xs` com o nome em `text-sm`, células `px-2 py-1`, célula de ações `px-2 py-0` (é o alvo de 28 px que define a altura), cabeçalho `px-2 py-1.5` em `text-xs uppercase tracking-wide`, conforme D1. Alvo: linha de 29 px, igual com e sem selo.
- [ ] 1.2 Selos compactos (`px-1 text-[11px] leading-4`) com `whitespace-nowrap` no selo **e** na célula Type; o selo de push mostra `push`, com a dica "Also receives push" no `title`, e **mantém** o `data-testid="admin-accepts-push-<chave>"` que o `push.sh` espera. Conferir `WEBSOCKET` com selo em 768 px e, se não couber em `7rem`, subir a coluna Type para `8rem` e refazer as larguras.
- [ ] 1.3 Aviso de conflito e de definição inválida como ícone `AlertTriangle` ao lado do nome, `shrink-0`, com `title` e nome acessível, conforme D2.1 — sai a segunda linha dentro da célula.
- [ ] 1.4 Larguras e alinhamentos das três listas conforme as tabelas de D3, incluindo a remoção do `h-12` da célula de ações das chaves de push, mantendo o que a change anterior garantiu (sem rolagem horizontal, colunas escondidas entre `md` e `lg`, cartões abaixo de `md`).

## 2. Ações por ícone

- [ ] 2.1 Componente `AdminActionButton.vue` conforme D4: alvo de 28 px com ícone de 14 px, `type="button"`, `title` e `aria-label` iguais, ícone `aria-hidden`, variante destrutiva em vermelho com `dark:`, anel de foco interno, estado indisponível com `aria-disabled` mantendo a dica, e `href` opcional para renderizar `<a>` (com `target`/`rel`) no caso de "Abrir página pública".
- [ ] 2.2 Trocar as ações das três listas pelos ícones de D4 (`Pencil`, `Eye`, `CirclePause`, `CirclePlay`, `Trash2`, `ExternalLink`, `Link2`), mantendo os identificadores de teste e os diálogos de confirmação.
- [ ] 2.3 Mesmos ícones nos cartões das telas estreitas, com alvo de 36 px.
- [ ] 2.4 `npm run lint`.

## 3. Testes e entrega

- [ ] 3.1 `test/e2e/push.sh`: conferir, em 1000 e 900 px, que **todas** as linhas da lista de endpoints têm a mesma altura e que ela é de no máximo 32 px — o roteiro já tem `push.endpoints` e os selos de push. Conferir também que as ações não têm texto visível e têm `aria-label` com a ação e o nome do endpoint.
- [ ] 3.2 `test/e2e/push.sh`: baixar a janela do passo da barra de rolagem fina de 1280×420 para 1280×300, porque com as linhas de 29 px a lista deixaria de transbordar e o passo falharia; e a mesma conferência de altura e de nome acessível na lista de chaves de push.
- [ ] 3.3 `test/e2e/status-pages.sh`: a mesma conferência na lista de status pages, incluindo um item em conflito ou inválido se o roteiro já tiver um; senão, conferir a altura igual entre linhas de origem Web e YAML.
- [ ] 3.4 Rodar `status-pages.sh`, `push.sh`, `admin.sh` e `admin-backup.sh` e conferir os prints.
- [ ] 3.5 Conferência manual das três listas em 1280×900, 1000×800 e 900×700, nos dois temas, com endpoint que recebe push, endpoint em conflito, tipo `WEBSOCKET` e nome longo; cartões em 390 px; e o anel de foco na primeira e na última linha da área que rola.
- [ ] 3.6 Recapturar `docs/screenshots/admin-endpoints.png`, `admin-status-pages.png` e `admin-push-keys.png`, e atualizar a versão citada em `docs/screenshots/README.md`.
- [ ] 3.7 `make frontend-build` com o `web/static` no commit, `make lint` e `openspec validate refine-admin-endpoint-list --strict`.
- [ ] 3.8 Entrega: PR em `jniltinho/gatus` com CI verde, release da próxima versão da série com notas em inglês, imagem no Docker Hub, pacote `mariadb`, versões dos exemplos e PR de arquivamento.
