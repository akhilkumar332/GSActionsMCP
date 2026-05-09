import { useEffect, useState, useCallback } from 'react';
import { useAuth } from '../context/AuthContext';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { Crown, ListChecks, Key, RefreshCw, Copy, Check, ShieldCheck, Zap, ArrowRight, Bell, ShieldAlert } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { useSSE } from '../hooks/useSSE';

const Dashboard = () => {
  const { user, checkAuth } = useAuth();
  const [taskCount, setTaskCount] = useState(0);
  const [copied, setCopied] = useState(false);
  const [rotating, setRotating] = useState(false);
  const [toasts, setToasts] = useState([]);
  const [pendingApprovals, setPendingApprovals] = useState([]);

  const fetchData = useCallback(async () => {
    try {
      const res = await axios.get('/api/dashboard');
      if (res.data.success) {
        setTaskCount(res.data.data.taskCount);
      }
    } catch {
      console.error('Failed to fetch dashboard data');
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
    const init = async () => {
      await fetchData();
    };
    init();
  }, [fetchData]);

  useSSE(useCallback((event) => {
    console.log('Dashboard received SSE event:', event);
    
    if (event.event_type === 'task_executed') {
      try {
        const payload = typeof event.payload === 'string' ? JSON.parse(event.payload) : event.payload;
        addToast(
          `Task ${payload.task_name || payload.task_id.slice(0, 8)} executed: ${payload.status}`, 
          payload.status === 'success' ? 'success' : 'error'
        );
        fetchData();
      } catch (e) {
        console.error('Error parsing task_executed payload', e);
      }
    }

    if (event.event_type === 'task_status_changed') {
      addToast('Task status updated');
      fetchData();
    }

    if (event.event_type === 'approval_required') {
      try {
        const payload = typeof event.payload === 'string' ? JSON.parse(event.payload) : event.payload;
        setPendingApprovals(prev => {
          // Avoid duplicates
          if (prev.find(a => a.task_id === payload.task_id)) return prev;
          return [...prev, payload];
        });
        addToast(`Manual Approval Required: ${payload.task_name}`, 'error');
      } catch (e) {
        console.error('Error parsing approval_required payload', e);
      }
    }
  }, [addToast, fetchData]));

  const handleApprove = async (taskId) => {
    try {
      await axios.post(`/api/tasks/${taskId}/approve`);
      setPendingApprovals(prev => prev.filter(a => a.task_id !== taskId));
      addToast('Task approved and resumed');
      fetchData();
    } catch {
      addToast('Failed to approve task', 'error');
    }
  };

  const handleDeny = async (taskId) => {
    try {
      await axios.post(`/api/tasks/${taskId}/deny`);
      setPendingApprovals(prev => prev.filter(a => a.task_id !== taskId));
      addToast('Task execution denied');
      fetchData();
    } catch {
      addToast('Failed to deny task', 'error');
    }
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(user?.api_key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleRotate = async () => {
    if (!confirm('Are you sure you want to rotate your API key? The current key will stop working immediately.')) return;
    
    setRotating(true);
    try {
      await axios.post('/api/rotate-api-key');
      await checkAuth();
      alert('API Key rotated successfully');
    } catch {
      alert('Failed to rotate API Key');
    } finally {
      setRotating(false);
    }
  };

  const handleUpgrade = async () => {
    try {
      const res = await axios.post('/api/billing/create-checkout-session');
      if (res.data.success && res.data.data.url) {
        window.location.assign(res.data.data.url);
      }
    } catch {
      alert('Failed to initiate upgrade');
    }
  };

  return (
    <DashboardLayout>
      <header className="mb-12">
        <motion.h1 
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl font-black text-white tracking-tight mb-2"
        >
          Control Center
        </motion.h1>
        <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em]">Operational Status: <span className="text-emerald-500">Nominal</span></p>
      </header>

      {/* Pending Approvals Section */}
      <AnimatePresence>
        {pendingApprovals.length > 0 && (
          <motion.div 
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="mb-12 overflow-hidden"
          >
            <h2 className="text-xl font-black text-white uppercase tracking-widest mb-6 flex items-center gap-3">
              <ShieldAlert className="text-red-500 animate-pulse" size={20} />
              Manual Intervention Required
            </h2>
            <div className="flex flex-col gap-4">
              {pendingApprovals.map((approval) => (
                <motion.div 
                  key={approval.task_id}
                  layout
                  initial={{ x: -20, opacity: 0 }}
                  animate={{ x: 0, opacity: 1 }}
                  className="bg-red-500/5 border border-red-500/20 p-8 rounded-[2rem] flex flex-col md:flex-row items-center justify-between gap-6 backdrop-blur-xl"
                >
                  <div>
                    <h4 className="text-white font-bold text-lg mb-1">{approval.task_name}</h4>
                    <p className="text-slate-500 text-xs font-mono">ID: {approval.task_id}</p>
                  </div>
                  <div className="flex items-center gap-4">
                    <button 
                      onClick={() => handleDeny(approval.task_id)}
                      className="px-8 py-4 rounded-xl text-xs font-black uppercase tracking-widest text-slate-400 hover:text-white transition-colors"
                    >
                      Deny
                    </button>
                    <button 
                      onClick={() => handleApprove(approval.task_id)}
                      className="bg-red-500 text-white px-10 py-4 rounded-xl text-xs font-black uppercase tracking-widest shadow-[0_10px_30px_rgba(239,68,68,0.3)] hover:scale-105 transition-transform"
                    >
                      Approve Execution
                    </button>
                  </div>
                </motion.div>
              ))}
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {new URLSearchParams(window.location.search).get('payment') === 'success' && (
        <div className="bg-emerald-500/10 text-emerald-400 p-6 rounded-3xl border border-emerald-500/20 mb-12 font-bold flex items-center gap-4 shadow-[0_0_50px_rgba(16,185,129,0.1)]">
          <ShieldCheck size={24} />
          Payment successful! Your neural capacity has been upgraded to PRO.
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 mb-12">
        {/* Tier Card */}
        <div className="bg-white/5 p-10 rounded-[2.5rem] border border-white/10 shadow-2xl flex flex-col group hover:bg-white/[0.08] transition-all duration-500 backdrop-blur-xl">
          <div className="flex items-center gap-4 mb-8">
            <div className="bg-accent-orange/10 p-3 rounded-2xl text-accent-orange group-hover:scale-110 transition-transform">
              <Crown size={24} />
            </div>
            <h3 className="font-bold text-slate-300 uppercase tracking-widest text-xs">Node Tier</h3>
          </div>
          <div className="text-4xl font-black text-white uppercase tracking-tighter mb-4 glow-text">
            {user?.tier}
          </div>
          <p className="text-slate-500 text-sm font-medium flex-1">
            {user?.tier === 'free' ? 'Legacy throughput: 2 concurrent streams.' : 'High-capacity throughput: 50 concurrent streams.'}
          </p>
          {user?.tier === 'free' && (
            <button 
              onClick={handleUpgrade}
              className="mt-8 text-xs font-black text-accent-orange uppercase tracking-[0.2em] hover:text-white transition-colors text-left flex items-center gap-2"
            >
              Upgrade Neural Capacity <ArrowRight size={14} />
            </button>
          )}
        </div>

        {/* Task Stats Card */}
        <div className="bg-white/5 p-10 rounded-[2.5rem] border border-white/10 shadow-2xl flex flex-col group hover:bg-white/[0.08] transition-all duration-500 backdrop-blur-xl">
          <div className="flex items-center gap-4 mb-8">
            <div className="bg-blue-500/10 p-3 rounded-2xl text-blue-400 group-hover:scale-110 transition-transform">
              <ListChecks size={24} />
            </div>
            <h3 className="font-bold text-slate-300 uppercase tracking-widest text-xs">Active Streams</h3>
          </div>
          <div className="text-5xl font-black text-white mb-4 tracking-tighter">
            {taskCount}
          </div>
          <p className="text-slate-500 text-sm font-medium flex-1">
            Durable schedules currently active across your global node network.
          </p>
        </div>

        {/* Identity Card */}
        <div className="bg-white/5 p-10 rounded-[2.5rem] border border-white/10 shadow-2xl flex flex-col group hover:bg-white/[0.08] transition-all duration-500 backdrop-blur-xl">
          <div className="flex items-center gap-4 mb-8">
            <div className="bg-emerald-500/10 p-3 rounded-2xl text-emerald-400 group-hover:scale-110 transition-transform">
              <Zap size={24} />
            </div>
            <h3 className="font-bold text-slate-300 uppercase tracking-widest text-xs">System Uptime</h3>
          </div>
          <div className="text-4xl font-black text-white mb-4 tracking-tighter">
            99.99<span className="text-slate-600">%</span>
          </div>
          <p className="text-slate-500 text-sm font-medium flex-1">
            Guaranteed low-latency delivery through our distributed reaper network.
          </p>
        </div>

        {/* API Key Card */}
        <div className="bg-white/5 p-10 rounded-[3rem] border border-white/10 shadow-2xl md:col-span-2 lg:col-span-3 flex flex-col backdrop-blur-2xl relative overflow-hidden">
          <div className="absolute top-0 right-0 w-64 h-64 bg-accent-orange/5 rounded-full blur-[100px] -translate-y-1/2 translate-x-1/2"></div>
          
          <div className="flex items-center gap-4 mb-10 relative z-10">
            <div className="bg-white/5 p-3 rounded-2xl text-slate-400">
              <Key size={24} />
            </div>
            <h3 className="font-bold text-slate-300 uppercase tracking-widest text-xs">Neural Access Key</h3>
          </div>

          <div className="flex flex-col md:flex-row gap-6 items-stretch md:items-center relative z-10">
            <div className="flex-1 bg-black/60 text-emerald-400 p-6 rounded-[1.5rem] font-mono text-sm break-all flex items-center justify-between border border-white/5 shadow-inner">
              <code className="tracking-widest">{user?.api_key}</code>
              <button onClick={handleCopy} className="ml-6 p-2 hover:bg-white/5 rounded-xl transition-all">
                {copied ? <Check className="w-5 h-5 text-emerald-500" /> : <Copy className="w-5 h-5" />}
              </button>
            </div>
            <button 
              onClick={handleRotate}
              disabled={rotating}
              className="bg-red-500/10 text-red-400 px-10 py-6 rounded-2xl font-black text-xs uppercase tracking-[0.2em] border border-red-500/20 hover:bg-red-500/20 transition-all flex items-center gap-3 justify-center shadow-xl active:scale-95"
            >
              <RefreshCw className={`w-4 h-4 ${rotating ? 'animate-spin' : ''}`} />
              Rotate Key
            </button>
          </div>
          
          <div className="mt-8 flex items-center gap-2 text-[10px] font-bold text-slate-600 uppercase tracking-widest relative z-10">
             <ShieldCheck size={12} className="text-red-900" /> Key rotation will instantly terminate all existing client connections.
          </div>
        </div>
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

export default Dashboard;
