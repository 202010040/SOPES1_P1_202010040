// server.js
const express = require('express');
const mysql = require('mysql2/promise');
const bodyParser = require('body-parser');

// En tu API Node.js
const cors = require('cors');
const app = express();

app.use(cors({
  origin: process.env.CORS_ORIGIN || 'http://localhost:3000'
}));

const PORT = 4001;

// Middleware
app.use(cors());
app.use(bodyParser.json());
app.use(express.json());

// Configuración de la base de datos usando variables de entorno
const dbConfig = {
  host: process.env.DB_HOST || '0.0.0.0', // 'db-monitoreo', // Nombre del contenedor MySQL
  port: process.env.DB_PORT || 3306,
  user: process.env.DB_USER || 'user_monitoreo',
  password: process.env.DB_PASSWORD || 'Ingenieria2025.',
  database: process.env.DB_NAME || 'sistema_monitoreo',
  waitForConnections: true,
  connectionLimit: 10,
  queueLimit: 0
};

let pool;

// Inicializar conexión a la base de datos con reintentos
async function initDatabase() {
  let retries = 5;
  
  while (retries > 0) {
    try {
      pool = mysql.createPool(dbConfig);
      
      // Probar la conexión
      const connection = await pool.getConnection();
      await connection.ping();
      connection.release();
      
      console.log('Base de datos inicializada correctamente');
      return;
    } catch (error) {
      console.error(`Error al conectar a la base de datos. Intentos restantes: ${retries - 1}`, error.message);
      retries--;
      
      if (retries === 0) {
        console.error('No se pudo conectar a la base de datos después de varios intentos');
        process.exit(1);
      }
      
      // Esperar 5 segundos antes del siguiente intento
      await new Promise(resolve => setTimeout(resolve, 5000));
    }
  }
}

// Insertar métricas del sistema
app.post('/api/metrics', async (req, res) => {
  try {
    const {
      total_ram,
      ram_libre,
      uso_ram,
      porcentaje_ram,
      porcentaje_cpu_uso,
      porcentaje_cpu_libre,
      procesos_corriendo,
      total_procesos,
      procesos_durmiendo,
      procesos_zombie,
      procesos_parados,
      hora
    } = req.body;


    // Validar campos requeridos
    if (
      total_ram === undefined ||
      ram_libre === undefined ||
      uso_ram === undefined ||
      porcentaje_ram === undefined || 
      porcentaje_cpu_uso === undefined || 
      porcentaje_cpu_libre === undefined ||
      procesos_corriendo === undefined ||
      total_procesos === undefined ||
      procesos_durmiendo === undefined ||
      procesos_zombie === undefined ||
      procesos_parados === undefined ||
      hora === undefined
    ) {
      console.log('Faltan campos requeridos en la petición:', req.body);
      return res.status(400).json({ 
        error: 'Faltan campos requeridos en el JSON' 
      });
    }


    const query = `
      INSERT INTO metricas_sistema (
        memoria_total, 
        memoria_libre, 
        memoria_usada, 
        porcentaje_ram,
        porcentaje_cpu_uso,
        porcentaje_cpu_libre,
        procesos_corriendo,
        total_procesos,
        procesos_durmiendo,
        procesos_zombie,
        procesos_parados,
        hora,
        api
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `;
    
    const [result] = await pool.execute(query, [
      total_ram,
      ram_libre,
      uso_ram,
      porcentaje_ram,
      porcentaje_cpu_uso,
      porcentaje_cpu_libre,
      procesos_corriendo,
      total_procesos,
      procesos_durmiendo,
      procesos_zombie,
      procesos_parados,
      hora,
      'NodeJS'
    ]);
    
    res.status(201).json({ 
      message: 'Métricas del sistema guardadas exitosamente',
      id: result.insertId 
    });
  } catch (error) {
    console.error('Error al guardar métricas del sistema:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});

// Ruta raíz
app.get('/', (req, res) => {
  res.json({ 
    message: 'API de Monitoreo de Servicios Linux',
    endpoints: [
      'POST /api/metrics - Insertar métricas del sistema'
    ]
  });
});

// Inicializar servidor
async function startServer() {
  await initDatabase();
  
  app.listen(PORT, '0.0.0.0', () => {
    console.log(`Servidor corriendo en puerto ${PORT}`);
  });
}

// Manejo de errores no capturados
process.on('unhandledRejection', (err) => {
  console.error('Unhandled Promise rejection:', err);
  process.exit(1);
});

// Manejo graceful de shutdown
process.on('SIGTERM', async () => {
  console.log('Cerrando servidor...');
  if (pool) {
    await pool.end();
  }
  process.exit(0);
});

process.on('SIGINT', async () => {
  console.log('Cerrando servidor...');
  if (pool) {
    await pool.end();
  }
  process.exit(0);
});

startServer();