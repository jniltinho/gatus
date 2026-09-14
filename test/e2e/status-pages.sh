#!/usr/bin/env bash
# Testes ponta a ponta das status pages públicas e das telas de administração delas, com agent-browser.
#
#   test/e2e/status-pages.sh
#
# Sobe o Gatus compilado localmente (SQLite temporário, basic auth, admin habilitado, endpoints locais), abre a página
# pública numa sessão sem credenciais e as telas de administração numa sessão com credenciais, e salva capturas em
# dist/prints/status-pages/ (dist/ está no .gitignore). Exige agent-browser com Chrome instalado.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"
PORT=${E2E_PORT:-18091}
BASE="http://127.0.0.1:$PORT"
PRINTS="$ROOT/dist/prints/status-pages"
WORK=$(mktemp -d)
USERNAME=admin
PASSWORD='e2e-senha'
# bcrypt (custo 10) de PASSWORD, em base64 com alfabeto de URL (como o Gatus decodifica)
PASSWORD_HASH='JDJhJDEwJHo1LnE5empYYkN5Vm1Vd1RmNXZPMS5SeWRCdlc3UlMxMXBHdmpwcDBUUTZiMXlIQ1R3RVRT'

command -v agent-browser >/dev/null || { echo "agent-browser não encontrado"; exit 1; }
mkdir -p "$PRINTS"

echo "==> Compilando"
make -s build

cat > "$WORK/config.yaml" <<CONFIG
web:
  address: 127.0.0.1
  port: $PORT
storage:
  type: sqlite
  path: $WORK/gatus.db
security:
  basic:
    username: $USERNAME
    password-bcrypt-base64: "$PASSWORD_HASH"
admin:
  enabled: true
endpoints:
  - name: health
    group: core
    url: $BASE/health
    interval: 5s
    conditions:
      - "[STATUS] == 200"
  - name: fora-do-ar
    group: core
    url: http://127.0.0.1:1/
    interval: 5s
    conditions:
      - "[STATUS] == 200"
  - name: painel
    url: $BASE/health
    interval: 5s
    conditions:
      - "[STATUS] == 200"
status-pages:
  rate-limit: 0
  pages:
    - slug: servicos
      title: "Serviços"
      description: "Página de teste ponta a ponta"
      groups: [core]
      endpoints: [_painel]
    - slug: rascunho
      title: "Rascunho"
      groups: [core]
      enabled: false
CONFIG

