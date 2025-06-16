// server.js
const express = require('express');
const mysql = require('mysql2/promise');
const cors = require('cors');
const bodyParser = require('body-parser');

const app = express();
const PORT = 3001;

// Middleware
app.use(cors());
app.use(bodyParser.json());
app.use(express.json());

// Configuración de la base de datos
const dbConfig = {
  host: 'localhost', 
  user: 'user_monitoreo',
  password: 'Ingenieria2025.',
  database: 'sistema_monitoreo',
  waitForConnections: true,
  connectionLimit: 10,
  queueLimit: 0
};

let pool;

// Inicializar conexión a la base de datos
async function initDatabase() {
  try {
    pool = mysql.createPool(dbConfig);
    
    console.log('Base de datos inicializada correctamente');
  } catch (error) {
    console.error('Error al inicializar la base de datos:', error);
    process.exit(1);
  }
}

// Insertar métricas de RAM
app.post('/api/ram', async (req, res) => {
  try {

    const data = req.body.data;

    // Mapear claves estandar a las de base de datos
    const mappedData = {
      memoria_total: data.total,
      memoria_libre: data.libre,
      memoria_usada: data.uso,
      porcentaje_uso: data.porcentaje
    };

    const { memoria_total, memoria_libre, memoria_usada, porcentaje_uso } = mappedData;

    if (!memoria_total || !memoria_libre || !memoria_usada || porcentaje_uso === undefined) {
      console.log('Faltan campos requeridos:', {
        memoria_total, 
        memoria_libre, 
        memoria_usada, 
        porcentaje_uso
      });
      // Respuesta de error si faltan campos
      return res.status(400).json({ 
        error: 'Faltan campos requeridos: memoria_total, memoria_libre, memoria_usada, porcentaje_uso' 
      });
    }

    const query = `
      INSERT INTO tabla_ram (memoria_total, memoria_libre, memoria_usada, porcentaje_uso)
      VALUES (?, ?, ?, ?)
    `;
    
    const [result] = await pool.execute(query, [
      memoria_total, 
      memoria_libre, 
      memoria_usada, 
      porcentaje_uso
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
    const data = req.body.data;
    // Mapear claves estandar a las de base de datos
    const mappedData = {
      porcentaje_cpu: data.porcentajeUso,
    };
    const { porcentaje_cpu } = mappedData;
    
    if (porcentaje_cpu === undefined) {
      return res.status(400).json({ 
        error: 'Faltan campos requeridos: porcentaje_cpu' 
      });
    }

    const query = `
      INSERT INTO tabla_cpu (porcentaje_cpu)
      VALUES (?)
    `;
    
    const [result] = await pool.execute(query, [
      porcentaje_cpu
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


// Obtener última métrica de RAM
app.get('/api/ram/latest', async (req, res) => {
  try {
    const query = `
      SELECT * FROM tabla_ram 
      ORDER BY fecha DESC 
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
      SELECT * FROM tabla_cpu 
      ORDER BY fecha DESC 
      LIMIT 1
    `;
    
    const [rows] = await pool.execute(query);
    res.json(rows[0] || null);
  } catch (error) {
    console.error('Error al obtener última métrica de CPU:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});


// Ruta raíz
app.get('/', (req, res) => {
  res.json({ 
    message: 'API de Monitoreo de Servicios Linux',
    endpoints: [
      'POST /api/ram - Insertar métricas de RAM',
      'POST /api/cpu - Insertar métricas de CPU',
      'GET /api/ram/latest - Obtener última métrica de RAM',
      'GET /api/cpu/latest - Obtener última métrica de CPU',
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