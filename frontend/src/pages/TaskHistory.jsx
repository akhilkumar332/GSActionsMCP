import { useEffect, useState, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { History, ArrowLeft, RefreshCw, Clock, CheckCircle2, AlertCircle } from 'lucide-react';
import { motion } from 'framer-motion';

const TaskHistory = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(true);
  const [restoring, setRestoring] = useState(null);

  const fetchHistory = useCallback(async () => {
    try {
      const res = await axios.get(`/api/v1/tasks/${id}/versions`);
      if (res.data.success) {
        setHistory(res.data.data || []);
      }
    } catch {
      console.error('Failed to fetch task history');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    const init = async () => {
      await fetchHistory();
    };
    init();
  }, [fetchHistory]);

  const handleRestore = async (versionId) => {
    if (!confirm('Are you sure you want to restore this version? Current configuration will be saved as a new version.')) return;
    
    setRestoring(versionId);
    try {
      await axios.post(`/api/v1/tasks/${id}/restore/${versionId}`);
      alert('Task restored successfully');
      fetchHistory();
    } catch {
      alert('Failed to restore task');
    } finally {
      setRestoring(null);
    }
  };

  return (
    <DashboardLayout>
      <header className="mb-12 flex items-center justify-between">
        <div>
          <button 
            onClick={() => navigate('/tasks')}
            className="flex items-center gap-2 text-slate-500 hover:text-white transition-colors text-xs font-black uppercase tracking-widest mb-6"
          >
            <ArrowLeft size={14} /> Back to Tasks
          </button>
          <motion.h1 
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            className="text-4xl font-black text-white tracking-tight mb-2"
          >
            Version History
          </motion.h1>
          <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em]">Task: {id}</p>
        </div>
        <div className="bg-blue-500/10 p-4 rounded-3xl text-blue-400">
          <History size={32} />
        </div>
      </header>

      <div className="space-y-6">
        {loading ? (
          <div className="flex flex-col items-center py-32 gap-4">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-orange"></div>
            <p className="text-slate-500 font-black uppercase tracking-widest text-xs">Loading Timeline...</p>
          </div>
        ) : history.length === 0 ? (
          <div className="bg-white/5 border border-white/10 rounded-[2.5rem] p-20 text-center">
            <AlertCircle className="mx-auto text-slate-600 mb-6" size={48} />
            <p className="text-slate-500 font-black uppercase tracking-widest text-sm">No history found for this task.</p>
          </div>
        ) : (
          <div className="relative">
            {/* Timeline Line */}
            <div className="absolute left-[31px] top-0 bottom-0 w-px bg-gradient-to-b from-blue-500/50 via-white/10 to-transparent"></div>
            
            <div className="space-y-12">
              {history.map((version, index) => (
                <motion.div 
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: index * 0.1 }}
                  key={version.id}
                  className="relative pl-20"
                >
                  {/* Timeline Dot */}
                  <div className={`absolute left-0 w-16 h-16 flex items-center justify-center rounded-2xl border backdrop-blur-xl z-10 ${
                    index === 0 ? 'bg-blue-500/20 border-blue-500/50 text-blue-400' : 'bg-white/5 border-white/10 text-slate-500'
                  }`}>
                    {index === 0 ? <CheckCircle2 size={24} /> : <Clock size={24} />}
                  </div>

                  <div className="bg-white/5 border border-white/10 rounded-[2.5rem] p-8 hover:bg-white/[0.08] transition-all duration-500 group">
                    <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-4">
                          <span className="text-white font-black text-xl tracking-tighter">
                            {index === 0 ? 'Current Version' : `Version ${history.length - index}`}
                          </span>
                          <span className="text-[10px] font-mono text-slate-500 bg-white/5 px-2 py-1 rounded-lg">
                            {version.id.substring(0, 8)}
                          </span>
                        </div>
                        
                        <div className="space-y-4">
                          <div>
                            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest mb-2">Prompt Configuration</p>
                            <div className="bg-black/40 p-4 rounded-2xl border border-white/5 font-mono text-xs text-slate-300 leading-relaxed max-h-32 overflow-y-auto">
                              {version.agent_prompt}
                            </div>
                          </div>
                          
                          <div className="flex items-center gap-6">
                            <div>
                              <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest mb-1">Created At</p>
                              <p className="text-xs font-bold text-slate-400">{new Date(version.created_at).toLocaleString()}</p>
                            </div>
                            <div>
                              <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest mb-1">Trigger</p>
                              <p className="text-xs font-bold text-slate-400 uppercase tracking-widest">{version.trigger_type}</p>
                            </div>
                          </div>
                        </div>
                      </div>

                      {index !== 0 && (
                        <div className="flex items-center">
                          <button 
                            onClick={() => handleRestore(version.id)}
                            disabled={restoring === version.id}
                            className="bg-blue-500 text-white px-8 py-4 rounded-2xl font-black text-xs uppercase tracking-widest shadow-[0_10px_30px_rgba(59,130,246,0.3)] hover:scale-105 active:scale-95 transition-all flex items-center gap-3 disabled:opacity-50"
                          >
                            <RefreshCw size={14} className={restoring === version.id ? 'animate-spin' : ''} />
                            {restoring === version.id ? 'Restoring...' : 'Restore This Version'}
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </motion.div>
              ))}
            </div>
          </div>
        )}
      </div>
    </DashboardLayout>
  );
};

export default TaskHistory;
