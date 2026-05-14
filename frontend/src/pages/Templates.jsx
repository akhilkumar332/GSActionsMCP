import { useEffect, useState } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import { Layout, Search, Download, Loader2, Sparkles, Plus } from 'lucide-react';
import { motion } from 'framer-motion';
import axios from 'axios';
import TaskWizard from '../components/TaskWizard';

const Templates = () => {
    const [templates, setTemplates] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState('');
    const [isWizardOpen, setIsWizardOpen] = useState(false);
    const [selectedTemplate, setSelectedTemplate] = useState(null);
    
    const fetchTemplates = async (query = '') => {
        setLoading(true);
        try {
            const res = await axios.get(`/api/v1/templates?search=${encodeURIComponent(query)}`);
            if (res.data.success) {
                setTemplates(res.data.data || []);
            }
        } catch (err) {
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        const timer = setTimeout(() => {
            fetchTemplates(search);
        }, 500);
        return () => clearTimeout(timer);
    }, [search]);

    const handleUseBlueprint = async (template) => {
        // config might be a string if not automatically parsed, but axios usually parses it
        const config = typeof template.config === 'string' ? JSON.parse(template.config) : template.config;

        if (Array.isArray(config)) {
            // It's a multi-task blueprint bundle
            if (!confirm(`This blueprint contains ${config.length} tasks. Deploy this bundle to your workspace?`)) return;
            
            setLoading(true);
            try {
                const res = await axios.post('/api/v1/blueprints/deploy', {
                    template_id: template.id,
                    variables: {} // Future: show a modal to collect variables
                });
                if (res.data.success) {
                    // Redirect to canvas to see the new workflow
                    window.location.href = '/canvas';
                }
            } catch (err) {
                console.error('Failed to deploy blueprint bundle', err);
                alert('Failed to deploy blueprint bundle. Please try again.');
            } finally {
                setLoading(false);
            }
        } else {
            // Single task blueprint
            setSelectedTemplate({
                id: template.id,
                name: `${template.name} (Copy)`,
                ...config
            });
            setIsWizardOpen(true);
        }
    };

    return (
        <DashboardLayout>
            <TaskWizard 
                isOpen={isWizardOpen} 
                onClose={() => setIsWizardOpen(false)} 
                onTaskCreated={async () => {
                    if (selectedTemplate && selectedTemplate.id) {
                        try {
                            await axios.post(`/api/v1/templates/${selectedTemplate.id}/increment-uses`);
                            fetchTemplates(search);
                        } catch (err) {
                            console.error('Failed to increment uses', err);
                        }
                    }
                    alert('Task created successfully from blueprint!');
                }}
                initialData={selectedTemplate}
            />
            <header className="mb-12 flex flex-col md:flex-row md:items-end justify-between gap-6">
                <div>
                    <motion.h1 
                        initial={{ opacity: 0, y: -20 }}
                        animate={{ opacity: 1, y: 0 }}
                        className="text-4xl font-black text-white tracking-tight mb-2"
                    >
                        Marketplace
                    </motion.h1>
                    <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em]">Pre-built AI Workflow Blueprints</p>
                </div>
                <div className="flex items-center gap-4">
                    <div className="relative group">
                        <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500 group-focus-within:text-accent-orange transition-colors" size={16} />
                        <input 
                            type="text" 
                            placeholder="Search blueprints..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className="bg-white/5 border border-white/10 rounded-2xl pl-12 pr-6 py-4 text-xs text-white placeholder:text-slate-600 focus:outline-none focus:border-accent-orange/50 transition-all w-full md:w-64"
                        />
                    </div>
                </div>
            </header>

            {loading ? (
                <div className="py-32 flex flex-col items-center gap-4">
                    <Loader2 className="animate-spin text-accent-orange" size={32} />
                    <p className="text-xs font-black uppercase tracking-widest text-slate-500">Curating Marketplace...</p>
                </div>
            ) : templates.length === 0 ? (
                <div className="py-32 flex flex-col items-center gap-6 text-center">
                    <div className="bg-white/5 p-8 rounded-[2.5rem] text-slate-600">
                        <Layout size={48} />
                    </div>
                    <div>
                        <h3 className="text-xl font-bold text-white mb-2">No templates available yet</h3>
                        <p className="text-sm text-slate-500 max-w-sm">The marketplace is currently being updated with new blueprints. Check back soon!</p>
                    </div>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
                    {templates.map((t, idx) => (
                        <motion.div 
                            key={t.id}
                            initial={{ opacity: 0, y: 20 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: idx * 0.1 }}
                            className="bg-white/5 border border-white/10 rounded-[2.5rem] p-8 hover:bg-white/[0.08] transition-all group flex flex-col h-full"
                        >
                            <div className="flex items-start justify-between mb-6">
                                <div className="bg-accent-orange/10 p-4 rounded-2xl text-accent-orange">
                                    <Sparkles size={24} />
                                </div>
                                {t.is_premium && (
                                    <span className="px-3 py-1 bg-yellow-500/10 text-yellow-500 text-[10px] font-black uppercase tracking-widest rounded-full border border-yellow-500/20">
                                        Premium
                                    </span>
                                )}
                            </div>
                            <h3 className="text-2xl font-black text-white uppercase tracking-tighter mb-2 group-hover:text-accent-orange transition-colors">{t.name}</h3>
                            <p className="text-slate-400 text-xs font-medium leading-relaxed mb-8 flex-grow">
                                {t.description || "No description provided for this template."}
                            </p>
                            <div className="flex items-center justify-between pt-6 border-t border-white/5">
                                <div className="flex items-center gap-2 text-[10px] font-black text-slate-500 uppercase tracking-widest">
                                    <Download size={12} /> {t.uses_count || 0} Uses
                                </div>
                                <button 
                                    onClick={() => handleUseBlueprint(t)}
                                    className="bg-white text-black px-6 py-3 rounded-xl text-[10px] font-black uppercase tracking-widest hover:bg-accent-orange hover:text-white transition-all flex items-center gap-2 active:scale-95"
                                >
                                    <Plus size={12} /> Use Blueprint
                                </button>
                            </div>
                        </motion.div>
                    ))}
                </div>
            )}
        </DashboardLayout>
    );
};

export default Templates;
