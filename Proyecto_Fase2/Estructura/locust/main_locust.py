import json
import time
from datetime import datetime
from locust import HttpUser, task, between
from locust.env import Environment
import requests
import random
from http.server import HTTPServer, BaseHTTPRequestHandler
import threading

# Configuraciones globales - ACTUALIZADAS CON IPs CORRECTAS
BALANCED_API_ENDPOINT = "http://34.121.110.88/api/metrics"  # nginx-lb-service que balancea automáticamente
METRICS_SOURCE_ENDPOINT = "http://34.27.149.243:3001"
API_LECTURA_ENDPOINT = "http://34.55.86.146:8080"  # Para WebSocket si es necesario

# Cola para los datos recibidos del endpoint de métricas
received_data_queue = []
phase1_complete = False
direct_mode = False  # Modo directo sin guardar en JSON
target_records = 2000  # Objetivo de registros

class MetricsFetcherUser(HttpUser):
    """FASE 1: Usuario para obtener datos del endpoint de métricas hasta completar 2000 registros"""
    wait_time = between(1, 2)  # Entre 1-2 segundos según requisitos
    host = METRICS_SOURCE_ENDPOINT
    
    def on_start(self):
        print("🎯 FASE 1: Obteniendo métricas desde endpoint iniciado")
    
    @task
    def fetch_metrics_data(self):
        """Obtiene datos del endpoint de métricas"""
        global target_records
        
        # Si ya alcanzamos el objetivo, no hacer más peticiones
        if len(received_data_queue) >= target_records:
            print(f"🎯 Objetivo alcanzado: {len(received_data_queue)}/{target_records} registros")
            return
            
        try:
            # Hacer petición GET al endpoint de métricas
            response = self.client.get("/api/metrics", timeout=10)
            
            if response.status_code == 200:
                try:
                    data = response.json()
                    
                    # Validar que tenga todos los campos requeridos
                    required_fields = [
                        'total_ram', 'ram_libre', 'uso_ram', 'porcentaje_ram',
                        'porcentaje_cpu_uso', 'porcentaje_cpu_libre',
                        'procesos_corriendo', 'total_procesos', 'procesos_durmiendo',
                        'procesos_zombie', 'procesos_parados', 'hora'
                    ]
                    
                    missing_fields = [field for field in required_fields if field not in data]
                    if missing_fields:
                        print(f"⚠️ Datos incompletos. Faltan: {missing_fields}")
                        return
                    
                    # Agregar timestamp e ID único si no existen
                    if 'timestamp' not in data:
                        data['timestamp'] = datetime.now().isoformat()
                    if 'id' not in data:
                        data['id'] = f"{int(time.time() * 1000)}_{random.randint(1000, 9999)}"
                    
                    # En modo directo, enviar inmediatamente a la API balanceada
                    if direct_mode:
                        self.send_data_to_balanced_api(data)
                    else:
                        # Agregar a la cola solo si no hemos alcanzado el objetivo
                        if len(received_data_queue) < target_records:
                            received_data_queue.append(data)
                    
                    current_count = len(received_data_queue)
                    print(f"📊 Datos obtenidos - {'Enviado directo' if direct_mode else f'Cola: {current_count}/{target_records}'} - RAM: {data.get('porcentaje_ram', 'N/A')}% - CPU: {data.get('porcentaje_cpu_uso', 'N/A')}%")
                    
                except json.JSONDecodeError:
                    print(f"❌ Error decodificando JSON de la respuesta")
                    
            elif response.status_code != 0:
                print(f"⚠️ Endpoint respuesta {response.status_code}")
                
        except requests.exceptions.Timeout:
            print(f"⏱️ Timeout en petición al endpoint")
        except Exception as e:
            print(f"❌ Error obteniendo métricas: {e}")
    
    def send_data_to_balanced_api(self, data):
        """Envía datos directamente a la API balanceada (nginx hace el balanceo automáticamente)"""
        try:
            data_copy = data.copy()
            data_copy['api'] = 'Balanced'  # Marcador para identificar que va a la API balanceada
            
            response = requests.post(
                BALANCED_API_ENDPOINT,
                json=data_copy,
                headers={"Content-Type": "application/json"},
                timeout=5
            )
            
            if response.status_code in [200, 201]:
                print(f"✅ Directo→Balanceada - RAM: {data.get('porcentaje_ram', 'N/A')}% - CPU: {data.get('porcentaje_cpu_uso', 'N/A')}%")
            else:
                print(f"⚠️ Directo→Balanceada error {response.status_code}")
                    
        except Exception as e:
            print(f"❌ Error enviando datos a API balanceada: {e}")

