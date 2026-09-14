---
name: create-release
description: Cria a tag de versão e a GitHub Release do fork jniltinho/gatus (tags v<versão-upstream>-fork.<N>), publica a imagem jniltinho/gatus no Docker Hub com make docker-release e ajusta as notas. Use quando pedirem para criar ou publicar uma release, gerar uma tag de versão ou publicar a imagem do fork.
---

# Releases do fork jniltinho/gatus

Esta skill guia o processo completo de criar a tag de versão, a GitHub Release e a imagem de container do fork.

---

## Antes de começar

```bash
gh auth status                 # precisa do escopo repo
docker info | grep Username    # precisa de login no Docker Hub para publicar a imagem
git status                     # precisa estar limpo
git fetch origin --tags && git checkout master && git pull --ff-only

# Última release do fork (ignora as tags do upstream, como v5.36.0)
LAST_TAG=$(git describe --tags --abbrev=0 --match 'v*-fork.*' 2>/dev/null || echo "none")
echo "Última release do fork: $LAST_TAG"
```

---

## Esquema de versão

`v<versão-upstream-base>-fork.<N>`: `v5.36.0-fork.1`, `v5.36.0-fork.2`, …

- A base é a versão do upstream em que o fork está (o commit está em `UPSTREAM_BASE` no `Makefile`).
- `N` começa em 1 e sobe a cada release sobre a mesma base.
- Depois de sincronizar com uma versão nova do upstream (ex.: v5.37.0), volta a `v5.37.0-fork.1`.
- Nunca criar tags `vX.Y.Z` sem o sufixo: elas pertencem ao upstream.
- Pela precedência do SemVer, `v5.36.0-fork.1` ordena abaixo de `v5.36.0` (Renovate, `sort -V`).

```bash
BASE=v5.36.0   # confira a base atual antes de continuar
if [ "$LAST_TAG" = "none" ] || [ "${LAST_TAG%-fork.*}" != "$BASE" ]; then
  NEXT="$BASE-fork.1"
else
  NEXT="$BASE-fork.$(( ${LAST_TAG##*-fork.} + 1 ))"
fi
echo "Próxima versão: $NEXT"
```

---

## Processo recomendado

### 1. Revisar as mudanças desde a última release

```bash
START=$([ "$LAST_TAG" = "none" ] && echo "$BASE" || echo "$LAST_TAG")
git log "$START"..HEAD --oneline
```

### 2. Categorizar os commits

Seções nesta ordem; omita as vazias:

| Seção | Prefixo(s) |
|-------|-----------|
| **✨ Novidades** | `feat:` |
| **🔧 Melhorias e correções** | `fix:`, `perf:`, `refactor:`, `ci:` |
| **🧹 Manutenção** | `chore:`, `cleanup:` |
| **📚 Documentação** | `docs:` |

### 3. Criar e enviar a tag (dispara o workflow)

```bash
git tag -a "$NEXT" -m "Release $NEXT"
git push origin "$NEXT"
```

O `release.yml` vai:

1. Rodar os testes Go com race
2. Rodar `make release-cross` → tarballs `linux/amd64` e `linux/arm64`
3. Criar a GitHub Release com notas geradas a partir da tag anterior do fork e anexar os `.tar.gz`

### 4. Publicar a imagem no Docker Hub (desta máquina)

O workflow não publica imagens. Com o checkout na tag:

```bash
git checkout "$NEXT"
make docker-release VERSION="${NEXT#v}"      # publica jniltinho/gatus:$NEXT (amd64 e arm64), nunca latest
git checkout master
```

### 5. Acompanhar o workflow e ajustar as notas

```bash
gh run list --workflow release.yml --limit 3
gh release view "$NEXT"          # espere os assets aparecerem

cat > "/tmp/release-notes-$NEXT.md" << 'NOTES'
# Release vX.Y.Z-fork.N

## ✨ Novidades
- ...

**Imagem:** `jniltinho/gatus:vX.Y.Z-fork.N` (linux/amd64, linux/arm64)

**Full Changelog**: https://github.com/jniltinho/gatus/compare/vANTERIOR...vNOVA
NOTES

gh release edit "$NEXT" --title "$NEXT" --notes-file "/tmp/release-notes-$NEXT.md"
```

### 6. Verificar

```bash
gh release view "$NEXT"
docker buildx imagetools inspect "jniltinho/gatus:$NEXT"
```

- Título igual a `$NEXT`, sem marca de pré-release
- Assets: `gatus_X.Y.Z-fork.N_linux_amd64.tar.gz` e `gatus_X.Y.Z-fork.N_linux_arm64.tar.gz`
- Imagem com as plataformas `linux/amd64` e `linux/arm64`; a tag `latest` não foi alterada
- Link de Full Changelog correto

---

## O que o processo gera

| Artefato | Onde | Status |
|----------|------|--------|
| Tarball `linux/amd64` e `linux/arm64` | GitHub Release (workflow) | ✅ |
| Imagem `jniltinho/gatus:<tag>` amd64/arm64 | Docker Hub (`make docker-release`) | ✅ |
| Tag `latest` | — | ❌ intencionalmente: só tags de versão |
| Publicação da imagem pelo workflow | — | ❌ ainda não; exigirá secrets do Docker Hub |
| `.deb`/`.rpm`, outros sistemas | — | ❌ fora do escopo |
