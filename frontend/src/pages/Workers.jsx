import { useEffect, useState, useCallback } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { Cpu, ShieldCheck, Activity, Loader2, ArrowLeft, RefreshCw, Server } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { useNavigate } from 'react-router-dom';

const Workers = () => {
  const [workers, setWorkers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const navigate = useNavigate();

  const fetchWorkers = useCallback(async () => {
    setRefreshing(true);
    try {
      const res = await axios.get('/api/admin/insights'); // We'll need a specific endpoint or use insights
      if (res.data.success) {
        // For now, we'll derive or use the insights data. 
        // Backend actually needs a GetWorkers endpoint for full detail, 
        // but we can start with the count from insights or mock for the design phase.
        setWorkers([{
          worker_id: 'node-alpha-1',
          hostname: 'worker-01.prod.internal',
          last_heartbeat: new Date().toISOString(),
          status: 'online',
          task_count: 5
        }]);
      }
    } catch {
      console.error('Failed to fetch workers');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchWorkers();
    const interval = setInterval(fetchWorkers, 30000);
    return () => clearInterval(interval);
  }, [fetchWorkers]);

  return (
    <DashboardLayout>
      <header className="mb-12">
        <button 
          onClick={() => navigate('/insights')}
          className="flex items-center gap-2 text-slate-500 hover:text-white transition-colors text-xs font-black uppercase tracking-widest mb-6"
        >
          <ArrowLeft size={14} /> Back to Insights
        </button>
        <div className="flex items-center justify-between">
          <div>
            <motion.h1 
              initial={{ opacity: 0, y: -20 }}
              animate={{ opacity: 1, y: 0 }}
              className="text-4xl font-black text-white tracking-tight mb-2"
            >
              Node Registry
            </motion.h1>
            <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em]">Active Execution Infrastructure</p>
          </div>
          <button 
            onClick={fetchWorkers}
            className="p-4 bg-white/5 border border-white/10 rounded-2xl text-slate-400 hover:text-white transition-all"
          >
            <RefreshCw size={20} className={refreshing ? 'animate-spin' : ''} />
          </button>
        </div>
      </header>

      <div className="grid grid-cols-1 gap-6">
        {loading ? (
          <div className="py-32 flex flex-col items-center gap-4">
            <Loader2 className="animate-spin text-accent-orange" size={32} />
            <p className="text-xs font-black uppercase tracking-widest text-slate-500">Scanning Cluster...</p>
          </div>
        ) : (
          workers.map((worker) => (
            <motion.div 
              key={worker.worker_id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              className="bg-white/5 border border-white/10 rounded-[2.5rem] p-8 hover:bg-white/[0.08] transition-all group"
            >
              <div className="flex flex-col md:flex-row md:items-center justify-between gap-8">
                <div className="flex items-center gap-6">
                  <div className="bg-blue-500/10 p-5 rounded-[1.5rem] text-blue-400">
                    <Server size={28} />
                  </div>
                  <div>
                    <div className="flex items-center gap-3 mb-1">
                      <h2 className="text-2xl font-black text-white uppercase tracking-tighter">{worker.hostname}</h2>
                      <span className="flex items-center gap-1.5 px-3 py-1 bg-emerald-500/10 text-emerald-400 text-[10px] font-black uppercase tracking-widest rounded-full">
                        <Activity size={10} className="animate-pulse" /> {worker.status}
                      </span>
                    </div>
                    <p className="text-[10px] font-mono text-slate-500 uppercase tracking-widest">ID: {worker.worker_id}</p>
                  </div>
                </div>

                <div className="grid grid-cols-2 md:flex items-center gap-12">
                  <div className="text-center md:text-left">
                    <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest mb-1">Active Tasks</p>
                    <p className="text-xl font-black text-white">{worker.task_count}</p>
                  </div>
                  <div className="text-center md:text-left">
                    <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest mb-1">Last Heartbeat</p>
                    <p className="text-xs font-bold text-slate-300 uppercase tracking-tight">{new Date(worker.last_heartbeat).toLocaleTimeString()}</p>
                  </div>
                </div>
              </div>
            </motion.div>
          ))
        )}
      </div>
    </DashboardLayout>
  );
};

export default Workers;
