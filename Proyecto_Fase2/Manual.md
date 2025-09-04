### Universidad de San Carlos de Guatemala
### Facultad de Ingeniería
### Escuela de Ciencias y Sistemas
### Sistemas Operativos 1

---

<div style="text-align: center; position: absolute; right: 50px; top: 2px;">
    <img src="manual/fiusac-logo.png" alt="Portada" width="300px" height="300px">
</div>

# Manual Técnico Proyecto Fase 2
## Monitoreo Cloud de VMs

## Descripción del Proyecto

Este proyecto implementa un sistema completo de monitoreo en la nube para máquinas virtuales, utilizando tecnologías de contenedorización, orquestación con Kubernetes, servicios en la nube y pruebas de carga. El sistema evalúa la capacidad de una VM para soportar cargas de trabajo específicas mediante la generación de tráfico controlado y el análisis de métricas en tiempo real.

## Objetivos

- Desplegar un clúster de Kubernetes para gestionar y escalar servicios de monitoreo
- Implementar enfoque Serverless con Cloud Run para aplicaciones ligeras
- Utilizar Cloud SQL para almacenamiento seguro y escalable de métricas
- Configurar balanceadores de carga para distribución eficiente del tráfico
- Desarrollar una aplicación web interactiva para visualización de estadísticas

## Arquitectura del Sistema

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                KUBERNETES CLUSTER                               │
│                                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐            │
│  │    INGRESS      │    │  TRAFFIC SPLIT  │    │  LOAD BALANCER  │            │
│  │                 │◄──►│   (50%/50%)     │◄──►│                 │            │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘            │
│           │                       │                       │                    │
│           ▼                       ▼                       ▼                    │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐            │
│  │  API Python     │    │   API NodeJS    │    │ WebSocket API   │            │
│  │   (Ruta 1)      │    │   (Ruta 2)      │    │   (NodeJS)      │            │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘            │
│           │                       │                       │                    │
│           └───────────────────────┼───────────────────────┘                    │
│                                   ▼                                            │
│                          ┌─────────────────┐                                   │
│                          │   CLOUD SQL     │                                   │
│                          │    (MySQL)      │                                   │
│                          └─────────────────┘                                   │
└─────────────────────────────────────────────────────────────────────────────────┘
                                   ▲
                                   │
                          ┌─────────────────┐
                          │   CLOUD RUN     │
                          │  Frontend React │
                          │   (Dashboard)   │
                          └─────────────────┘
                                   ▲
                                   │ (WebSocket)
                          ┌─────────────────┐
                          │     LOCUST      │
                          │ Traffic Gen.    │
                          │   (Local)       │
                          └─────────────────┘
                                   ▲
                                   │
                          ┌─────────────────┐
                          │  VM TARGET      │
                          │ Agente Go +     │
                          │ Kernel Modules  │
                          │ + Stress Tests  │
                          └─────────────────┘
```

## Tecnologías Utilizadas

### Infraestructura
- **Orquestación**: Kubernetes (GKE)
- **Serverless**: Google Cloud Run
- **Base de Datos**: Google Cloud SQL (MySQL)
- **Balanceador**: Google Cloud Load Balancer
- **Contenedores**: Docker + DockerHub

### Desarrollo
- **Módulos del Kernel**: C
- **Agente de Monitoreo**: Go (Golang) - Dockerizado
- **APIs**: Python (Ruta 1), Node.js (Ruta 2), Node.js + Socket.IO (WebSocket)
- **Frontend**: React + Socket.IO Client
- **Generador de Tráfico**: Locust (Python)
- **Pruebas de Estrés**: Docker stress container

## Estructura del Proyecto

```
Proyecto_Fase2/
├── kubernetes/                 # Configuraciones K8s
│   ├── namespace.yaml
│   ├── ingress.yaml
│   ├── traffic-split.yaml
│   └── services/
├── apis/
│   ├── python-api/            # API Ruta 1 (Python)
│   │   ├── Dockerfile
│   │   └── app.py
│   ├── nodejs-api/            # API Ruta 2 (NodeJS)
│   │   ├── Dockerfile
│   │   └── server.js
│   └── websocket-api/         # API WebSocket (NodeJS)
│       ├── Dockerfile
│       └── socket-server.js
├── agente/                    # Agente de monitoreo en Go
│   ├── Dockerfile
│   └── main.go
├── frontend/                  # Frontend React + Socket.IO
│   ├── Dockerfile
│   ├── package.json
│   └── src/
├── kernel/                    # Módulos del kernel
│   ├── Makefile
│   ├── cpu_<carnet>.c
│   ├── ram_<carnet>.c
│   └── procesos_<carnet>.c    # NUEVO: Módulo de procesos
├── locust/                    # Generador de tráfico
│   ├── traffic_generator.py
│   └── config.py
├── cloud-sql/                 # Scripts de base de datos
│   └── init-db.sql
└── scripts/                   # Scripts de automatización
    ├── deploy-k8s.sh
    ├── setup-gcp.sh
    └── cleanup.sh
