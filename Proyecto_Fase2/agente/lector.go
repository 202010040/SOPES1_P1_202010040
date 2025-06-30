package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Leer información de RAM desde el módulo del kernel
func (ma *MonitoringAgent) readRAMInfo() (RAMInfo, error) {
	data, err := ioutil.ReadFile(ma.config.RAMProcFile)
	if err != nil {
		return RAMInfo{}, fmt.Errorf("no se pudo leer %s: %v", ma.config.RAMProcFile, err)
	}

	var ramInfo RAMInfo
	err = json.Unmarshal(data, &ramInfo)
	if err != nil {
		return RAMInfo{}, fmt.Errorf("error parsing JSON de RAM: %v", err)
	}

	return ramInfo, nil
}

// Leer información de CPU desde el módulo del kernel
func (ma *MonitoringAgent) readCPUInfo() (CPUInfo, error) {
	data, err := ioutil.ReadFile(ma.config.CPUProcFile)
	if err != nil {
		return CPUInfo{}, fmt.Errorf("no se pudo leer %s: %v", ma.config.CPUProcFile, err)
	}

	var cpuInfo CPUInfo
	err = json.Unmarshal(data, &cpuInfo)
	if err != nil {
		return CPUInfo{}, fmt.Errorf("error parsing JSON de CPU: %v", err)
	}

	// Calcular el porcentaje libre (no viene del módulo del kernel)
	cpuInfo.PorcentajeLibre = 100 - cpuInfo.PorcentajeUso

	return cpuInfo, nil
}

// Leer información de Procesos desde el módulo del kernel
func (ma *MonitoringAgent) readProcesosInfo() (ProcesosInfo, error) {
	data, err := ioutil.ReadFile(ma.config.ProcesosProcFile)
	if err != nil {
		return ProcesosInfo{}, fmt.Errorf("no se pudo leer %s: %v", ma.config.ProcesosProcFile, err)
	}

	var procesosInfo ProcesosInfo
	err = json.Unmarshal(data, &procesosInfo)
	if err != nil {
		return ProcesosInfo{}, fmt.Errorf("error parsing JSON de Procesos: %v", err)
	}

	return procesosInfo, nil
}

// Función de utilidad para verificar si los archivos /proc existen
func (ma *MonitoringAgent) CheckProcFiles() error {
	// Verificar archivo de RAM
	if _, err := ioutil.ReadFile(ma.config.RAMProcFile); err != nil {
		return fmt.Errorf("archivo RAM no disponible (%s): %v. ¿Está el módulo del kernel cargado?",
			ma.config.RAMProcFile, err)
	}

	// Verificar archivo de CPU
	if _, err := ioutil.ReadFile(ma.config.CPUProcFile); err != nil {
		return fmt.Errorf("archivo CPU no disponible (%s): %v. ¿Está el módulo del kernel cargado?",
			ma.config.CPUProcFile, err)
	}

	// Verificar archivo de Procesos
	if _, err := ioutil.ReadFile(ma.config.ProcesosProcFile); err != nil {
		return fmt.Errorf("archivo Procesos no disponible (%s): %v. ¿Está el módulo del kernel cargado?",
			ma.config.ProcesosProcFile, err)
	}

	return nil
}

// Función para probar la lectura de los módulos (útil para debugging)
func (ma *MonitoringAgent) TestReading() {
	fmt.Println("🧪 Probando lectura de módulos del kernel...")

	// Probar RAM
	ramInfo, err := ma.readRAMInfo()
	if err != nil {
		fmt.Printf("❌ Error leyendo RAM: %v\n", err)
	} else {
		fmt.Printf("✅ RAM OK: Total=%d, Libre=%d, Uso=%d, Porcentaje=%d%%\n",
			ramInfo.Total, ramInfo.Libre, ramInfo.Uso, ramInfo.Porcentaje)
	}

	// Probar CPU
	cpuInfo, err := ma.readCPUInfo()
	if err != nil {
		fmt.Printf("❌ Error leyendo CPU: %v\n", err)
	} else {
		fmt.Printf("✅ CPU OK: Uso=%d%%, Libre=%d%%\n",
			cpuInfo.PorcentajeUso, cpuInfo.PorcentajeLibre)
	}

	// Probar Procesos
	procesosInfo, err := ma.readProcesosInfo()
	if err != nil {
		fmt.Printf("❌ Error leyendo Procesos: %v\n", err)
	} else {
		fmt.Printf("✅ Procesos OK: Corriendo=%d, Total=%d, Durmiendo=%d, Zombie=%d, Parados=%d\n",
			procesosInfo.ProcesosCorriendo, procesosInfo.TotalProcesos,
			procesosInfo.ProcesosDurmiendo, procesosInfo.ProcesosZombie,
			procesosInfo.ProcesosParados)
	}
}
