---
name: create-release
description: Cria a tag de versão e a GitHub Release de jniltinho/gatus (tags vX.Y.Z, SemVer simples), publica a imagem jniltinho/gatus no Docker Hub com make docker-release e ajusta as notas. Use quando pedirem para criar ou publicar uma release, gerar uma tag de versão ou publicar a imagem do fork.
---

# Releases de jniltinho/gatus

Esta skill guia o processo completo de criar a tag de versão, a GitHub Release e a imagem de container.

---

## Antes de começar

```bash
gh auth status                 # precisa do escopo repo
docker info | grep Username    # precisa de login no Docker Hub para publicar a imagem
git status                     # precisa estar limpo
git fetch origin --tags && git checkout master && git pull --ff-only

# Última release (a maior tag; até a v6.0.0 elas eram v5.36.0-fork.N)
LAST_TAG=$(git tag --list 'v[0-9]*' --sort=-v:refname | head -n1)
echo "Última release: ${LAST_TAG:-none}"
```

---

## Esquema de versão

SemVer simples, `vX.Y.Z`, a partir da `v6.0.0`. O projeto deixou de ser fork do TwiN/gatus, então a versão não segue mais a do upstream, e as tags `v5.36.0-fork.N` ficam só como histórico: nunca criar outra.

- **Z** (patch): correções, sem mudança de comportamento para quem opera.
- **Y** (minor): recurso novo, compatível com a configuração e a API existentes.
- **X** (major): mudança que exige ação de quem opera (configuração, API, linha de comando).
- O módulo Go continua `gatus/v5`: nada fora de `internal/` é importável, então a versão maior da tag não precisa acompanhar.

```bash
NEXT=v6.0.1   # escolha pelo que mudou desde $LAST_TAG: git log --oneline "$LAST_TAG"..HEAD
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
3. Criar a GitHub Release com notas geradas a partir da tag anterior e anexar os `.tar.gz`

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
# Release vX.Y.Z

## ✨ Novidades
- ...

**Imagem:** `jniltinho/gatus:vX.Y.Z` (linux/amd64, linux/arm64)

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
- Assets: `gatus_X.Y.Z_linux_amd64.tar.gz` e `gatus_X.Y.Z_linux_arm64.tar.gz`
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
