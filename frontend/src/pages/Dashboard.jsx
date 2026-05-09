import React, { useEffect, useState } from 'react';
import { useAuth } from '../context/AuthContext';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { Crown, ListChecks, Key, RefreshCw, Copy, Check } from 'lucide-react';

const Dashboard = () => {
  const { user, checkAuth } = useAuth();
  const [taskCount, setTaskCount] = useState(0);
  const [copied, setCopied] = useState(false);
  const [rotating, setRotating] = useState(false);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await axios.get('/api/dashboard');
        if (res.data.success) {
          setTaskCount(res.data.data.taskCount);
        }
      } catch (err) {
        console.error('Failed to fetch dashboard data', err);
      }
    };
    fetchData();
  }, []);

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
    } catch (err) {
      alert('Failed to rotate API Key');
    } finally {
      setRotating(false);
    }
  };

  return (
    <DashboardLayout>
      <header className="mb-8">
        <h1 className="text-3xl font-bold text-[#141413]">Dashboard</h1>
        <p className="text-slate-500 mt-1">Manage your account and scheduled actions.</p>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {/* Tier Card */}
        <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm flex flex-col">
          <div className="flex items-center gap-3 mb-4">
            <div className="bg-amber-100 p-2 rounded-lg text-amber-600">
              <Crown className="w-5 h-5" />
            </div>
            <h3 className="font-semibold text-slate-900">Current Tier</h3>
          </div>
          <div className="text-3xl font-bold text-[#d97706] uppercase tracking-tight mb-2">
            {user?.tier}
          </div>
          <p className="text-sm text-slate-500 flex-1">
            {user?.tier === 'free' ? 'Limited to 5 active tasks.' : 'Up to 50 active tasks.'}
          </p>
          <button className="mt-4 text-sm font-semibold text-[#d97706] hover:underline text-left">
            Upgrade Plan →
          </button>
        </div>

        {/* Task Stats Card */}
        <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm flex flex-col">
          <div className="flex items-center gap-3 mb-4">
            <div className="bg-blue-100 p-2 rounded-lg text-blue-600">
              <ListChecks className="w-5 h-5" />
            </div>
            <h3 className="font-semibold text-slate-900">Total Tasks</h3>
          </div>
          <div className="text-3xl font-bold text-slate-900 mb-2">
            {taskCount}
          </div>
          <p className="text-sm text-slate-500 flex-1">
            Active and scheduled tasks across all tools.
          </p>
          <button className="mt-4 text-sm font-semibold text-blue-600 hover:underline text-left">
            View All Tasks →
          </button>
        </div>

        {/* API Key Card */}
        <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm md:col-span-2 lg:col-span-3 flex flex-col">
          <div className="flex items-center gap-3 mb-4">
            <div className="bg-emerald-100 p-2 rounded-lg text-emerald-600">
              <Key className="w-5 h-5" />
            </div>
            <h3 className="font-semibold text-slate-900">API Key</h3>
          </div>
          <div className="flex flex-col md:flex-row gap-4 items-stretch md:items-center">
            <div className="flex-1 bg-slate-900 text-emerald-400 p-4 rounded-xl font-mono text-sm break-all flex items-center justify-between border border-slate-800">
              <code>{user?.api_key}</code>
              <button onClick={handleCopy} className="ml-4 hover:text-white transition-colors">
                {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
              </button>
            </div>
            <button 
              onClick={handleRotate}
              disabled={rotating}
              className="bg-red-50 text-red-600 px-6 py-4 rounded-xl font-semibold hover:bg-red-100 transition-colors flex items-center gap-2 justify-center"
            >
              <RefreshCw className={`w-4 h-4 ${rotating ? 'animate-spin' : ''}`} />
              Rotate Key
            </button>
          </div>
          <p className="mt-4 text-xs text-slate-400 italic">
            Warning: Rotating your key will immediately invalidate the previous one. Update your MCP configs accordingly.
          </p>
        </div>
      </div>
    </DashboardLayout>
  );
};

export default Dashboard;
