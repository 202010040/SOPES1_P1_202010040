#!/bin/bash

# Script para desinstalar los módulos del Kernel
# Autor: 202010040

echo "=== Desinstalando módulos del kernel ==="

# Verificar si se ejecuta como root
if [ "$EUID" -ne 0 ]; then
    echo "Este script debe ejecutarse como root o con sudo"
    exit 1
fi

# Descargar módulos
echo "Descargando módulos..."

if lsmod | grep -q "ram_202010040"; then
    echo "Descargando módulo ram_202010040..."
    rmmod ram_202010040
    if [ $? -eq 0 ]; then
        echo "✓ Módulo ram_202010040 descargado"
    else
        echo "✗ Error al descargar ram_202010040"
    fi
else
    echo "- Módulo ram_202010040 no está cargado"
fi

if lsmod | grep -q "cpu_202010040"; then
    echo "Descargando módulo cpu_202010040..."
    rmmod cpu_202010040
    if [ $? -eq 0 ]; then
        echo "✓ Módulo cpu_202010040 descargado"
    else
        echo "✗ Error al descargar cpu_202010040"
    fi
else
    echo "- Módulo cpu_202010040 no está cargado"
fi

# Verificar que se descargaron
echo ""
echo "Verificando descarga..."
if ! lsmod | grep -q "ram_202010040" && ! lsmod | grep -q "cpu_202010040"; then
    echo "✓ Todos los módulos descargados exitosamente"
else
    echo "✗ Algunos módulos siguen cargados"
    lsmod | grep "202010040"
fi

# Verificar que se eliminaron los archivos /proc
if [ ! -f "/proc/ram_202010040" ] && [ ! -f "/proc/cpu_202010040" ]; then
    echo "✓ Archivos /proc eliminados exitosamente"
else
    echo "✗ Algunos archivos /proc siguen existiendo"
fi

# Limpiar archivos compilados
echo ""
echo "Limpiando archivos compilados..."
make clean 2>/dev/null || echo "No hay Makefile o ya está limpio"

echo ""
echo "=== Desinstalación completada ==="