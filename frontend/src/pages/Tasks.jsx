import { useEffect, useState, useCallback } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import TaskWizard from '../components/TaskWizard';
import axios from 'axios';
import { Play, Pause, Trash2, CheckCircle2, ShieldAlert, Cpu, Link, History, Globe, Plus } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { useNavigate } from 'react-router-dom';

const Tasks = () => {
  const navigate = useNavigate();
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [isWizardOpen, setIsWizardOpen] = useState(false);

  const fetchTasks = useCallback(async () => {
    try {
      const res = await axios.get('/api/v1/tasks');
      if (res.data.success) {
        setTasks(res.data.data || []);
      }
    } catch {
      console.error('Failed to fetch tasks');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const init = async () => {
      await fetchTasks();
    };
    init();
  }, [fetchTasks]);

  const handleAction = async (taskId, action) => {
    if (action === 'delete' && !confirm('Are you sure you want to delete this task?')) return;
    
    try {
      if (action === 'delete') {
        await axios.delete(`/api/v1/tasks/${taskId}`);
      } else {
        await axios.post(`/api/v1/tasks/${taskId}/${action}`);
      }
      fetchTasks();
    } catch {
      alert(`Failed to ${action} task`);
    }
  };

  return (
    <DashboardLayout>
      <header className="mb-12 flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div>
          <motion.h1 
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            className="text-4xl font-black text-white tracking-tight mb-2"
          >
            Task Management
          </motion.h1>
          <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em]">Active schedules and dependencies</p>
        </div>
        <button 
          onClick={() => setIsWizardOpen(true)}
          className="bg-accent-orange text-white px-8 py-4 rounded-2xl text-xs font-black uppercase tracking-widest shadow-[0_10px_30px_rgba(217,119,6,0.3)] hover:scale-105 transition-transform flex items-center gap-2"
        >
          <Plus size={16} /> New Task
        </button>
      </header>

      <AnimatePresence>
        {isWizardOpen && (
          <TaskWizard 
            isOpen={isWizardOpen} 
            onClose={() => setIsWizardOpen(false)} 
            onTaskCreated={() => fetchTasks()} 
          />
        )}
      </AnimatePresence>

      <div className="bg-black/60 rounded-[2.5rem] border border-white/10 shadow-[0_40px_100px_rgba(0,0,0,0.5)] overflow-hidden backdrop-blur-3xl">
        <div className="overflow-x-auto min-h-[400px]">
          <table className="w-full text-left border-collapse text-sm">
            <thead className="bg-white/5 text-slate-500 uppercase tracking-widest text-[10px] font-black">
              <tr>
                <th className="px-8 py-5 border-b border-white/5">Task Name</th>
                <th className="px-6 py-5 border-b border-white/5">Schedule</th>
                <th className="px-6 py-5 border-b border-white/5">Status</th>
                <th className="px-6 py-5 border-b border-white/5">Next Run</th>
                <th className="px-8 py-5 border-b border-white/5 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {loading ? (
                <tr>
                  <td colSpan="5" className="px-10 py-32 text-center text-slate-500 font-bold uppercase tracking-widest">
                     <div className="flex flex-col items-center gap-4">
                        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-orange"></div>
                        Loading...
                     </div>
                  </td>
                </tr>
              ) : tasks.length === 0 ? (
                <tr>
                  <td colSpan="5" className="px-10 py-32 text-center text-slate-500 font-bold uppercase tracking-widest italic opacity-50">
                    No active tasks.
                  </td>
                </tr>
              ) : tasks.map((task) => (
                <motion.tr 
                  layout 
                  initial={{ opacity: 0 }} 
                  animate={{ opacity: 1 }}
                  key={task.id} 
                  className="hover:bg-white/[0.03] transition-colors group"
                >
                  <td className="px-8 py-5">
                    <div className="flex items-center gap-3">
                       <Cpu size={16} className="text-accent-orange" />
                       <div>
                          <div className="flex items-center gap-2">
                            <div className="text-white font-bold">{task.name}</div>
                            {task.version_count > 0 && (
                              <span className="text-[9px] font-black bg-blue-500/20 text-blue-400 px-1.5 py-0.5 rounded-md uppercase tracking-tighter" title={`${task.version_count} versions available`}>
                                v{task.version_count}
                              </span>
                            )}
                            <span className="text-[10px] text-slate-500 font-mono">
                              {task.agent_prompt ? (task.agent_prompt.includes('SELECT') ? 'SQL' : 'PROMPT') : 'N/A'}
                            </span>
                            {task.agent_prompt?.includes('{{env.') && (
                              <Globe size={10} className="text-emerald-500" title="Uses Workspace Environment Variables" />
                            )}
                          </div>
                          {task.depends_on_task_id && (
                            <div className="text-[10px] text-slate-500 flex items-center gap-1 mt-1 uppercase tracking-widest font-bold">
                              <Link size={10} className="text-blue-400" /> Depends on: {task.depends_on_task_id.substring(0, 8)}...
                            </div>
                          )}
                       </div>
                    </div>
                  </td>
                  <td className="px-6 py-5">
                     <div className="text-slate-300 font-mono text-xs">{task.trigger_type}</div>
                  </td>
                  <td className="px-6 py-5">
                    {task.status === 'active' ? (
                      <span className="text-emerald-400 flex items-center gap-1.5 font-black uppercase tracking-tighter text-xs">
                        <CheckCircle2 className="w-4 h-4 shadow-[0_0_10px_rgba(52,211,153,0.5)]" /> Active
                      </span>
                    ) : task.status === 'paused' ? (
                      <span className="text-amber-400 flex items-center gap-1.5 font-black uppercase tracking-tighter text-xs">
                        <Pause className="w-4 h-4" /> Paused
                      </span>
                    ) : (
                      <span className="text-slate-400 flex items-center gap-1.5 font-black uppercase tracking-tighter text-xs">
                        <ShieldAlert className="w-4 h-4" /> {task.status}
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-5 text-slate-400 text-xs font-mono">
                    {new Date(task.next_run).toLocaleString()}
                  </td>
                  <td className="px-8 py-5 text-right">
                     <div className="flex justify-end gap-3 opacity-50 group-hover:opacity-100 transition-opacity">
                       <button 
                         onClick={() => navigate(`/tasks/${task.id}/history`)} 
                         className="p-2 bg-white/5 hover:bg-blue-500/20 text-blue-500 rounded-xl transition-all" 
                         title="View History"
                       >
                         <History size={16} />
                       </button>
                       {task.status === 'active' ? (
                         <button onClick={() => handleAction(task.id, 'pause')} className="p-2 bg-white/5 hover:bg-amber-500/20 text-amber-500 rounded-xl transition-all" title="Pause Task"><Pause size={16} /></button>
                       ) : (
                         <button onClick={() => handleAction(task.id, 'resume')} className="p-2 bg-white/5 hover:bg-emerald-500/20 text-emerald-500 rounded-xl transition-all" title="Resume Task"><Play size={16} /></button>
                       )}
                       <button onClick={() => handleAction(task.id, 'delete')} className="p-2 bg-white/5 hover:bg-red-500/20 text-red-500 rounded-xl transition-all" title="Delete Task"><Trash2 size={16} /></button>
                     </div>
                  </td>
                </motion.tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </DashboardLayout>
  );
};

export default Tasks;
