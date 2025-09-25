#!/bin/bash

# Script principal del cronjob - se ejecuta cada minuto
LOG_FILE="/var/log/docker_cronjob.log"
HIGH_CONSUMPTION_IMAGES=("high-cpu-image" "high-ram-image")
LOW_CONSUMPTION_IMAGE="low-consumption-image"

# Función para logging
log_message() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Función para número aleatorio
random_number() {
    echo $(($RANDOM % ($2 - $1 + 1) + $1))
}

# Función para crear contenedor de alto consumo
create_high_consumption_container() {
    local image_type=$(random_number 0 1)
    local image=${HIGH_CONSUMPTION_IMAGES[$image_type]}
    local container_name="high_consumption_$(date +%s)_$RANDOM"
    
    if docker run -d --name "$container_name" "$image" >/dev/null 2>&1; then
        log_message "✓ Contenedor alto consumo creado: $container_name ($image)"
        return 0
    else
        log_message "✗ Error creando: $container_name"
        return 1
    fi
}

# Función para crear contenedor de bajo consumo
create_low_consumption_container() {
    local container_name="low_consumption_$(date +%s)_$RANDOM"
    
    if docker run -d --name "$container_name" "$LOW_CONSUMPTION_IMAGE" >/dev/null 2>&1; then
        log_message "✓ Contenedor bajo consumo creado: $container_name"
        return 0
    else
        log_message "✗ Error creando: $container_name"
        return 1
    fi
}

# MAIN: Crear 10 contenedores
log_message "=== INICIANDO CREACIÓN DE 10 CONTENEDORES ==="

for i in {1..10}; do
    # 70% alto consumo, 30% bajo consumo
    if [ $(random_number 1 10) -le 7 ]; then
        create_high_consumption_container
    else
        create_low_consumption_container
    fi
    sleep 0.5
done

log_message "=== CREACIÓN COMPLETADA ==="
