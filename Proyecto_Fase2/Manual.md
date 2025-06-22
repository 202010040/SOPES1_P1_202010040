### Universidad de San Carlos de Guatemala
### Facultad de Ingenieria
### Ingenieria en Ciencias y Sistemas
### Sistemas operativos 1

---

<div style="text-align: center; position: absolute; right: 50px; top: 2px;">
    <img src="manual/fiusac-logo.png" alt="Portada" width="300px" height="300px">
</div>

# Manual Técnico Proyecto 1
 
## Descripción del Proyecto

El objetivo de este proyecto es aplicar todos los conocimientos adquiridos en la unidad 1, con la implementación de un gestor de contenedores mediante el uso de scripts, módulos de kernel, lenguajes de programación y la herramienta para la creación y manejo de contenedores más popular, Docker. Con la ayuda de este gestor de contenedores se podrá observar de manera más detallada los recursos y la representación de los contenedores a nivel de procesos de Linux y como de manera flexible pueden ser creados, destruidos y conectados por otros servicios.

## Arquitectura del Sistema

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │    API NodeJS   │    │  Base de Datos  │
│   (React)       │◄──►│   (Puerto 3001) │◄──►│    (MySQL)      │
│  (Puerto 3000)  │    │                 │    │  (Puerto 3306)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                       ▲
         │                       │
         └───────────────────────┼─────────────────────────┘
                                 ▼
                    ┌─────────────────┐
                    │ Agente Go       │
                    │ (Puerto 8080)   │
                    └─────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Módulos Kernel  │
                    │ - ram_202010040 │
                    │ - cpu_202010040 │
                    └─────────────────┘
```

## Tecnologías Utilizadas

- **Módulos del Kernel**: C
- **Agente de Monitoreo**: Go (Golang)
- **API**: Node.js con Express
- **Frontend**: React
- **Base de Datos**: MySQL 8.0
- **Containerización**: Docker & Docker Compose
- **Scripts de Automatización**: Bash

## Estructura del Proyecto

```
Proyecto1_Fase1/
├── agente/                 # Agente de monitoreo en Go
│   ├── Dockerfile
│   └── main.go
├── api/                    # API en Node.js
│   ├── Dockerfile
│   ├── package.json
│   └── src/
├── bd/                     # Scripts de base de datos
│   └── init-db.sql
├── bash/                   # Scripts de automatización
│   ├── cleanup.sh
│   ├── deploy_app.sh
│   ├── install_modules.sh
│   └── stress_containers.sh
├── frontend/               # Frontend en React
│   ├── Dockerfile
│   ├── package.json
│   └── src/
├── kernel/                 # Módulos del kernel
│   ├── Makefile
│   ├── cpu_202010040.c
│   └── ram_202010040.c
└── docker-compose.yml      # Configuración de servicios
```

## Prerrequisitos

- **Sistema Operativo**: Ubuntu 20.04 o 22.04
- **Docker**: Versión 20.10 o superior
- **Docker Compose**: Versión 2.0 o superior
- **Herramientas de desarrollo del kernel**:
  ```bash
  sudo apt-get install build-essential linux-headers-$(uname -r) make gcc
  ```

## Instalación y Despliegue

### 1. Clonar el Repositorio
```bash
git clone <url-del-repositorio>
cd Proyecto1_Fase1
```

### 2. Dar Permisos de Ejecución a los Scripts
```bash
cd bash
chmod +x *.sh
```

### 3. Instalación de Módulos del Kernel
```bash
./install_modules.sh
```
Este script:
- Instala dependencias del sistema
- Compila los módulos del kernel
- Carga los módulos ram_202010040 y cpu_202010040
- Verifica la instalación correcta

### 4. Despliegue de la Aplicación
```bash
./deploy_app.sh
```
Este script:
- Construye las imágenes Docker
- Opcionalmente sube las imágenes a DockerHub
- Levanta todos los servicios con Docker Compose
- Verifica el estado de los servicios

### 5. Pruebas de Estrés (Opcional)
```bash
./stress_containers.sh
```
Crea 10 contenedores (5 para CPU, 5 para RAM) que generan carga en el sistema para probar el monitoreo.

## 🖥️ Acceso a los Servicios

Una vez desplegado el sistema, los servicios estarán disponibles en:

- **Frontend**: http://localhost:3000
- **API**: http://localhost:3001
- **Agente de Monitoreo**: http://localhost:8080
- **Base de Datos MySQL**: localhost:3306

### Credenciales de Base de Datos
- **Usuario**: user_monitoreo
- **Contraseña**: Ingenieria2025.
- **Base de Datos**: sistema_monitoreo

## Módulos del Kernel

### Módulo de RAM (ram_202010040)
- **Ubicación**: `/proc/ram_202010040`
- **Función**: Obtiene información de memoria RAM del sistema
- **Librerías utilizadas**: `<sys/sysinfo.h>`, `<linux/mm.h>`
- **Formato de salida**: JSON con información de memoria total, libre y utilizada

**Ejemplo de salida**:
```json
{
  "total": 8192,
  "libre": 2048,
  "uso": 6144,
  "porcentaje": 75.0
}
```

### Módulo de CPU (cpu_202010040)
- **Ubicación**: `/proc/cpu_202010040`
- **Función**: Obtiene información de utilización de CPU
- **Librerías utilizadas**: `<linux/sched.h>`, `<linux/sched/signal.h>`
- **Formato de salida**: JSON con información de procesos y utilización

**Ejemplo de salida**:
```json
{
  "porcentajeUso": 45.2
}
```

## Comandos Útiles

### Ver Estado de los Servicios
```bash
docker-compose ps
```

### Ver Logs de un Servicio Específico
```bash
docker-compose logs -f [nombre_servicio]
# Ejemplo: docker-compose logs -f monitor_api
```

### Reiniciar un Servicio
```bash
docker-compose restart [nombre_servicio]
```

### Verificar Módulos del Kernel Cargados
```bash
lsmod | grep 202010040
```

### Ver Contenido de los Módulos
```bash
cat /proc/ram_202010040
cat /proc/cpu_202010040
```

### Ver Contenedores de Estrés
```bash
docker ps --filter name=stress-
```

## Limpieza del Sistema

Para limpiar completamente el sistema y liberar recursos:

```bash
./cleanup.sh
```

Este script:
- Detiene y elimina todos los contenedores
- Elimina imágenes del proyecto
- Limpia volumes y networks huérfanos
- Descarga los módulos del kernel
- Limpia archivos compilados

## Docker Compose

### Servicios Definidos

1. **monitor_db**: Base de datos MySQL con persistencia
2. **monitor_api**: API en Node.js para comunicación con BD
3. **monitor_agente**: Agente de monitoreo en Go
4. **monitor_frontend**: Frontend web en React

### Volúmenes
- `db_data`: Persistencia de datos de MySQL

### Redes
- `monitor-network`: Red bridge para comunicación entre servicios

## Solución de Problemas

### Error: Módulos del Kernel No Cargan
```bash
# Verificar headers del kernel
sudo apt-get install linux-headers-$(uname -r)

