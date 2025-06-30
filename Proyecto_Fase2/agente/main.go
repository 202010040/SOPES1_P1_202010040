package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Estructuras para los datos de monitoreo
type RAMInfo struct {
	Total      int64 `json:"total"`
	Libre      int64 `json:"libre"`
	Uso        int64 `json:"uso"`
	Porcentaje int64 `json:"porcentaje"`
}

type CPUInfo struct {
	PorcentajeUso   int64 `json:"porcentajeUso"`
	PorcentajeLibre int64 `json:"porcentaje_libre"` // Campo calculado
}

type ProcesosInfo struct {
	ProcesosCorriendo int64 `json:"procesos_corriendo"`
	TotalProcesos     int64 `json:"total_procesos"`
	ProcesosDurmiendo int64 `json:"procesos_durmiendo"`
	ProcesosZombie    int64 `json:"procesos_zombie"`
	ProcesosParados   int64 `json:"procesos_parados"`
}

// FIXED: Estructura SystemMetrics con nombres de campos JSON exactos que requiere la API
type SystemMetrics struct {
	// RAM fields - CAMBIOS: Usar los nombres exactos requeridos
	TotalRAM      int64 `json:"total_ram"`
	RAMLibre      int64 `json:"ram_libre"`
	UsoRAM        int64 `json:"uso_ram"`
	PorcentajeRAM int64 `json:"porcentaje_ram"`

	// CPU fields - CAMBIOS: Usar los nombres exactos requeridos
	PorcentajeCPUUso   int64 `json:"porcentaje_cpu_uso"`
	PorcentajeCPULibre int64 `json:"porcentaje_cpu_libre"`

	// Procesos fields - Los nombres ya están correctos
	ProcesosCorriendo int64 `json:"procesos_corriendo"`
	TotalProcesos     int64 `json:"total_procesos"`
	ProcesosDurmiendo int64 `json:"procesos_durmiendo"`
	ProcesosZombie    int64 `json:"procesos_zombie"`
	ProcesosParados   int64 `json:"procesos_parados"`

	// Timestamp
	Hora string `json:"hora"`
}

// Canales para comunicación entre goroutines
type Channels struct {
	RAMChan      chan RAMInfo
	CPUChan      chan CPUInfo
	ProcesosChan chan ProcesosInfo
	MetricsChan  chan SystemMetrics
	ErrorChan    chan error
	StopChan     chan bool
}

// Configuración del agente
type Config struct {
	APIEndpoint      string
	MonitorInterval  time.Duration
	RAMProcFile      string
	CPUProcFile      string
	ProcesosProcFile string
	MaxRetries       int
}

// Agente de monitoreo principal
type MonitoringAgent struct {
	config   Config
	channels Channels
	wg       sync.WaitGroup
	client   *APIClient

	// Almacenamiento temporal de métricas
	latestRAM      RAMInfo
	latestCPU      CPUInfo
	latestProcesos ProcesosInfo
	metricsMutex   sync.RWMutex

	// Flags para saber si tenemos datos válidos
	hasRAMData      bool
	hasCPUData      bool
	hasProcesosData bool
}