class BalancedTrafficUser(HttpUser):
    """FASE 2: Usuario para enviar datos a la API balanceada (nginx hace el balanceo automáticamente)"""
    wait_time = between(1, 4)  # Entre 1-4 segundos según requisitos
    host = "http://34.121.110.88"  # nginx-lb-service
    
    def on_start(self):
        print("⚖️ FASE 2: Enviando a API balanceada - nginx distribuye automáticamente")
    
    def get_next_data_item(self):
        """Obtiene el siguiente elemento de datos de la cola"""
        if received_data_queue:
            return received_data_queue.pop(0)
        return None
    
    @task
    def send_to_balanced_api(self):
        """Envía objeto individual a la API balanceada"""
        data_item = self.get_next_data_item()
        
        if data_item:
            # Agregar identificador para la API balanceada
            data_item['api'] = 'Balanced'
            data_item['load_balancer'] = 'nginx'
            
            try:
                response = self.client.post(
                    "/api/metrics",
                    json=data_item,
                    headers={"Content-Type": "application/json"}
                )
                
                if response.status_code in [200, 201]:
                    print(f"✅ Balanceada - RAM: {data_item.get('porcentaje_ram', 'N/A')}% - CPU: {data_item.get('porcentaje_cpu_uso', 'N/A')}%")
                else:
                    print(f"⚠️ Balanceada error {response.status_code}")
                    # Devolver a cola sin los campos adicionales
                    if 'api' in data_item:
                        del data_item['api']
                    if 'load_balancer' in data_item:
                        del data_item['load_balancer']
                    received_data_queue.insert(0, data_item)
                    
            except Exception as e:
                print(f"❌ API Balanceada error: {e}")
                if 'api' in data_item:
                    del data_item['api']
                if 'load_balancer' in data_item:
                    del data_item['load_balancer']
                received_data_queue.insert(0, data_item)

def generate_dummy_data(num_records=2000):
    """Genera datos dummy simulando métricas del sistema"""
    dummy_data = []
    
    print(f"🎲 Generando {num_records} registros dummy...")
    
    for i in range(num_records):
        # Generar datos realistas
        total_ram = random.randint(8000, 32000)  # MB
        ram_usado = random.randint(int(total_ram * 0.3), int(total_ram * 0.9))
        ram_libre = total_ram - ram_usado
        porcentaje_ram = round((ram_usado / total_ram) * 100, 2)
        
        cpu_uso = round(random.uniform(10, 95), 2)
        cpu_libre = round(100 - cpu_uso, 2)
        
        total_procesos = random.randint(200, 500)
        procesos_corriendo = random.randint(1, 10)
        procesos_durmiendo = total_procesos - procesos_corriendo - random.randint(0, 5)
        procesos_zombie = random.randint(0, 3)
        procesos_parados = total_procesos - procesos_corriendo - procesos_durmiendo - procesos_zombie
        
        # Crear timestamp con variación
        base_time = datetime.now()
        timestamp_variation = random.randint(-3600, 0)  # Hasta 1 hora atrás
        record_time = base_time.timestamp() + timestamp_variation
        
        dummy_record = {
            "id": f"dummy_{int(time.time() * 1000)}_{random.randint(1000, 9999)}",
            "timestamp": datetime.fromtimestamp(record_time).isoformat(),
            "total_ram": total_ram,
            "ram_libre": ram_libre,
            "uso_ram": ram_usado,
            "porcentaje_ram": porcentaje_ram,
            "porcentaje_cpu_uso": cpu_uso,
            "porcentaje_cpu_libre": cpu_libre,
            "procesos_corriendo": procesos_corriendo,
            "total_procesos": total_procesos,
            "procesos_durmiendo": procesos_durmiendo,
            "procesos_zombie": procesos_zombie,
            "procesos_parados": procesos_parados,
            "hora": datetime.fromtimestamp(record_time).strftime("%H:%M:%S"),
            "dummy": True  # Marcador para identificar datos dummy
        }
        
        dummy_data.append(dummy_record)
        
        # Mostrar progreso cada 500 registros
        if (i + 1) % 500 == 0:
            print(f"📊 Generados {i + 1}/{num_records} registros dummy...")
    
    return dummy_data

