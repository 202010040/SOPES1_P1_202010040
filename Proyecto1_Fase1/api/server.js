// server.js
const express = require('express');
const mysql = require('mysql2/promise');
const cors = require('cors');
const bodyParser = require('body-parser');

const app = express();
const PORT = process.env.PORT || 3001;

// Middleware
app.use(cors());
app.use(bodyParser.json());
app.use(express.json());

// Configuración de la base de datos
const dbConfig = {
  host: process.env.DB_HOST || 'mysql-db',
  user: process.env.DB_USER || 'root',
  password: process.env.DB_PASSWORD || 'password123',
  database: process.env.DB_NAME || 'monitoring_db',
  waitForConnections: true,
  connectionLimit: 10,
  queueLimit: 0
};

let pool;

// Inicializar conexión a la base de datos
async function initDatabase() {
  try {
    pool = mysql.createPool(dbConfig);
    
    // Crear tablas si no existen
    await createTables();
    console.log('Base de datos inicializada correctamente');
  } catch (error) {
    console.error('Error al inicializar la base de datos:', error);
    process.exit(1);
  }
}

// Crear tablas
async function createTables() {
  const createRamTable = `
    CREATE TABLE IF NOT EXISTS ram_metrics (
      id INT AUTO_INCREMENT PRIMARY KEY,
      total_memory BIGINT NOT NULL,
      free_memory BIGINT NOT NULL,
      used_memory BIGINT NOT NULL,
      percentage_used DECIMAL(5,2) NOT NULL,
      timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      INDEX idx_timestamp (timestamp)
    )
  `;

  const createCpuTable = `
    CREATE TABLE IF NOT EXISTS cpu_metrics (
      id INT AUTO_INCREMENT PRIMARY KEY,
      cpu_percentage DECIMAL(5,2) NOT NULL,
      processes INT NOT NULL,
      running INT NOT NULL,
      sleeping INT NOT NULL,
      zombie INT NOT NULL,
      stopped INT NOT NULL,
      timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      INDEX idx_timestamp (timestamp)
    )
  `;

  await pool.execute(createRamTable);
  await pool.execute(createCpuTable);
}

// RUTAS DE ESCRITURA (para el agente de monitoreo)