```

## Nuevas Funcionalidades - Fase 2

### 1. Módulo de Procesos del Kernel
**Archivo**: `procesos_<carnet>.c`

Nuevo módulo que recolecta información sobre procesos del sistema:

```json
{
  "procesos_corriendo": 123,
  "total_procesos": 233,
  "procesos_durmiendo": 65,
  "procesos_zombie": 65,
  "procesos_parados": 65
}
```

**Compilación e instalación**:
```bash
cd kernel
make
sudo insmod procesos_<carnet>.ko
cat /proc/procesos_<carnet>
```

### 2. Generador de Tráfico con Locust

**Configuración**: `locust/traffic_generator.py`

**Fase 1 - Generación de datos**:
- 300 usuarios concurrentes
- Peticiones cada 1-2 segundos
- Nuevos usuarios cada segundo
- Duración mínima: 3 minutos
- Genera ~2000 registros JSON

**Fase 2 - Envío al Traffic Split**:
- 150 usuarios concurrentes
- Peticiones cada 1-4 segundos
- Envío a endpoints de Kubernetes

**Ejemplo de uso**:
```bash
# Instalar Locust
pip install locust

# Ejecutar generador
cd locust
locust -f traffic_generator.py --host=http://VM-IP:8080
```

### 3. Kubernetes con Traffic Split

**Namespace**: `so1_fase2`

**Configuración del Ingress**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: monitor-ingress
  namespace: so1_fase2
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-weight: "50"
spec:
  rules:
  - host: monitor.example.com
    http:
      paths:
      - path: /api/python
        pathType: Prefix
        backend:
          service:
            name: python-api-service
            port:
              number: 5000
      - path: /api/nodejs
        pathType: Prefix
        backend:
          service:
            name: nodejs-api-service
            port:
              number: 3000
```

**Despliegue**:
```bash
# Crear namespace
kubectl create namespace so1_fase2

# Aplicar configuraciones
kubectl apply -f kubernetes/ -n so1_fase2

# Verificar pods
kubectl get pods -n so1_fase2
```

### 4. APIs Multi-Lenguaje

#### API Python (Ruta 1)
**Puerto**: 5000
**Función**: Recibe métricas y las almacena en Cloud SQL
```python
# Ejemplo de endpoint
@app.route('/metrics', methods=['POST'])
def save_metrics():
    data = request.json
    data['api'] = 'Python'
    # Guardar en Cloud SQL
    return jsonify({"status": "saved"})
```

#### API NodeJS (Ruta 2)
**Puerto**: 3000
**Función**: Funcionalidad idéntica a Python API
```javascript
// Ejemplo de endpoint
app.post('/metrics', (req, res) => {
    const data = req.body;
    data.api = 'NodeJS';
    // Guardar en Cloud SQL
    res.json({"status": "saved"});
});
```

