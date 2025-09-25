package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Estructuras para parsear los datos del kernel
type SystemInfo struct {
	Timestamp      string         `json:"timestamp"`
	System         SystemDetails  `json:"system"`
	Memory         MemoryInfo     `json:"memory"`
	ProcessSummary ProcessSummary `json:"process_summary"`
	Processes      []Process      `json:"processes"`
}

type SystemDetails struct {
	Kernel       string `json:"kernel"`
	Architecture string `json:"architecture"`
	Hostname     string `json:"hostname"`
}

type ContainerInfo struct {
	Timestamp  string      `json:"timestamp"`
	Memory     MemoryInfo  `json:"memory"`
	Containers []Container `json:"containers"`
}

type MemoryInfo struct {
	TotalKB int64 `json:"total_kb"`
	FreeKB  int64 `json:"free_kb"`
	UsedKB  int64 `json:"used_kb"`
}

type ProcessSummary struct {
	Total    int `json:"total"`
	Running  int `json:"running"`
	Sleeping int `json:"sleeping"`
	Other    int `json:"other"`
}

type Process struct {
	PID           int    `json:"pid"`
	PPID          int    `json:"ppid"`
	Name          string `json:"name"`
	Cmdline       string `json:"cmdline"`
	VSZKB         int64  `json:"vsz_kb"`
	RSSKB         int64  `json:"rss_kb"`
	MemoryPercent int    `json:"memory_percent"`
	CPUPercent    int    `json:"cpu_percent"`
	State         string `json:"state"`
}

type Container struct {
	PID           int    `json:"pid"`
	PPID          int    `json:"ppid"`
	Name          string `json:"name"`
	Cmdline       string `json:"cmdline"`
	VSZKB         int64  `json:"vsz_kb"`
	RSSKB         int64  `json:"rss_kb"`
	MemoryPercent int    `json:"memory_percent"`
	CPUPercent    int    `json:"cpu_percent"`
}

// Configuración del daemon
type DaemonConfig struct {
	ContainerInfoPath      string
	SystemInfoPath         string
	DBPath                 string
	LoopInterval           time.Duration
	MinLowConsumption      int
	MinHighConsumption     int
	MemoryThreshold        int64 // KB
	CPUThreshold           int
	CreateContainersScript string
	CleanContainersScript  string
	KernelModulesScript    string
	BashDir                string
}

type Daemon struct {
	config         *DaemonConfig
	db             *sql.DB
	grafanaStarted bool
	cronJobActive  bool
}

func main() {
	// Obtener directorio actual del proyecto
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error obteniendo directorio actual: %v", err)
	}

	// El directorio raíz del proyecto (subir un nivel desde /Daemon)
	projectRoot := filepath.Dir(currentDir)

	config := &DaemonConfig{
		ContainerInfoPath:      "/proc/continfo_so1_202100265",
		SystemInfoPath:         "/proc/sysinfo_so1_202100265",
		DBPath:                 "./monitoring.db",
		LoopInterval:           20 * time.Second,
		MinLowConsumption:      3,
		MinHighConsumption:     2,
		MemoryThreshold:        30000, // 30MB en KB
		CPUThreshold:           80,    // 80%
		CreateContainersScript: filepath.Join(projectRoot, "Bash", "create_containers.sh"),
		CleanContainersScript:  filepath.Join(projectRoot, "Bash", "clean_containers.sh"),
		KernelModulesScript:    filepath.Join(projectRoot, "load_kernel_modules.sh"),
		BashDir:                filepath.Join(projectRoot, "Bash"),
	}

	daemon := &Daemon{
		config: config,
	}

	// Verificar que los scripts existen
	if err := daemon.validateScripts(); err != nil {
		log.Fatalf("Error validando scripts: %v", err)
	}

	// Inicializar la base de datos
	if err := daemon.initDB(); err != nil {
		log.Fatalf("Error inicializando la base de datos: %v", err)
	}
	defer daemon.db.Close()

	// Manejar señales para limpieza
	daemon.setupSignalHandlers()

	// Iniciar el daemon
	daemon.start()
}