def save_dummy_json(num_records=2000):
    """Genera y guarda un archivo JSON con datos dummy"""
    print("🎲 GENERANDO DATOS DUMMY")
    
    try:
        # Generar datos dummy
        dummy_data = generate_dummy_data(num_records)
        
        # Crear archivo JSON
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
        filename = f"dummy_metrics_data_{timestamp}.json"
        
        json_data = {
            "metadata": {
                "total_records": len(dummy_data),
                "generated_at": datetime.now().isoformat(),
                "type": "dummy_data",
                "description": "Datos de métricas simuladas para testing",
                "records_requested": num_records,
                "load_balancer": "nginx",
                "api_endpoint": BALANCED_API_ENDPOINT
            },
            "metrics": dummy_data
        }
        
        with open(filename, 'w', encoding='utf-8') as f:
            json.dump(json_data, f, indent=2, ensure_ascii=False)
        
        print(f"✅ Archivo dummy generado: {filename}")
        print(f"📊 Registros creados: {len(dummy_data)}")
        
        # Cargar datos en memoria para distribución inmediata si se desea
        load_choice = input("\n¿Cargar datos dummy en memoria para distribución? (y/n): ").strip().lower()
        if load_choice == 'y':
            received_data_queue.extend(dummy_data)
            print(f"📥 {len(dummy_data)} registros dummy cargados en memoria")
            return filename, True
        
        return filename, False
        
    except Exception as e:
        print(f"❌ Error generando datos dummy: {e}")
        return None, False

def save_generated_json():
    """Guarda el JSON con los datos generados en la Fase 1"""
    if received_data_queue:
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
        filename = f"metrics_data_{timestamp}.json"
        
        # Crear una copia de los datos para el JSON
        json_data = {
            "metadata": {
                "total_records": len(received_data_queue),
                "generated_at": datetime.now().isoformat(),
                "phase": "Fase 1 - Obtención de datos del endpoint",
                "source_endpoint": METRICS_SOURCE_ENDPOINT,
                "target_records": target_records,
                "users": 300,
                "load_balancer": "nginx",
                "balanced_api_endpoint": BALANCED_API_ENDPOINT
            },
            "metrics": received_data_queue.copy()
        }
        
        with open(filename, 'w', encoding='utf-8') as f:
            json.dump(json_data, f, indent=2, ensure_ascii=False)
        
        print(f"💾 JSON generado: {filename}")
        print(f"📊 Registros guardados: {len(received_data_queue)}")
        return filename
    else:
        print("❌ No hay datos para generar JSON")
        return None

def run_phase1_metrics_collection():
    """FASE 1: Obtiene datos del endpoint hasta completar 2000 registros"""
    global phase1_complete, target_records
    
    print("=== FASE 1: OBTENCIÓN DE MÉTRICAS INICIADA ===")
    print(f"🎯 Objetivo: {target_records} registros")
    print(f"📡 Endpoint: {METRICS_SOURCE_ENDPOINT}")
    print("👥 300 usuarios, 1-2 seg entre peticiones, +1 usuario/seg")
    
    # Configurar Locust para Fase 1
    env = Environment(user_classes=[MetricsFetcherUser])
    env.create_local_runner()
    
    users = 300
    spawn_rate = 1  # 1 usuario por segundo según requisitos
    
    print(f"🚀 Iniciando {users} usuarios obteniendo métricas")
    env.runner.start(user_count=users, spawn_rate=spawn_rate)
    
    start_time = time.time()
    
    print(f"⏱️ Ejecutando hasta completar {target_records} registros...")
    
    while len(received_data_queue) < target_records:
        elapsed = int(time.time() - start_time)
        queue_size = len(received_data_queue)
        progress = (queue_size / target_records) * 100
        
        print(f"📈 Tiempo: {elapsed}s - Progreso: {queue_size}/{target_records} ({progress:.1f}%) - Faltantes: {target_records - queue_size}")
        
        # Verificar si han pasado muchos minutos sin progreso
        if elapsed > 600 and queue_size < 100:  # 10 minutos sin muchos registros
            print("⚠️ ADVERTENCIA: Poco progreso en 10 minutos. Verifica la conectividad del endpoint.")
        
        time.sleep(10)  # Reporte cada 10 segundos
    
    env.runner.stop()
    phase1_complete = True
    
    final_count = len(received_data_queue)
    elapsed_total = int(time.time() - start_time)
    print(f"✅ FASE 1 COMPLETADA en {elapsed_total}s - Registros obtenidos: {final_count}")
    
    # GENERAR EL ARCHIVO JSON REQUERIDO
    json_file = save_generated_json()
    if json_file:
        print(f"📁 Archivo JSON creado: {json_file}")
    
    return final_count

