import { useEffect, useState } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { Settings, Save, Trash2, RefreshCw, AlertCircle, CheckCircle2 } from 'lucide-react';
import { motion } from 'framer-motion';

const AdminSettings = () => {
  const [settings, setSettings] = useState({ worker_prune_days: 7 });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [pruning, setPruning] = useState(false);
  const [message, setMessage] = useState({ type: '', text: '' });

  const fetchSettings = async () => {
    try {
      const res = await axios.get('/api/v1/admin/settings');
      if (res.data.success) {
        setSettings(res.data.data || { worker_prune_days: 7 });
      }
    } catch (err) {
      console.error('Failed to fetch settings', err);
      setMessage({ type: 'error', text: 'Failed to load system settings.' });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchSettings();
  }, []);

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    setMessage({ type: '', text: '' });
    try {
      const res = await axios.post('/api/v1/admin/settings', settings);
      if (res.data.success) {
        setMessage({ type: 'success', text: 'Settings updated successfully.' });
      } else {
        setMessage({ type: 'error', text: res.data.message || 'Failed to update settings.' });
      }
    } catch (err) {
      console.error('Failed to save settings', err);
      setMessage({ type: 'error', text: 'An error occurred while saving settings.' });
    } finally {
      setSaving(false);
    }
  };

  const handlePruneNow = async () => {
    if (!window.confirm('Are you sure you want to prune inactive workers now? This action cannot be undone.')) {
      return;
    }
    setPruning(true);
    setMessage({ type: '', text: '' });
    try {
      const res = await axios.post('/api/v1/admin/prune');
      if (res.data.success) {
        setMessage({ type: 'success', text: res.data.message || 'Worker pruning completed.' });
      } else {
        setMessage({ type: 'error', text: res.data.message || 'Failed to prune workers.' });
      }
    } catch (err) {
      console.error('Failed to prune workers', err);
      setMessage({ type: 'error', text: 'An error occurred while pruning workers.' });
    } finally {
      setPruning(false);
    }
  };

  return (
    <DashboardLayout>
      <header className="mb-12">
        <h1 className="text-4xl font-black text-white tracking-tight">System Control</h1>
        <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em] mt-2">Core Platform Configuration & Maintenance</p>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div className="lg:col-span-2 space-y-8">
          <motion.div 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="bg-black/40 rounded-[2.5rem] border border-white/5 shadow-2xl overflow-hidden backdrop-blur-3xl p-10"
          >
            <div className="flex items-center gap-4 mb-8">
              <div className="bg-accent-orange/10 p-3 rounded-2xl text-accent-orange border border-accent-orange/20">
                <Settings className="w-6 h-6" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-white">Worker Management</h2>
                <p className="text-xs text-slate-500 font-medium tracking-wider uppercase">Auto-maintenance settings</p>
              </div>
            </div>

            <form onSubmit={handleSave} className="space-y-6">
              <div>
                <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] mb-3 ml-1">
                  Worker Pruning Threshold (Days)
                </label>
                <div className="relative group">
                  <input 
                    type="number" 
                    min="1"
                    max="365"
                    value={settings.worker_prune_days}
                    onChange={(e) => setSettings({ ...settings, worker_prune_days: parseInt(e.target.value) || 0 })}
                    className="w-full bg-white/5 border border-white/10 rounded-2xl px-6 py-4 text-white outline-none focus:border-accent-orange/50 transition-all backdrop-blur-xl"
                    placeholder="7"
                    disabled={loading}
                  />
                </div>
                <p className="mt-3 text-[11px] text-slate-500 italic px-1">
                  Inactive workers will be automatically removed from the registry after this many days of silence.
                </p>
              </div>

              <div className="pt-4 flex items-center gap-4">
                <button 
                  type="submit"
                  disabled={loading || saving}
                  className="flex items-center gap-3 px-8 py-4 bg-accent-orange hover:bg-accent-orange/90 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-2xl font-bold text-sm transition-all shadow-[0_0_30px_rgba(217,119,6,0.3)] group"
                >
                  {saving ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4 group-hover:scale-110 transition-transform" />}
                  Save Configuration
                </button>

                {message.text && (
                  <div className={`flex items-center gap-2 text-xs font-bold uppercase tracking-widest ${message.type === 'success' ? 'text-emerald-400' : 'text-red-400'}`}>
                    {message.type === 'success' ? <CheckCircle2 className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
                    {message.text}
                  </div>
                )}
              </div>
            </form>
          </motion.div>

          <motion.div 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="bg-red-500/5 rounded-[2.5rem] border border-red-500/10 shadow-2xl overflow-hidden backdrop-blur-3xl p-10"
          >
            <div className="flex items-center gap-4 mb-8">
              <div className="bg-red-500/10 p-3 rounded-2xl text-red-400 border border-red-500/20">
                <Trash2 className="w-6 h-6" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-white">Manual Pruning</h2>
                <p className="text-xs text-slate-500 font-medium tracking-wider uppercase">Danger Zone</p>
              </div>
            </div>

            <p className="text-sm text-slate-400 mb-8 leading-relaxed">
              Trigger an immediate cleanup of inactive workers based on the current threshold. 
              This will forcefully remove any worker that has not sent a heartbeat within the configured window.
            </p>

            <button 
              onClick={handlePruneNow}
              disabled={loading || pruning}
              className="flex items-center gap-3 px-8 py-4 bg-red-500/10 border border-red-500/20 hover:bg-red-500/20 text-red-400 rounded-2xl font-bold text-sm transition-all group"
            >
              {pruning ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4 group-hover:scale-110 transition-transform" />}
              Prune Inactive Workers Now
            </button>
          </motion.div>
        </div>

        <div className="space-y-8">
          <motion.div 
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            className="bg-white/[0.02] rounded-[2.5rem] border border-white/5 p-8 backdrop-blur-3xl"
          >
            <h3 className="text-sm font-black text-white uppercase tracking-[0.2em] mb-6">Maintenance Intelligence</h3>
            <div className="space-y-6">
              <div className="flex items-start gap-4">
                <div className="w-1 h-1 rounded-full bg-accent-orange mt-1.5 shrink-0" />
                <p className="text-xs text-slate-400 leading-relaxed">
                  Worker pruning ensures the System Monitor remains accurate by removing stale worker registrations.
                </p>
              </div>
              <div className="flex items-start gap-4">
                <div className="w-1 h-1 rounded-full bg-accent-orange mt-1.5 shrink-0" />
                <p className="text-xs text-slate-400 leading-relaxed">
                  Default threshold is 7 days. Reducing this may lead to active workers being pruned if they have connectivity issues.
                </p>
              </div>
              <div className="flex items-start gap-4">
                <div className="w-1 h-1 rounded-full bg-accent-orange mt-1.5 shrink-0" />
                <p className="text-xs text-slate-400 leading-relaxed">
                  Pruning is a background process that runs every 24 hours automatically.
                </p>
              </div>
            </div>
          </motion.div>
        </div>
      </div>
    </DashboardLayout>
  );
};

export default AdminSettings;