func (d *Daemon) validateScripts() error {
	scripts := []string{
		d.config.CreateContainersScript,
		d.config.CleanContainersScript,
	}

	for _, script := range scripts {
		if _, err := os.Stat(script); os.IsNotExist(err) {
			return fmt.Errorf("script no encontrado: %s", script)
		}

		// Hacer el script ejecutable
		if err := os.Chmod(script, 0755); err != nil {
			log.Printf("Advertencia: No se pudo hacer ejecutable %s: %v", script, err)
		}
	}

	return nil
}

func (d *Daemon) initDB() error {
	var err error
	d.db, err = sql.Open("sqlite3", d.config.DBPath)
	if err != nil {
		return err
	}

	// Crear tablas
	tables := []string{
		`CREATE TABLE IF NOT EXISTS system_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			total_memory_kb INTEGER,
			free_memory_kb INTEGER,
			used_memory_kb INTEGER,
			total_processes INTEGER,
			running_processes INTEGER,
			sleeping_processes INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS container_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			pid INTEGER,
			name TEXT,
			cmdline TEXT,
			vsz_kb INTEGER,
			rss_kb INTEGER,
			memory_percent INTEGER,
			cpu_percent INTEGER,
			status TEXT DEFAULT 'active'
		)`,
		`CREATE TABLE IF NOT EXISTS container_actions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			action TEXT,
			container_pid INTEGER,
			container_name TEXT,
			reason TEXT
		)`,
	}

	for _, table := range tables {
		if _, err := d.db.Exec(table); err != nil {
			return err
		}
	}

	return nil
}

func (d *Daemon) start() {
	log.Println("Iniciando daemon de monitoreo...")

	// 1. Ejecutar script de limpieza inicial
	if err := d.executeCleanContainers(); err != nil {
		log.Printf("Error en limpieza inicial: %v", err)
	}

	// 2. Crear contenedor de Grafana
	if err := d.startGrafana(); err != nil {
		log.Printf("Error iniciando Grafana: %v", err)
	}

	// 3. Iniciar cronjob
	if err := d.startCronJob(); err != nil {
		log.Printf("Error iniciando cronjob: %v", err)
	}

	// 4. Construir imágenes Docker si no existen
	if err := d.buildDockerImages(); err != nil {
		log.Printf("Error construyendo imágenes Docker: %v", err)
	}

	// 5. Cargar módulos de kernel
	if err := d.loadKernelModules(); err != nil {
		log.Printf("Error cargando módulos de kernel: %v", err)
	}

	// 5. Crear contenedores iniciales
	if err := d.executeCreateContainers(); err != nil {
		log.Printf("Error creando contenedores iniciales: %v", err)
	}

	// 6. Loop principal
	d.mainLoop()
}

func (d *Daemon) executeCreateContainers() error {
	log.Println("Ejecutando script de creación de contenedores...")

	cmd := exec.Command("bash", d.config.CreateContainersScript)
	cmd.Dir = d.config.BashDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Output del script create_containers: %s", string(output))
		return fmt.Errorf("error ejecutando create_containers.sh: %v", err)
	}

	log.Printf("Script create_containers ejecutado exitosamente: %s", string(output))
	return nil
}

func (d *Daemon) executeCleanContainers() error {
	log.Println("Ejecutando script de limpieza de contenedores...")

	cmd := exec.Command("bash", d.config.CleanContainersScript)
	cmd.Dir = d.config.BashDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Output del script clean_containers: %s", string(output))
		return fmt.Errorf("error ejecutando clean_containers.sh: %v", err)
	}

	log.Printf("Script clean_containers ejecutado exitosamente: %s", string(output))
	return nil
}

