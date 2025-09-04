// websocket-server.js
const express = require('express');
const http = require('http');
const socketIo = require('socket.io');
const mysql = require('mysql2/promise');
const cors = require('cors');

const app = express();
const server = http.createServer(app);

// Configuración de Socket.IO con CORS
const io = socketIo(server, {
  cors: {
    origin: process.env.CORS_ORIGIN || "*",
    methods: ["GET", "POST"],
    credentials: true
  }
});

const PORT = process.env.WEBSOCKET_PORT || 6001;

// Middleware 
app.use(cors({
  origin: process.env.CORS_ORIGIN || '*',
  credentials: true
})); 
app.use(express.json());

// Configuración de la base de datos
const dbConfig = {
  host: '35.202.138.192',
  port: process.env.DB_PORT || 3306,
  user: process.env.DB_USER || 'user_monitoreo',
  password: process.env.DB_PASSWORD || 'Ingenieria2025.',
  database: process.env.DB_NAME || 'sistema_monitoreo',
  waitForConnections: true,
  connectionLimit: 10,
  queueLimit: 0
};

let pool;

// Inicializar conexión a la base de datos
async function initDatabase() {
  let retries = 5;
  
  while (retries > 0) {
    try {
      pool = mysql.createPool(dbConfig);
      
      const connection = await pool.getConnection();
      await connection.ping();
      connection.release();
      
      console.log('Base de datos WebSocket inicializada correctamente');
      return;
    } catch (error) {
      console.error(`Error al conectar a la base de datos. Intentos restantes: ${retries - 1}`, error.message);
      retries--;
      
      if (retries === 0) {
        console.error('No se pudo conectar a la base de datos después de varios intentos');
        process.exit(1);
      }
      
      await new Promise(resolve => setTimeout(resolve, 5000));
    }
  }
}

// Función para obtener las métricas más recientes
async function getLatestMetrics() {
  try {
    const query = `
      SELECT 
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
        hora
      FROM metricas_sistema 
      ORDER BY id DESC 
      LIMIT 1
    `;
    
    const [rows] = await pool.execute(query);
    return rows[0] || null;
  } catch (error) {
    console.error('Error al obtener métricas:', error);
    return null;
  }
}

// Configuración de Socket.IO
io.on('connection', (socket) => {
  console.log('Cliente conectado:', socket.id);

  // Enviar datos iniciales al cliente
  const sendInitialData = async () => {
    try {
      const metrics = await getLatestMetrics();

      socket.emit('initial-data', {
        metrics
      });
    } catch (error) {
      console.error('Error enviando datos iniciales:', error);
    }
  };

  sendInitialData();

  // Manejar desconexión
  socket.on('disconnect', () => {
    console.log('Cliente desconectado:', socket.id);
  });
});

// Función para enviar actualizaciones periódicas a todos los clientes
const broadcastUpdates = async () => {
  try {
    const metrics = await getLatestMetrics();

    if (metrics) {
      io.emit('metrics-update', {
        metrics,
        timestamp: new Date().toISOString()
      });
    }
  } catch (error) {
    console.error('Error en broadcast:', error);
  }
};

// API REST endpoint para obtener métricas más recientes
app.get('/api/metrics/latest', async (req, res) => {
  try {
    const metrics = await getLatestMetrics();
    res.json(metrics);
  } catch (error) {
    console.error('Error obteniendo métricas:', error);
    res.status(500).json({ error: 'Error interno del servidor' });
  }
});

// Ruta raíz
app.get('/', (req, res) => {
  res.json({ 
    message: 'WebSocket API de Monitoreo - Socket.IO Server',
    endpoints: [
      'GET /api/metrics/latest - Obtener métricas más recientes',
      'WebSocket events: connection, initial-data, metrics-update'
    ]
  });
});

// Inicializar servidor
async function startServer() {
  await initDatabase();
  
  server.listen(PORT, '0.0.0.0', () => {
    console.log(`Servidor WebSocket corriendo en puerto ${PORT}`);
    
    // Iniciar broadcast periódico cada segundo
    setInterval(broadcastUpdates, 1000);
  });
}

// Manejo de errores
process.on('unhandledRejection', (err) => {
  console.error('Unhandled Promise rejection:', err);
  process.exit(1);
});

process.on('SIGTERM', async () => {
  console.log('Cerrando servidor WebSocket...');
  if (pool) {
    await pool.end();
  }
  process.exit(0);
});

process.on('SIGINT', async () => {
  console.log('Cerrando servidor WebSocket...');
  if (pool) {
    await pool.end();
  }
  process.exit(0);
});

startServer();