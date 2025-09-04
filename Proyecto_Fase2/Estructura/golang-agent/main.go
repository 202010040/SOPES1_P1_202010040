package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
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
	PorcentajeLibre int64 `json:"porcentaje_libre"`
}

type ProcesosInfo struct {
	ProcesosCorriendo int64 `json:"procesos_corriendo"`
	TotalProcesos     int64 `json:"total_procesos"`
	ProcesosDurmiendo int64 `json:"procesos_durmiendo"`
	ProcesosZombie    int64 `json:"procesos_zombie"`
	ProcesosParados   int64 `json:"procesos_parados"`
}

// Estructura combinada para todas las métricas
type SystemMetrics struct {
	// RAM fields
	TotalRAM      int64 `json:"total_ram"`
	RAMLibre      int64 `json:"ram_libre"`
	UsoRAM        int64 `json:"uso_ram"`
	PorcentajeRAM int64 `json:"porcentaje_ram"`

	// CPU fields
	PorcentajeCPUUso   int64 `json:"porcentaje_cpu_uso"`
	PorcentajeCPULibre int64 `json:"porcentaje_cpu_libre"`

	// Procesos fields
	ProcesosCorriendo int64 `json:"procesos_corriendo"`
	TotalProcesos     int64 `json:"total_procesos"`
	ProcesosDurmiendo int64 `json:"procesos_durmiendo"`
	ProcesosZombie    int64 `json:"procesos_zombie"`
	ProcesosParados   int64 `json:"procesos_parados"`

	// Timestamp
	Hora string `json:"hora"`
}

// Configuración de la API
type Config struct {
	Port             string
	RAMProcFile      string
	CPUProcFile      string
	ProcesosProcFile string
}

// API Server principal
type MonitoringAPI struct {
	config Config
	router *mux.Router
}

func main() {
	// Obtener configuración desde variables de entorno
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "3001"
	}

	// Configuración de la API
	config := Config{
		Port:             port,
		RAMProcFile:      "/proc/ram_202010040",
		CPUProcFile:      "/proc/cpu_202010040",
		ProcesosProcFile: "/proc/procesos_202010040",
	}

	// Crear API
	api := NewMonitoringAPI(config)

	log.Printf("🚀 Iniciando API de Monitoreo del Sistema en puerto %s", port)
	log.Printf("📊 Endpoint disponible:")
	log.Printf("   GET /api/metrics - Todas las métricas del sistema")

	// Verificar archivos /proc
	if err := api.checkProcFiles(); err != nil {
		log.Fatalf("❌ Error verificando archivos /proc: %v", err)
	}

	// Iniciar servidor
	log.Printf("🌐 API disponible en http://localhost:%s/api/metrics", port)
	log.Fatal(http.ListenAndServe(":"+port, api.router))
}

// Crear nueva instancia de la API
func NewMonitoringAPI(config Config) *MonitoringAPI {
	api := &MonitoringAPI{
		config: config,
		router: mux.NewRouter(),
	}

	api.setupRoutes()
	return api
}

// Configurar rutas de la API
func (api *MonitoringAPI) setupRoutes() {
	// Middleware para CORS y logging
	api.router.Use(api.corsMiddleware)
	api.router.Use(api.loggingMiddleware)

	// Ruta principal
	api.router.HandleFunc("/api/metrics", api.getAllMetrics).Methods("GET")
}