func (d *Daemon) startGrafana() error {
	log.Println("Iniciando Grafana...")

	// Verificar si ya existe
	cmd := exec.Command("docker", "ps", "-q", "-f", "name=grafana-monitoring")
	if output, _ := cmd.Output(); len(strings.TrimSpace(string(output))) > 0 {
		log.Println("Grafana ya está ejecutándose")
		d.grafanaStarted = true
		return nil
	}

	// Obtener directorio del proyecto (un nivel arriba del Daemon)
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error obteniendo directorio actual: %v", err)
	}

	projectRoot := filepath.Dir(currentDir)

	// Intentar primero con docker compose (nuevo comando)
	cmd = exec.Command("docker", "compose", "up", "-d", "grafana")
	cmd.Dir = projectRoot

	if err := cmd.Run(); err != nil {
		log.Printf("Docker compose falló, intentando con docker-compose: %v", err)

		// Intentar con docker-compose (comando legacy)
		cmd = exec.Command("docker-compose", "up", "-d", "grafana")
		cmd.Dir = projectRoot

		if err := cmd.Run(); err != nil {
			log.Printf("Docker-compose falló, intentando con docker run: %v", err)
			return d.startGrafanaWithDocker()
		}
	}

	d.grafanaStarted = true
	log.Println("Grafana iniciado con compose en puerto 3000")
	return nil
}

func (d *Daemon) startGrafanaWithDocker() error {
	log.Println("Iniciando contenedor de Grafana con docker run...")

	// Primero detener cualquier contenedor existente con el mismo nombre
	exec.Command("docker", "stop", "grafana-monitoring").Run()
	exec.Command("docker", "rm", "grafana-monitoring").Run()

	cmd := exec.Command("docker", "run", "-d",
		"--name", "grafana-monitoring",
		"-p", "3000:3000",
		"-e", "GF_SECURITY_ADMIN_PASSWORD=admin",
		"-v", "grafana-data:/var/lib/grafana",
		"--restart", "unless-stopped",
		"grafana/grafana:latest")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error detallado de Docker: %s", string(output))
		return fmt.Errorf("error creando contenedor de Grafana: %v", err)
	}

	log.Printf("Grafana iniciado exitosamente: %s", string(output))
	return nil
}

func (d *Daemon) startCronJob() error {
	log.Println("Configurando cronjob para creación de contenedores...")

	// Crear entrada de cron que ejecute el script cada minuto
	cronEntry := fmt.Sprintf("* * * * * %s", d.config.CreateContainersScript)

	cmd := exec.Command("bash", "-c", fmt.Sprintf("(crontab -l 2>/dev/null; echo '%s') | crontab -", cronEntry))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error configurando cronjob: %v", err)
	}

	d.cronJobActive = true
	log.Println("Cronjob configurado correctamente")
	return nil
}

func (d *Daemon) loadKernelModules() error {
	log.Println("Cargando módulos de kernel...")

	// Si existe el script de módulos, ejecutarlo
	if _, err := os.Stat(d.config.KernelModulesScript); err == nil {
		cmd := exec.Command("bash", d.config.KernelModulesScript)
		if err := cmd.Run(); err != nil {
			log.Printf("Error ejecutando script de módulos: %v", err)
		}
	}

	log.Println("Módulos de kernel verificados")
	return nil
}

func (d *Daemon) mainLoop() {
	log.Printf("Iniciando loop principal (cada %v)...", d.config.LoopInterval)

	ticker := time.NewTicker(d.config.LoopInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.processIteration()
		}
	}
}

func (d *Daemon) processIteration() {
	log.Println("=== Nueva iteración ===")

	// Leer información del sistema
	systemInfo, err := d.readSystemInfo()
	if err != nil {
		log.Printf("Error leyendo información del sistema: %v", err)
		log.Printf("Verificando si el archivo existe: %s", d.config.SystemInfoPath)
		if _, statErr := os.Stat(d.config.SystemInfoPath); os.IsNotExist(statErr) {
			log.Printf("ADVERTENCIA: Archivo de sistema no existe. ¿Están cargados los módulos de kernel?")
		}
		return
	}

	// Leer información de contenedores
	containerInfo, err := d.readContainerInfo()
	if err != nil {
		log.Printf("Error leyendo información de contenedores: %v", err)
		log.Printf("Verificando si el archivo existe: %s", d.config.ContainerInfoPath)
		if _, statErr := os.Stat(d.config.ContainerInfoPath); os.IsNotExist(statErr) {
			log.Printf("ADVERTENCIA: Archivo de contenedores no existe. ¿Están cargados los módulos de kernel?")
		}
		return
	}

	// Almacenar métricas en la base de datos
	d.storeSystemMetrics(systemInfo)
	d.storeContainerMetrics(containerInfo)

	// Analizar y gestionar contenedores
	d.analyzeAndManageContainers(containerInfo)

	log.Printf("Memoria total: %d KB, Libre: %d KB, Contenedores activos: %d",
		containerInfo.Memory.TotalKB, containerInfo.Memory.FreeKB, len(containerInfo.Containers))
}

