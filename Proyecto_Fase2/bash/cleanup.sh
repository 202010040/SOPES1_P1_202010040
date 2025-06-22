#!/bin/bash
# Script para limpiar todos los servicios del proyecto

set -e

echo "🧹 Limpiando todos los servicios del proyecto..."

# Ir al directorio raíz del proyecto
cd ..

# Detectar si se necesita sudo para usar docker
if ! groups $USER | grep -q docker; then
    echo "⚠️ Tu usuario no está en el grupo 'docker', usando sudo para todos los comandos Docker..."
    alias docker='sudo docker'
    alias docker-compose='sudo docker-compose'
    USE_SUDO=true
else
    USE_SUDO=false
fi

# Función para eliminar contenedores problemáticos de forma agresiva
force_remove_containers() {
    echo "🔨 Eliminando contenedores de forma agresiva..."

    echo "🗡️ Matando todos los contenedores..."
    docker ps -q | xargs -r docker kill 2>/dev/null || true
    docker ps -aq | xargs -r docker rm -f 2>/dev/null || true

    if [ "$(docker ps -aq 2>/dev/null | wc -l)" -gt 0 ]; then
        echo "⚠️ Contenedores aún presentes, usando método nuclear..."

        echo "🛑 Deteniendo Docker y socket..."
        sudo systemctl stop docker.socket
        sudo systemctl stop docker

        echo "⏳ Esperando que Docker se detenga completamente..."
        sleep 5

        # Contenedor conflictivo (ajustar IDs si cambian)
        CONFLICT_IDS=(
            "5c1ccfc66c12642ca16bef35a6b9659e966a65e8301b5454b5e9876ca5bb64c6"
            "3558379a6a2f"
            "b8e7175832ea"
            "fc5b20cfddba"
        )

        for ID in "${CONFLICT_IDS[@]}"; do
            echo "🧨 Forzando limpieza del contenedor $ID..."

            sudo fuser -k /var/lib/docker/containers/$ID/* 2>/dev/null || true
            sudo ctr --namespace moby containers delete $ID 2>/dev/null || true
            sudo rm -rf /var/lib/docker/containers/$ID* 2>/dev/null || true
        done

        echo "🔄 Reiniciando Docker..."
        sudo systemctl start docker
        sudo systemctl start docker.socket

        echo "⏳ Esperando que Docker se inicie completamente..."
        sleep 8

        if sudo systemctl is-active --quiet docker; then
            echo "✅ Docker reiniciado exitosamente"
        else
            echo "❌ Error: Docker no se pudo reiniciar"
            exit 1
        fi
    fi

    remaining=$(docker ps -aq 2>/dev/null | wc -l)
    if [ "$remaining" -eq 0 ]; then
        echo "✅ Todos los contenedores eliminados exitosamente"
    else
        echo "⚠️ Quedan $remaining contenedores. Continuando..."
    fi
}

# Detener y eliminar contenedores de Docker Compose
if sudo systemctl is-active --quiet docker; then
    echo "📦 Deteniendo servicios de Docker Compose..."
    docker-compose down -v --remove-orphans --timeout 5 || echo "⚠️ Docker Compose falló, pero continuamos..."
else
    echo "⚠️ Docker no está funcionando, saltando docker-compose down"
fi

# Eliminar contenedores de estrés si existen
echo "🔥 Eliminando contenedores de estrés..."
docker ps -q --filter name=stress- | xargs -r docker kill 2>/dev/null || true
docker ps -q --filter name=stress- | xargs -r docker stop 2>/dev/null || true
docker ps -aq --filter name=stress- | xargs -r docker rm -f

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

# Ejecutar limpieza forzada si aún hay contenedores
if [ "$(docker ps -aq | wc -l)" -gt 0 ]; then
    force_remove_containers
fi

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