func main() {
	// Verificar si se ejecuta en modo testing
	testMode := os.Getenv("TEST_MODE")
	if testMode == "true" || testMode == "1" {
		runTestMode()
		return
	}

	// Obtener configuración desde variables de entorno
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "localhost" // Para testing local
	}

	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "3001"
	}

	// Configuración del agente
	config := Config{
		APIEndpoint:      fmt.Sprintf("http://%s:%s/api/metrics", apiHost, apiPort),
		MonitorInterval:  5 * time.Second,
		RAMProcFile:      "/proc/ram_202010040",
		CPUProcFile:      "/proc/cpu_202010040",
		ProcesosProcFile: "/proc/procesos_202010040",
		MaxRetries:       3,
	}

	// Inicializar canales
	channels := Channels{
		RAMChan:      make(chan RAMInfo, 10),
		CPUChan:      make(chan CPUInfo, 10),
		ProcesosChan: make(chan ProcesosInfo, 10),
		MetricsChan:  make(chan SystemMetrics, 10),
		ErrorChan:    make(chan error, 10),
		StopChan:     make(chan bool, 10),
	}

	// Crear cliente API
	client := NewAPIClient(config.APIEndpoint, config.MaxRetries)

	// Crear agente de monitoreo
	agent := &MonitoringAgent{
		config:   config,
		channels: channels,
		client:   client,
	}

	log.Println("🚀 Iniciando Agente de Monitoreo de Sistema")
	log.Printf("📊 Intervalo de monitoreo: %v", config.MonitorInterval)
	log.Printf("🔗 API Endpoint: %s", config.APIEndpoint)

	// Verificar que los archivos /proc existan
	if err := agent.CheckProcFiles(); err != nil {
		log.Fatalf("❌ Error verificando archivos /proc: %v", err)
	}

	// Hacer una lectura inicial para poblar los datos antes de iniciar goroutines
	log.Println("📊 Realizando lectura inicial de métricas...")
	if err := agent.initialDataRead(); err != nil {
		log.Fatalf("❌ Error en lectura inicial: %v", err)
	}

	// Iniciar agente
	agent.Start()

	// Manejar señales de sistema para cierre graceful
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Esperar señal de cierre
	<-sigChan
	log.Println("🛑 Recibida señal de cierre, deteniendo agente...")

	// Detener agente
	agent.Stop()
	log.Println("✅ Agente de monitoreo detenido correctamente")
}

// Función para hacer lectura inicial de datos
func (ma *MonitoringAgent) initialDataRead() error {
	log.Println("🔄 Leyendo datos iniciales...")

	// Leer RAM
	if ramInfo, err := ma.readRAMInfo(); err != nil {
		return fmt.Errorf("error leyendo RAM inicial: %v", err)
	} else {
		ma.metricsMutex.Lock()
		ma.latestRAM = ramInfo
		ma.hasRAMData = true
		ma.metricsMutex.Unlock()
		log.Printf("✅ RAM inicial: %d%% (%d/%d KB)", ramInfo.Porcentaje, ramInfo.Uso, ramInfo.Total)
	}

	// Leer CPU
	if cpuInfo, err := ma.readCPUInfo(); err != nil {
		return fmt.Errorf("error leyendo CPU inicial: %v", err)
	} else {
		ma.metricsMutex.Lock()
		ma.latestCPU = cpuInfo
		ma.hasCPUData = true
		ma.metricsMutex.Unlock()
		log.Printf("✅ CPU inicial: %d%% uso", cpuInfo.PorcentajeUso)
	}

	// Leer Procesos
	if procesosInfo, err := ma.readProcesosInfo(); err != nil {
		return fmt.Errorf("error leyendo Procesos inicial: %v", err)
	} else {
		ma.metricsMutex.Lock()
		ma.latestProcesos = procesosInfo
		ma.hasProcesosData = true
		ma.metricsMutex.Unlock()
		log.Printf("✅ Procesos inicial: %d total", procesosInfo.TotalProcesos)
	}

	return nil
}

