import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  X, ChevronRight, ChevronLeft, Check, Cpu, Globe, 
  Terminal, Calendar, Zap, Shield, Loader2, Info
} from 'lucide-react';
import axios from 'axios';

const TaskWizard = ({ isOpen, onClose, onTaskCreated, initialData }) => {
  const [step, setStep] = useState(1);
  const [workspaces, setWorkspaces] = useState([]);
  const [loadingWorkspaces, setLoadingWorkspaces] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);

  // Form State
  const [formData, setFormData] = useState({
    name: '',
    workspace_id: '',
    task_type: 'mcp_sampling', // 'mcp_sampling' or 'native_action'
    agent_prompt: '',
    native_code: '',
    trigger_type: 'cron', // 'cron', 'interval', 'webhook'
    trigger_config: { cron: '0 * * * *' }, // Default to hourly
    requires_approval: false,
    missed_task_policy: 'skip'
  });

  useEffect(() => {
    if (isOpen) {
      fetchWorkspaces();
      if (initialData) {
        setFormData(prev => ({
          ...prev,
          ...initialData,
          // Ensure nested objects are handled
          trigger_config: initialData.trigger_config || prev.trigger_config
        }));
      } else {
        resetForm();
      }
    }
  }, [isOpen, initialData]);

  const resetForm = () => {
    setStep(1);
    setFormData({
      name: '',
      workspace_id: '',
      task_type: 'mcp_sampling',
      agent_prompt: '',
      native_code: '',
      trigger_type: 'cron',
      trigger_config: { cron: '0 * * * *' },
      requires_approval: false,
      missed_task_policy: 'skip'
    });
    setError(null);
  };

  const fetchWorkspaces = async () => {
    setLoadingWorkspaces(true);
    try {
      const res = await axios.get('/api/v1/workspaces');
      if (res.data.success) {
        setWorkspaces(res.data.data || []);
        if (res.data.data?.length > 0) {
          setFormData(prev => ({ ...prev, workspace_id: res.data.data[0].id }));
        }
      }
    } catch (err) {
      console.error('Failed to fetch workspaces', err);
    } finally {
      setLoadingWorkspaces(false);
    }
  };

  const handleNext = () => setStep(s => Math.min(s + 1, 4));
  const handleBack = () => setStep(s => Math.max(s - 1, 1));

  const updateFormData = (field, value) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    setError(null);
    try {
      // Prepare payload
      const payload = {
        name: formData.name,
        workspace_id: formData.workspace_id,
        task_type: formData.task_type,
        agent_prompt: formData.task_type === 'mcp_sampling' ? formData.agent_prompt : '',
        native_code: formData.task_type === 'native_action' ? formData.native_code : '',
        trigger_type: formData.trigger_type,
        trigger_config: formData.trigger_config,
        requires_approval: formData.requires_approval,
        missed_task_policy: formData.missed_task_policy
      };

      const res = await axios.post('/api/v1/tasks', payload);
      if (res.data.success) {
        onTaskCreated(res.data.data);
        onClose();
      } else {
        setError(res.data.error || 'Failed to create task');
      }
    } catch (err) {
      setError(err.response?.data?.error || 'An error occurred during submission');
    } finally {
      setSubmitting(false);
    }
  };

  const steps = [
    { id: 1, name: 'Identity', icon: Globe },
    { id: 2, name: 'Config', icon: Cpu },
    { id: 3, name: 'Trigger', icon: Calendar },
    { id: 4, name: 'Review', icon: Check }
  ];

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[110] flex items-center justify-center p-6">
      <motion.div 
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
        className="absolute inset-0 bg-black/80 backdrop-blur-md"
      />
      
      <motion.div 
        initial={{ opacity: 0, scale: 0.9, y: 20 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.9, y: 20 }}
        className="bg-zinc-900 border border-white/10 rounded-[2.5rem] shadow-2xl w-full max-w-2xl relative z-10 overflow-hidden flex flex-col max-h-[90vh]"
      >
        {/* Header */}
        <div className="p-8 border-b border-white/5 flex items-center justify-between bg-white/[0.02]">
          <div>
            <h2 className="text-2xl font-black text-white uppercase tracking-tighter">New Task Wizard</h2>
            <p className="text-[10px] text-slate-500 font-black uppercase tracking-[0.2em] mt-1">Configure your automated workflow</p>
          </div>
          <button onClick={onClose} className="text-slate-500 hover:text-white transition-colors p-2 hover:bg-white/5 rounded-xl">
            <X size={24} />
          </button>
        </div>

        {/* Progress Bar */}
        <div className="flex px-8 py-4 bg-black/20 gap-2">
          {steps.map((s) => (
            <div key={s.id} className="flex-1 flex flex-col gap-2">
              <div className={`h-1 rounded-full transition-all duration-500 ${step >= s.id ? 'bg-accent-orange shadow-[0_0_10px_rgba(217,119,6,0.5)]' : 'bg-white/10'}`} />
              <div className="flex items-center gap-2 px-1">
                <s.icon size={10} className={step >= s.id ? 'text-accent-orange' : 'text-slate-600'} />
                <span className={`text-[8px] font-black uppercase tracking-widest ${step >= s.id ? 'text-white' : 'text-slate-600'}`}>
                  {s.name}
                </span>
              </div>
            </div>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-8 custom-scrollbar">
          <AnimatePresence mode="wait">
            {step === 1 && (
              <motion.div 
                key="step1"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="space-y-8"
              >
                <div className="space-y-4">
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Task Name</label>
                  <input 
                    type="text"
                    value={formData.name}
                    onChange={(e) => updateFormData('name', e.target.value)}
                    placeholder="e.g. Daily Analytics Report"
                    className="w-full bg-black/40 border border-white/5 rounded-2xl p-5 text-white font-mono text-sm focus:outline-none focus:border-accent-orange/50 transition-colors"
                    autoFocus
                  />
                </div>

                <div className="space-y-4">
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Workspace Context</label>
                  {loadingWorkspaces ? (
                    <div className="flex items-center gap-3 text-slate-500 py-4">
                      <Loader2 size={16} className="animate-spin" />
                      <span className="text-xs font-bold uppercase tracking-widest">Loading Workspaces...</span>
                    </div>
                  ) : (
                    <div className="grid grid-cols-1 gap-3">
                      {workspaces.map(w => (
                        <div 
                          key={w.id}
                          onClick={() => updateFormData('workspace_id', w.id)}
                          className={`p-5 rounded-2xl border transition-all cursor-pointer flex items-center justify-between group ${formData.workspace_id === w.id ? 'bg-accent-orange/10 border-accent-orange/50' : 'bg-black/20 border-white/5 hover:border-white/20'}`}
                        >
                          <div className="flex items-center gap-4">
                            <div className={`p-3 rounded-xl ${formData.workspace_id === w.id ? 'bg-accent-orange text-white' : 'bg-white/5 text-slate-500'}`}>
                              <Globe size={18} />
                            </div>
                            <div>
                              <div className="text-sm font-bold text-white uppercase tracking-tight">{w.name}</div>
                              <div className="text-[10px] text-slate-500 font-mono">ID: {w.id.substring(0, 8)}...</div>
                            </div>
                          </div>
                          {formData.workspace_id === w.id && <Check size={20} className="text-accent-orange" />}
                        </div>
                      ))}
                      {workspaces.length === 0 && (
                        <div className="p-8 border border-dashed border-white/10 rounded-2xl text-center">
                          <p className="text-xs text-slate-500 font-medium">No workspaces found. You might need to create one first.</p>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </motion.div>
            )}

            {step === 2 && (
              <motion.div 
                key="step2"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="space-y-8"
              >
                <div className="space-y-4">
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Task Execution Mode</label>
                  <div className="grid grid-cols-2 gap-4">
                    <button 
                      onClick={() => updateFormData('task_type', 'mcp_sampling')}
                      className={`p-6 rounded-2xl border transition-all flex flex-col items-center gap-3 text-center ${formData.task_type === 'mcp_sampling' ? 'bg-accent-orange/10 border-accent-orange/50' : 'bg-black/20 border-white/5 hover:border-white/20'}`}
                    >
                      <Cpu size={32} className={formData.task_type === 'mcp_sampling' ? 'text-accent-orange' : 'text-slate-500'} />
                      <div>
                        <div className="text-xs font-black text-white uppercase tracking-widest mb-1">LLM Sampling</div>
                        <div className="text-[9px] text-slate-500 uppercase tracking-tighter leading-relaxed">Let the AI decide the actions based on your prompt</div>
                      </div>
                    </button>
                    <button 
                      onClick={() => updateFormData('task_type', 'native_action')}
                      className={`p-6 rounded-2xl border transition-all flex flex-col items-center gap-3 text-center ${formData.task_type === 'native_action' ? 'bg-blue-500/10 border-blue-500/50' : 'bg-black/20 border-white/5 hover:border-white/20'}`}
                    >
                      <Terminal size={32} className={formData.task_type === 'native_action' ? 'text-blue-400' : 'text-slate-500'} />
                      <div>
                        <div className="text-xs font-black text-white uppercase tracking-widest mb-1">Native Action</div>
                        <div className="text-[9px] text-slate-500 uppercase tracking-tighter leading-relaxed">Execute deterministic JavaScript/TypeScript code</div>
                      </div>
                    </button>
                  </div>
                </div>

                <div className="space-y-4">
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">
                    {formData.task_type === 'mcp_sampling' ? 'Agent Prompt' : 'Native Code'}
                  </label>
                  <textarea 
                    value={formData.task_type === 'mcp_sampling' ? formData.agent_prompt : formData.native_code}
                    onChange={(e) => updateFormData(formData.task_type === 'mcp_sampling' ? 'agent_prompt' : 'native_code', e.target.value)}
                    placeholder={formData.task_type === 'mcp_sampling' ? "Describe what the AI should do..." : "// Write your action code here..."}
                    className="w-full bg-black/40 border border-white/5 rounded-2xl p-5 text-white font-mono text-sm focus:outline-none focus:border-accent-orange/50 transition-colors h-48 resize-none"
                  />
                  <div className="flex items-center gap-2 text-slate-500">
                    <Info size={12} />
                    <span className="text-[9px] font-medium uppercase tracking-wider italic">
                      {formData.task_type === 'mcp_sampling' ? 'The LLM will use this as instructions for every run.' : 'This code runs in a sandboxed V8 environment.'}
                    </span>
                  </div>
                </div>
              </motion.div>
            )}

            {step === 3 && (
              <motion.div 
                key="step3"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="space-y-8"
              >
                <div className="space-y-4">
                  <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Trigger Strategy</label>
                  <div className="grid grid-cols-3 gap-3">
                    {['cron', 'interval', 'webhook'].map(type => (
                      <button 
                        key={type}
                        onClick={() => {
                          const defaultConfig = type === 'cron' ? { cron: '0 * * * *' } : type === 'interval' ? { minutes: 10 } : { manual: true };
                          setFormData(prev => ({ ...prev, trigger_type: type, trigger_config: defaultConfig }));
                        }}
                        className={`p-4 rounded-xl border transition-all text-[10px] font-black uppercase tracking-widest ${formData.trigger_type === type ? 'bg-white/10 border-white/40 text-white' : 'bg-black/20 border-white/5 text-slate-500 hover:border-white/20'}`}
                      >
                        {type}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="space-y-6">
                  {formData.trigger_type === 'cron' && (
                    <div className="space-y-4">
                      <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Cron Expression</label>
                      <input 
                        type="text"
                        value={formData.trigger_config.cron}
                        onChange={(e) => updateFormData('trigger_config', { cron: e.target.value })}
                        placeholder="* * * * *"
                        className="w-full bg-black/40 border border-white/5 rounded-2xl p-5 text-white font-mono text-sm focus:outline-none focus:border-accent-orange/50 transition-colors"
                      />
                      <div className="p-4 bg-blue-500/5 border border-blue-500/20 rounded-xl">
                        <p className="text-[10px] text-blue-400 font-medium">Standard 5-field cron expression supported (Min, Hour, Day, Month, Weekday).</p>
                      </div>
                    </div>
                  )}

                  {formData.trigger_type === 'interval' && (
                    <div className="space-y-4">
                      <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Interval (Minutes)</label>
                      <input 
                        type="number"
                        value={formData.trigger_config.minutes}
                        onChange={(e) => updateFormData('trigger_config', { minutes: parseInt(e.target.value) })}
                        placeholder="10"
                        className="w-full bg-black/40 border border-white/5 rounded-2xl p-5 text-white font-mono text-sm focus:outline-none focus:border-accent-orange/50 transition-colors"
                      />
                    </div>
                  )}

                  {formData.trigger_type === 'webhook' && (
                    <div className="p-8 border border-dashed border-white/10 rounded-2xl text-center space-y-4">
                      <Zap size={32} className="mx-auto text-amber-500" />
                      <div>
                        <p className="text-xs text-white font-bold uppercase tracking-wider">Inbound Webhook</p>
                        <p className="text-[10px] text-slate-500 mt-1 uppercase tracking-tight">Task will trigger via a unique URL generated after creation.</p>
                      </div>
                    </div>
                  )}

                  <div className="pt-6 border-t border-white/5 space-y-6">
                    <div className="flex items-center justify-between p-6 bg-black/20 rounded-[2rem] border border-white/5">
                      <div className="flex items-center gap-4">
                        <div className={`p-3 rounded-xl ${formData.requires_approval ? 'bg-amber-500 text-white' : 'bg-white/5 text-slate-500'}`}>
                          <Shield size={20} />
                        </div>
                        <div>
                          <div className="text-xs font-black text-white uppercase tracking-widest">Manual Approval</div>
                          <div className="text-[9px] text-slate-500 uppercase tracking-tighter">Require confirmation before each run</div>
                        </div>
                      </div>
                      <button 
                        onClick={() => updateFormData('requires_approval', !formData.requires_approval)}
                        className={`w-12 h-6 rounded-full transition-colors relative ${formData.requires_approval ? 'bg-amber-500' : 'bg-white/10'}`}
                      >
                        <motion.div 
                          animate={{ x: formData.requires_approval ? 26 : 4 }}
                          className="absolute top-1 w-4 h-4 bg-white rounded-full shadow-lg"
                        />
                      </button>
                    </div>

                    <div className="space-y-4">
                       <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Missed Task Policy</label>
                       <div className="flex gap-4">
                          {['skip', 'run_immediately'].map(policy => (
                            <button 
                              key={policy}
                              onClick={() => updateFormData('missed_task_policy', policy)}
                              className={`flex-1 py-4 rounded-xl border transition-all text-[9px] font-black uppercase tracking-widest ${formData.missed_task_policy === policy ? 'bg-white/10 border-white/40 text-white' : 'bg-black/20 border-white/5 text-slate-500'}`}
                            >
                              {policy.replace('_', ' ')}
                            </button>
                          ))}
                       </div>
                    </div>
                  </div>
                </div>
              </motion.div>
            )}

            {step === 4 && (
              <motion.div 
                key="step4"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="space-y-8"
              >
                <div className="bg-black/40 border border-white/10 rounded-3xl p-8 space-y-6">
                   <div className="grid grid-cols-2 gap-8">
                      <div>
                        <div className="text-[9px] font-black text-slate-500 uppercase tracking-widest mb-2">Task Name</div>
                        <div className="text-white font-bold">{formData.name || 'Untitled Task'}</div>
                      </div>
                      <div>
                        <div className="text-[9px] font-black text-slate-500 uppercase tracking-widest mb-2">Mode</div>
                        <div className="text-white font-bold flex items-center gap-2">
                           {formData.task_type === 'mcp_sampling' ? <Cpu size={14} className="text-accent-orange" /> : <Terminal size={14} className="text-blue-400" />}
                           {formData.task_type === 'mcp_sampling' ? 'LLM' : 'Native'}
                        </div>
                      </div>
                      <div>
                        <div className="text-[9px] font-black text-slate-500 uppercase tracking-widest mb-2">Trigger</div>
                        <div className="text-white font-bold uppercase tracking-widest text-[10px]">{formData.trigger_type}</div>
                      </div>
                      <div>
                        <div className="text-[9px] font-black text-slate-500 uppercase tracking-widest mb-2">Approval</div>
                        <div className={`text-[10px] font-black uppercase tracking-widest ${formData.requires_approval ? 'text-amber-500' : 'text-slate-500'}`}>
                          {formData.requires_approval ? 'Required' : 'Optional'}
                        </div>
                      </div>
                   </div>

                   <div className="pt-6 border-t border-white/5">
                      <div className="text-[9px] font-black text-slate-500 uppercase tracking-widest mb-4">Payload Preview</div>
                      <div className="bg-black/60 rounded-2xl p-6 font-mono text-[10px] text-slate-400 overflow-hidden text-ellipsis max-h-32">
                         {formData.task_type === 'mcp_sampling' ? formData.agent_prompt : formData.native_code}
                      </div>
                   </div>
                </div>

                {error && (
                  <div className="p-4 bg-red-500/10 border border-red-500/50 rounded-2xl flex items-center gap-3 text-red-400">
                    <Shield size={16} />
                    <span className="text-xs font-bold uppercase tracking-tight">{error}</span>
                  </div>
                )}
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* Footer */}
        <div className="p-8 border-t border-white/5 flex items-center justify-between bg-white/[0.01]">
          <button 
            onClick={handleBack}
            disabled={step === 1 || submitting}
            className="flex items-center gap-2 text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] hover:text-white transition-colors disabled:opacity-0"
          >
            <ChevronLeft size={14} /> Back
          </button>
          
          <div className="flex gap-4">
             {step < 4 ? (
               <button 
                 onClick={handleNext}
                 disabled={!formData.name || (step === 1 && !formData.workspace_id)}
                 className="bg-white text-black px-10 py-4 rounded-2xl text-[10px] font-black uppercase tracking-[0.2em] shadow-xl hover:scale-105 active:scale-95 transition-all disabled:opacity-30 flex items-center gap-2"
               >
                 Continue <ChevronRight size={14} />
               </button>
             ) : (
               <button 
                 onClick={handleSubmit}
                 disabled={submitting}
                 className="bg-accent-orange text-white px-10 py-4 rounded-2xl text-[10px] font-black uppercase tracking-[0.2em] shadow-[0_10px_30px_rgba(217,119,6,0.3)] hover:scale-105 active:scale-95 transition-all flex items-center gap-2 disabled:opacity-50"
               >
                 {submitting ? <Loader2 size={14} className="animate-spin" /> : <Check size={14} />}
                 Launch Task
               </button>
             )}
          </div>
        </div>
      </motion.div>
    </div>
  );
};

export default TaskWizard;