func (d *Daemon) readSystemInfo() (*SystemInfo, error) {
	data, err := ioutil.ReadFile(d.config.SystemInfoPath)
	if err != nil {
		return nil, err
	}

	var info SystemInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func (d *Daemon) readContainerInfo() (*ContainerInfo, error) {
	data, err := ioutil.ReadFile(d.config.ContainerInfoPath)
	if err != nil {
		return nil, err
	}

	var info ContainerInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func (d *Daemon) storeSystemMetrics(info *SystemInfo) {
	query := `INSERT INTO system_metrics 
		(total_memory_kb, free_memory_kb, used_memory_kb, total_processes, running_processes, sleeping_processes)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query,
		info.Memory.TotalKB,
		info.Memory.FreeKB,
		info.Memory.UsedKB,
		info.ProcessSummary.Total,
		info.ProcessSummary.Running,
		info.ProcessSummary.Sleeping)

	if err != nil {
		log.Printf("Error guardando métricas del sistema: %v", err)
	}
}

func (d *Daemon) storeContainerMetrics(info *ContainerInfo) {
	for _, container := range info.Containers {
		query := `INSERT INTO container_metrics 
			(pid, name, cmdline, vsz_kb, rss_kb, memory_percent, cpu_percent)
			VALUES (?, ?, ?, ?, ?, ?, ?)`

		_, err := d.db.Exec(query,
			container.PID,
			container.Name,
			container.Cmdline,
			container.VSZKB,
			container.RSSKB,
			container.MemoryPercent,
			container.CPUPercent)

		if err != nil {
			log.Printf("Error guardando métricas del contenedor %s: %v", container.Name, err)
		}
	}
}

func (d *Daemon) analyzeAndManageContainers(info *ContainerInfo) {
	// Filtrar contenedores (excluir Grafana)
	containers := d.filterContainers(info.Containers)

	// Clasificar contenedores
	lowConsumption, highConsumption := d.classifyContainers(containers)

	log.Printf("Contenedores de bajo consumo: %d, alto consumo: %d", len(lowConsumption), len(highConsumption))

	// Verificar y ajustar según restricciones
	d.enforceContainerLimits(lowConsumption, highConsumption)

	// Si necesitamos más contenedores, crear algunos
	totalNeeded := d.config.MinLowConsumption + d.config.MinHighConsumption
	totalCurrent := len(lowConsumption) + len(highConsumption)

	if totalCurrent < totalNeeded {
		log.Printf("Se necesitan más contenedores. Actual: %d, Necesario: %d", totalCurrent, totalNeeded)
		if err := d.executeCreateContainers(); err != nil {
			log.Printf("Error creando contenedores adicionales: %v", err)
		}
	}
}

func (d *Daemon) filterContainers(containers []Container) []Container {
	var filtered []Container
	for _, container := range containers {
		// Excluir Grafana y otros servicios del sistema
		if !strings.Contains(strings.ToLower(container.Name), "grafana") &&
			!strings.Contains(strings.ToLower(container.Name), "containerd") &&
			!strings.Contains(strings.ToLower(container.Name), "dockerd") &&
			!strings.Contains(strings.ToLower(container.Cmdline), "grafana") {
			filtered = append(filtered, container)
		}
	}
	return filtered
}

func (d *Daemon) classifyContainers(containers []Container) ([]Container, []Container) {
	var low, high []Container

	for _, container := range containers {
		// Clasificar basado en consumo de memoria y CPU
		if container.RSSKB > d.config.MemoryThreshold || container.CPUPercent > d.config.CPUThreshold {
			high = append(high, container)
		} else {
			low = append(low, container)
		}
	}

	// Ordenar por consumo de recursos
	sort.Slice(low, func(i, j int) bool {
		return low[i].RSSKB > low[j].RSSKB
	})

	sort.Slice(high, func(i, j int) bool {
		return high[i].RSSKB > high[j].RSSKB
	})

	return low, high
}

func (d *Daemon) enforceContainerLimits(low, high []Container) {
	// Eliminar exceso de contenedores de bajo consumo
	if len(low) > d.config.MinLowConsumption {
		excess := low[d.config.MinLowConsumption:]
		for _, container := range excess {
			d.killContainer(container, "Exceso de contenedores de bajo consumo")
		}
	}

	// Eliminar exceso de contenedores de alto consumo
	if len(high) > d.config.MinHighConsumption {
		excess := high[d.config.MinHighConsumption:]
		for _, container := range excess {
			d.killContainer(container, "Exceso de contenedores de alto consumo")
		}
	}
}

func (d *Daemon) killContainer(container Container, reason string) {
	log.Printf("Eliminando contenedor: PID %d, Nombre: %s, Razón: %s", container.PID, container.Name, reason)

	// Buscar ID del contenedor por PID
	containerID, err := d.getContainerIDByPID(container.PID)
	if err != nil {
		log.Printf("Error obteniendo ID del contenedor: %v", err)
		return
	}

	if containerID != "" {
		// Detener y eliminar contenedor
		exec.Command("docker", "stop", containerID).Run()
		exec.Command("docker", "rm", containerID).Run()

		// Registrar acción
		d.logContainerAction("KILLED", container.PID, container.Name, reason)
	}
}

func (d *Daemon) getContainerIDByPID(pid int) (string, error) {
	cmd := exec.Command("docker", "ps", "-q", "--no-trunc")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, id := range containerIDs {
		if id == "" {
			continue
		}

		// Obtener PID del contenedor
		inspectCmd := exec.Command("docker", "inspect", "-f", "{{.State.Pid}}", id)
		pidOutput, err := inspectCmd.Output()
		if err != nil {
			continue
		}

		containerPID, err := strconv.Atoi(strings.TrimSpace(string(pidOutput)))
		if err != nil {
			continue
		}

		if containerPID == pid {
			return id, nil
		}
	}

	return "", fmt.Errorf("container not found for PID %d", pid)
}

func (d *Daemon) logContainerAction(action string, pid int, name, reason string) {
	query := `INSERT INTO container_actions (action, container_pid, container_name, reason)
		VALUES (?, ?, ?, ?)`

	_, err := d.db.Exec(query, action, pid, name, reason)
	if err != nil {
		log.Printf("Error registrando acción del contenedor: %v", err)
	}
}

func (d *Daemon) setupSignalHandlers() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Recibida señal de terminación, limpiando...")
		d.cleanup()
		os.Exit(0)
	}()
}

func (d *Daemon) buildDockerImages() error {
	log.Println("Verificando y construyendo imágenes Docker...")

	images := map[string]string{
		"high-cpu-image":        "high-cpu",
		"high-ram-image":        "high-ram",
		"low-consumption-image": "low-consumption",
	}

	for imageName, dirName := range images {
		// Verificar si la imagen ya existe
		cmd := exec.Command("docker", "images", "-q", imageName)
		if output, err := cmd.Output(); err == nil && len(strings.TrimSpace(string(output))) > 0 {
			log.Printf("Imagen %s ya existe", imageName)
			continue
		}

		log.Printf("Construyendo imagen %s...", imageName)

		// Construir la imagen
		dockerDir := filepath.Join(d.config.BashDir, "docker-images", dirName)
		cmd = exec.Command("docker", "build", "-t", imageName, ".")
		cmd.Dir = dockerDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Error construyendo %s: %v", imageName, err)
			log.Printf("Output: %s", string(output))
			continue
		}

		log.Printf("Imagen %s construida exitosamente", imageName)
	}

	return nil
}
func (d *Daemon) cleanup() {
	// Ejecutar script de limpieza de contenedores
	log.Println("Ejecutando limpieza de contenedores...")
	d.executeCleanContainers()

	// Eliminar cronjob
	if d.cronJobActive {
		cmd := exec.Command("bash", "-c", fmt.Sprintf("crontab -l | grep -v '%s' | crontab -", d.config.CreateContainersScript))
		cmd.Run()
		log.Println("Cronjob eliminado")
	}

	// Cerrar base de datos
	if d.db != nil {
		d.db.Close()
	}

	log.Println("Limpieza completada")
}