// Función para ejecutar en modo testing
func runTestMode() {
	log.Println("🧪 MODO TESTING ACTIVADO")

	// Crear archivos mock temporales
	if err := createMockFiles(); err != nil {
		log.Printf("❌ Error creando archivos mock: %v", err)
		log.Println("📝 Creando archivos mock básicos...")
		createBasicMockFiles()
	}

	// Configuración para testing
	config := Config{
		APIEndpoint:      "http://localhost:3001/api/metrics",
		MonitorInterval:  2 * time.Second, // Más rápido para testing
		RAMProcFile:      "/tmp/ram_202010040",
		CPUProcFile:      "/tmp/cpu_202010040",
		ProcesosProcFile: "/tmp/procesos_202010040",
		MaxRetries:       2,
	}

	// Test de lectura de archivos
	log.Println("🔍 Testeando lectura de archivos...")
	testAgent := &MonitoringAgent{config: config}
	testAgent.TestReading()

	// Si no hay API disponible, solo mostrar los datos leídos
	log.Println("⚠️  Si no tienes una API corriendo, los datos se mostrarán aquí:")

	// Leer y mostrar datos cada 3 segundos
	for i := 0; i < 5; i++ {
		log.Printf("📊 Test #%d:", i+1)

		if ram, err := testAgent.readRAMInfo(); err == nil {
			log.Printf("   💾 RAM: %d%% (%d/%d KB)", ram.Porcentaje, ram.Uso, ram.Total)
		}

		if cpu, err := testAgent.readCPUInfo(); err == nil {
			log.Printf("   ⚡ CPU: %d%% uso", cpu.PorcentajeUso)
		}

		if proc, err := testAgent.readProcesosInfo(); err == nil {
			log.Printf("   🔄 Procesos: %d total (%d corriendo, %d durmiendo)",
				proc.TotalProcesos, proc.ProcesosCorriendo, proc.ProcesosDurmiendo)
		}

		// FIXED: Generar JSON con la estructura exacta requerida
		var ram RAMInfo
		var cpu CPUInfo
		var proc ProcesosInfo

		ram, _ = testAgent.readRAMInfo()
		cpu, _ = testAgent.readCPUInfo()
		proc, _ = testAgent.readProcesosInfo()

		// Calcular PorcentajeLibre para CPU (no viene del JSON)
		cpu.PorcentajeLibre = 100 - cpu.PorcentajeUso

		// FIXED: Crear SystemMetrics con los nombres de campo exactos
		metrics := SystemMetrics{
			// RAM fields - NUEVOS NOMBRES
			TotalRAM:      ram.Total,
			RAMLibre:      ram.Libre,
			UsoRAM:        ram.Uso,
			PorcentajeRAM: ram.Porcentaje,

			// CPU fields - NUEVOS NOMBRES
			PorcentajeCPUUso:   cpu.PorcentajeUso,
			PorcentajeCPULibre: cpu.PorcentajeLibre,

			// Procesos fields (sin cambios)
			ProcesosCorriendo: proc.ProcesosCorriendo,
			TotalProcesos:     proc.TotalProcesos,
			ProcesosDurmiendo: proc.ProcesosDurmiendo,
			ProcesosZombie:    proc.ProcesosZombie,
			ProcesosParados:   proc.ProcesosParados,

			// Timestamp
			Hora: time.Now().Format("2006-01-02 15:04:05"),
		}

		jsonData, _ := json.MarshalIndent(metrics, "", "  ")
		log.Printf("📄 JSON generado:\n%s", string(jsonData))

		time.Sleep(3 * time.Second)
	}

	// Limpiar archivos temporales
	cleanupMockFiles()
	log.Println("✅ Testing completado")
}

// FIXED: Crear archivos mock básicos con datos que coincidan con el formato esperado
func createBasicMockFiles() {
	ramData := `{"total":2072,"libre":1110552576,"uso":442,"porcentaje":22}`
	cpuData := `{"porcentajeUso":22}`
	procesosData := `{"procesos_corriendo":123,"total_procesos":233,"procesos_durmiendo":65,"procesos_zombie":65,"procesos_parados":65}`

	ioutil.WriteFile("/tmp/ram_202010040", []byte(ramData), 0644)
	ioutil.WriteFile("/tmp/cpu_202010040", []byte(cpuData), 0644)
	ioutil.WriteFile("/tmp/procesos_202010040", []byte(procesosData), 0644)

	log.Println("✅ Archivos mock básicos creados en /tmp/")
}