// Handler para obtener todas las métricas
func (api *MonitoringAPI) getAllMetrics(w http.ResponseWriter, r *http.Request) {
	// Leer todas las métricas
	ramInfo, err := api.readRAMInfo()
	if err != nil {
		api.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error leyendo RAM: %v", err))
		return
	}

	cpuInfo, err := api.readCPUInfo()
	if err != nil {
		api.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error leyendo CPU: %v", err))
		return
	}

	procesosInfo, err := api.readProcesosInfo()
	if err != nil {
		api.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error leyendo procesos: %v", err))
		return
	}

	// Combinar métricas
	metrics := SystemMetrics{
		// RAM fields
		TotalRAM:      ramInfo.Total,
		RAMLibre:      ramInfo.Libre,
		UsoRAM:        ramInfo.Uso,
		PorcentajeRAM: ramInfo.Porcentaje,

		// CPU fields
		PorcentajeCPUUso:   cpuInfo.PorcentajeUso,
		PorcentajeCPULibre: cpuInfo.PorcentajeLibre,

		// Procesos fields
		ProcesosCorriendo: procesosInfo.ProcesosCorriendo,
		TotalProcesos:     procesosInfo.TotalProcesos,
		ProcesosDurmiendo: procesosInfo.ProcesosDurmiendo,
		ProcesosZombie:    procesosInfo.ProcesosZombie,
		ProcesosParados:   procesosInfo.ProcesosParados,

		// Timestamp
		Hora: time.Now().Format("2006-01-02 15:04:05"),
	}

	api.sendJSON(w, http.StatusOK, metrics)
}

// Middleware para CORS
func (api *MonitoringAPI) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Middleware para logging
func (api *MonitoringAPI) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf("📝 %s %s - %v", r.Method, r.RequestURI, time.Since(start))
	})
}

// Función para enviar respuesta JSON
func (api *MonitoringAPI) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("❌ Error encoding JSON: %v", err)
	}
}

// Función para enviar error
func (api *MonitoringAPI) sendError(w http.ResponseWriter, statusCode int, message string) {
	errorResponse := map[string]interface{}{
		"error":     true,
		"message":   message,
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}

	api.sendJSON(w, statusCode, errorResponse)
}

// Leer información de RAM
func (api *MonitoringAPI) readRAMInfo() (RAMInfo, error) {
	data, err := ioutil.ReadFile(api.config.RAMProcFile)
	if err != nil {
		return RAMInfo{}, fmt.Errorf("no se pudo leer %s: %v", api.config.RAMProcFile, err)
	}

	var ramInfo RAMInfo
	err = json.Unmarshal(data, &ramInfo)
	if err != nil {
		return RAMInfo{}, fmt.Errorf("error parsing JSON de RAM: %v", err)
	}

	return ramInfo, nil
}

// Leer información de CPU
func (api *MonitoringAPI) readCPUInfo() (CPUInfo, error) {
	data, err := ioutil.ReadFile(api.config.CPUProcFile)
	if err != nil {
		return CPUInfo{}, fmt.Errorf("no se pudo leer %s: %v", api.config.CPUProcFile, err)
	}

	var cpuInfo CPUInfo
	err = json.Unmarshal(data, &cpuInfo)
	if err != nil {
		return CPUInfo{}, fmt.Errorf("error parsing JSON de CPU: %v", err)
	}

	// Calcular porcentaje libre
	cpuInfo.PorcentajeLibre = 100 - cpuInfo.PorcentajeUso

	return cpuInfo, nil
}

// Leer información de procesos
func (api *MonitoringAPI) readProcesosInfo() (ProcesosInfo, error) {
	data, err := ioutil.ReadFile(api.config.ProcesosProcFile)
	if err != nil {
		return ProcesosInfo{}, fmt.Errorf("no se pudo leer %s: %v", api.config.ProcesosProcFile, err)
	}

	var procesosInfo ProcesosInfo
	err = json.Unmarshal(data, &procesosInfo)
	if err != nil {
		return ProcesosInfo{}, fmt.Errorf("error parsing JSON de procesos: %v", err)
	}

	return procesosInfo, nil
}

// Verificar si los archivos /proc existen
func (api *MonitoringAPI) checkProcFiles() error {
	files := []string{
		api.config.RAMProcFile,
		api.config.CPUProcFile,
		api.config.ProcesosProcFile,
	}

	for _, file := range files {
		if _, err := ioutil.ReadFile(file); err != nil {
			return fmt.Errorf("archivo %s no disponible: %v. ¿Está el módulo del kernel cargado?", file, err)
		}
	}

	return nil
}
