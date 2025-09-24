#!/usr/bin/env python3
import time
import random
import threading
import gc

class MemoryStressTest:
    def __init__(self):
        self.memory_chunks = []
        self.chunk_size = 10 * 1024 * 1024  # 10MB
        self.running = True
        
    def log(self, message):
        print(f"[{time.strftime('%H:%M:%S')}] {message}", flush=True)
        
    def memory_consumer(self):
        self.log("=== INICIANDO CONSUMO INTENSIVO DE RAM ===")
        chunk_count = 0
        
        try:
            while self.running:
                # Crear chunk de datos aleatorios
                chunk = bytearray(random.getrandbits(8) for _ in range(self.chunk_size))
                self.memory_chunks.append(chunk)
                
                # String grande
                big_string = ''.join(random.choices('ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789', k=1024*1024))
                self.memory_chunks.append(big_string)
                
                # Lista de números
                number_list = [random.randint(1, 1000000) for _ in range(100000)]
                self.memory_chunks.append(number_list)
                
                chunk_count += 1
                
                if chunk_count % 10 == 0:
                    memory_mb = len(self.memory_chunks) * 4  # Estimación
                    self.log(f"Memoria consumida: ~{memory_mb}MB - Chunks: {len(self.memory_chunks)}")
                
                time.sleep(0.2)
                
                # Gestión de memoria para evitar crash
                if len(self.memory_chunks) > 300:
                    self.memory_chunks = self.memory_chunks[100:]  # Mantener solo los últimos 200
                    gc.collect()
                    self.log("Liberando memoria antigua...")
                    
        except MemoryError:
            self.log("Límite de memoria alcanzado - manteniendo presión")
            while self.running:
                time.sleep(30)
                self.log("Manteniendo presión de memoria...")
    
    def cpu_background(self):
        """CPU en segundo plano"""
        while self.running:
            result = sum(i * i * random.random() for i in range(10000))
            time.sleep(1)

def signal_handler(signum, frame):
    print(f"\nSeñal {signum} recibida, terminando...")
    exit(0)

if __name__ == "__main__":
    import signal
    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)
    
    memory_test = MemoryStressTest()
    
    # CPU thread en segundo plano
    cpu_thread = threading.Thread(target=memory_test.cpu_background)
    cpu_thread.daemon = True
    cpu_thread.start()
    
    # Consumo principal de memoria
    memory_test.memory_consumer()
