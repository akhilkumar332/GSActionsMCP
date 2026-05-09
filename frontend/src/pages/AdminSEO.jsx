import { useEffect, useState } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { Search, Save, Globe, Image as ImageIcon, Layout as LayoutIcon, FileText } from 'lucide-react';

const AdminSEO = () => {
  const [settings, setSettings] = useState({
    title: '',
    description: '',
    keywords: '',
    og_image: ''
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState(null);

  useEffect(() => {
    const fetchSEO = async () => {
      try {
        const res = await axios.get('/api/admin/seo');
        if (res.data.success) {
          const d = res.data.data;
          setSettings({
            title: d.title,
            description: d.description,
            keywords: d.keywords,
            og_image: d.og_image?.String || ''
          });
        }
      } catch (err) {
        console.error('Failed to fetch SEO settings', err);
      } finally {
        setLoading(false);
      }
    };
    fetchSEO();
  }, []);

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    setMessage(null);
    try {
      await axios.post('/api/admin/seo', settings);
      setMessage({ type: 'success', text: 'SEO configuration published to edge nodes successfully.' });
    } catch {
      setMessage({ type: 'error', text: 'Failed to update SEO configuration.' });
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <DashboardLayout><div className="text-white">Loading SEO Engine...</div></DashboardLayout>;

  return (
    <DashboardLayout>
      <header className="mb-12">
        <div className="flex items-center gap-3 mb-2">
          <Search className="text-accent-orange" size={24} />
          <h1 className="text-3xl font-bold text-white tracking-tight">Search Engine Optimization</h1>
        </div>
        <p className="text-slate-400">Manage global metadata, AI-crawler visibility, and social sharing presence.</p>
      </header>

      {message && (
        <div className={`p-4 rounded-2xl mb-8 border ${
          message.type === 'success' ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400' : 'bg-red-500/10 border-red-500/20 text-red-400'
        }`}>
          {message.text}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-12">
        {/* Editor Form */}
        <form onSubmit={handleSave} className="lg:col-span-2 space-y-8">
          <div className="bg-white/5 border border-white/10 rounded-[2rem] p-10 space-y-8 backdrop-blur-xl">
            <div>
              <label className="block text-xs font-black uppercase tracking-[0.2em] text-slate-500 mb-4 flex items-center gap-2">
                <LayoutIcon size={14} /> Meta Title
              </label>
              <input 
                type="text"
                value={settings.title}
                onChange={(e) => setSettings({...settings, title: e.target.value})}
                className="w-full bg-black/40 border border-white/10 rounded-xl px-6 py-4 text-white outline-none focus:border-accent-orange transition-colors"
                placeholder="Schedule MCP - Persistent AI Workflows"
              />
            </div>

            <div>
              <label className="block text-xs font-black uppercase tracking-[0.2em] text-slate-500 mb-4 flex items-center gap-2">
                <FileText size={14} /> Meta Description
              </label>
              <textarea 
                rows={4}
                value={settings.description}
                onChange={(e) => setSettings({...settings, description: e.target.value})}
                className="w-full bg-black/40 border border-white/10 rounded-xl px-6 py-4 text-white outline-none focus:border-accent-orange transition-colors resize-none"
                placeholder="Durable scheduling for Model Context Protocol..."
              />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
               <div>
                  <label className="block text-xs font-black uppercase tracking-[0.2em] text-slate-500 mb-4 flex items-center gap-2">
                    <Globe size={14} /> Keywords
                  </label>
                  <input 
                    type="text"
                    value={settings.keywords}
                    onChange={(e) => setSettings({...settings, keywords: e.target.value})}
                    className="w-full bg-black/40 border border-white/10 rounded-xl px-6 py-4 text-white outline-none focus:border-accent-orange transition-colors"
                    placeholder="AI, MCP, Scheduler, Go"
                  />
               </div>
               <div>
                  <label className="block text-xs font-black uppercase tracking-[0.2em] text-slate-500 mb-4 flex items-center gap-2">
                    <ImageIcon size={14} /> OG Image URL
                  </label>
                  <input 
                    type="text"
                    value={settings.og_image}
                    onChange={(e) => setSettings({...settings, og_image: e.target.value})}
                    className="w-full bg-black/40 border border-white/10 rounded-xl px-6 py-4 text-white outline-none focus:border-accent-orange transition-colors"
                    placeholder="https://yoursite.com/og.png"
                  />
               </div>
            </div>
          </div>

          <button 
            type="submit"
            disabled={saving}
            className="flex items-center gap-3 bg-white text-ink-900 px-10 py-5 rounded-2xl font-black text-sm uppercase tracking-widest hover:bg-slate-100 transition-all active:scale-95 shadow-[0_0_40px_rgba(255,255,255,0.1)]"
          >
            {saving ? 'Syncing...' : <><Save size={20} /> Publish SEO Changes</>}
          </button>
        </form>

        {/* Live Preview */}
        <div className="space-y-8">
           <h4 className="text-xs font-black uppercase tracking-[0.2em] text-slate-500">Live Preview</h4>
           
           {/* Google Preview */}
           <div className="bg-white p-8 rounded-3xl shadow-2xl">
              <p className="text-[14px] text-[#202124] mb-1">https://schedulemcp.com</p>
              <h3 className="text-[20px] text-[#1a0dab] hover:underline cursor-pointer truncate mb-2">{settings.title || 'Page Title'}</h3>
              <p className="text-[14px] text-[#4d5156] line-clamp-2">{settings.description || 'Enter a description to see a preview of how this page might appear in Google search results.'}</p>
           </div>

           {/* Social Preview */}
           <div className="bg-[#141413] border border-white/10 rounded-3xl overflow-hidden shadow-2xl">
              <div className="aspect-[1.91/1] bg-white/5 flex items-center justify-center">
                 {settings.og_image ? (
                   <img src={settings.og_image} className="w-full h-full object-cover" alt="OG Preview" />
                 ) : (
                   <ImageIcon size={48} className="text-white/10" />
                 )}
              </div>
              <div className="p-6">
                <p className="text-[11px] font-bold text-slate-500 uppercase tracking-widest mb-2">ScheduleMCP.com</p>
                <h4 className="text-white font-bold mb-2">{settings.title || 'Social Title'}</h4>
                <p className="text-slate-400 text-xs line-clamp-2">{settings.description}</p>
              </div>
           </div>
        </div>
      </div>
    </DashboardLayout>
  );
};

export default AdminSEO;

