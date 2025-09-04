#!/bin/bash
# Script para desplegar la aplicación completa

set -e

echo "🚀 Desplegando aplicación de monitoreo..."

# Ir al directorio raíz del proyecto
cd ..

# Construir imágenes
echo "🔨 Construyendo imágenes Docker..."
docker-compose build

# Subir imágenes a DockerHub (opcional)
read -p "¿Deseas subir las imágenes a DockerHub? (y/n): " upload_choice
if [[ $upload_choice == "y" || $upload_choice == "Y" ]]; then
    read -p "Ingresa tu usuario de DockerHub: " dockerhub_user
    
    # Etiquetar imágenes (CORREGIDO - usando los nombres reales)
    docker tag proyecto1_fase1-monitor_api $dockerhub_user/monitor-api:latest
    docker tag proyecto1_fase1-monitor_agente $dockerhub_user/monitor-agente:latest
    docker tag proyecto1_fase1-monitor_frontend $dockerhub_user/monitor-frontend:latest
    
    # Subir imágenes
    docker push $dockerhub_user/monitor-api:latest
    docker push $dockerhub_user/monitor-agente:latest
    docker push $dockerhub_user/monitor-frontend:latest
    
    echo "✅ Imágenes subidas a DockerHub"
fi

# Levantar servicios
echo "🆙 Levantando servicios..."
docker-compose up -d

# Esperar a que los servicios estén listos
echo "⏳ Esperando a que los servicios estén listos..."
sleep 30

# Verificar estado de servicios
echo "📋 Estado de los servicios:"
docker-compose ps

# Mostrar información de acceso
echo ""
echo "🎉 ¡Aplicación desplegada exitosamente!"
echo ""
echo "📊 Accesos:"
echo "   Frontend: http://localhost"
echo "   API: http://localhost:3000"
echo "   Agente: http://localhost:8080"
echo "   Base de datos: localhost:3306"
echo ""
echo "📝 Para ver logs:"
echo "   docker-compose logs -f [servicio]"
echo "🔄 Para reiniciar:"
echo "   docker-compose restart"
echo "🛑 Para detener:"
echo "   docker-compose down"