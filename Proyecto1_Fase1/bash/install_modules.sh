#!/bin/bash

# Script para instalar y configurar los módulos del Kernel
# Autor: 202010040

echo "=== Instalando módulos del kernel ==="

# Verificar si se ejecuta como root
if [ "$EUID" -ne 0 ]; then
    echo "Este script debe ejecutarse como root o con sudo"
    exit 1
fi

# Verificar que los archivos necesarios existen
if [ ! -f "ram_202010040.c" ] || [ ! -f "cpu_202010040.c" ] || [ ! -f "Makefile" ]; then
    echo "Error: Faltan archivos necesarios (ram_202010040.c, cpu_202010040.c, Makefile)"
    exit 1
fi

# Instalar dependencias si es necesario
echo "Verificando dependencias..."
apt-get update
apt-get install -y build-essential linux-headers-$(uname -r)

# Compilar los módulos
echo "Compilando módulos..."
make clean
make

# Verificar que la compilación fue exitosa
if [ ! -f "ram_202010040.ko" ] || [ ! -f "cpu_202010040.ko" ]; then
    echo "Error: Falló la compilación de los módulos"
    exit 1
fi

# Descargar módulos anteriores si existen
echo "Descargando módulos anteriores..."
if lsmod | grep -q "ram_202010040"; then
    rmmod ram_202010040
fi

if lsmod | grep -q "cpu_202010040"; then
    rmmod cpu_202010040
fi

# Cargar los nuevos módulos
echo "Cargando módulos..."
insmod ram_202010040.ko
insmod cpu_202010040.ko

# Verificar que se cargaron correctamente
echo "Verificando módulos cargados..."
if lsmod | grep -q "ram_202010040" && lsmod | grep -q "cpu_202010040"; then
    echo "✓ Módulos cargados exitosamente"
else
    echo "✗ Error al cargar los módulos"
    exit 1
fi

# Verificar que se crearon los archivos en /proc
if [ -f "/proc/ram_202010040" ] && [ -f "/proc/cpu_202010040" ]; then
    echo "✓ Archivos /proc creados exitosamente"
else
    echo "✗ Error: No se crearon los archivos en /proc"
    exit 1
fi

# Mostrar información de prueba
echo ""
echo "=== Prueba de módulos ==="
echo "Información de RAM:"
cat /proc/ram_202010040
echo ""
echo "Información de CPU:"
cat /proc/cpu_202010040

echo ""
echo "=== Instalación completada ==="
echo "Los módulos están disponibles en:"
echo "  - /proc/ram_202010040"
echo "  - /proc/cpu_202010040"