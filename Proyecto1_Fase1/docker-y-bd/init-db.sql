-- Crear base de datos
CREATE DATABASE IF NOT EXISTS sistema_monitoreo 
CHARACTER SET utf8mb4;

USE sistema_monitoreo;

-- Tabla para métricas de RAM
CREATE TABLE IF NOT EXISTS tabla_ram (
    id INT AUTO_INCREMENT PRIMARY KEY,
    memoria_total BIGINT NOT NULL,
    memoria_libre BIGINT NOT NULL,
    memoria_usada BIGINT NOT NULL,
    porcentaje_uso DECIMAL(5,2) NOT NULL,
    fecha TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

-- Tabla para métricas de CPU
CREATE TABLE IF NOT EXISTS tabla_cpu (
    id INT AUTO_INCREMENT PRIMARY KEY,
    porcentaje_cpu DECIMAL(5,2) NOT NULL,
    fecha TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

-- Crear usuario para la aplicación
CREATE USER IF NOT EXISTS 'user_monitoreo'@'%' IDENTIFIED BY 'Ingenieria2025.';
GRANT SELECT, INSERT, UPDATE, DELETE ON sistema_monitoreo.* TO 'user_monitoreo'@'%';

-- Permisos para root desde cualquier host
GRANT ALL PRIVILEGES ON sistema_monitoreo.* TO 'root'@'%';
FLUSH PRIVILEGES;

-- Datos de ejemplo RAM
INSERT INTO tabla_ram (memoria_total, memoria_libre, memoria_usada, porcentaje_uso) VALUES
(8589934592, 4294967296, 4294967296, 50.00),
(8589934592, 3865470976, 4724463616, 55.00),
(8589934592, 3435973632, 5153960960, 60.00);

-- Datos de ejemplo CPU
INSERT INTO tabla_cpu (porcentaje_cpu) VALUES
(25.50),
(30.00),
(35.25),
(40.75),
(45.00);

-- Mostrar tablas y estructura
SHOW TABLES;
DESCRIBE tabla_ram;
DESCRIBE tabla_cpu;

-- Consultas de ejemplo
SELECT 'Datos de ejemplo - RAM:' AS info;
SELECT * FROM tabla_ram ORDER BY fecha DESC LIMIT 3;

SELECT 'Datos de ejemplo - CPU:' AS info;
SELECT * FROM tabla_cpu ORDER BY fecha DESC LIMIT 3;
