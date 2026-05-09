import { useAuth } from '../context/AuthContext';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { LayoutDashboard, Activity, Users, LogOut, Clock, Search } from 'lucide-react';

const DashboardLayout = ({ children }) => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const handleLogout = async () => {
    await logout();
    navigate('/');
  };

  const navItems = [
    { icon: LayoutDashboard, label: 'Dashboard', path: '/dashboard', roles: ['user', 'staff', 'admin'] },
    { icon: Activity, label: 'System Monitor', path: '/monitor', roles: ['staff', 'admin'] },
    { icon: Users, label: 'User Management', path: '/admin/users', roles: ['admin'] },
    { icon: Search, label: 'SEO Manager', path: '/admin/seo', roles: ['admin'] },
  ];

  return (
    <div className="flex min-h-screen bg-ai-black text-white">
      {/* Premium Sidebar */}
      <aside className="w-72 bg-black/40 backdrop-blur-2xl border-r border-white/5 flex flex-col sticky top-0 h-screen">
        <div className="p-8 border-b border-white/5 flex items-center gap-3">
          <div className="bg-accent-orange p-2 rounded-xl text-white shadow-[0_0_20px_rgba(217,119,6,0.3)]">
            <Clock size={20} />
          </div>
          <span className="font-bold text-xl tracking-tighter">Schedule MCP</span>
        </div>

        <nav className="flex-1 p-6 space-y-2">
          {navItems.map((item) => (
            item.roles.includes(user?.role) && (
              <Link
                key={item.path}
                to={item.path}
                className={`flex items-center gap-4 px-4 py-3 rounded-2xl transition-all duration-300 ${
                  location.pathname === item.path
                    ? 'bg-accent-orange/10 text-accent-orange font-bold border border-accent-orange/20 shadow-[0_0_30px_rgba(217,119,6,0.1)]'
                    : 'text-slate-400 hover:text-white hover:bg-white/5 border border-transparent'
                }`}
              >
                <item.icon size={20} className={location.pathname === item.path ? 'animate-pulse' : ''} />
                <span className="text-[13px] uppercase tracking-widest">{item.label}</span>
              </Link>
            )
          ))}
        </nav>

        <div className="p-6 border-t border-white/5 bg-white/[0.02]">
          <div className="px-5 py-4 bg-black/40 rounded-2xl border border-white/5 mb-6">
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] mb-2">Authenticated As</p>
            <p className="text-sm font-bold text-white truncate mb-1">{user?.email}</p>
            <span className="inline-flex px-2 py-0.5 bg-accent-orange/10 text-accent-orange text-[9px] font-black rounded-lg uppercase tracking-widest border border-accent-orange/20">
              {user?.role}
            </span>
          </div>
          <button
            onClick={handleLogout}
            className="flex items-center gap-4 w-full px-5 py-3 text-slate-400 hover:text-red-400 hover:bg-red-500/10 rounded-2xl transition-all font-bold text-[11px] uppercase tracking-[0.2em]"
          >
            <LogOut size={18} />
            Termination
          </button>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 p-10 md:p-16 overflow-y-auto">
        <div className="max-w-6xl mx-auto">
          {children}
        </div>
      </main>
    </div>
  );
};

export default DashboardLayout;
