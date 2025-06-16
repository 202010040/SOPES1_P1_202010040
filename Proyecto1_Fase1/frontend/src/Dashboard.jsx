import React, { useEffect, useState } from 'react';
import axios from 'axios';
import {
  PieChart, Pie, Cell, Tooltip, ResponsiveContainer
} from 'recharts';
import './Dashboard.css';

// Utilidad para convertir KB a MB y redondear
const kbToMb = (kb) => (kb / 1024).toFixed(1);

const Dashboard = () => {
  const [ram, setRam] = useState({ total: 0, usada: 0, libre: 0, porcentaje: 0 });
  const [cpu, setCpu] = useState(0);

  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        const [ramRes, cpuRes] = await Promise.all([
          axios.get('http://monitor_api:3001/api/ram/latest'),
          axios.get('http://monitor_api:3001/api/cpu/latest'),
        ]);

        const ramData = ramRes.data;
        const porcentajeUso = parseFloat(ramData?.porcentaje_uso || 0);
        setRam({
          total: ramData.memoria_total,
          usada: ramData.memoria_usada,
          libre: ramData.memoria_libre,
          porcentaje: porcentajeUso
        });

        const cpuPorcentaje = parseFloat(cpuRes.data?.porcentaje_cpu || 0);
        setCpu(cpuPorcentaje);
      } catch (err) {
        console.error('Error al obtener métricas:', err);
      }
    };

    fetchMetrics();
    const interval = setInterval(fetchMetrics, 1000);
    return () => clearInterval(interval);
  }, []);

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
      const porcentaje = ((data.value / ram.total) * 100).toFixed(1);
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
            <p><strong>Total:</strong> {(cpu)} %</p>
            <p><strong>Libre:</strong> {(100-cpu)} %</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