// Insertar métricas de RAM
app.post('/api/ram', async (req, res) => {
  try {
    const { total_memory, free_memory, used_memory, percentage_used } = req.body;
    
    if (!total_memory || !free_memory || !used_memory || percentage_used === undefined) {
      return res.status(400).json({ 
        error: 'Faltan campos requeridos: total_memory, free_memory, used_memory, percentage_used' 
      });
    }

    const query = `
      INSERT INTO ram_metrics (total_memory, free_memory, used_memory, percentage_used)
      VALUES (?, ?, ?, ?)
    `;
    
    const [result] = await pool.execute(query, [
      total_memory, 
      free_memory, 
      used_memory, 
      percentage_used
    ]);
    
    res.status(201).json({ 
      message: 'Métricas de RAM guardadas exitosamente',
      id: result.insertId 
    });
  } catch (error) {
    console.error('Error al guardar métricas de RAM:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});

// Insertar métricas de CPU
app.post('/api/cpu', async (req, res) => {
  try {
    const { cpu_percentage, processes, running, sleeping, zombie, stopped } = req.body;
    
    if (cpu_percentage === undefined || !processes || running === undefined || 
        sleeping === undefined || zombie === undefined || stopped === undefined) {
      return res.status(400).json({ 
        error: 'Faltan campos requeridos: cpu_percentage, processes, running, sleeping, zombie, stopped' 
      });
    }

    const query = `
      INSERT INTO cpu_metrics (cpu_percentage, processes, running, sleeping, zombie, stopped)
      VALUES (?, ?, ?, ?, ?, ?)
    `;
    
    const [result] = await pool.execute(query, [
      cpu_percentage, 
      processes, 
      running, 
      sleeping, 
      zombie, 
      stopped
    ]);
    
    res.status(201).json({ 
      message: 'Métricas de CPU guardadas exitosamente',
      id: result.insertId 
    });
  } catch (error) {
    console.error('Error al guardar métricas de CPU:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});

// RUTAS DE LECTURA (para el frontend)

// Obtener métricas recientes de RAM
app.get('/api/ram/recent', async (req, res) => {
  try {
    const limit = req.query.limit || 50;
    const query = `
      SELECT * FROM ram_metrics 
      ORDER BY timestamp DESC 
      LIMIT ?
    `;
    
    const [rows] = await pool.execute(query, [parseInt(limit)]);
    res.json(rows);
  } catch (error) {
    console.error('Error al obtener métricas de RAM:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});

// Obtener métricas recientes de CPU
app.get('/api/cpu/recent', async (req, res) => {
  try {
    const limit = req.query.limit || 50;
    const query = `
      SELECT * FROM cpu_metrics 
      ORDER BY timestamp DESC 
      LIMIT ?
    `;
    
    const [rows] = await pool.execute(query, [parseInt(limit)]);
    res.json(rows);
  } catch (error) {
    console.error('Error al obtener métricas de CPU:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});

// Obtener última métrica de RAM
app.get('/api/ram/latest', async (req, res) => {
  try {
    const query = `
      SELECT * FROM ram_metrics 
      ORDER BY timestamp DESC 
      LIMIT 1
    `;
    
    const [rows] = await pool.execute(query);
    res.json(rows[0] || null);
  } catch (error) {
    console.error('Error al obtener última métrica de RAM:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});

// Obtener última métrica de CPU
app.get('/api/cpu/latest', async (req, res) => {
  try {
    const query = `
      SELECT * FROM cpu_metrics 
      ORDER BY timestamp DESC 
      LIMIT 1
    `;
    
    const [rows] = await pool.execute(query);
    res.json(rows[0] || null);
  } catch (error) {
    console.error('Error al obtener última métrica de CPU:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});

// Obtener métricas por rango de tiempo
app.get('/api/ram/range', async (req, res) => {
  try {
    const { start, end } = req.query;
    
    if (!start || !end) {
      return res.status(400).json({ 
        error: 'Se requieren parámetros start y end (formato: YYYY-MM-DD HH:MM:SS)' 
      });
    }

    const query = `
      SELECT * FROM ram_metrics 
      WHERE timestamp BETWEEN ? AND ?
      ORDER BY timestamp ASC
    `;
    
    const [rows] = await pool.execute(query, [start, end]);
    res.json(rows);
  } catch (error) {
    console.error('Error al obtener métricas de RAM por rango:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});

app.get('/api/cpu/range', async (req, res) => {
  try {
    const { start, end } = req.query;
    
    if (!start || !end) {
      return res.status(400).json({ 
        error: 'Se requieren parámetros start y end (formato: YYYY-MM-DD HH:MM:SS)' 
      });
    }

    const query = `
      SELECT * FROM cpu_metrics 
      WHERE timestamp BETWEEN ? AND ?
      ORDER BY timestamp ASC
    `;
    
    const [rows] = await pool.execute(query, [start, end]);
    res.json(rows);
  } catch (error) {
    console.error('Error al obtener métricas de CPU por rango:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});

// Ruta de salud
app.get('/health', async (req, res) => {
  try {
    await pool.execute('SELECT 1');
    res.json({ status: 'OK', database: 'Connected' });
  } catch (error) {
    res.status(500).json({ status: 'ERROR', database: 'Disconnected' });
  }
});

// Ruta raíz
app.get('/', (req, res) => {
  res.json({ 
    message: 'API de Monitoreo de Servicios Linux',
    endpoints: [
      'POST /api/ram - Insertar métricas de RAM',
      'POST /api/cpu - Insertar métricas de CPU',
      'GET /api/ram/recent?limit=N - Obtener métricas recientes de RAM',
      'GET /api/cpu/recent?limit=N - Obtener métricas recientes de CPU',
      'GET /api/ram/latest - Obtener última métrica de RAM',
      'GET /api/cpu/latest - Obtener última métrica de CPU',
      'GET /api/ram/range?start=...&end=... - Obtener métricas de RAM por rango',
      'GET /api/cpu/range?start=...&end=... - Obtener métricas de CPU por rango',
      'GET /health - Estado del servicio'
    ]
  });
});

// Inicializar servidor
async function startServer() {
  await initDatabase();
  
  app.listen(PORT, () => {
    console.log(`Servidor corriendo en puerto ${PORT}`);
  });
}

// Manejo de errores no capturados
process.on('unhandledRejection', (err) => {
  console.error('Unhandled Promise rejection:', err);
  process.exit(1);
});

startServer();