# Recompilar módulos
cd kernel
make clean
make
sudo insmod ram_202010040.ko
sudo insmod cpu_202010040.ko
```

### Error: Puerto en Uso
```bash
# Verificar puertos ocupados
sudo netstat -tulpn | grep -E ':300[0-1]|:3306|:8080'

# Detener servicios que ocupan puertos
sudo docker-compose down
```

### Error: Permisos Insuficientes
```bash
# Dar permisos a scripts
chmod +x bash/*.sh

# Ejecutar con sudo si es necesario
sudo ./install_modules.sh
```

### Error: Imágenes Docker No Encontradas
```bash
# Reconstruir imágenes
docker-compose build --no-cache

# Verificar nombres de imágenes
docker images | grep proyecto1
```

## Funcionalidades del Sistema

### Frontend
- Dashboard en tiempo real
- Gráficas de utilización de CPU y RAM
- Actualización automática de métricas
- Interfaz responsive

### API
- Endpoints RESTful para métricas
- Almacenamiento de datos históricos
- Validación de datos
- Manejo de errores

### Agente
- Recolección de datos mediante goroutines
- Comunicación con módulos del kernel
- Envío periódico de métricas a la API
- Manejo de concurrencia con channels

## Despliegue en DockerHub

Para subir las imágenes a DockerHub:

1. **Modificar el script deploy_app.sh** con los nombres correctos de imágenes:
   ```bash
   docker tag proyecto1_fase1-monitor_api tu_usuario/monitor-api:latest
   docker tag proyecto1_fase1-monitor_agente tu_usuario/monitor-agente:latest
   docker tag proyecto1_fase1-monitor_frontend tu_usuario/monitor-frontend:latest
   ```

2. **Ejecutar el despliegue** y seleccionar "y" cuando pregunte por DockerHub

## Notas Importantes

- En caso no se pueda eliminar los contenedores, debe modificarse el codigo para utilizar el metodo nuclear

