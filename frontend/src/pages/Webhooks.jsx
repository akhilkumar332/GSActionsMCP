import { useEffect, useState, useCallback } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { Webhook, Trash2, Plus, ShieldCheck, Zap, Bell, Loader2, X, Activity } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';

const Webhooks = () => {
  const [webhooks, setWebhooks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [toasts, setToasts] = useState([]);
  const [showAddForm, setShowAddForm] = useState(false);
  const [newWebhook, setNewWebhook] = useState({ endpoint_url: '', event_types: ['task_executed'] });
  const [submitting, setSubmitting] = useState(false);

  const addToast = useCallback((message, type = 'success') => {
    const id = Date.now();
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 5000);
  }, []);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await axios.get('/api/v1/webhooks');
      if (res.data.success) {
        setWebhooks(res.data.data || []);
      }
    } catch {
      addToast('Failed to fetch webhooks', 'error');
    } finally {
      setLoading(false);
    }
  }, [addToast]);

  useEffect(() => {
    const init = async () => {
      await fetchData();
    };
    init();
  }, [fetchData]);

  const handleDelete = async (id) => {
    if (!confirm(`Are you sure you want to delete this webhook?`)) return;
    try {
      await axios.delete(`/api/v1/webhooks/${id}`);
      addToast(`Webhook deleted`);
      fetchData();
    } catch {
      addToast(`Failed to delete webhook`, 'error');
    }
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    if (!newWebhook.endpoint_url) {
      addToast('Endpoint URL is required', 'error');
      return;
    }
    setSubmitting(true);
    try {
      await axios.post('/api/v1/webhooks', newWebhook);
      addToast(`Webhook registered successfully`);
      setNewWebhook({ endpoint_url: '', event_types: ['task_executed'] });
      setShowAddForm(false);
      fetchData();
    } catch {
      addToast('Failed to register webhook', 'error');
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
            Outbound Webhooks
          </motion.h1>
          <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em]">Event-driven integrations</p>
        </div>
        <button 
          onClick={() => setShowAddForm(true)}
          className="bg-accent-orange text-white px-8 py-4 rounded-2xl text-xs font-black uppercase tracking-widest shadow-[0_10px_30px_rgba(217,119,6,0.3)] hover:scale-105 transition-transform flex items-center gap-2"
        >
          <Plus size={16} /> Add Webhook
        </button>
      </header>

      {/* Add Webhook Form Modal */}
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
                <h2 className="text-2xl font-black text-white uppercase tracking-tighter">New Webhook</h2>
                <button onClick={() => setShowAddForm(false)} className="text-slate-500 hover:text-white transition-colors">
                  <X size={24} />
                </button>
              </div>
              <form onSubmit={handleCreate} className="space-y-6">
                <div>
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] mb-3">Endpoint URL</label>
                  <input 
                    type="url"
                    value={newWebhook.endpoint_url}
                    onChange={(e) => setNewWebhook({...newWebhook, endpoint_url: e.target.value})}
                    placeholder="https://api.yourdomain.com/webhook"
                    className="w-full bg-black/40 border border-white/5 rounded-2xl p-5 text-white font-mono text-sm focus:outline-none focus:border-accent-orange/50 transition-colors"
                  />
                </div>
                <button 
                  disabled={submitting}
                  className="w-full bg-accent-orange text-white py-5 rounded-2xl text-xs font-black uppercase tracking-widest shadow-[0_10px_30px_rgba(217,119,6,0.3)] hover:brightness-110 transition-all flex items-center justify-center gap-3 disabled:opacity-50"
                >
                  {submitting ? <Loader2 size={16} className="animate-spin" /> : <ShieldCheck size={16} />}
                  Register Webhook
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
            <p className="text-xs font-black uppercase tracking-[0.2em]">Loading Webhooks...</p>
          </div>
        ) : webhooks.length === 0 ? (
          <div className="p-20 flex flex-col items-center justify-center text-slate-500 gap-6 text-center">
            <div className="bg-white/5 p-6 rounded-3xl">
              <Webhook size={48} className="text-slate-600" />
            </div>
            <div>
              <p className="text-white font-bold text-lg mb-1">No webhooks configured</p>
              <p className="text-sm max-w-xs">Register a webhook to receive real-time HTTP POST requests when events occur.</p>
            </div>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-white/5 bg-black/40">
                  <th className="p-8 text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Endpoint</th>
                  <th className="p-8 text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Status</th>
                  <th className="p-8 text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Created</th>
                  <th className="p-8 text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {webhooks.map((webhook) => (
                  <motion.tr 
                    layout
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    key={webhook.id} 
                    className="border-b border-white/5 hover:bg-white/[0.02] transition-colors group"
                  >
                    <td className="p-8">
                      <div className="flex items-center gap-4">
                        <div className="bg-blue-500/10 p-2 rounded-lg text-blue-500">
                          <Activity size={14} />
                        </div>
                        <span className="text-sm font-bold text-white font-mono truncate max-w-xs block">{webhook.endpoint_url}</span>
                      </div>
                    </td>
                    <td className="p-8">
                      {webhook.is_active ? (
                        <span className="text-emerald-400 flex items-center gap-1.5 font-black uppercase tracking-tighter text-xs">
                           Active
                        </span>
                      ) : (
                        <span className="text-slate-400 flex items-center gap-1.5 font-black uppercase tracking-tighter text-xs">
                           Inactive
                        </span>
                      )}
                    </td>
                    <td className="p-8">
                      <span className="text-xs text-slate-500 font-medium">{new Date(webhook.created_at).toLocaleString()}</span>
                    </td>
                    <td className="p-8 text-right">
                      <button 
                        onClick={() => handleDelete(webhook.id)}
                        className="p-3 text-slate-500 hover:text-red-400 hover:bg-red-500/10 rounded-xl transition-all"
                        title="Delete Webhook"
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

export default Webhooks;