#### API WebSocket (NodeJS + Socket.IO)
**Puerto**: 8080
**Función**: Comunicación en tiempo real con frontend
```javascript
io.on('connection', (socket) => {
    console.log('Cliente conectado');
    
    // Enviar métricas en tiempo real
    setInterval(() => {
        socket.emit('metrics', getCurrentMetrics());
    }, 1000);
});
```

### 5. Frontend con WebSockets

**Tecnologías**: React + Socket.IO Client + Chart.js

**Características**:
- Dashboard en tiempo real
- Gráficas de CPU y RAM actualizadas en vivo
- Tabla de información de procesos
- Conexión WebSocket para datos en tiempo real

**Componentes principales**:
```jsx
import io from 'socket.io-client';
import { Line } from 'react-chartjs-2';

function Dashboard() {
    const [metrics, setMetrics] = useState({});
    
    useEffect(() => {
        const socket = io('ws://websocket-api-url');
        
        socket.on('metrics', (data) => {
            setMetrics(data);
        });
        
        return () => socket.disconnect();
    }, []);
    
    return (
        <div>
            <CPUChart data={metrics.cpu} />
            <RAMChart data={metrics.ram} />
            <ProcessTable data={metrics.processes} />
        </div>
    );
}
```

### 6. Cloud SQL Configuration

**Base de datos**: MySQL 8.0
**Instancia**: Cloud SQL (Google Cloud)

**Esquema de base de datos**:
```sql
CREATE DATABASE sistema_monitoreo_fase2;

CREATE TABLE metricas (
    id INT AUTO_INCREMENT PRIMARY KEY,
    total_ram BIGINT,
    ram_libre BIGINT,
    uso_ram BIGINT,
    porcentaje_ram DECIMAL(5,2),
    porcentaje_cpu_uso DECIMAL(5,2),
    porcentaje_cpu_libre DECIMAL(5,2),
    procesos_corriendo INT,
    total_procesos INT,
    procesos_durmiendo INT,
    procesos_zombie INT,
    procesos_parados INT,
    hora TIMESTAMP,
    api VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 7. Cloud Run Deployment

**Despliegue del Frontend**:
```bash
# Build de la imagen
docker build -t gcr.io/PROJECT-ID/monitor-frontend .

# Push a Google Container Registry
docker push gcr.io/PROJECT-ID/monitor-frontend

# Deploy a Cloud Run
gcloud run deploy monitor-frontend \
  --image gcr.io/PROJECT-ID/monitor-frontend \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

## Instalación y Despliegue

### 1. Prerrequisitos

**Sistema Local**:
- Ubuntu 22.04 o 24.xx
- Docker y Docker Compose
- kubectl
- Locust (`pip install locust`)

**Google Cloud**:
- Proyecto GCP activo
- GKE cluster
- Cloud SQL instance
- Cloud Run habilitado

### 2. Configuración de Google Cloud

```bash
# Instalar gcloud CLI
curl https://sdk.cloud.google.com | bash

# Configurar proyecto
gcloud config set project YOUR-PROJECT-ID

# Crear cluster GKE
gcloud container clusters create monitor-cluster \
  --zone us-central1-a \
  --num-nodes 3

# Configurar kubectl
gcloud container clusters get-credentials monitor-cluster --zone us-central1-a
```

### 3. Despliegue de Módulos del Kernel

```bash
cd kernel
make clean && make

# Cargar módulos
sudo insmod cpu_<carnet>.ko
sudo insmod ram_<carnet>.ko
sudo insmod procesos_<carnet>.ko

# Verificar
lsmod | grep <carnet>
cat /proc/procesos_<carnet>
```

### 4. Construcción y Publicación de Imágenes

```bash
# Build de todas las imágenes
docker build -t <dockerhub-user>/python-api:latest apis/python-api/
docker build -t <dockerhub-user>/nodejs-api:latest apis/nodejs-api/
docker build -t <dockerhub-user>/websocket-api:latest apis/websocket-api/
docker build -t <dockerhub-user>/monitor-agente:latest agente/
docker build -t <dockerhub-user>/monitor-frontend:latest frontend/

# Push a DockerHub
docker push <dockerhub-user>/python-api:latest
docker push <dockerhub-user>/nodejs-api:latest
docker push <dockerhub-user>/websocket-api:latest
docker push <dockerhub-user>/monitor-agente:latest
docker push <dockerhub-user>/monitor-frontend:latest
```

