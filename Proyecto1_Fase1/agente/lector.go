package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
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

	// Agregar timestamp
	ramInfo.Timestamp = time.Now().Unix()

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

	// Agregar timestamp
	cpuInfo.Timestamp = time.Now().Unix()

	return cpuInfo, nil
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
		fmt.Printf("✅ RAM OK: %+v\n", ramInfo)
	}

	// Probar CPU
	cpuInfo, err := ma.readCPUInfo()
	if err != nil {
		fmt.Printf("❌ Error leyendo CPU: %v\n", err)
	} else {
		fmt.Printf("✅ CPU OK: %+v\n", cpuInfo)
	}
}
