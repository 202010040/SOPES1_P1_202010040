#!/bin/bash

# Script para crear 10 contenedores que estresen CPU y memoria RAM
# Autor: 202010040

echo "=== Creando contenedores de stress ==="

# Función para limpiar contenedores existentes
cleanup_containers() {
    echo "Limpiando contenedores anteriores..."
    for i in {1..10}; do
        docker stop stress-container-$i 2>/dev/null || true
        docker rm stress-container-$i 2>/dev/null || true
    done
}

# Función para crear contenedores de stress
create_stress_containers() {
    echo "Creando 10 contenedores de stress..."
    
    for i in {1..10}; do
        echo "Creando contenedor stress-container-$i..."
        
        # Crear contenedor con stress de CPU y memoria
        docker run -d \
            --name stress-container-$i \
            --memory="256m" \
            --cpus="0.5" \
            ubuntu:20.04 \
            bash -c "
                apt-get update && apt-get install -y stress;
                stress --cpu 2 --vm 2 --vm-bytes 128M --timeout 3600s
            " 2>/dev/null
        
        if [ $? -eq 0 ]; then
            echo "✓ Contenedor stress-container-$i creado"
        else
            echo "✗ Error creando contenedor stress-container-$i"
        fi
        
        # Pequeña pausa entre creaciones
        sleep 2
    done
}

# Función para mostrar estado de contenedores
show_containers_status() {
    echo ""
    echo "=== Estado de contenedores de stress ==="
    docker ps --filter "name=stress-container" --format "table {{.Names}}\t{{.Status}}\t{{.CPUPerc}}\t{{.MemUsage}}"
}

# Función para monitorear recursos
monitor_resources() {
    echo ""
    echo "=== Monitoreando recursos del sistema ==="
    echo "Presiona Ctrl+C para detener el monitoreo..."
    
    while true; do
        clear
        echo "=== Monitor de Recursos - $(date) ==="
        echo ""
        
        # Mostrar información de RAM desde nuestro módulo
        if [ -f "/proc/ram_202010040" ]; then
            echo "RAM (desde módulo kernel):"
            cat /proc/ram_202010040
        else
            echo "Módulo RAM no disponible"
        fi
        
        echo ""
        
        # Mostrar información de CPU desde nuestro módulo
        if [ -f "/proc/cpu_202010040" ]; then
            echo "CPU (desde módulo kernel):"
            cat /proc/cpu_202010040
        else
            echo "Módulo CPU no disponible"
        fi
        
        echo ""
        echo "Contenedores activos:"
        docker ps --filter "name=stress-container" --format "{{.Names}}: {{.Status}}" | head -5
        
        sleep 3
    done
}

# Menú principal
case "$1" in
    "start")
        cleanup_containers
        create_stress_containers
        show_containers_status
        echo ""
        echo "Contenedores de stress creados. Usa '$0 monitor' para ver el impacto."
        ;;
    "stop")
        cleanup_containers
        echo "✓ Todos los contenedores de stress eliminados"
        ;;
    "status")
        show_containers_status
        ;;
    "monitor")
        monitor_resources
        ;;
    *)
        echo "Uso: $0 {start|stop|status|monitor}"
        echo ""
        echo "Comandos:"
        echo "  start   - Crear 10 contenedores de stress"
        echo "  stop    - Eliminar todos los contenedores de stress"
        echo "  status  - Mostrar estado de contenedores"
        echo "  monitor - Monitorear recursos en tiempo real"
        exit 1
        ;;
esac