// FIXED: Crear archivos mock con datos dinámicos que coincidan con el formato esperado
func createMockFiles() error {
	ram := RAMInfo{
		Total:      2072,
		Libre:      1110552576,
		Uso:        442,
		Porcentaje: 22,
	}

	cpu := CPUInfo{
		PorcentajeUso:   22,
		PorcentajeLibre: 78, // Este se calcula, no viene del JSON
	}

	procesos := ProcesosInfo{
		ProcesosCorriendo: 123,
		TotalProcesos:     233,
		ProcesosDurmiendo: 65,
		ProcesosZombie:    65,
		ProcesosParados:   65,
	}

	// Escribir archivos
	ramData, _ := json.Marshal(ram)
	cpuData, _ := json.Marshal(map[string]int64{"porcentajeUso": cpu.PorcentajeUso})
	procesosData, _ := json.Marshal(procesos)

	if err := ioutil.WriteFile("/tmp/ram_202010040", ramData, 0644); err != nil {
		return err
	}
	if err := ioutil.WriteFile("/tmp/cpu_202010040", cpuData, 0644); err != nil {
		return err
	}
	if err := ioutil.WriteFile("/tmp/procesos_202010040", procesosData, 0644); err != nil {
		return err
	}

	log.Println("✅ Archivos mock creados en /tmp/")
	return nil
}

// Limpiar archivos temporales
func cleanupMockFiles() {
	os.Remove("/tmp/ram_202010040")
	os.Remove("/tmp/cpu_202010040")
	os.Remove("/tmp/procesos_202010040")
	log.Println("🧹 Archivos temporales eliminados")
}

// Iniciar todas las goroutines del agente
func (ma *MonitoringAgent) Start() {
	log.Println("🔄 Iniciando goroutines de monitoreo...")

	// Iniciar monitores individuales
	ma.wg.Add(1)
	go ma.monitorRAM()

	ma.wg.Add(1)
	go ma.monitorCPU()

	ma.wg.Add(1)
	go ma.monitorProcesos()

	// IMPORTANTE: Esperar un poco antes de iniciar el combinador
	// para asegurar que las goroutines de monitoreo lean datos frescos
	time.Sleep(1 * time.Second)

	// Iniciar combinador de métricas
	ma.wg.Add(1)
	go ma.combineMetrics()

	// Iniciar enviador de datos a API
	ma.wg.Add(1)
	go ma.sendMetricsToAPI()

	// Iniciar manejador de errores
	ma.wg.Add(1)
	go ma.handleErrors()

	log.Println("✅ Todas las goroutines iniciadas correctamente")
}

// Detener todas las goroutines
func (ma *MonitoringAgent) Stop() {
	log.Println("🔄 Deteniendo goroutines...")

	// Enviar señal de stop a todas las goroutines
	for i := 0; i < 10; i++ {
		select {
		case ma.channels.StopChan <- true:
		default:
		}
	}

	// Esperar que todas las goroutines terminen
	ma.wg.Wait()

	// Cerrar canales
	close(ma.channels.RAMChan)
	close(ma.channels.CPUChan)
	close(ma.channels.ProcesosChan)
	close(ma.channels.MetricsChan)
	close(ma.channels.ErrorChan)
	close(ma.channels.StopChan)

	log.Println("✅ Todas las goroutines detenidas")
}

// Goroutine para monitorear RAM
func (ma *MonitoringAgent) monitorRAM() {
	defer ma.wg.Done()
	ticker := time.NewTicker(ma.config.MonitorInterval)
	defer ticker.Stop()

	log.Println("📊 Iniciando monitoreo de RAM")

	for {
		select {
		case <-ticker.C:
			ramInfo, err := ma.readRAMInfo()
			if err != nil {
				ma.channels.ErrorChan <- fmt.Errorf("error leyendo RAM: %v", err)
				continue
			}

			// Actualizar inmediatamente los datos en memoria
			ma.metricsMutex.Lock()
			ma.latestRAM = ramInfo
			ma.hasRAMData = true
			ma.metricsMutex.Unlock()

			select {
			case ma.channels.RAMChan <- ramInfo:
				log.Printf("💾 RAM: %d%% usado (%d/%d KB)",
					ramInfo.Porcentaje, ramInfo.Uso, ramInfo.Total)
			default:
				log.Println("⚠️ Canal RAM lleno, descartando datos")
			}

		case <-ma.channels.StopChan:
			log.Println("🛑 Deteniendo monitoreo de RAM")
			return
		}
	}
}