### 5. Despliegue en Kubernetes

```bash
# Crear namespace
kubectl create namespace so1_fase2

# Aplicar configuraciones
kubectl apply -f kubernetes/ -n so1_fase2

# Verificar estado
kubectl get pods -n so1_fase2
kubectl get services -n so1_fase2
kubectl get ingress -n so1_fase2
```

### 6. Configuración de Cloud SQL

```bash
# Crear instancia
gcloud sql instances create monitor-db \
  --database-version=MYSQL_8_0 \
  --cpu=1 \
  --memory=3840MB \
  --region=us-central1

# Crear base de datos
gcloud sql databases create sistema_monitoreo_fase2 --instance=monitor-db

# Crear usuario
gcloud sql users create monitor_user \
  --instance=monitor-db \
  --password=SecurePassword123
```

### 7. Despliegue de Frontend en Cloud Run

```bash
# Deploy
gcloud run deploy monitor-frontend \
  --image <dockerhub-user>/monitor-frontend:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --port 3000
```

## Pruebas de Carga con Locust

### Configuración de Stress Testing

**Container de estrés**: `polinux/stress`

```bash
# Ejecutar contenedor de estrés en la VM
docker run --rm -d --name stress-test \
  polinux/stress \
  stress --cpu 2 --io 1 --vm 1 --vm-bytes 128M --timeout 300s
```

### Ejecución de Locust

**Fase 1 - Generación de datos**:
```bash
cd locust
locust -f traffic_generator.py \
  --host=http://VM-IP:8080 \
  -u 300 \
  -r 1 \
  -t 180s \
  --headless
```

**Fase 2 - Envío al cluster**:
```bash
locust -f traffic_generator.py \
  --host=http://INGRESS-IP \
  -u 150 \
  -r 1 \
  -t 300s \
  --headless
```

## Acceso a los Servicios

### URLs de Acceso

- **Frontend (Cloud Run)**: https://monitor-frontend-xxx-uc.a.run.app
- **Ingress (Kubernetes)**: http://EXTERNAL-IP
- **VM Agente**: http://VM-IP:8080
- **Cloud SQL**: Acceso via APIs

### Endpoints de APIs

**Python API (Ruta 1)**:
- `POST /api/python/metrics` - Guardar métricas

**NodeJS API (Ruta 2)**:
- `POST /api/nodejs/metrics` - Guardar métricas

**WebSocket API**:
- `WS /socket.io` - Conexión en tiempo real

## Monitoreo y Debugging

### Comandos Kubernetes Útiles

```bash
# Ver logs de pods
kubectl logs -f POD-NAME -n so1_fase2

# Describir recursos
kubectl describe pod POD-NAME -n so1_fase2
kubectl describe service SERVICE-NAME -n so1_fase2

# Port-forward para debugging
kubectl port-forward service/python-api-service 5000:5000 -n so1_fase2

# Ver métricas del cluster
kubectl top nodes
kubectl top pods -n so1_fase2
```

### Verificación de Traffic Split

```bash
# Verificar distribución de tráfico
kubectl get ingress -n so1_fase2 -o yaml

# Monitorear logs de ambas APIs
kubectl logs -f deployment/python-api -n so1_fase2 &
kubectl logs -f deployment/nodejs-api -n so1_fase2 &
```

### Debugging de Cloud SQL

```bash
# Conectar a Cloud SQL
gcloud sql connect monitor-db --user=monitor_user

# Ver conexiones activas
SHOW PROCESSLIST;

# Ver métricas guardadas
SELECT api, COUNT(*) FROM metricas GROUP BY api;
```

## Limpieza del Sistema

