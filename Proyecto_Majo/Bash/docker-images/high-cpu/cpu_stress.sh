#!/bin/bash
echo "=== STRESS INTENSIVO DE CPU ==="

# Obtener CPUs disponibles
CPUS=$(nproc)
echo "CPUs detectadas: $CPUS"

# Cleanup function
cleanup() {
    killall stress-ng 2>/dev/null
    exit 0
}
trap cleanup SIGTERM SIGINT

# Iniciar stress-ng con 90% de carga
stress-ng --cpu $CPUS --cpu-load 90 --timeout 0 &

# Bucle infinito con cálculos matemáticos intensivos
iteration=0
while true; do
    iteration=$((iteration + 1))
    
    # Cálculos intensivos en paralelo
    echo "scale=2000; 4*a(1)" | bc -l > /dev/null 2>&1 &
    echo "scale=1500; sqrt(2)" | bc -l > /dev/null 2>&1 &
    echo "scale=1000; e(10)" | bc -l > /dev/null 2>&1 &
    
    # Hash operations
    for i in {1..10}; do
        echo "CPU stress $iteration-$i $(date)" | sha256sum > /dev/null &
    done
    
    # Operaciones aritméticas
    result=0
    for i in {1..10000}; do
        result=$((result + i * i % 1000))
    done
    
    if [ $((iteration % 100)) -eq 0 ]; then
        echo "Iteración $iteration - CPU al máximo"
    fi
    
    sleep 0.1
done