def run_phase2_balanced_distribution():
    """FASE 2: Distribuye los datos a la API balanceada con 150 usuarios"""
    print("\n=== FASE 2: DISTRIBUCIÓN A API BALANCEADA INICIADA ===")
    print("⚖️ 150 usuarios, 1-4 seg entre peticiones, +1 usuario/seg")
    print(f"📊 API Balanceada: {BALANCED_API_ENDPOINT}")
    print("🔄 nginx-lb-service distribuye automáticamente entre NodeJS y Python")
    
    # Configurar Locust para Fase 2
    env = Environment(user_classes=[BalancedTrafficUser])
    env.create_local_runner()
    
    users = 150  # Según requisitos
    spawn_rate = 1  # 1 usuario por segundo
    
    print(f"🚀 Iniciando {users} usuarios enviando a API balanceada")
    env.runner.start(user_count=users, spawn_rate=spawn_rate)
    
    print("🎯 Enviando datos a:")
    print("   - nginx-lb-service (balanceador)")
    print("   - Distribuirá automáticamente entre:")
    print("     * api-monitoreo (NodeJS - puerto 4001)")
    print("     * api-monitoreo-python (Python - puerto 5001)")
    print("\nPresiona Ctrl+C para parar...")
    
    try:
        while True:
            time.sleep(10)
            queue_size = len(received_data_queue)
            if queue_size > 0:
                print(f"📈 Cola restante: {queue_size} elementos")
            else:
                print("⏳ Cola vacía, esperando más datos o presiona Ctrl+C...")
                
    except KeyboardInterrupt:
        print("\n🛑 Parando distribución...")
    
    env.runner.stop()
    print("✅ FASE 2 COMPLETADA")

def run_direct_balanced_mode():
    """Modo directo: obtiene datos del endpoint y los envía inmediatamente a la API balanceada"""
    global direct_mode
    direct_mode = True
    
    print("=== MODO DIRECTO BALANCEADO INICIADO ===")
    print(f"🎯 Endpoint {METRICS_SOURCE_ENDPOINT} → API Balanceada directamente")
    print(f"⚖️ nginx-lb-service: {BALANCED_API_ENDPOINT}")
    print("⚡ Sin almacenamiento en JSON, sin cola intermedia")
    print("🔄 nginx distribuye automáticamente entre NodeJS y Python")
    
    # Configurar Locust para modo directo
    env = Environment(user_classes=[MetricsFetcherUser])
    env.create_local_runner()
    
    users = 50  # Menos usuarios para modo directo
    spawn_rate = 1
    
    print(f"🚀 Iniciando {users} usuarios en modo directo balanceado")
    env.runner.start(user_count=users, spawn_rate=spawn_rate)
    
    print("🔄 Esperando datos del endpoint...")
    print("📊 Los datos se enviarán automáticamente a la API balanceada")
    print("⚖️ nginx-lb-service distribuirá entre:")
    print("   - api-monitoreo (NodeJS - puerto 4001)")
    print("   - api-monitoreo-python (Python - puerto 5001)")
    print("\nPresiona Ctrl+C para parar...")
    
    try:
        while True:
            time.sleep(5)
            print("⏳ Modo directo balanceado activo, obteniendo y enviando datos...")
            
    except KeyboardInterrupt:
        print("\n🛑 Parando modo directo balanceado...")
    finally:
        env.runner.stop()
        direct_mode = False
        print("✅ MODO DIRECTO BALANCEADO COMPLETADO")

def load_json_data(filename):
    """Carga datos desde un archivo JSON generado previamente"""
    try:
        with open(filename, 'r', encoding='utf-8') as f:
            data = json.load(f)
        
        # Si el JSON tiene metadata, extraer solo las métricas
        if 'metrics' in data:
            metrics = data['metrics']
        else:
            metrics = data
        
        # Cargar los datos en la cola
        received_data_queue.extend(metrics)
        
        print(f"📁 JSON cargado: {filename}")
        print(f"📊 Registros cargados: {len(metrics)}")
        return len(metrics)
        
    except FileNotFoundError:
        print(f"❌ Archivo no encontrado: {filename}")
        return 0
    except Exception as e:
        print(f"❌ Error cargando JSON: {e}")
        return 0

def save_remaining_data():
    """Guarda datos restantes en archivo JSON"""
    if received_data_queue:
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
        filename = f"remaining_data_{timestamp}.json"
        
        with open(filename, 'w', encoding='utf-8') as f:
            json.dump(received_data_queue, f, indent=2, ensure_ascii=False)
        
        print(f"💾 {len(received_data_queue)} registros guardados en {filename}")
        return filename
    else:
        print("✅ No hay datos restantes para guardar")
        return None

