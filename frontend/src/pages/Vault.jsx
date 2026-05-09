import { useEffect, useState, useCallback } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { Key, Trash2, Plus, ShieldCheck, ShieldAlert, Zap, Bell, Loader2, X } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';

const Vault = () => {
  const [secrets, setSecrets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [toasts, setToasts] = useState([]);
  const [showAddForm, setShowAddForm] = useState(false);
  const [newSecret, setNewSecret] = useState({ name: '', value: '' });
  const [submitting, setSubmitting] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await axios.get('/api/secrets');
      if (res.data.success) {
        setSecrets(res.data.data);
      }
    } catch {
      addToast('Failed to fetch secrets', 'error');
    } finally {
      setLoading(false);
    }
  }, []);

  const addToast = useCallback((message, type = 'success') => {
    const id = Date.now();
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 5000);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleDelete = async (name) => {
    if (!confirm(`Are you sure you want to delete secret "${name}"?`)) return;
    try {
      await axios.delete(`/api/secrets/${name}`);
      addToast(`Secret "${name}" deleted`);
      fetchData();
    } catch {
      addToast(`Failed to delete secret "${name}"`, 'error');
    }
  };

  const handleUpsert = async (e) => {
    e.preventDefault();
    if (!newSecret.name || !newSecret.value) {
      addToast('Name and value are required', 'error');
      return;
    }
    setSubmitting(true);
    try {
      await axios.post('/api/secrets', newSecret);
      addToast(`Secret "${newSecret.name}" stored successfully`);
      setNewSecret({ name: '', value: '' });
      setShowAddForm(false);
      fetchData();
    } catch {
      addToast('Failed to store secret', 'error');
    } finally {
      setSubmitting(false);
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
            Secret Vault
          </motion.h1>
          <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em]">Security Status: <span className="text-emerald-500">Encrypted</span></p>
        </div>
        <button 
          onClick={() => setShowAddForm(true)}
          className="bg-accent-orange text-white px-8 py-4 rounded-2xl text-xs font-black uppercase tracking-widest shadow-[0_10px_30px_rgba(217,119,6,0.3)] hover:scale-105 transition-transform flex items-center gap-2"
        >
          <Plus size={16} /> Store New Secret
        </button>
      </header>

      {/* Add Secret Form Modal */}
      <AnimatePresence>
        {showAddForm && (
          <div className="fixed inset-0 z-[110] flex items-center justify-center p-6">
            <motion.div 
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              onClick={() => setShowAddForm(false)}
              className="absolute inset-0 bg-black/80 backdrop-blur-md"
            />
            <motion.div 
              initial={{ opacity: 0, scale: 0.9, y: 20 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.9, y: 20 }}
              className="bg-zinc-900 border border-white/10 p-10 rounded-[2.5rem] shadow-2xl w-full max-w-lg relative z-10"
            >
              <div className="flex items-center justify-between mb-8">
                <h2 className="text-2xl font-black text-white uppercase tracking-tighter">New Secret</h2>
                <button onClick={() => setShowAddForm(false)} className="text-slate-500 hover:text-white transition-colors">
                  <X size={24} />
                </button>
              </div>
              <form onSubmit={handleUpsert} className="space-y-6">
                <div>
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] mb-3">Secret Name</label>
                  <input 
                    type="text"
                    value={newSecret.name}
                    onChange={(e) => setNewSecret({...newSecret, name: e.target.value})}
                    placeholder="e.g. OPENAI_API_KEY"
                    className="w-full bg-black/40 border border-white/5 rounded-2xl p-5 text-white font-mono text-sm focus:outline-none focus:border-accent-orange/50 transition-colors"
                  />
                </div>
                <div>
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] mb-3">Secret Value</label>
                  <textarea 
                    value={newSecret.value}
                    onChange={(e) => setNewSecret({...newSecret, value: e.target.value})}
                    placeholder="Sensitive data will be encrypted before storage"
                    rows={4}
                    className="w-full bg-black/40 border border-white/5 rounded-2xl p-5 text-white font-mono text-sm focus:outline-none focus:border-accent-orange/50 transition-colors resize-none"
                  />
                </div>
                <button 
                  disabled={submitting}
                  className="w-full bg-accent-orange text-white py-5 rounded-2xl text-xs font-black uppercase tracking-widest shadow-[0_10px_30px_rgba(217,119,6,0.3)] hover:brightness-110 transition-all flex items-center justify-center gap-3 disabled:opacity-50"
                >
                  {submitting ? <Loader2 size={16} className="animate-spin" /> : <ShieldCheck size={16} />}
                  Secure Storage
                </button>
              </form>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      <div className="bg-white/5 rounded-[2.5rem] border border-white/10 shadow-2xl overflow-hidden backdrop-blur-xl">
        {loading ? (
          <div className="p-20 flex flex-col items-center justify-center text-slate-500 gap-4">
            <Loader2 className="animate-spin" size={32} />
            <p className="text-xs font-black uppercase tracking-[0.2em]">Synchronizing Vault...</p>
          </div>
        ) : secrets.length === 0 ? (
          <div className="p-20 flex flex-col items-center justify-center text-slate-500 gap-6 text-center">
            <div className="bg-white/5 p-6 rounded-3xl">
              <Key size={48} className="text-slate-600" />
            </div>
            <div>
              <p className="text-white font-bold text-lg mb-1">Your vault is empty</p>
              <p className="text-sm max-w-xs">Securely store API keys and credentials for use in your scheduled tasks.</p>
            </div>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-white/5">
                  <th className="p-8 text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Secret Name</th>
                  <th className="p-8 text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Created At</th>
                  <th className="p-8 text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {secrets.map((secret) => (
                  <motion.tr 
                    layout
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    key={secret.name} 
                    className="border-b border-white/5 hover:bg-white/[0.02] transition-colors group"
                  >
                    <td className="p-8">
                      <div className="flex items-center gap-4">
                        <div className="bg-emerald-500/10 p-2 rounded-lg text-emerald-500">
                          <Key size={14} />
                        </div>
                        <span className="text-sm font-bold text-white font-mono">{secret.name}</span>
                      </div>
                    </td>
                    <td className="p-8">
                      <span className="text-xs text-slate-500 font-medium">{new Date(secret.created_at).toLocaleString()}</span>
                    </td>
                    <td className="p-8 text-right">
                      <button 
                        onClick={() => handleDelete(secret.name)}
                        className="p-3 text-slate-500 hover:text-red-400 hover:bg-red-500/10 rounded-xl transition-all"
                        title="Delete Secret"
                      >
                        <Trash2 size={18} />
                      </button>
                    </td>
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Real-time Toast Notifications */}
      <div className="fixed bottom-8 right-8 z-[100] flex flex-col gap-3 pointer-events-none">
        <AnimatePresence>
          {toasts.map((toast) => (
            <motion.div
              key={toast.id}
              initial={{ opacity: 0, x: 50, scale: 0.9 }}
              animate={{ opacity: 1, x: 0, scale: 1 }}
              exit={{ opacity: 0, x: 20, scale: 0.9 }}
              className={`pointer-events-auto px-6 py-4 rounded-2xl shadow-[0_20px_50px_rgba(0,0,0,0.5)] border flex items-center gap-4 backdrop-blur-2xl min-w-[300px] ${
                toast.type === 'success' 
                  ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400' 
                  : 'bg-red-500/10 border-red-500/20 text-red-400'
              }`}
            >
              <div className={`p-2 rounded-xl ${toast.type === 'success' ? 'bg-emerald-500/20' : 'bg-red-500/20'}`}>
                {toast.type === 'success' ? <Zap size={16} /> : <Bell size={16} />}
              </div>
              <span className="text-xs font-black uppercase tracking-widest">{toast.message}</span>
            </motion.div>
          ))}
        </AnimatePresence>
      </div>
    </DashboardLayout>
  );
};

export default Vault;
