package main

import (
	"fmt"
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
	Timestamp  int64 `json:"timestamp"`
}

type CPUInfo struct {
	PorcentajeUso int64 `json:"porcentajeUso"`
	Timestamp     int64 `json:"timestamp"`
}

type SystemMetrics struct {
	RAM RAMInfo `json:"ram"`
	CPU CPUInfo `json:"cpu"`
}

// Canales para comunicación entre goroutines
type Channels struct {
	RAMChan   chan RAMInfo
	CPUChan   chan CPUInfo
	ErrorChan chan error
	StopChan  chan bool
}

// Configuración del agente
type Config struct {
	RAMAPIEndpoint  string
	CPUAPIEndpoint  string
	MonitorInterval time.Duration
	RAMProcFile     string
	CPUProcFile     string
	MaxRetries      int
}

// Agente de monitoreo principal
type MonitoringAgent struct {
	config    Config
	channels  Channels
	wg        sync.WaitGroup
	ramClient *APIClient
	cpuClient *APIClient
}

func main() {
	// Configuración del agente
	config := Config{
		RAMAPIEndpoint:  "http://localhost:3001/api/ram",
		CPUAPIEndpoint:  "http://localhost:3001/api/cpu",
		MonitorInterval: 5 * time.Second,
		RAMProcFile:     "/proc/ram_202010040",
		CPUProcFile:     "/proc/cpu_202010040",
		MaxRetries:      3,
	}

	// Inicializar canales
	channels := Channels{
		RAMChan:   make(chan RAMInfo, 10),
		CPUChan:   make(chan CPUInfo, 10),
		ErrorChan: make(chan error, 10),
		StopChan:  make(chan bool, 5),
	}

	// Crear clientes API separados
	ramClient := NewAPIClient(config.RAMAPIEndpoint, config.MaxRetries)
	cpuClient := NewAPIClient(config.CPUAPIEndpoint, config.MaxRetries)

	// Crear agente de monitoreo
	agent := &MonitoringAgent{
		config:    config,
		channels:  channels,
		ramClient: ramClient,
		cpuClient: cpuClient,
	}

	log.Println("🚀 Iniciando Agente de Monitoreo de Sistema")
	log.Printf("📊 Intervalo de monitoreo: %v", config.MonitorInterval)
	log.Printf("🔗 RAM API Endpoint: %s", config.RAMAPIEndpoint)
	log.Printf("🔗 CPU API Endpoint: %s", config.CPUAPIEndpoint)

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

// Iniciar todas las goroutines del agente
func (ma *MonitoringAgent) Start() {
	log.Println("🔄 Iniciando goroutines de monitoreo...")

	// Iniciar monitores
	ma.wg.Add(1)
	go ma.monitorRAM()

	ma.wg.Add(1)
	go ma.monitorCPU()

	// Iniciar enviadores de datos a API (separados)
	ma.wg.Add(1)
	go ma.sendRAMToAPI()

	ma.wg.Add(1)
	go ma.sendCPUToAPI()

	// Iniciar manejador de errores
	ma.wg.Add(1)
	go ma.handleErrors()

	log.Println("✅ Todas las goroutines iniciadas correctamente")
}

// Detener todas las goroutines
func (ma *MonitoringAgent) Stop() {
	log.Println("🔄 Deteniendo goroutines...")

	// Enviar señal de stop a todas las goroutines
	for i := 0; i < 5; i++ {
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

			select {
			case ma.channels.CPUChan <- cpuInfo:
				log.Printf("⚡ CPU: %d%% uso", cpuInfo.PorcentajeUso)
			default:
				log.Println("⚠️ Canal CPU lleno, descartando datos")
			}

		case <-ma.channels.StopChan:
			log.Println("🛑 Deteniendo monitoreo de CPU")
			return
		}
	}
}

// Goroutine para enviar datos de RAM a la API
func (ma *MonitoringAgent) sendRAMToAPI() {
	defer ma.wg.Done()

	log.Println("🌐 Iniciando enviador de datos de RAM a API")

	for {
		select {
		case ramInfo := <-ma.channels.RAMChan:
			err := ma.ramClient.SendRAMMetrics(ramInfo)
			if err != nil {
				ma.channels.ErrorChan <- fmt.Errorf("error enviando RAM a API: %v", err)
			} else {
				log.Printf("✅ Métricas de RAM enviadas a API: %d%% (%d/%d KB)",
					ramInfo.Porcentaje, ramInfo.Uso, ramInfo.Total)
			}

		case <-ma.channels.StopChan:
			log.Println("🛑 Deteniendo enviador de RAM API")
			return
		}
	}
}

// Goroutine para enviar datos de CPU a la API
func (ma *MonitoringAgent) sendCPUToAPI() {
	defer ma.wg.Done()

	log.Println("🌐 Iniciando enviador de datos de CPU a API")

	for {
		select {
		case cpuInfo := <-ma.channels.CPUChan:
			err := ma.cpuClient.SendCPUMetrics(cpuInfo)
			if err != nil {
				ma.channels.ErrorChan <- fmt.Errorf("error enviando CPU a API: %v", err)
			} else {
				log.Printf("✅ Métricas de CPU enviadas a API: %d%% uso",
					cpuInfo.PorcentajeUso)
			}

		case <-ma.channels.StopChan:
			log.Println("🛑 Deteniendo enviador de CPU API")
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