GATUS_CONFIG_PATH="$WORK/config.yaml" dist/gatus > "$WORK/gatus.log" 2>&1 &
GATUS_PID=$!
public() { agent-browser --session e2e-status-publico "$@"; }
admin() { agent-browser --session e2e-status-admin "$@"; }
cleanup() {
  public close >/dev/null 2>&1 || true
  admin close >/dev/null 2>&1 || true
  kill "$GATUS_PID" >/dev/null 2>&1 || true
  wait "$GATUS_PID" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

for _ in $(seq 1 60); do
  curl -sf "$BASE/health" >/dev/null && break
  sleep 1
done
curl -sf "$BASE/health" >/dev/null || { echo "Gatus não subiu"; cat "$WORK/gatus.log"; exit 1; }

STEP=0
step() {
  STEP=$((STEP + 1))
  echo "==> $STEP. $*"
}
fail() {
  echo "FALHOU: $*"
  public screenshot --full "$PRINTS/erro-publico.png" >/dev/null 2>&1 || true
  admin screenshot --full "$PRINTS/erro-admin.png" >/dev/null 2>&1 || true
  tail -20 "$WORK/gatus.log"
  exit 1
}
testid() {
  echo "[data-testid=\"$1\"]"
}
api_status() {
  curl -s -o /dev/null -w '%{http_code}' "$@"
}
body_text() {
  "$1" eval "document.body.innerText" 2>/dev/null
}
js() {
  local session=$1
  shift
  "$session" eval "$*" 2>/dev/null | tr -d '"'
}

step "API pública sem credenciais e rotas protegidas"
[ "$(api_status "$BASE/api/v1/status-pages/servicos")" = 200 ] || fail "esperado 200 na API pública"
[ "$(api_status "$BASE/api/v1/status-pages/rascunho")" = 404 ] || fail "esperado 404 para a página desabilitada"
[ "$(api_status "$BASE/api/v1/status-pages/a/b")" = 404 ] || fail "esperado 404 para caminho inválido, sem 401"
[ "$(api_status "$BASE/status/nao-existe")" = 200 ] || fail "esperado 200 na rota HTML"
[ "$(api_status "$BASE/api/v1/endpoints/statuses")" = 401 ] || fail "esperado 401 nas rotas protegidas"
if curl -s "$BASE/api/v1/status-pages/servicos" | grep -qE '127\.0\.0\.1|connection refused|core_health'; then
  fail "a API pública expôs URL, erro ou chave"
fi

step "Página pública no tema claro, sem credenciais"
public set viewport 1280 900 >/dev/null
public set media light >/dev/null
public open "$BASE/status/servicos" >/dev/null
public wait --text "Degradação parcial" >/dev/null || fail "a página não mostrou a degradação parcial"
public wait 700 >/dev/null
public screenshot --full "$PRINTS/01-servicos-claro.png" >/dev/null
[ "$(js public 'document.title')" = "Serviços" ] || fail "document.title não é o título da página"
[ "$(js public 'document.documentElement.lang')" = "pt-BR" ] || fail "o layout público não definiu lang=pt-BR"
grep -q "Outros serviços" <<<"$(body_text public)" || fail "endpoint sem grupo fora da seção Outros serviços"
requests=$(public network requests 2>/dev/null)
grep -q "/api/v1/config" <<<"$requests" && fail "a página pública chamou /api/v1/config"
grep -qE '\b401\b' <<<"$requests" && fail "alguma requisição da página pública recebeu 401"

step "Detalhe de uma verificação pelo teclado"
public eval "document.querySelector('[data-testid=\"status-endpoint-health\"] [role=group]').focus()" >/dev/null
public press ArrowLeft >/dev/null
public wait 300 >/dev/null
grep -q " ms" <<<"$(js public "document.querySelector('[data-testid=\"status-endpoint-health\"] [data-testid=status-endpoint-detail]').textContent")" || fail "o teclado não mostrou o detalhe da verificação"
public screenshot "$PRINTS/02-servicos-detalhe-teclado.png" >/dev/null

step "Tema escuro e tela de 390 px"
public set media dark >/dev/null
public open "$BASE/status/servicos" >/dev/null
public wait "$(testid status-summary)" >/dev/null || fail "a faixa de estado não apareceu"
public wait 700 >/dev/null
public screenshot --full "$PRINTS/03-servicos-escuro.png" >/dev/null
public set viewport 390 844 >/dev/null
public open "$BASE/status/servicos" >/dev/null
public wait "$(testid status-summary)" >/dev/null || fail "a faixa de estado não apareceu em 390 px"
public wait 700 >/dev/null
public screenshot --full "$PRINTS/04-servicos-390px-escuro.png" >/dev/null
[ "$(js public "document.querySelectorAll('[data-testid=\"status-endpoint-health\"] [role=group] > span').length")" = 25 ] || fail "esperadas 25 barras em tela estreita"
public set viewport 1280 900 >/dev/null
public set media light >/dev/null

step "Página inexistente, desabilitada e slug malformado"
public network requests --clear >/dev/null 2>&1 || true
for path in nao-existe rascunho "a%2Fb" "a/b"; do
  public open "$BASE/status/$path" >/dev/null
  public wait --text "Página não encontrada" >/dev/null || fail "/status/$path não mostrou Página não encontrada"
done
public screenshot "$PRINTS/05-pagina-nao-encontrada.png" >/dev/null
requests=$(public network requests 2>/dev/null)
grep -q "/api/v1/status-pages/a" <<<"$requests" && fail "o slug malformado chamou a API"
grep -qE '\b401\b' <<<"$requests" && fail "alguma página não encontrada recebeu 401"

step "OIDC simulado: sem tela de login na página pública; controle positivo no dashboard"
public network route "**/api/v1/config" --body '{"oidc":true,"authenticated":false}' >/dev/null
public open "$BASE/status/servicos" >/dev/null
public wait "$(testid status-page-title)" >/dev/null || fail "a página pública não carregou com OIDC simulado"
grep -q "Login with OIDC" <<<"$(body_text public)" && fail "a página pública mostrou a tela de login"
public open "$BASE/" >/dev/null
public wait --text "Login with OIDC" >/dev/null || fail "o controle positivo não mostrou a tela de login"
public screenshot "$PRINTS/06-controle-oidc-dashboard.png" >/dev/null
public network unroute >/dev/null 2>&1 || true

step "Administração: lista com as páginas do arquivo de configuração"
admin set credentials "$USERNAME" "$PASSWORD" >/dev/null
admin set viewport 1280 900 >/dev/null
admin set media light >/dev/null
admin open "$BASE/admin/status-pages" >/dev/null
admin wait "$(testid status-page-row-config-servicos)" >/dev/null || fail "a lista não mostrou a página servicos"
admin wait "$(testid status-page-row-config-rascunho)" >/dev/null || fail "a lista não mostrou a página rascunho"
admin screenshot --full "$PRINTS/07-admin-lista.png" >/dev/null

step "Administração: criar pelo formulário e validar"
admin click "$(testid admin-new-status-page)" >/dev/null
admin wait "$(testid status-page-field-slug)" >/dev/null || fail "o formulário não abriu"
admin fill "$(testid status-page-field-slug)" "equipe" >/dev/null
admin fill "$(testid status-page-field-title)" "Equipe" >/dev/null
admin fill "$(testid status-page-field-description)" "Serviços usados pela equipe" >/dev/null
admin click "$(testid status-page-group-core)" >/dev/null
admin fill "$(testid status-page-endpoint-search)" "painel" >/dev/null
admin click "$(testid status-page-endpoint-_painel)" >/dev/null
admin click "$(testid status-page-validate)" >/dev/null
admin wait --text "A página mostrará 3 endpoints" >/dev/null || fail "a validação não contou os 3 endpoints"
admin screenshot --full "$PRINTS/08-admin-validacao.png" >/dev/null

step "Administração: salvar (nasce desabilitada) e pré-visualizar"
admin click "$(testid status-page-save)" >/dev/null
admin wait --text "Status page criada" >/dev/null || fail "a criação não foi confirmada"
[ "$(js admin 'location.pathname')" = "/admin/status-pages/equipe/edit" ] || fail "a criação não levou à edição"
[ "$(api_status "$BASE/api/v1/status-pages/equipe")" = 404 ] || fail "a página recém-criada não deveria ser pública"
admin click "$(testid status-page-preview-button)" >/dev/null
admin wait "$(testid status-page-preview)" >/dev/null || fail "a pré-visualização não apareceu"
admin screenshot --full "$PRINTS/09-admin-previa.png" >/dev/null

step "Administração: publicar e abrir sem credenciais"
admin click "$(testid status-page-field-enabled)" >/dev/null
admin click "$(testid status-page-save)" >/dev/null
admin wait --text "salva e publicada" >/dev/null || fail "a publicação não foi confirmada"
public open "$BASE/status/equipe" >/dev/null
public wait --text "Serviços usados pela equipe" >/dev/null || fail "a página publicada não abriu sem credenciais"
public screenshot --full "$PRINTS/10-equipe-publica.png" >/dev/null

step "Administração: aviso de exposição no formulário de endpoints"
admin open "$BASE/admin/endpoints/new" >/dev/null
admin wait "$(testid admin-field-group)" >/dev/null || fail "o formulário de endpoints não abriu"
admin fill "$(testid admin-field-group)" "core" >/dev/null
admin fill "$(testid admin-field-name)" "novo" >/dev/null
admin wait "$(testid admin-endpoint-exposure)" >/dev/null || fail "o aviso de exposição não apareceu"
exposure=$(js admin "document.querySelector('[data-testid=admin-endpoint-exposure]').innerText")
grep -q "Equipe" <<<"$exposure" && grep -q "pelo grupo" <<<"$exposure" || fail "o aviso de exposição não citou a página Equipe pelo grupo"
admin screenshot "$PRINTS/11-admin-exposicao.png" >/dev/null

step "Administração: tema escuro e remoção"
admin set media dark >/dev/null
admin open "$BASE/admin/status-pages" >/dev/null
admin wait "$(testid status-page-row-admin-equipe)" >/dev/null || fail "a lista não mostrou a página equipe"
grep -q "Publicada" <<<"$(js admin "document.querySelector('[data-testid=\"status-page-row-admin-equipe\"]').innerText")" || fail "a página equipe não aparece publicada"
admin wait 700 >/dev/null
admin screenshot --full "$PRINTS/12-admin-lista-escuro.png" >/dev/null
admin click "$(testid status-page-remove-equipe)" >/dev/null
admin click "$(testid confirm-accept)" >/dev/null
admin wait --text "removida" >/dev/null || fail "a remoção não foi confirmada"
[ "$(api_status "$BASE/api/v1/status-pages/equipe")" = 404 ] || fail "a página removida ainda é pública"

echo "OK: $STEP etapas; capturas em $PRINTS"