// Goroutine para monitorear CPU
func (ma *MonitoringAgent) monitorCPU() {
	defer ma.wg.Done()
	ticker := time.NewTicker(ma.config.MonitorInterval)
	defer ticker.Stop()

	log.Println("📊 Iniciando monitoreo de CPU")

	for {
		select {
		case <-ticker.C:
			cpuInfo, err := ma.readCPUInfo()
			if err != nil {
				ma.channels.ErrorChan <- fmt.Errorf("error leyendo CPU: %v", err)
				continue
			}

			// Calcular porcentaje libre (no viene del JSON del módulo)
			cpuInfo.PorcentajeLibre = 100 - cpuInfo.PorcentajeUso

			// Actualizar inmediatamente los datos en memoria
			ma.metricsMutex.Lock()
			ma.latestCPU = cpuInfo
			ma.hasCPUData = true
			ma.metricsMutex.Unlock()

			select {
			case ma.channels.CPUChan <- cpuInfo:
				log.Printf("⚡ CPU: %d%% uso, %d%% libre",
					cpuInfo.PorcentajeUso, cpuInfo.PorcentajeLibre)
			default:
				log.Println("⚠️ Canal CPU lleno, descartando datos")
			}

		case <-ma.channels.StopChan:
			log.Println("🛑 Deteniendo monitoreo de CPU")
			return
		}
	}
}

// Goroutine para monitorear Procesos
func (ma *MonitoringAgent) monitorProcesos() {
	defer ma.wg.Done()
	ticker := time.NewTicker(ma.config.MonitorInterval)
	defer ticker.Stop()

	log.Println("📊 Iniciando monitoreo de Procesos")

	for {
		select {
		case <-ticker.C:
			procesosInfo, err := ma.readProcesosInfo()
			if err != nil {
				ma.channels.ErrorChan <- fmt.Errorf("error leyendo Procesos: %v", err)
				continue
			}

			// Actualizar inmediatamente los datos en memoria
			ma.metricsMutex.Lock()
			ma.latestProcesos = procesosInfo
			ma.hasProcesosData = true
			ma.metricsMutex.Unlock()

			select {
			case ma.channels.ProcesosChan <- procesosInfo:
				log.Printf("🔄 Procesos: %d corriendo, %d total, %d durmiendo, %d zombie, %d parados",
					procesosInfo.ProcesosCorriendo, procesosInfo.TotalProcesos,
					procesosInfo.ProcesosDurmiendo, procesosInfo.ProcesosZombie,
					procesosInfo.ProcesosParados)
			default:
				log.Println("⚠️ Canal Procesos lleno, descartando datos")
			}

		case <-ma.channels.StopChan:
			log.Println("🛑 Deteniendo monitoreo de Procesos")
			return
		}
	}
}

// CAMBIO CLAVE: El combinador ahora solo escucha por cambios en los datos,
// no tiene su propio ticker
func (ma *MonitoringAgent) combineMetrics() {
	defer ma.wg.Done()

	log.Println("🔄 Iniciando combinador de métricas")

	// Variables para tracking de último envío
	var lastSentTime time.Time
	minSendInterval := ma.config.MonitorInterval

	for {
		select {
		case ramInfo := <-ma.channels.RAMChan:
			// Los datos ya se actualizaron en la goroutine de monitoreo
			_ = ramInfo
			ma.tryGenerateMetrics(&lastSentTime, minSendInterval)

		case cpuInfo := <-ma.channels.CPUChan:
			// Los datos ya se actualizaron en la goroutine de monitoreo
			_ = cpuInfo
			ma.tryGenerateMetrics(&lastSentTime, minSendInterval)

		case procesosInfo := <-ma.channels.ProcesosChan:
			// Los datos ya se actualizaron en la goroutine de monitoreo
			_ = procesosInfo
			ma.tryGenerateMetrics(&lastSentTime, minSendInterval)

		case <-ma.channels.StopChan:
			log.Println("🛑 Deteniendo combinador de métricas")
			return
		}
	}
}

