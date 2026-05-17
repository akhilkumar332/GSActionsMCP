import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { Menu, X, Activity } from 'lucide-react';
import { useAuth } from '../context/AuthContext';

const Navbar = () => {
  const [isScrolled, setIsScrolled] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const { user } = useAuth();

  useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 20);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const navLinks = [
    { name: 'Features', href: '#features' },
    { name: 'Pricing', href: '#pricing' },
    { name: 'Docs', href: '/docs/overview' },
  ];

  return (
    <nav className={`fixed top-0 left-0 right-0 z-50 transition-all duration-500 ${
      isScrolled ? 'bg-ai-black/60 backdrop-blur-xl border-b border-white/5 py-4' : 'bg-transparent py-8'
    }`}>
      <div className="container mx-auto px-6">
        <div className="flex items-center justify-between">
          {/* Brand Logo */}
          <Link to="/" className="flex items-center gap-3 group relative">
            <img src="/logo-icon.svg" className="w-10 h-10 text-accent-orange group-hover:rotate-[360deg] transition-transform duration-700" alt="Actionfy Logo" />
            <div className="flex flex-col">
              <span className="font-bold text-xl text-white tracking-tighter leading-tight group-hover:text-accent-orange transition-colors">Actionfy</span>
              <span className="text-[9px] font-bold text-slate-500 uppercase tracking-[0.3em] leading-tight">Engine v1.0</span>
            </div>
          </Link>

          {/* Premium Desktop Nav */}
          <div className="hidden md:flex items-center gap-10">
            <div className="flex items-center gap-8">
              {navLinks.map((link) => (
                <a
                  key={link.name}
                  href={link.href}
                  className="text-[13px] font-bold text-slate-400 hover:text-white uppercase tracking-widest transition-colors"
                >
                  {link.name}
                </a>
              ))}
            </div>
            
            <div className="flex items-center gap-4">
              {user ? (
                <Link
                  to="/dashboard"
                  className="group relative text-sm font-bold text-white bg-white/5 border border-white/10 px-6 py-2.5 rounded-2xl hover:bg-white/10 transition-all flex items-center gap-3"
                >
                  <Activity size={16} className="text-accent-orange animate-pulse" />
                  Dashboard
                </Link>
              ) : (
                <>
                  <Link
                    to="/login"
                    className="text-sm font-bold text-slate-400 hover:text-white transition-colors"
                  >
                    Login
                  </Link>
                  <Link
                    to="/signup"
                    className="relative px-7 py-3 text-sm font-bold text-white bg-accent-orange rounded-2xl hover:bg-amber-700 transition-all shadow-[0_10px_30px_rgba(217,119,6,0.2)] active:scale-95 overflow-hidden"
                  >
                    Get Started
                  </Link>
                </>
              )}
            </div>
          </div>

          {/* Mobile Menu Button */}
          <button
            className="md:hidden text-white p-2 hover:bg-white/5 rounded-lg transition-colors"
            onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
          >
            {isMobileMenuOpen ? <X size={28} /> : <Menu size={28} />}
          </button>
        </div>
      </div>

      {/* Mobile Sidebar Overlay */}
      {isMobileMenuOpen && (
        <div className="md:hidden fixed inset-0 z-40 bg-ai-black/95 backdrop-blur-2xl animate-in fade-in duration-300">
          <div className="flex flex-col items-center justify-center h-full gap-8">
            {navLinks.map((link) => (
              <a
                key={link.name}
                href={link.href}
                className="text-3xl font-bold text-white tracking-tighter"
                onClick={() => setIsMobileMenuOpen(false)}
              >
                {link.name}
              </a>
            ))}
            <div className="w-12 h-px bg-white/10"></div>
            {user ? (
              <Link
                to="/dashboard"
                className="text-2xl font-bold text-accent-orange"
                onClick={() => setIsMobileMenuOpen(false)}
              >
                Dashboard
              </Link>
            ) : (
              <>
                <Link
                  to="/login"
                  className="text-2xl font-bold text-white"
                  onClick={() => setIsMobileMenuOpen(false)}
                >
                  Login
                </Link>
                <Link
                  to="/signup"
                  className="px-10 py-4 bg-accent-orange text-white rounded-2xl font-bold text-xl shadow-2xl"
                  onClick={() => setIsMobileMenuOpen(false)}
                >
                  Sign Up
                </Link>
              </>
            )}
          </div>
        </div>
      )}
    </nav>
  );
};

export default Navbar;
