import { useEffect, useState } from 'react';
import DashboardLayout from '../components/DashboardLayout';
import axios from 'axios';
import { UserCog, Shield, Star, UserCircle, Search } from 'lucide-react';
import { motion } from 'framer-motion';

const AdminUsers = () => {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  const fetchUsers = async (query = '') => {
    try {
      const res = await axios.get(`/api/v1/admin/users?search=${encodeURIComponent(query)}`);
      if (res.data.success) {
        setUsers(res.data.data || []);
      }
    } catch {
      console.error('Failed to fetch users');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const timer = setTimeout(() => {
      fetchUsers(search);
    }, 500);
    return () => clearTimeout(timer);
  }, [search]);

  const handleUpdate = async (userId, role, tier) => {
    try {
      await axios.post('/api/v1/admin/users/update', { user_id: userId, role, tier });
      await fetchUsers();
    } catch {
      alert('Failed to update user');
    }
  };

  return (
    <DashboardLayout>
      <header className="mb-12 flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div>
          <h1 className="text-4xl font-black text-white tracking-tight">Identity Nexus</h1>
          <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em] mt-2">Centralized User & Permission Registry</p>
        </div>
        
        <div className="flex gap-4">
           <div className="relative group">
              <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500 group-focus-within:text-accent-orange transition-colors" size={18} />
              <input 
                type="text" 
                placeholder="Search Identity..." 
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="bg-white/5 border border-white/10 rounded-2xl pl-12 pr-6 py-3 text-sm text-white outline-none focus:border-accent-orange/50 transition-all w-64 backdrop-blur-xl"
              />
           </div>
        </div>
      </header>

      <motion.div 
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="bg-black/40 rounded-[2.5rem] border border-white/5 shadow-2xl overflow-hidden backdrop-blur-3xl"
      >
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead className="bg-white/5 text-slate-500 text-[10px] font-black uppercase tracking-[0.2em]">
              <tr>
                <th className="px-10 py-6 border-b border-white/5">Neural Identity</th>
                <th className="px-6 py-6 border-b border-white/5">Access Key</th>
                <th className="px-6 py-6 border-b border-white/5 text-center">Privilege</th>
                <th className="px-6 py-6 border-b border-white/5 text-center">Tier</th>
                <th className="px-10 py-6 border-b border-white/5 text-right">Overrides</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {loading ? (
                <tr>
                  <td colSpan="5" className="px-10 py-32 text-center">
                     <div className="flex flex-col items-center gap-4">
                        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-orange"></div>
                        <span className="text-slate-500 font-bold uppercase tracking-widest">Querying User Data...</span>
                     </div>
                  </td>
                </tr>
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan="5" className="px-10 py-32 text-center text-slate-500 font-bold uppercase tracking-widest italic opacity-50">Identity registry is currently empty.</td>
                </tr>
              ) : users.map((u) => (
                <tr key={u.id} className="hover:bg-white/[0.02] transition-colors group">
                  <td className="px-10 py-6">
                    <div className="flex items-center gap-4">
                      <div className="bg-gradient-to-br from-slate-800 to-slate-900 p-3 rounded-2xl text-slate-500 border border-white/5 group-hover:border-accent-orange/30 transition-colors">
                        <UserCircle className="w-6 h-6" />
                      </div>
                      <div>
                        <div className="font-bold text-slate-200">{u.email}</div>
                        <div className="text-[10px] text-slate-500 font-mono tracking-tighter opacity-60 uppercase">{u.id}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-6">
                    <code className="text-[10px] bg-black/40 px-3 py-1.5 rounded-lg text-emerald-500 font-mono border border-white/5 shadow-inner">
                      {u.api_key}
                    </code>
                  </td>
                  <td className="px-6 py-6 text-center">
                    {u.role === 'admin' ? (
                      <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[10px] font-black uppercase tracking-widest bg-purple-500/10 text-purple-400 border border-purple-500/20 shadow-[0_0_15px_rgba(168,85,247,0.2)]">
                        <Shield className="w-3 h-3" /> admin
                      </span>
                    ) : u.role === 'staff' ? (
                      <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[10px] font-black uppercase tracking-widest bg-blue-500/10 text-blue-400 border border-blue-500/20">
                        <UserCog className="w-3 h-3" /> staff
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[10px] font-black uppercase tracking-widest bg-white/5 text-slate-500 border border-white/5">
                        user
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-6 text-center text-sm font-medium">
                    {u.tier === 'pro' ? (
                      <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[10px] font-black uppercase tracking-widest bg-amber-500/10 text-accent-orange border border-accent-orange/20 shadow-[0_0_15px_rgba(217,119,6,0.2)]">
                        <Star className="w-3 h-3 fill-accent-orange" /> pro
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[10px] font-black uppercase tracking-widest bg-white/5 text-slate-500 border border-white/5">
                        free
                      </span>
                    )}
                  </td>
                  <td className="px-10 py-6 text-right">
                    <div className="flex justify-end gap-3">
                      <button 
                        onClick={() => handleUpdate(u.id, u.role === 'user' ? 'staff' : 'user', u.tier)}
                        className="p-2.5 bg-white/5 border border-white/5 rounded-xl text-slate-500 hover:text-blue-400 hover:border-blue-400/30 hover:bg-blue-400/5 transition-all"
                        title={u.role === 'user' ? 'Promote to Staff' : 'Revoke Staff'}
                      >
                        <UserCog className="w-4 h-4" />
                      </button>
                      <button 
                        onClick={() => handleUpdate(u.id, u.role, u.tier === 'free' ? 'pro' : 'free')}
                        className="p-2.5 bg-white/5 border border-white/5 rounded-xl text-slate-500 hover:text-accent-orange hover:border-accent-orange/30 hover:bg-accent-orange/5 transition-all shadow-xl"
                        title={u.tier === 'free' ? 'Upgrade to Pro' : 'Downgrade to Free'}
                      >
                        <Crown className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </motion.div>
    </DashboardLayout>
  );
};

const Crown = ({ className }) => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" className={className}>
    <path d="m2 4 3 12h14l3-12-6 7-4-7-4 7-6-7zm3 16h14" />
  </svg>
);

export default AdminUsers;
