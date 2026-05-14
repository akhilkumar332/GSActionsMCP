import { useEffect, useState } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { 
  Terminal, ShieldAlert, CheckCircle2, Clock, Activity, Users, 
  AlertTriangle, Database, Zap, RefreshCcw
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';

const StatsTab = ({ usage }) => {
  if (!usage) return null;
  
  const metrics = [
    { label: 'Total Nodes', value: usage.users, icon: Users, color: 'text-blue-400' },
    { label: 'Total Tasks', value: usage.tasks, icon: Activity, color: 'text-white' },
    { label: 'Successes', value: usage.task_successes, icon: CheckCircle2, color: 'text-emerald-400' },
    { label: 'Failures', value: usage.task_failures, icon: AlertTriangle, color: 'text-red-400' },
    { label: 'Missed', value: usage.task_missed, icon: Clock, color: 'text-amber-400' },
    { label: 'Audit Events', value: usage.audit_log_events, icon: Database, color: 'text-purple-400' },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {metrics.map((m, idx) => (
        <motion.div
          key={m.label}
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: idx * 0.1 }}
          className="bg-white/5 border border-white/10 rounded-[2rem] p-8 backdrop-blur-xl hover:border-white/20 transition-all group"
        >
          <div className="flex items-center justify-between mb-4">
            <m.icon className={`w-6 h-6 ${m.color}`} />
            <Zap className="w-4 h-4 text-white/10 group-hover:text-white/30 transition-colors" />
          </div>
          <p className="text-3xl font-black text-white mb-1 tabular-nums tracking-tighter">{m.value}</p>
          <p className="text-[10px] font-bold text-slate-500 uppercase tracking-[0.2em]">{m.label}</p>
        </motion.div>
      ))}
    </div>
  );
};

const LogsTab = ({ logs }) => (
  <motion.div 
    initial={{ opacity: 0 }}
    animate={{ opacity: 1 }}
    className="bg-black/60 rounded-[2.5rem] border border-white/10 shadow-[0_40px_100px_rgba(0,0,0,0.5)] overflow-hidden backdrop-blur-3xl"
  >
    <div className="flex items-center gap-3 px-10 py-6 bg-white/5 border-b border-white/10">
      <Terminal className="w-5 h-5 text-accent-orange" />
      <span className="text-xs font-mono font-bold text-slate-400 tracking-widest uppercase">system_audit_trail.log</span>
    </div>
    
    <div className="p-0 overflow-x-auto">
      <table className="w-full text-left border-collapse font-mono text-[11px]">
        <thead className="bg-black/40 text-slate-500">
          <tr>
            <th className="px-10 py-5 font-bold uppercase tracking-widest border-b border-white/5">Timestamp</th>
            <th className="px-6 py-5 font-bold uppercase tracking-widest border-b border-white/5">Subject</th>
            <th className="px-6 py-5 font-bold uppercase tracking-widest border-b border-white/5">Action</th>
            <th className="px-6 py-5 font-bold uppercase tracking-widest border-b border-white/5">Resource</th>
            <th className="px-10 py-5 font-bold uppercase tracking-widest border-b border-white/5 text-right">Metadata</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-white/5">
          {logs.length === 0 ? (
            <tr>
              <td colSpan="5" className="px-10 py-20 text-center text-slate-500 uppercase tracking-widest font-bold opacity-50">
                No audit logs detected in this sector.
              </td>
            </tr>
          ) : logs.map((log) => (
            <tr key={log.id} className="hover:bg-white/[0.03] transition-colors group">
              <td className="px-10 py-5 text-slate-500 whitespace-nowrap tabular-nums">
                {new Date(log.created_at).toLocaleString()}
              </td>
              <td className="px-6 py-5">
                <span className="text-blue-400 font-bold">{log.user_id || 'SYSTEM'}</span>
              </td>
              <td className="px-6 py-5">
                <span className="text-white font-bold uppercase">{log.action}</span>
              </td>
              <td className="px-6 py-5">
                <span className="text-slate-400">{log.resource_type}</span>
                {log.resource_id && <span className="text-slate-600 text-[9px] ml-1">({log.resource_id})</span>}
              </td>
              <td className="px-10 py-5 text-right">
                <code className="text-slate-500 group-hover:text-slate-300 transition-colors truncate block max-w-xs ml-auto">
                  {JSON.stringify(log.metadata)}
                </code>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  </motion.div>
);

const Monitor = () => {
  const [activeTab, setActiveTab] = useState('stats'); // 'stats' or 'logs'
  const [usage, setUsage] = useState(null);
  const [auditLogs, setAuditLogs] = useState([]);
  const [loading, setLoading] = useState(true);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [usageRes, auditRes] = await Promise.all([
        axios.get('/api/v1/admin/usage'),
        axios.get('/api/v1/admin/audit-logs?limit=100')
      ]);
      if (usageRes.data.success) setUsage(usageRes.data.data);
      if (auditRes.data.success) setAuditLogs(auditRes.data.data);
    } catch (err) {
      console.error('Failed to fetch monitor data', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 60000); // Refresh every minute
    return () => clearInterval(interval);
  }, []);

  return (
    <DashboardLayout>
      <header className="mb-12 flex items-center justify-between">
        <div>
          <h1 className="text-4xl font-black text-white tracking-tight">System Monitor</h1>
          <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em] mt-2">Neural Node Execution Stream</p>
        </div>
        <div className="flex items-center gap-4">
          <button 
            onClick={fetchData}
            className="p-3 bg-white/5 border border-white/10 rounded-full hover:bg-white/10 transition-all text-slate-400 hover:text-white"
            title="Force Neural Sync"
          >
            <RefreshCcw size={18} className={loading ? 'animate-spin text-accent-orange' : ''} />
          </button>
          <div className="flex items-center gap-3 bg-white/5 border border-white/10 px-5 py-2 rounded-full backdrop-blur-xl">
            <span className="w-2 h-2 bg-emerald-500 rounded-full animate-pulse shadow-[0_0_10px_#10b981]"></span>
            <span className="text-[10px] font-black text-white uppercase tracking-widest">Active Link</span>
          </div>
        </div>
      </header>

      <div className="flex gap-4 mb-8">
        {['stats', 'logs'].map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-6 py-2 rounded-full text-xs font-black uppercase tracking-widest transition-all ${
              activeTab === tab 
                ? 'bg-accent-orange text-white shadow-[0_0_20px_rgba(217,119,6,0.3)]' 
                : 'bg-white/5 text-slate-400 hover:bg-white/10'
            }`}
          >
            {tab === 'stats' ? 'Neural Pulse' : 'Audit Stream'}
          </button>
        ))}
      </div>

      <AnimatePresence mode="wait">
        {loading && !usage && auditLogs.length === 0 ? (
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="flex flex-col items-center justify-center py-32 gap-6"
          >
            <div className="w-12 h-12 border-4 border-accent-orange/20 border-t-accent-orange rounded-full animate-spin"></div>
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-[0.4em] animate-pulse">Synchronizing Neural Data...</p>
          </motion.div>
        ) : (
          <motion.div
            key={activeTab}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            transition={{ duration: 0.2 }}
          >
            {activeTab === 'stats' ? (
              <StatsTab usage={usage} />
            ) : (
              <LogsTab logs={auditLogs} />
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </DashboardLayout>
  );
};

export default Monitor;
