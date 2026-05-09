import React, { useEffect, useState } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { Terminal, ShieldAlert, CheckCircle2, Clock } from 'lucide-react';

const Monitor = () => {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);

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
      <header className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-[#141413]">System Monitor</h1>
          <p className="text-slate-500 mt-1">Live execution logs across the entire node network.</p>
        </div>
        <div className="flex items-center gap-2 text-xs font-medium text-slate-400 uppercase tracking-widest bg-slate-100 px-3 py-1 rounded-full">
          <span className="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></span>
          Live
        </div>
      </header>

      <div className="bg-[#141413] rounded-2xl border border-slate-800 shadow-xl overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-3 bg-slate-900 border-b border-slate-800">
          <Terminal className="w-4 h-4 text-slate-400" />
          <span className="text-xs font-mono text-slate-400">task_execution.log</span>
        </div>
        
        <div className="p-0 overflow-x-auto">
          <table className="w-full text-left border-collapse font-mono text-xs">
            <thead className="bg-slate-900/50 text-slate-500">
              <tr>
                <th className="px-4 py-2 font-normal border-b border-slate-800">Timestamp</th>
                <th className="px-4 py-2 font-normal border-b border-slate-800">User</th>
                <th className="px-4 py-2 font-normal border-b border-slate-800">Task Name</th>
                <th className="px-4 py-2 font-normal border-b border-slate-800">Status</th>
                <th className="px-4 py-2 font-normal border-b border-slate-800">Result/Error</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/50">
              {loading ? (
                <tr>
                  <td colSpan="5" className="px-4 py-8 text-center text-slate-500">Loading system logs...</td>
                </tr>
              ) : logs.length === 0 ? (
                <tr>
                  <td colSpan="5" className="px-4 py-8 text-center text-slate-500">No logs found in the last 100 executions.</td>
                </tr>
              ) : logs.map((log) => (
                <tr key={log.id} className="hover:bg-slate-900 transition-colors">
                  <td className="px-4 py-2 text-slate-500 whitespace-nowrap">
                    {new Date(log.execution_time).toLocaleString()}
                  </td>
                  <td className="px-4 py-2 text-blue-400">{log.user_email}</td>
                  <td className="px-4 py-2 text-slate-300">{log.task_name}</td>
                  <td className="px-4 py-2">
                    {log.status === 'success' ? (
                      <span className="text-emerald-500 flex items-center gap-1">
                        <CheckCircle2 className="w-3 h-3" /> success
                      </span>
                    ) : log.status === 'failure' ? (
                      <span className="text-red-500 flex items-center gap-1">
                        <ShieldAlert className="w-3 h-3" /> failure
                      </span>
                    ) : (
                      <span className="text-amber-500 flex items-center gap-1">
                        <Clock className="w-3 h-3" /> missed
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-2 text-slate-400 max-w-xs truncate">
                    {log.error_message || log.llm_response || '-'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </DashboardLayout>
  );
};

export default Monitor;
