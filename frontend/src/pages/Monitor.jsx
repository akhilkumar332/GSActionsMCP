import { useEffect, useState, useCallback } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { Terminal, ShieldAlert, CheckCircle2, Clock, Cpu } from 'lucide-react';
import { motion } from 'framer-motion';
import { useSSE } from '../hooks/useSSE';

const Monitor = () => {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);

  useSSE(useCallback((event) => {
    if (event.event_type === 'task_executed') {
      try {
        const newLog = typeof event.payload === 'string' ? JSON.parse(event.payload) : event.payload;
        setLogs(prev => [newLog, ...prev]);
      } catch (err) {
        console.error('Failed to parse task_executed payload', err);
      }
    }
  }, []));

  useEffect(() => {
    const fetchLogs = async () => {
      try {
        const res = await axios.get('/api/monitor');
        if (res.data.success) {
          setLogs(res.data.data || []);
        }
      } catch (err) {
        console.error('Failed to fetch logs', err);
      } finally {
        setLoading(false);
      }
    };
    fetchLogs();
    
    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchLogs, 30000);
    return () => clearInterval(interval);
  }, []);

  return (
    <DashboardLayout>
      <header className="mb-12 flex items-center justify-between">
        <div>
          <h1 className="text-4xl font-black text-white tracking-tight">System Monitor</h1>
          <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em] mt-2">Neural Node Execution Stream</p>
        </div>
        <div className="flex items-center gap-3 bg-white/5 border border-white/10 px-5 py-2 rounded-full backdrop-blur-xl">
           <span className="w-2 h-2 bg-emerald-500 rounded-full animate-pulse shadow-[0_0_10px_#10b981]"></span>
           <span className="text-[10px] font-black text-white uppercase tracking-widest">Active Link</span>
        </div>
      </header>

      <motion.div 
        initial={{ opacity: 0, scale: 0.98 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.5 }}
        className="bg-black/60 rounded-[2.5rem] border border-white/10 shadow-[0_40px_100px_rgba(0,0,0,0.5)] overflow-hidden backdrop-blur-3xl"
      >
        <div className="flex items-center gap-3 px-10 py-6 bg-white/5 border-b border-white/10">
          <Terminal className="w-5 h-5 text-accent-orange" />
          <span className="text-xs font-mono font-bold text-slate-400 tracking-widest uppercase">cluster_execution_history.log</span>
          <div className="ml-auto flex gap-2">
             <div className="w-2 h-2 rounded-full bg-white/10"></div>
             <div className="w-2 h-2 rounded-full bg-white/10"></div>
          </div>
        </div>
        
        <div className="p-0 overflow-x-auto min-h-[600px]">
          <table className="w-full text-left border-collapse font-mono text-[11px]">
            <thead className="bg-black/40 text-slate-500">
              <tr>
                <th className="px-10 py-5 font-bold uppercase tracking-widest border-b border-white/5">Sequence</th>
                <th className="px-6 py-5 font-bold uppercase tracking-widest border-b border-white/5">Neural Identity</th>
                <th className="px-6 py-5 font-bold uppercase tracking-widest border-b border-white/5">Task Vector</th>
                <th className="px-6 py-5 font-bold uppercase tracking-widest border-b border-white/5">Status</th>
                <th className="px-10 py-5 font-bold uppercase tracking-widest border-b border-white/5 text-right">Telemetry</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {loading ? (
                <tr>
                  <td colSpan="5" className="px-10 py-32 text-center">
                    <div className="flex flex-col items-center gap-4">
                       <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-orange"></div>
                       <span className="text-slate-500 font-bold uppercase tracking-[0.3em]">Synchronizing Logs...</span>
                    </div>
                  </td>
                </tr>
              ) : logs.length === 0 ? (
                <tr>
                  <td colSpan="5" className="px-10 py-32 text-center text-slate-500 font-bold uppercase tracking-widest italic opacity-50">No execution cycles detected in this window.</td>
                </tr>
              ) : logs.map((log) => (
                <tr key={log.id} className="hover:bg-white/[0.03] transition-colors group">
                  <td className="px-10 py-5 text-slate-500 whitespace-nowrap tabular-nums">
                    {new Date(log.execution_time).toLocaleString()}
                  </td>
                  <td className="px-6 py-5">
                    <span className="text-blue-400 font-bold">{log.user_email}</span>
                  </td>
                  <td className="px-6 py-5">
                    <div className="flex items-center gap-2">
                       <Cpu size={12} className="text-slate-600" />
                       <span className="text-slate-200 font-bold">{log.task_name}</span>
                    </div>
                  </td>
                  <td className="px-6 py-5">
                    {log.status === 'success' ? (
                      <span className="text-emerald-400 flex items-center gap-1.5 font-black uppercase tracking-tighter">
                        <CheckCircle2 className="w-3.5 h-3.5 shadow-[0_0_10px_rgba(52,211,153,0.5)]" /> success
                      </span>
                    ) : log.status === 'failure' ? (
                      <span className="text-red-400 flex items-center gap-1.5 font-black uppercase tracking-tighter">
                        <ShieldAlert className="w-3.5 h-3.5" /> failure
                      </span>
                    ) : (
                      <span className="text-amber-400 flex items-center gap-1.5 font-black uppercase tracking-tighter">
                        <Clock className="w-3.5 h-3.5" /> missed
                      </span>
                    )}
                  </td>
                  <td className="px-10 py-5 text-right max-w-md">
                    <code className="text-slate-500 group-hover:text-slate-300 transition-colors truncate block">
                      {log.error_message || log.llm_response || '-'}
                    </code>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </motion.div>
    </DashboardLayout>
  );
};

export default Monitor;
