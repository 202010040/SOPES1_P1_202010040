import React, { useEffect, useState } from 'react';
import io from 'socket.io-client';
import {
  PieChart, Pie, Cell, Tooltip, ResponsiveContainer
} from 'recharts';
import './Dashboard.css';

// Utilidad para convertir KB a MB y redondear
const kbToMb = (kb) => (kb / 1024).toFixed(1);

// URLs de conexión para GCP
const GCP_WEBSOCKET_URLS = {
  // Usando el LoadBalancer externo de api-lectura
  primary: 'http://35.202.100.27:6001',
};

// Determinar la URL a usar basándose en el entorno
const getWebSocketURL = () => {
  // Si hay una variable de entorno específica, usarla
  if (process.env.REACT_APP_WEBSOCKET_URL) {
    return process.env.REACT_APP_WEBSOCKET_URL;
  }
  
  // Por defecto, usar la IP externa del LoadBalancer de GCP
  return GCP_WEBSOCKET_URLS.primary;
};

const Dashboard = () => {
  const [ram, setRam] = useState({ total: 0, usada: 0, libre: 0, porcentaje: 0 });
  const [cpu, setCpu] = useState(0);
  const [processes, setProcesses] = useState({
    corriendo: 0,
    durmiendo: 0,
    parados: 0,
    zombie: 0,
    total: 0
  });
  const [connected, setConnected] = useState(false);
  const [socket, setSocket] = useState(null);
  const [connectionAttempts, setConnectionAttempts] = useState(0);
  const [currentUrl, setCurrentUrl] = useState('');

  const connectToWebSocket = (url, attempt = 0) => {
    console.log(`Intentando conectar a: ${url} (intento ${attempt + 1})`);
    setCurrentUrl(url);
    
    const newSocket = io(url, {
      transports: ['polling', 'websocket'], // Permitir fallback a polling
      upgrade: true,
      timeout: 10000,
      forceNew: true,
      reconnection: true,
      reconnectionAttempts: 5,
      reconnectionDelay: 2000,
    });

    setSocket(newSocket);

    // Manejar conexión exitosa
    newSocket.on('connect', () => {
      console.log(`Conectado exitosamente a: ${url}`);
      setConnected(true);
      setConnectionAttempts(0);
    });

    // Manejar errores de conexión
    newSocket.on('connect_error', (error) => {
      console.error(`Error conectando a ${url}:`, error);
      setConnected(false);
      
      // Si falla, intentar con la siguiente URL
      const urls = Object.values(GCP_WEBSOCKET_URLS);
      const currentIndex = urls.indexOf(url);
      const nextIndex = (currentIndex + 1) % urls.length;
      
      if (attempt < urls.length - 1) {
        setTimeout(() => {
          newSocket.close();
          connectToWebSocket(urls[nextIndex], attempt + 1);
        }, 3000); 
      }
    });

    // Manejar desconexión
    newSocket.on('disconnect', (reason) => {
      console.log(`Desconectado de ${url}. Razón:`, reason);
      setConnected(false);
      
      // Si la desconexión no fue intencional, intentar reconectar
      if (reason !== 'io client disconnect') {
        setConnectionAttempts(prev => prev + 1);
      }
    });

    // Manejar datos iniciales
    newSocket.on('initial-data', (data) => {
      console.log('Datos iniciales recibidos:', data);
      if (data.metrics) {
        updateMetrics(data.metrics);
      }
    });

    // Manejar actualizaciones de métricas
    newSocket.on('metrics-update', (data) => {
      console.log('Actualización de métricas:', data);
      if (data.metrics) {
        updateMetrics(data.metrics);
      }
    });

    // Eventos específicos para la API de lectura de GCP
    newSocket.on('system-metrics', (data) => {
      console.log('Métricas del sistema recibidas:', data);
      updateMetrics(data);
    });

    return newSocket;
  };

  useEffect(() => {
    const websocketUrl = getWebSocketURL();
    const socket = connectToWebSocket(websocketUrl);

    // Cleanup al desmontar el componente
    return () => {
      if (socket) {
        socket.close();
      }
    };
  }, []);

  // Función para reconectar manualmente
  const handleReconnect = () => {
    if (socket) {
      socket.close();
    }
    const websocketUrl = getWebSocketURL();
    connectToWebSocket(websocketUrl);
  };

  const updateMetrics = (metrics) => {
    try {
      // Actualizar RAM - manejar diferentes formatos de datos
      const ramData = {
        total: metrics.memoria_total || metrics.ram_total || metrics.totalMemory || 0,
        usada: metrics.memoria_usada || metrics.ram_used || metrics.usedMemory || 0,
        libre: metrics.memoria_libre || metrics.ram_free || metrics.freeMemory || 0,
        porcentaje: parseFloat(metrics.porcentaje_ram || metrics.ram_percentage || metrics.memoryUsagePercent || 0)
      };
      
      // Si no tenemos porcentaje pero tenemos total y usado, calcularlo
      if (!ramData.porcentaje && ramData.total > 0) {
        ramData.porcentaje = (ramData.usada / ramData.total) * 100;
      }
      
      setRam(ramData);

      // Actualizar CPU - manejar diferentes formatos
      const cpuUsage = parseFloat(
        metrics.porcentaje_cpu_uso || 
        metrics.cpu_usage || 
        metrics.cpuUsagePercent || 
        0
      );
      setCpu(cpuUsage);

      // Actualizar Procesos - manejar diferentes formatos
      const processData = {
        corriendo: metrics.procesos_corriendo || metrics.running_processes || metrics.runningProcesses || 0,
        durmiendo: metrics.procesos_durmiendo || metrics.sleeping_processes || metrics.sleepingProcesses || 0,
        parados: metrics.procesos_parados || metrics.stopped_processes || metrics.stoppedProcesses || 0,
        zombie: metrics.procesos_zombie || metrics.zombie_processes || metrics.zombieProcesses || 0,
        total: metrics.total_procesos || metrics.total_processes || metrics.totalProcesses || 0
      };
      
      // Si no tenemos total, calcularlo
      if (!processData.total) {
        processData.total = processData.corriendo + processData.durmiendo + 
                           processData.parados + processData.zombie;
      }
      
      setProcesses(processData);
    } catch (error) {
      console.error('Error actualizando métricas:', error);
    }
  };

  const ramChartData = [
    { name: 'En uso', value: ram.usada },
    { name: 'Sin usar', value: ram.libre }
  ];

  const cpuChartData = [
    { name: 'En uso', value: cpu },
    { name: 'Sin usar', value: 100 - cpu }
  ];

  const RAM_COLORS = ['#3b4c63', '#353e4a']; // colores mate para RAM
  const CPU_COLORS = ['#5f3269', '#3d1c45']; // colores mate para CPU

  // Tooltip personalizado para RAM
  const renderRamTooltip = ({ active, payload }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      const porcentaje = ram.total > 0 ? ((data.value / ram.total) * 100).toFixed(1) : '0';
      const mb = kbToMb(data.value);
      return (
        <div style={{ background: '#fff', padding: 8, border: '1px solid #ccc' }}>
          <strong>{data.name}</strong>
          <br />
          {porcentaje}% - {mb} MB
        </div>
      );
    }
    return null;
  };

  return (
    <div className="dashboard-container">
      <h1 className="dashboard-title">SOPES1 PROYECTO - 202010040</h1>
      
      {/* Indicador de conexión mejorado */}
      <div className="connection-status">
        <span className={`status-indicator ${connected ? 'connected' : 'disconnected'}`}>
          {connected ? '🟢 Conectado' : '🔴 Desconectado'}
        </span>
        <div className="connection-details">
          <small>URL: {currentUrl}</small>
          {!connected && (
            <button onClick={handleReconnect} className="reconnect-btn">
              Reconectar
            </button>
          )}
          {connectionAttempts > 0 && (
            <small>Intentos de reconexión: {connectionAttempts}</small>
          )}
        </div>
      </div>

      <div className="charts-container">
        {/* RAM */}
        <div className="chart-box"> 
          <h2>Métricas RAM</h2>
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie
                data={ramChartData}
                dataKey="value"
                nameKey="name"
                cx="50%"
                cy="50%"
                outerRadius={80}
                label
              >
                {ramChartData.map((_, index) => (
                  <Cell key={`ram-${index}`} fill={RAM_COLORS[index % RAM_COLORS.length]} />
                ))}
              </Pie>
              <Tooltip content={renderRamTooltip} />
            </PieChart>
          </ResponsiveContainer>
          <div className="metrics-text">
            <p><strong>Total:</strong> {kbToMb(ram.total)} MB</p>
            <p><strong>Libre:</strong> {kbToMb(ram.libre)} MB</p>
            <p><strong>En uso:</strong> {ram.porcentaje.toFixed(1)}%</p>
          </div>
        </div>

        {/* CPU */}
        <div className="chart-box">
          <h2>Métricas CPU</h2>
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie
                data={cpuChartData}
                dataKey="value"
                nameKey="name"
                cx="50%"
                cy="50%"
                outerRadius={80}
                label
              >
                {cpuChartData.map((_, index) => (
                  <Cell key={`cpu-${index}`} fill={CPU_COLORS[index % CPU_COLORS.length]} />
                ))}
              </Pie>
              <Tooltip formatter={(value, name) => [`${value.toFixed(1)}%`, name]} />
            </PieChart>
          </ResponsiveContainer>
          <div className="metrics-text">
            <p><strong>En uso:</strong> {cpu.toFixed(1)}%</p>
            <p><strong>Libre:</strong> {(100 - cpu).toFixed(1)}%</p>
          </div>
        </div>
      </div>

      {/* Tabla de Procesos */}
      <div className="processes-section">
        <h2>Estado de Procesos</h2>
        <div className="processes-table-container">
          <table className="processes-table">
            <thead>
              <tr>
                <th>Estado</th>
                <th>Cantidad</th>
                <th>Porcentaje</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>
                  <span className="process-status running">Corriendo</span>
                </td>
                <td>{processes.corriendo}</td>
                <td>{processes.total > 0 ? ((processes.corriendo / processes.total) * 100).toFixed(1) : 0}%</td>
              </tr>
              <tr>
                <td>
                  <span className="process-status sleeping">Durmiendo</span>
                </td>
                <td>{processes.durmiendo}</td>
                <td>{processes.total > 0 ? ((processes.durmiendo / processes.total) * 100).toFixed(1) : 0}%</td>
              </tr>
              <tr>
                <td>
                  <span className="process-status stopped">Parados</span>
                </td>
                <td>{processes.parados}</td>
                <td>{processes.total > 0 ? ((processes.parados / processes.total) * 100).toFixed(1) : 0}%</td>
              </tr>
              <tr>
                <td>
                  <span className="process-status zombie">Zombie</span>
                </td>
                <td>{processes.zombie}</td>
                <td>{processes.total > 0 ? ((processes.zombie / processes.total) * 100).toFixed(1) : 0}%</td>
              </tr>
              <tr className="total-row">
                <td><strong>Total de Procesos</strong></td>
                <td><strong>{processes.total}</strong></td>
                <td><strong>100%</strong></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;