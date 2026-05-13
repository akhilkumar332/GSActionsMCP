import { useEffect, useState } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { 
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, 
  AreaChart, Area, Cell 
} from 'recharts';
import { BarChart3, Activity, Zap, Users, ShieldCheck, ArrowRight, Server } from 'lucide-react';
import { motion } from 'framer-motion';
import { useNavigate } from 'react-router-dom';

const Insights = () => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    const fetchInsights = async () => {
      try {
        const res = await axios.get('/api/admin/insights');
        if (res.data.success) {
          setData(res.data.data);
        }
      } catch (err) {
        console.error('Failed to fetch insights', err);
      } finally {
        setLoading(false);
      }
    };
    fetchInsights();
  }, []);

  if (loading) {
    return (
      <DashboardLayout>
        <div className="flex items-center justify-center min-h-[600px]">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-accent-orange"></div>
        </div>
      </DashboardLayout>
    );
  }

  const chartColors = {
    primary: '#d97706', // accent-orange
    secondary: '#3b82f6', // blue-500
    success: '#10b981', // emerald-500
    grid: 'rgba(255, 255, 255, 0.05)',
    text: '#94a3b8' // slate-400
  };

  return (
    <DashboardLayout>
      <header className="mb-12">
        <h1 className="text-4xl font-black text-white tracking-tight">System Insights</h1>
        <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em] mt-2">Global Analytics & Performance Metrics</p>
      </header>

      {/* Metric Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-12">
        <MetricCard 
          icon={Zap} 
          label="P99 Latency" 
          value={`${data?.p99_latency}ms`} 
          trend="+2.4%" 
          color="text-amber-400"
        />
        <MetricCard 
          icon={ShieldCheck} 
          label="Success Rate" 
          value={`${data?.success_rate}%`} 
          trend="+0.2%" 
          color="text-emerald-400"
        />
        <MetricCard 
          icon={Users} 
          label="Active Workers" 
          value={data?.active_workers} 
          trend="Stable" 
          color="text-blue-400"
        />
      </div>

      <div className="mb-12">
        <motion.div 
          whileHover={{ y: -5 }}
          onClick={() => navigate('/admin/workers')}
          className="bg-white/5 border border-white/10 rounded-[2.5rem] p-8 backdrop-blur-3xl cursor-pointer hover:bg-white/[0.08] transition-all flex items-center justify-between group"
        >
          <div className="flex items-center gap-6">
            <div className="bg-blue-500/10 p-5 rounded-[1.5rem] text-blue-400">
              <Server size={28} />
            </div>
            <div>
              <h2 className="text-2xl font-black text-white uppercase tracking-tighter mb-1">Infrastructure Health</h2>
              <p className="text-xs font-bold text-slate-500 uppercase tracking-widest">Manage and monitor active execution nodes</p>
            </div>
          </div>
          <div className="flex items-center gap-4 text-blue-400 font-black uppercase tracking-widest text-xs">
            View Registry <ArrowRight size={16} className="group-hover:translate-x-2 transition-transform" />
          </div>
        </motion.div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Daily Tasks Chart */}
        <motion.div 
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="bg-white/5 border border-white/10 rounded-[2.5rem] p-8 backdrop-blur-3xl"
        >
          <div className="flex items-center justify-between mb-8">
            <div className="flex items-center gap-3">
              <BarChart3 className="text-accent-orange w-5 h-5" />
              <h2 className="text-lg font-bold text-white uppercase tracking-wider">Daily Executions</h2>
            </div>
          </div>
          <div className="h-64 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={data?.daily_tasks}>
                <defs>
                  <linearGradient id="colorCount" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor={chartColors.primary} stopOpacity={0.3}/>
                    <stop offset="95%" stopColor={chartColors.primary} stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke={chartColors.grid} vertical={false} />
                <XAxis 
                  dataKey="date" 
                  stroke={chartColors.text} 
                  fontSize={10} 
                  tickLine={false} 
                  axisLine={false} 
                />
                <YAxis 
                  stroke={chartColors.text} 
                  fontSize={10} 
                  tickLine={false} 
                  axisLine={false} 
                />
                <Tooltip 
                  contentStyle={{ backgroundColor: '#000', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '12px' }}
                  itemStyle={{ color: chartColors.primary }}
                />
                <Area 
                  type="monotone" 
                  dataKey="count" 
                  stroke={chartColors.primary} 
                  strokeWidth={3}
                  fillOpacity={1} 
                  fill="url(#colorCount)" 
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </motion.div>

        {/* Task Distribution */}
        <motion.div 
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="bg-white/5 border border-white/10 rounded-[2.5rem] p-8 backdrop-blur-3xl"
        >
          <div className="flex items-center justify-between mb-8">
            <div className="flex items-center gap-3">
              <Activity className="text-blue-400 w-5 h-5" />
              <h2 className="text-lg font-bold text-white uppercase tracking-wider">Status Distribution</h2>
            </div>
          </div>
          <div className="h-64 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={[
                { name: 'Success', val: data?.success_rate },
                { name: 'Failure', val: 100 - (data?.success_rate || 0) }
              ]}>
                <CartesianGrid strokeDasharray="3 3" stroke={chartColors.grid} vertical={false} />
                <XAxis dataKey="name" stroke={chartColors.text} fontSize={10} tickLine={false} axisLine={false} />
                <YAxis stroke={chartColors.text} fontSize={10} tickLine={false} axisLine={false} />
                <Tooltip 
                  contentStyle={{ backgroundColor: '#000', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '12px' }}
                />
                <Bar dataKey="val" radius={[10, 10, 0, 0]}>
                  <Cell fill={chartColors.success} />
                  <Cell fill="#ef4444" />
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </motion.div>
      </div>
    </DashboardLayout>
  );
};

const MetricCard = ({ icon: Icon, label, value, trend, color }) => (
  <motion.div 
    whileHover={{ y: -5 }}
    className="bg-white/5 border border-white/10 rounded-3xl p-6 backdrop-blur-xl relative overflow-hidden group"
  >
    <div className="absolute top-0 right-0 p-8 opacity-5 group-hover:opacity-10 transition-opacity">
      <Icon size={80} />
    </div>
    <div className="flex items-start justify-between mb-4">
      <div className={`p-3 rounded-2xl bg-white/5 ${color}`}>
        <Icon size={20} />
      </div>
      <span className="text-[10px] font-black text-emerald-400 uppercase tracking-widest">{trend}</span>
    </div>
    <p className="text-slate-400 font-bold uppercase text-[10px] tracking-[0.2em] mb-1">{label}</p>
    <p className="text-3xl font-black text-white tracking-tighter">{value}</p>
  </motion.div>
);

export default Insights;
