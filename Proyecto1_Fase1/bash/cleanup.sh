#!/bin/bash
# Script para limpiar todos los servicios

set -e

echo "🧹 Limpiando todos los servicios del proyecto..."

# Ir al directorio raíz del proyecto
cd ..

# Detener y eliminar contenedores de Docker Compose
echo "📦 Deteniendo servicios de Docker Compose..."
docker-compose down -v --remove-orphans

# Eliminar contenedores de estrés si existen
echo "🔥 Eliminando contenedores de estrés..."
docker ps -q --filter name=stress- | xargs -r docker stop
docker ps -aq --filter name=stress- | xargs -r docker rm

# Limpiar imágenes del proyecto
echo "🖼️ Limpiando imágenes del proyecto..."
docker images --filter "reference=monitor-servicios-linux*" -q | xargs -r docker rmi -f

# Limpiar volumes huérfanos
echo "💾 Limpiando volumes huérfanos..."
docker volume prune -f

# Limpiar networks huérfanas
echo "🌐 Limpiando networks huérfanas..."
docker network prune -f

# Descargar módulos del kernel
echo "🔧 Descargando módulos del kernel..."
cd kernel
if lsmod | grep -q "ram_202010040"; then
    sudo rmmod ram_202010040 || echo "⚠️ No se pudo descargar módulo RAM"
fi

if lsmod | grep -q "cpu_202010040"; then
    sudo rmmod cpu_202010040 || echo "⚠️ No se pudo descargar módulo CPU"
fi

# Limpiar archivos compilados del kernel
echo "🗑️ Limpiando archivos compilados..."
make clean 2>/dev/null || echo "⚠️ No hay Makefile o archivos para limpiar"

cd ..

# Mostrar resumen
echo ""
echo "✅ Limpieza completada!"
echo ""
echo "📊 Resumen de la limpieza:"
echo "   ✓ Servicios Docker Compose detenidos"
echo "   ✓ Contenedores de estrés eliminados"
echo "   ✓ Imágenes del proyecto eliminadas"
echo "   ✓ Volumes huérfanos limpiados"
echo "   ✓ Networks huérfanas limpiadas"
echo "   ✓ Módulos del kernel descargados"
echo "   ✓ Archivos compilados limpiados"
echo ""
echo "🔍 Para verificar que todo está limpio:"
echo "   docker ps -a"
echo "   docker images"
echo "   lsmod | grep 202010040"