// Función helper para generar métricas solo cuando es apropiado
func (ma *MonitoringAgent) tryGenerateMetrics(lastSentTime *time.Time, minInterval time.Duration) {
	// Solo generar si han pasado suficiente tiempo desde el último envío
	if time.Since(*lastSentTime) < minInterval {
		return
	}

	// Solo generar métricas si tenemos todos los datos
	ma.metricsMutex.RLock()
	hasAllData := ma.hasRAMData && ma.hasCPUData && ma.hasProcesosData

	if !hasAllData {
		ma.metricsMutex.RUnlock()
		return
	}

	// Crear SystemMetrics con los nombres de campo exactos requeridos
	combinedMetrics := SystemMetrics{
		// RAM fields - NOMBRES EXACTOS
		TotalRAM:      ma.latestRAM.Total,
		RAMLibre:      ma.latestRAM.Libre,
		UsoRAM:        ma.latestRAM.Uso,
		PorcentajeRAM: ma.latestRAM.Porcentaje,

		// CPU fields - NOMBRES EXACTOS
		PorcentajeCPUUso:   ma.latestCPU.PorcentajeUso,
		PorcentajeCPULibre: ma.latestCPU.PorcentajeLibre,

		// Procesos fields (sin cambios)
		ProcesosCorriendo: ma.latestProcesos.ProcesosCorriendo,
		TotalProcesos:     ma.latestProcesos.TotalProcesos,
		ProcesosDurmiendo: ma.latestProcesos.ProcesosDurmiendo,
		ProcesosZombie:    ma.latestProcesos.ProcesosZombie,
		ProcesosParados:   ma.latestProcesos.ProcesosParados,

		// Timestamp
		Hora: time.Now().Format("2006-01-02 15:04:05"),
	}
	ma.metricsMutex.RUnlock()

	select {
	case ma.channels.MetricsChan <- combinedMetrics:
		*lastSentTime = time.Now()
		log.Printf("📊 Métricas combinadas generadas: RAM %d%%, CPU %d%%, Procesos %d",
			combinedMetrics.PorcentajeRAM, combinedMetrics.PorcentajeCPUUso,
			combinedMetrics.TotalProcesos)
	default:
		log.Println("⚠️ Canal métricas lleno, descartando datos")
	}
}

// Goroutine para enviar métricas combinadas a la API
func (ma *MonitoringAgent) sendMetricsToAPI() {
	defer ma.wg.Done()

	log.Println("🌐 Iniciando enviador de métricas a API")

	for {
		select {
		case metrics := <-ma.channels.MetricsChan:
			// Debug - mostrar JSON que se va a enviar
			if debugData, err := json.MarshalIndent(metrics, "", "  "); err == nil {
				log.Printf("🔍 JSON enviando a API:\n%s", string(debugData))
			}

			err := ma.client.SendMetrics(metrics)
			if err != nil {
				ma.channels.ErrorChan <- fmt.Errorf("error enviando métricas a API: %v", err)
			} else {
				log.Printf("✅ Métricas enviadas a API: RAM %d%%, CPU %d%%, Procesos %d - %s",
					metrics.PorcentajeRAM, metrics.PorcentajeCPUUso,
					metrics.TotalProcesos, metrics.Hora)
			}

		case <-ma.channels.StopChan:
			log.Println("🛑 Deteniendo enviador de métricas API")
			return
		}
	}
}

// Goroutine para manejar errores
func (ma *MonitoringAgent) handleErrors() {
	defer ma.wg.Done()

	log.Println("🚨 Iniciando manejador de errores")

	for {
		select {
		case err := <-ma.channels.ErrorChan:
			log.Printf("❌ Error: %v", err)
			// Aquí podrías implementar lógica adicional como reintentos,
			// alertas, etc.

		case <-ma.channels.StopChan:
			log.Println("🛑 Deteniendo manejador de errores")
			return
		}
	}
}
