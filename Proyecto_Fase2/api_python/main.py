
import os
import sys
import time
import signal
import asyncio
import logging
from typing import Optional
from contextlib import asynccontextmanager

import uvicorn
import aiomysql
from fastapi import FastAPI, HTTPException, status
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

# Configurar logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Modelos de datos
class MetricasRequest(BaseModel):
    total_ram: float
    ram_libre: float
    uso_ram: float
    porcentaje_ram: float
    porcentaje_cpu_uso: float
    porcentaje_cpu_libre: float
    procesos_corriendo: int
    total_procesos: int
    procesos_durmiendo: int
    procesos_zombie: int
    procesos_parados: int
    hora: str

class MetricasResponse(BaseModel):
    message: str
    id: int

class APIResponse(BaseModel):
    message: str
    endpoints: list[str]

# Variables globales
pool: Optional[aiomysql.Pool] = None

# Configuración de la base de datos
DB_CONFIG = {
    'host': os.getenv('DB_HOST', '0.0.0.0'),
    'port': int(os.getenv('DB_PORT', 3306)),
    'user': os.getenv('DB_USER', 'user_monitoreo'),
    'password': os.getenv('DB_PASSWORD', 'Ingenieria2025.'),
    'db': os.getenv('DB_NAME', 'sistema_monitoreo'),
    'minsize': 5,
    'maxsize': 10,
    'autocommit': True
}

async def init_database() -> None:
    """Inicializar conexión a la base de datos con reintentos"""
    global pool
    retries = 5
    
    while retries > 0:
        try:
            pool = await aiomysql.create_pool(**DB_CONFIG)
            
            # Probar la conexión
            async with pool.acquire() as conn:
                await conn.ping()
            
            logger.info('Base de datos inicializada correctamente')
            return
            
        except Exception as error:
            logger.error(f'Error al conectar a la base de datos. Intentos restantes: {retries - 1}. Error: {error}')
            retries -= 1
            
            if retries == 0:
                logger.error('No se pudo conectar a la base de datos después de varios intentos')
                sys.exit(1)
            
            # Esperar 5 segundos antes del siguiente intento
            await asyncio.sleep(5)

async def close_database() -> None:
    """Cerrar conexión a la base de datos"""
    global pool
    if pool:
        pool.close()
        await pool.wait_closed()
        logger.info('Conexión a la base de datos cerrada')

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Gestión del ciclo de vida de la aplicación"""
    # Startup
    await init_database()
    yield
    # Shutdown
    await close_database()

# Crear aplicación FastAPI
app = FastAPI(
    title="API de Monitoreo de Servicios Linux",
    description="API para monitoreo de métricas del sistema",
    version="1.0.0",
    lifespan=lifespan
)

# Configurar CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=[os.getenv('CORS_ORIGIN', 'http://localhost:3000')],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.post("/api/metrics", response_model=MetricasResponse, status_code=status.HTTP_201_CREATED)
async def insertar_metricas(metricas: MetricasRequest):
    """Insertar métricas del sistema"""
    try:
        query = """
            INSERT INTO metricas_sistema (
                memoria_total, 
                memoria_libre, 
                memoria_usada, 
                porcentaje_ram,
                porcentaje_cpu_uso,
                porcentaje_cpu_libre,
                procesos_corriendo,
                total_procesos,
                procesos_durmiendo,
                procesos_zombie,
                procesos_parados,
                hora,
                api
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        """
        
        values = (
            metricas.total_ram,
            metricas.ram_libre,
            metricas.uso_ram,
            metricas.porcentaje_ram,
            metricas.porcentaje_cpu_uso,
            metricas.porcentaje_cpu_libre,
            metricas.procesos_corriendo,
            metricas.total_procesos,
            metricas.procesos_durmiendo,
            metricas.procesos_zombie,
            metricas.procesos_parados,
            metricas.hora,
            'Python'
        )
        
        async with pool.acquire() as conn:
            async with conn.cursor() as cursor:
                await cursor.execute(query, values)
                insert_id = cursor.lastrowid
        
        return MetricasResponse(
            message="Métricas del sistema guardadas exitosamente",
            id=insert_id
        )
        
    except Exception as error:
        logger.error(f'Error al guardar métricas del sistema: {error}')
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Error interno del servidor"
        )

@app.get("/", response_model=APIResponse)
async def root():
    """Ruta raíz con información de la API"""
    return APIResponse(
        message="API de Monitoreo de Servicios Linux",
        endpoints=["POST /api/metrics - Insertar métricas del sistema"]
    )

@app.get("/health")
async def health_check():
    """Endpoint de verificación de salud"""
    try:
        async with pool.acquire() as conn:
            await conn.ping()
        return {"status": "healthy", "database": "connected"}
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=f"Database connection failed: {str(e)}"
        )

def signal_handler(signum, frame):
    """Manejador de señales para shutdown graceful"""
    logger.info(f'Recibida señal {signum}, cerrando servidor...')
    sys.exit(0)

if __name__ == "__main__":
    # Configurar manejadores de señales
    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)
    
    # Configuración del servidor
    PORT = int(os.getenv('PORT', 5001))
    HOST = os.getenv('HOST', '0.0.0.0')
    
    logger.info(f'Iniciando servidor en {HOST}:{PORT}')
    
    try:
        uvicorn.run(
            "main:app",  # Cambiar por el nombre de tu archivo si es diferente
            host=HOST,
            port=PORT,
            reload=False,
            log_level="info"
        )
    except KeyboardInterrupt:
        logger.info('Servidor interrumpido por el usuario')
    except Exception as e:
        logger.error(f'Error al iniciar el servidor: {e}')
        sys.exit(1)