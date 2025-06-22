package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Cliente para comunicarse con la API de Node.js
type APIClient struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
}

// Crear nuevo cliente API
func NewAPIClient(baseURL string, maxRetries int) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
			},
		},
		maxRetries: maxRetries,
	}
}

// Enviar métricas a la API con reintentos
func (c *APIClient) SendMetrics(metrics SystemMetrics) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("error serializando métricas: %v", err)
	}

	var lastErr error
	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		err := c.sendRequest(jsonData)
		if err == nil {
			return nil // Éxito
		}

		lastErr = err
		if attempt < c.maxRetries {
			waitTime := time.Duration(attempt) * 2 * time.Second
			log.Printf("⚠️ Intento %d/%d falló, reintentando en %v: %v",
				attempt, c.maxRetries, waitTime, err)
			time.Sleep(waitTime)
		}
	}

	return fmt.Errorf("falló después de %d intentos: %v", c.maxRetries, lastErr)
}

// Realizar la petición HTTP
func (c *APIClient) sendRequest(jsonData []byte) error {
	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creando request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "MonitoringAgent/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error enviando request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API respondió con código %d", resp.StatusCode)
	}

	return nil
}

// Verificar conectividad con la API
func (c *APIClient) HealthCheck() error {
	healthURL := c.baseURL + "/health"

	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("error creando request de health: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error en health check: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("API health check falló con código %d", resp.StatusCode)
	}

	return nil
}

// Enviar métricas individuales (alternativa)
func (c *APIClient) SendRAMMetrics(ram RAMInfo) error {
	jsonData, err := json.Marshal(map[string]interface{}{
		"type": "ram",
		"data": ram,
	})
	if err != nil {
		return fmt.Errorf("error serializando RAM: %v", err)
	}

	return c.sendRequest(jsonData)
}

func (c *APIClient) SendCPUMetrics(cpu CPUInfo) error {
	jsonData, err := json.Marshal(map[string]interface{}{
		"type": "cpu",
		"data": cpu,
	})
	if err != nil {
		return fmt.Errorf("error serializando CPU: %v", err)
	}

	return c.sendRequest(jsonData)
}

// Configurar timeouts personalizados
func (c *APIClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// Obtener estadísticas del cliente
func (c *APIClient) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"base_url":    c.baseURL,
		"max_retries": c.maxRetries,
		"timeout":     c.httpClient.Timeout,
	}
}
