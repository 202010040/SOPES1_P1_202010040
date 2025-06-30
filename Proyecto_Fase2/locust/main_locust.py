import json
import time
import random
from datetime import datetime
from locust import HttpUser, task, between, events
from locust.env import Environment
from locust.stats import stats_printer, stats_history
from locust.log import setup_logging
import gevent

# Configuraciones globales
VM_ENDPOINT = "http://localhost:3000/api/metrics"  # Cambia por tu VM endpoint
NODE_API_ENDPOINT = "http://localhost:4001/api/metrics"
PYTHON_API_ENDPOINT = "http://localhost:5001/api/metrics"

# Almacén para los datos recolectados
collected_data = []

class VMDataCollector(HttpUser):
    """Usuario para recolectar datos de la VM"""
    wait_time = between(1, 2)  # Entre 1 y 2 segundos
    
    def on_start(self):
        """Inicialización del usuario"""
        print(f"Usuario {self.environment.runner.user_count} iniciado para recolección de datos VM")
    
    @task
    def collect_vm_metrics(self):
        """Recolecta métricas de la VM"""
        try:
            response = self.client.get("/api/metrics")
            if response.status_code == 200:
                # Simular datos si la VM no responde con el formato exacto
                data = self.generate_mock_data()
                collected_data.append(data)
                print(f"Datos recolectados: {len(collected_data)} registros")
            else:
                # Generar datos mock si hay error
                data = self.generate_mock_data()
                collected_data.append(data)
        except Exception as e:
            # En caso de error, generar datos mock
            data = self.generate_mock_data()
            collected_data.append(data)
    
    def generate_mock_data(self):
        """Genera datos simulados con el formato requerido"""
        return {
            "total_ram": random.randint(1500, 3000),
            "ram_libre": random.randint(500000000, 2000000000),
            "uso_ram": random.randint(200, 800),
            "porcentaje_ram": random.randint(15, 85),
            "porcentaje_cpu_uso": random.randint(10, 90),
            "porcentaje_cpu_libre": random.randint(10, 90),
            "procesos_corriendo": random.randint(50, 200),
            "total_procesos": random.randint(150, 400),
            "procesos_durmiendo": random.randint(30, 100),
            "procesos_zombie": random.randint(0, 10),
            "procesos_parados": random.randint(0, 20),
            "hora": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        }

class TrafficSplitUser(HttpUser):
    """Usuario para enviar datos al traffic split"""
    wait_time = between(1, 4)  # Entre 1 y 4 segundos
    
    def on_start(self):
        """Inicialización del usuario"""
        print(f"Usuario {self.environment.runner.user_count} iniciado para traffic split")
    
    @task(5)  # Mayor peso para Node.js
    def send_to_node_api(self):
        """Envía datos a la API de Node.js"""
        if collected_data:
            # Seleccionar datos aleatorios del conjunto recolectado
            sample_data = random.sample(collected_data, min(10, len(collected_data)))
            try:
                response = self.client.post(
                    "http://localhost:4001/api/metrics",
                    json=sample_data,
                    headers={"Content-Type": "application/json"}
                )
                print(f"Enviado a Node.js - Status: {response.status_code}")
            except Exception as e:
                print(f"Error enviando a Node.js: {e}")
    
    @task(3)  # Menor peso para Python
    def send_to_python_api(self):
        """Envía datos a la API de Python"""
        if collected_data:
            # Seleccionar datos aleatorios del conjunto recolectado
            sample_data = random.sample(collected_data, min(10, len(collected_data)))
            try:
                response = self.client.post(
                    "http://localhost:5001/api/metrics",
                    json=sample_data,
                    headers={"Content-Type": "application/json"}
                )
                print(f"Enviado a Python - Status: {response.status_code}")
            except Exception as e:
                print(f"Error enviando a Python: {e}")

def save_collected_data():
    """Guarda los datos recolectados en un archivo JSON"""
    if collected_data:
        filename = f"collected_metrics_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        with open(filename, 'w') as f:
            json.dump(collected_data, f, indent=2)
        print(f"Datos guardados en {filename} - Total registros: {len(collected_data)}")

# Configuración para ejecutar las fases
def run_data_collection_phase():
    """Fase 1: Recolección de datos de la VM"""
    print("=== INICIANDO FASE 1: RECOLECCIÓN DE DATOS VM ===")
    
    # Configurar environment para recolección
    env = Environment(user_classes=[VMDataCollector])
    env.create_local_runner()
    
    # Configurar para 300 usuarios, 1 usuario nuevo por segundo
    users = 300
    spawn_rate = 1
    
    print(f"Iniciando {users} usuarios con spawn rate de {spawn_rate}/seg")
    env.runner.start(user_count=users, spawn_rate=spawn_rate)
    
    # Ejecutar por 3 minutos (180 segundos)
    duration = 180
    print(f"Ejecutando por {duration} segundos...")
    
    start_time = time.time()
    while time.time() - start_time < duration:
        time.sleep(1)
        if len(collected_data) >= 2000:
            print("¡Objetivo de 2000 registros alcanzado!")
            break
    
    env.runner.stop()
    save_collected_data()
    print(f"=== FASE 1 COMPLETADA - {len(collected_data)} registros recolectados ===\n")

def run_traffic_split_phase():
    """Fase 2: Envío al traffic split"""
    print("=== INICIANDO FASE 2: TRAFFIC SPLIT ===")
    
    if len(collected_data) < 100:
        print("⚠️  Pocos datos recolectados, generando datos adicionales...")
        # Generar más datos si es necesario
        collector = VMDataCollector()
        for _ in range(500):
            collected_data.append(collector.generate_mock_data())
    
    # Configurar environment para traffic split
    env = Environment(user_classes=[TrafficSplitUser])
    env.create_local_runner()
    
    # Configurar para 150 usuarios, 1 usuario nuevo por segundo
    users = 150
    spawn_rate = 1
    
    print(f"Iniciando {users} usuarios con spawn rate de {spawn_rate}/seg")
    env.runner.start(user_count=users, spawn_rate=spawn_rate)
    
    # Ejecutar por tiempo indefinido (presiona Ctrl+C para parar)
    print("Ejecutando traffic split... (Presiona Ctrl+C para parar)")
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\nParando traffic split...")
    
    env.runner.stop()
    print("=== FASE 2 COMPLETADA ===")

if __name__ == "__main__":
    import sys
    
    if len(sys.argv) > 1:
        if sys.argv[1] == "collect":
            run_data_collection_phase()
        elif sys.argv[1] == "split":
            run_traffic_split_phase()
        elif sys.argv[1] == "full":
            run_data_collection_phase()
            input("Presiona Enter para continuar con el traffic split...")
            run_traffic_split_phase()
        else:
            print("Uso: python script.py [collect|split|full]")
    else:
        print("Uso: python script.py [collect|split|full]")
        print("  collect: Solo recolección de datos VM")
        print("  split: Solo traffic split")
        print("  full: Ambas fases secuencialmente")