-- Crear base de datos
CREATE DATABASE IF NOT EXISTS sistema_monitoreo CHARACTER SET utf8mb4;
USE sistema_monitoreo;

-- Tabla unificada para todas las métricas del sistema
CREATE TABLE IF NOT EXISTS metricas_sistema (
    id INT AUTO_INCREMENT PRIMARY KEY,
    memoria_total BIGINT NOT NULL,
    memoria_libre BIGINT NOT NULL,
    memoria_usada BIGINT NOT NULL,
    porcentaje_ram DECIMAL(5,2) NOT NULL,
    porcentaje_cpu_uso DECIMAL(5,2) NOT NULL,
    porcentaje_cpu_libre DECIMAL(5,2) NOT NULL,
    procesos_corriendo INT NOT NULL,
    total_procesos INT NOT NULL,
    procesos_durmiendo INT NOT NULL,
    procesos_zombie INT NOT NULL,
    procesos_parados INT NOT NULL,
    hora VARCHAR(20) NOT NULL,
    api VARCHAR(20) NOT NULL
) DEFAULT CHARSET=utf8mb4;

-- Crear usuario y permisos
CREATE USER IF NOT EXISTS 'user_monitoreo'@'%' IDENTIFIED BY 'Ingenieria2025.';
GRANT SELECT, INSERT, UPDATE, DELETE ON sistema_monitoreo.* TO 'user_monitoreo'@'%';
GRANT ALL PRIVILEGES ON sistema_monitoreo.* TO 'root'@'%';
FLUSH PRIVILEGES;

-- Mostrar tablas y estructura
SHOW TABLES;
DESCRIBE metricas_sistema;
-- Consultas de ejemplo
SELECT 'Datos de ejemplo - Sistema:' AS info;
SELECT * FROM metricas_sistema ORDER BY hora DESC LIMIT 3;

