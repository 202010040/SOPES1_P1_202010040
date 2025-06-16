#!/bin/bash
# Script para instalar módulos del kernel

set -e

echo "🔧 Instalando dependencias del sistema..."
sudo apt-get update
sudo apt-get install -y build-essential linux-headers-$(uname -r) make gcc

echo "📦 Compilando módulos del kernel..."
cd ../kernel

# Compilar módulo de RAM
echo "Compilando módulo RAM..."
make clean
make

# Cargar módulos
echo "🚀 Cargando módulos del kernel..."
sudo insmod ram_202010040.ko
sudo insmod cpu_202010040.ko

# Verificar que se cargaron correctamente
echo "✅ Verificando módulos cargados:"
lsmod | grep -E "(ram_202010040|cpu_202010040)"

# Verificar archivos en /proc
echo "📁 Verificando archivos en /proc:"
ls -la /proc/ | grep -E "(ram_202010040|cpu_202010040)"

echo "✅ Módulos del kernel instalados correctamente!"
