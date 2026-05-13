import { useEffect, useState, useCallback } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { motion, AnimatePresence } from 'framer-motion';
import { Globe, ChevronDown, Trash2, Plus, Loader2, X } from 'lucide-react';

const WorkspaceEnvSection = ({ workspaceId }) => {
  const [envs, setEnvs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [newName, setNewName] = useState('');
  const [newValue, setNewValue] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const fetchEnvs = useCallback(async () => {
    setLoading(true);
    try {
      const res = await axios.get(`/api/v1/workspaces/${workspaceId}/env`);
      if (res.data.success) {
        setEnvs(res.data.data || []);
      }
    } catch {
      console.error('Failed to fetch env vars');
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  useEffect(() => {
    const init = async () => {
      await fetchEnvs();
    };
    init();
  }, [fetchEnvs]);

  const handleAdd = async (e) => {
    e.preventDefault();
    if (!newName || !newValue) return;
    setSubmitting(true);
    try {
      await axios.post(`/api/v1/workspaces/${workspaceId}/env`, {
        name: newName,
        value: newValue
      });
      setNewName('');
      setNewValue('');
      fetchEnvs();
    } catch {
      alert('Failed to add environment variable');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (name) => {
    if (!confirm(`Delete ${name}?`)) return;
    try {
      await axios.delete(`/api/v1/workspaces/${workspaceId}/env/${name}`);
      fetchEnvs();
    } catch {
      alert('Failed to delete environment variable');
    }
  };

  return (
    <div className="mt-4 pt-6 border-t border-white/5 space-y-6">
      <h3 className="text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Environment Variables</h3>
      
      {loading ? (
        <div className="py-4 flex items-center gap-3 text-xs text-slate-500 font-bold uppercase tracking-widest animate-pulse">
          <Loader2 size={14} className="animate-spin" /> Loading variables...
        </div>
      ) : (
        <div className="space-y-3">
          {envs.length === 0 ? (
            <div className="py-4 text-xs text-slate-600 italic font-medium uppercase tracking-wider">No environment variables defined.</div>
          ) : (
            envs.map(env => (
              <div key={env.name} className="flex items-center justify-between bg-black/40 p-4 rounded-2xl border border-white/5 group">
                <div className="flex flex-col gap-1">
                  <span className="text-xs font-mono font-black text-white tracking-tight">{env.name}</span>
                  <span className="text-[10px] font-mono text-slate-500 truncate max-w-[200px] md:max-w-md uppercase tracking-tighter">{env.value}</span>
                </div>
                <button 
                  onClick={() => handleDelete(env.name)}
                  className="p-3 text-slate-500 hover:text-red-400 hover:bg-red-500/10 rounded-xl transition-all opacity-0 group-hover:opacity-100"
                >
                  <Trash2 size={16} />
                </button>
              </div>
            ))
          )}
        </div>
      )}

      <form onSubmit={handleAdd} className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-2">
        <input 
          type="text" 
          placeholder="VARIABLE_NAME"
          value={newName}
          onChange={e => setNewName(e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, ''))}
          className="bg-black/60 border border-white/5 rounded-2xl px-6 py-4 text-xs text-white font-mono focus:outline-none focus:border-accent-orange/50 transition-colors"
        />
        <input 
          type="text" 
          placeholder="Value"
          value={newValue}
          onChange={e => setNewValue(e.target.value)}
          className="bg-black/60 border border-white/5 rounded-2xl px-6 py-4 text-xs text-white font-mono focus:outline-none focus:border-accent-orange/50 transition-colors"
        />
        <button 
          disabled={submitting || !newName || !newValue}
          className="bg-accent-orange text-white py-4 rounded-2xl text-[10px] font-black uppercase tracking-[0.2em] shadow-[0_10px_30px_rgba(217,119,6,0.2)] hover:scale-[1.02] active:scale-[0.98] transition-all disabled:opacity-50 disabled:grayscale flex items-center justify-center gap-3"
        >
          {submitting ? <Loader2 size={14} className="animate-spin" /> : <Plus size={14} />}
          Add Variable
        </button>
      </form>
    </div>
  );
};

const Workspaces = () => {
  const [workspaces, setWorkspaces] = useState([]);
  const [loading, setLoading] = useState(true);
  const [expandedId, setExpandedId] = useState(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newWorkspaceName, setNewWorkspaceName] = useState('');
  const [creating, setCreating] = useState(false);

  const fetchWorkspaces = useCallback(async () => {
    setLoading(true);
    try {
      const res = await axios.get('/api/v1/workspaces');
      if (res.data.success) setWorkspaces(res.data.data || []);
    } catch (err) {
      console.error('Failed to fetch workspaces', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    let isMounted = true;
    const load = async () => {
      if (isMounted) await fetchWorkspaces();
    };
    load();
    return () => { isMounted = false; };
  }, [fetchWorkspaces]);

  const handleCreateWorkspace = async (e) => {
    e.preventDefault();
    if (!newWorkspaceName) return;
    setCreating(true);
    try {
      const res = await axios.post('/api/v1/workspaces', { name: newWorkspaceName });
      if (res.data.success) {
        setNewWorkspaceName('');
        setShowCreateForm(false);
        fetchWorkspaces();
      }
    } catch (err) {
      console.error('Failed to create workspace', err);
      alert('Failed to create workspace');
    } finally {
      setCreating(false);
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
            Workspaces
          </motion.h1>
          <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em]">Compute contexts and isolation</p>
        </div>
        <button 
          onClick={() => setShowCreateForm(true)}
          className="bg-accent-orange text-white px-8 py-4 rounded-2xl text-xs font-black uppercase tracking-widest shadow-[0_10px_30px_rgba(217,119,6,0.3)] hover:scale-105 transition-transform flex items-center gap-2"
        >
          <Plus size={16} /> New Workspace
        </button>
      </header>

      {/* Create Workspace Modal */}
      <AnimatePresence>
        {showCreateForm && (
          <div className="fixed inset-0 z-[110] flex items-center justify-center p-6">
            <motion.div 
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              onClick={() => setShowCreateForm(false)}
              className="absolute inset-0 bg-black/80 backdrop-blur-md"
            />
            <motion.div 
              initial={{ opacity: 0, scale: 0.9, y: 20 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.9, y: 20 }}
              className="bg-zinc-900 border border-white/10 p-10 rounded-[2.5rem] shadow-2xl w-full max-w-lg relative z-10"
            >
              <div className="flex items-center justify-between mb-8">
                <h2 className="text-2xl font-black text-white uppercase tracking-tighter">New Workspace</h2>
                <button onClick={() => setShowCreateForm(false)} className="text-slate-500 hover:text-white transition-colors">
                  <X size={24} />
                </button>
              </div>
              <form onSubmit={handleCreateWorkspace} className="space-y-6">
                <div>
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] mb-3">Workspace Name</label>
                  <input 
                    type="text"
                    value={newWorkspaceName}
                    onChange={(e) => setNewWorkspaceName(e.target.value)}
                    placeholder="e.g. Production Context"
                    className="w-full bg-black/40 border border-white/5 rounded-2xl p-5 text-white font-mono text-sm focus:outline-none focus:border-accent-orange/50 transition-colors"
                  />
                </div>
                <button 
                  disabled={creating || !newWorkspaceName}
                  className="w-full bg-accent-orange text-white py-5 rounded-2xl text-xs font-black uppercase tracking-widest shadow-[0_10px_30px_rgba(217,119,6,0.3)] hover:brightness-110 transition-all flex items-center justify-center gap-3 disabled:opacity-50"
                >
                  {creating ? <Loader2 size={16} className="animate-spin" /> : <Plus size={16} />}
                  Create Workspace
                </button>
              </form>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      <div className="grid grid-cols-1 gap-6">
        {loading ? (
          <div className="p-20 flex flex-col items-center justify-center text-slate-500 gap-4">
            <Loader2 className="animate-spin" size={32} />
            <p className="text-xs font-black uppercase tracking-[0.2em]">Loading workspaces...</p>
          </div>
        ) : workspaces.length === 0 ? (
          <div className="p-20 flex flex-col items-center justify-center text-slate-500 gap-6 text-center">
            <div className="bg-white/5 p-6 rounded-3xl">
              <Globe size={48} className="text-slate-600" />
            </div>
            <div>
              <p className="text-white font-bold text-lg mb-1">No workspaces found</p>
              <p className="text-sm max-w-xs">Create your first workspace to start isolating your task environments.</p>
            </div>
          </div>
        ) : workspaces.map(w => (
          <motion.div 
            key={w.id} 
            layout
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="bg-black/40 border border-white/10 rounded-[2.5rem] p-8 hover:bg-black/60 transition-all cursor-pointer group backdrop-blur-3xl shadow-2xl"
            onClick={() => setExpandedId(expandedId === w.id ? null : w.id)}
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-6">
                <div className="bg-accent-orange/10 p-5 rounded-[1.5rem] text-accent-orange shadow-inner group-hover:scale-110 transition-transform duration-500">
                  <Globe size={28} />
                </div>
                <div>
                  <h2 className="text-2xl font-black text-white uppercase tracking-tighter mb-1">{w.name}</h2>
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] text-slate-500 font-mono uppercase tracking-[0.1em] px-2 py-0.5 bg-white/5 rounded-md">ID: {w.id ? w.id.substring(0, 8) : 'N/A'}...</span>
                    <span className="text-[10px] text-slate-500 font-medium uppercase tracking-widest">Created {new Date(w.created_at).toLocaleDateString()}</span>
                  </div>
                </div>
              </div>
              <motion.div 
                animate={{ rotate: expandedId === w.id ? 180 : 0 }}
                className={`w-12 h-12 rounded-2xl flex items-center justify-center border transition-colors ${expandedId === w.id ? 'bg-white/10 border-white/20 text-white' : 'bg-white/5 border-white/5 text-slate-500'}`}
              >
                <ChevronDown size={20} />
              </motion.div>
            </div>

            <AnimatePresence>
              {expandedId === w.id && (
                <motion.div
                  initial={{ height: 0, opacity: 0 }}
                  animate={{ height: 'auto', opacity: 1 }}
                  exit={{ height: 0, opacity: 0 }}
                  transition={{ duration: 0.4, ease: [0.23, 1, 0.32, 1] }}
                  className="overflow-hidden"
                  onClick={e => e.stopPropagation()}
                >
                  <WorkspaceEnvSection workspaceId={w.id} />
                </motion.div>
              )}
            </AnimatePresence>
          </motion.div>
        ))}
      </div>
    </DashboardLayout>
  );
};

export default Workspaces;
