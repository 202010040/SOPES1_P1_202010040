#!/bin/bash

echo "=== PROCESO DE BAJO CONSUMO ==="
echo "Contenedor: $(hostname)"
echo "PID: $$"

# Crear directorios
mkdir -p /tmp/app/logs

# Cleanup function
cleanup() {
    echo "$(date): Terminación limpia"
    exit 0
}
trap cleanup SIGTERM SIGINT

# Inicializar
echo "$(date): Contenedor iniciado" > /tmp/app/status.txt
counter=0

# Bucle principal - MUY eficiente
while true; do
    counter=$((counter + 1))
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    # Actividad muy ligera
    echo "[$counter] $timestamp - Sistema normal" >> /tmp/app/logs/activity.log
    
    # Actualizar estado
    cat > /tmp/app/status.txt << EOF
Estado: Funcionando
Última actualización: $timestamp
Contador: $counter
PID: $$
Contenedor: $(hostname)
EOF
    
    # Mantener logs pequeños
    if [ $((counter % 20)) -eq 0 ]; then
        tail -n 50 /tmp/app/logs/activity.log > /tmp/app/logs/temp.log
        mv /tmp/app/logs/temp.log /tmp/app/logs/activity.log
        echo "Ciclo $counter - $(date)"
    fi
    
    # Test conectividad muy ligero
    ping -c 1 -W 1 127.0.0.1 >/dev/null 2>&1
    
    # Dormir 30 segundos para minimizar recursos
    sleep 30
done