def run_complete_balanced_test():
    """Ejecuta el test completo: Fase 1 + Fase 2 con API balanceada"""
    try:        
        print("🌟 INICIANDO TEST COMPLETO DE CARGA CON API BALANCEADA")
        print("📋 Requisitos del proyecto:")
        print(f"   - Fase 1: 300 usuarios, hasta {target_records} registros")
        print("   - Fase 2: 150 usuarios, envío a API balanceada")
        print(f"   - Endpoint de métricas: {METRICS_SOURCE_ENDPOINT}")
        print(f"   - API Balanceada: {BALANCED_API_ENDPOINT}")
        print("   - nginx-lb-service distribuye automáticamente entre NodeJS y Python\n")
        
        # FASE 1: Obtención de datos
        records_generated = run_phase1_metrics_collection()
        
        if records_generated < 100:
            print("❌ ERROR: Muy pocos registros obtenidos. Verifica:")
            print(f"   1. Endpoint {METRICS_SOURCE_ENDPOINT} está accesible")
            print("   2. El endpoint devuelve datos en formato JSON válido")
            print("   3. Los datos contienen todos los campos requeridos")
            return
        
        print(f"\n⏸️ Pausa de 5 segundos entre fases...")
        time.sleep(5)
        
        # FASE 2: Distribución a API balanceada
        run_phase2_balanced_distribution()
        
        print("\n🎉 TEST COMPLETO CON API BALANCEADA FINALIZADO")
        
    except KeyboardInterrupt:
        print("\n🔄 Test interrumpido por usuario...")
    except Exception as e:
        print(f"❌ Error en test completo: {e}")
    finally:
        save_remaining_data()
        print("✅ Proceso terminado")

if __name__ == "__main__":
    print("🚀 LOCUST - PROYECTO FASE 2 - SO1 (API BALANCEADA)")
    print("=" * 50)
    
    # Verificar configuración
    print("🔧 Verificando configuración:")
    print(f"   - Endpoint métricas: {METRICS_SOURCE_ENDPOINT}")
    print(f"   - API Balanceada: {BALANCED_API_ENDPOINT}")
    print(f"   - API Lectura: {API_LECTURA_ENDPOINT}")
    print(f"   - Objetivo registros: {target_records}")
    print("   - Balanceador: nginx-lb-service (automático)")
    
    # Opciones de ejecución actualizadas
    print("\n📋 OPCIONES DE EJECUCIÓN:")
    print("1. Ejecutar test completo balanceado (Fase 1 + Fase 2)")
    print("2. Solo Fase 1 (obtener datos y generar JSON)")
    print("3. Solo Fase 2 balanceada (cargar JSON existente)")
    print("4. Continuar con datos en memoria")
    print("5. 🎲 Generar JSON con datos dummy")
    print("6. ⚡ Modo directo balanceado (endpoint → API balanceada)")
    
    try:
        choice = input("\nSelecciona una opción (1-6): ").strip()
        
        if choice == "1":
            run_complete_balanced_test()
        elif choice == "2":
            # Solo Fase 1
            run_phase1_metrics_collection()
        elif choice == "3":
            # Solo Fase 2 con JSON existente
            json_file = input("Ingresa el nombre del archivo JSON: ").strip()
            if json_file and load_json_data(json_file) > 0:
                run_phase2_balanced_distribution()
            else:
                print("❌ No se pudo cargar el archivo JSON")
        elif choice == "4":
            # Continuar con datos en memoria
            if len(received_data_queue) > 0:
                print(f"📊 Datos en memoria: {len(received_data_queue)}")
                run_phase2_balanced_distribution()
            else:
                print("❌ No hay datos en memoria")
        elif choice == "5":
            # Generar datos dummy
            try:
                num_records = input(f"Número de registros dummy (default {target_records}): ").strip()
                num_records = int(num_records) if num_records else target_records
                filename, loaded = save_dummy_json(num_records)
                
                if filename and loaded:
                    # Si se cargaron en memoria, ofrecer distribución
                    distribute = input("¿Iniciar distribución de datos dummy a API balanceada? (y/n): ").strip().lower()
                    if distribute == 'y':
                        run_phase2_balanced_distribution()
            except ValueError:
                print("❌ Número inválido")
        elif choice == "6":
            # Modo directo balanceado
            run_direct_balanced_mode()
        else:
            print("❌ Opción inválida")
            
    except KeyboardInterrupt:
        print("\n🔄 Interrumpido por usuario...")
        save_remaining_data()
    except Exception as e:
        print(f"❌ Error crítico: {e}")
        save_remaining_data()