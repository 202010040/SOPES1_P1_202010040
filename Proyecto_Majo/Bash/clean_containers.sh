#!/usr/bin/env bash
set -euo pipefail

# =============================
# clean_containers.sh
# - Detecta rootless vs rootful
# - Reinicia daemon (opcional)
# - Para y elimina contenedores por prefijo
# - Modos: default (prefijos), --all, --dry-run, --no-restart
# =============================

PREF1="high_consumption_"
PREF2="low_consumption_"
DO_RESTART=1
DO_ALL=0
DRY_RUN=0

log() { echo -e "[clean] $*"; }
die() { echo "[clean][ERROR] $*" >&2; exit 1; }

while [[ "${1:-}" =~ ^- ]]; do
  case "$1" in
    --all) DO_ALL=1 ;;
    --dry-run) DRY_RUN=1 ;;
    --no-restart) DO_RESTART=0 ;;
    --help|-h)
      cat <<EOF
Uso: $0 [opciones]

Opciones:
  --all         Limpia TODOS los contenedores (no solo con prefijos)
  --dry-run     Muestra lo que haría sin ejecutar acciones
  --no-restart  No reinicia el daemon Docker
  -h, --help    Ayuda

Por defecto:
  - Reinicia el daemon (rootful: systemd, rootless: systemd --user si aplica)
  - Limpia SOLO contenedores con prefijos: "${PREF1}" y "${PREF2}"
EOF
      exit 0
      ;;
    *)
      die "Opción no reconocida: $1"
      ;;
  esac
  shift
done

# ---------- Detectar modo rootless vs rootful ----------
ROOTLESS="false"
if docker info >/dev/null 2>&1; then
  ROOTLESS="$(docker info 2>/dev/null | awk -F': ' '/Rootless/ {print tolower($2)}' || true)"
else
  die "No puedo ejecutar 'docker info'. ¿Docker está instalado/corriendo?"
fi

if [[ "$ROOTLESS" == "true" ]]; then
  DOCKER="docker"               # sin sudo
  SVC_RESTART="systemctl --user restart docker"
  SVC_STATUS="systemctl --user is-active docker"
else
  DOCKER="sudo docker"          # con sudo
  SVC_RESTART="sudo systemctl restart docker"
  SVC_STATUS="systemctl is-active docker"
fi

log "Rootless: $ROOTLESS"
log "CLI: $DOCKER"

# ---------- Reiniciar daemon (opcional) ----------
if [[ "$DO_RESTART" -eq 1 ]]; then
  log "Reiniciando daemon Docker..."
  if [[ "$DRY_RUN" -eq 1 ]]; then
    echo "$SVC_RESTART"
  else
    eval "$SVC_RESTART" || die "Fallo al reiniciar Docker"
  fi
  # Esperar a que esté activo
  for i in {1..10}; do
    if eval "$SVC_STATUS" >/dev/null 2>&1; then
      log "Daemon activo."
      break
    fi
    sleep 0.5
  done
fi

# ---------- Construir filtros ----------
if [[ "$DO_ALL" -eq 1 ]]; then
  FILTER_CMD="$DOCKER ps -aq"
  log "Modo --all: se eliminarán TODOS los contenedores."
else
  FILTER_CMD="$DOCKER ps -aq --filter \"name=$PREF1\" --filter \"name=$PREF2\""
  log "Modo por prefijo: $PREF1 , $PREF2"
fi

# ---------- Listar contenedores objetivo ----------
TARGETS=$(eval "$FILTER_CMD" || true)
if [[ -z "${TARGETS}" ]]; then
  log "No hay contenedores que coincidan con el criterio. Nada que hacer."
  exit 0
fi

log "Contenedores objetivo:"
if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "$TARGETS"
else
  # Mostrar tabla informativa
  eval "$DOCKER ps -a --format 'table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Image}}' \
        --filter name=$PREF1 --filter name=$PREF2" || true
fi

# ---------- Intentar stop -> kill -> rm -f ----------
STOP_CMD="$DOCKER stop $TARGETS"
KILL_CMD="$DOCKER kill $TARGETS"
RMF_CMD="$DOCKER rm -f $TARGETS"

log "Deteniendo contenedores..."
if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "$STOP_CMD"
else
  eval "$STOP_CMD" || log "Algunos no se pudieron detener (ok, se intentará kill)."
fi

log "Forzando (kill) contenedores restantes en ejecución..."
if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "$KILL_CMD"
else
  eval "$KILL_CMD" || log "Kill no aplicó o ya estaban detenidos."
fi

log "Eliminando contenedores (rm -f)..."
if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "$RMF_CMD"
else
  eval "$RMF_CMD" || die "Fallo eliminando contenedores."
fi

log "✅ Limpieza completada."