### Limpiar Kubernetes

```bash
# Eliminar recursos del namespace
kubectl delete namespace so1_fase2

# Limpiar cluster (opcional)
gcloud container clusters delete monitor-cluster --zone us-central1-a
```

### Limpiar Cloud Resources

```bash
# Eliminar instancia Cloud SQL
gcloud sql instances delete monitor-db

# Eliminar servicio Cloud Run
gcloud run services delete monitor-frontend --region us-central1
```

### Limpiar Módulos del Kernel

```bash
sudo rmmod procesos_<carnet>
sudo rmmod cpu_<carnet>
sudo rmmod ram_<carnet>

cd kernel
make clean
```

## Formato de Datos

### JSON de Métricas Completo

```json
{
  "total_ram": 2072,
  "ram_libre": 1110552576,
  "uso_ram": 442,
  "porcentaje_ram": 22,
  "porcentaje_cpu_uso": 22,
  "porcentaje_cpu_libre": 88,
  "procesos_corriendo": 123,
  "total_procesos": 233,
  "procesos_durmiendo": 65,
  "procesos_zombie": 65,
  "procesos_parados": 65,
  "hora": "2025-06-17 02:21:54",
  "api": "Python"  // o "NodeJS"
}
```

## Consideraciones de Rendimiento

### Optimizaciones Kubernetes

- **Resource Limits**: Configurar limits y requests para todos los pods
- **Horizontal Pod Autoscaler**: Auto-escalado basado en CPU/memoria
- **Persistent Volumes**: Para datos que requieren persistencia

### Optimizaciones Base de Datos

- **Índices**: Crear índices en columnas frecuentemente consultadas
- **Connection Pooling**: Configurar pool de conexiones en las APIs
- **Particionamiento**: Para tablas con gran volumen de datos

### Optimizaciones Frontend

- **Lazy Loading**: Cargar componentes bajo demanda
- **Memoización**: React.memo para componentes pesados
- **WebSocket Throttling**: Limitar frecuencia de actualizaciones

## Solución de Problemas Comunes

### Error: Pods en estado Pending

```bash
# Verificar recursos del cluster
kubectl describe nodes

# Verificar eventos
kubectl get events -n so1_fase2 --sort-by='.lastTimestamp'
```

### Error: No se puede conectar a Cloud SQL

```bash
# Verificar Cloud SQL Proxy
cloud_sql_proxy -instances=PROJECT:REGION:INSTANCE=tcp:3306

# Verificar credenciales
kubectl get secret cloudsql-db-credentials -n so1_fase2 -o yaml
```

### Error: Traffic Split no funciona

```bash
# Verificar configuración del Ingress
kubectl get ingress -n so1_fase2 -o yaml

# Verificar servicios backend
kubectl get endpoints -n so1_fase2
```

### Error: WebSocket no conecta

```bash
# Verificar CORS en WebSocket API
# Verificar puerto expuesto en Cloud Run
# Revisar logs del frontend y WebSocket API
```

## Mejores Prácticas

### Seguridad

- Usar Service Accounts para acceso a Cloud SQL
- Configurar Network Policies en Kubernetes
- Implementar HTTPS en todos los endpoints
- Rotar credenciales regularmente

### Desarrollo

- Usar ConfigMaps para configuración
- Implementar Health Checks en todos los servicios
- Configurar Liveness y Readiness Probes
- Usar versioning en imágenes Docker

### Monitoreo

- Configurar Google Cloud Monitoring
- Implementar logging estructurado
- Configurar alertas para métricas críticas
- Usar Grafana para visualización avanzada

---

## Notas Importantes

- **Restricción importante**: Solo Locust debe ejecutarse localmente, todo lo demás debe estar en la nube
- **DockerHub**: Todas las imágenes deben estar publicadas y ser consumidas desde DockerHub
- **Tiempo de calificación**: 15 minutos, tener todo preparado
- **Autenticidad**: Prepararse para explicaciones y modificaciones de código durante la calificación