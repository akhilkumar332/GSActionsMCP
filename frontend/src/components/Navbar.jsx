import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { Clock, Menu, X, ArrowRight } from 'lucide-react';
import { useAuth } from '../context/AuthContext';

const Navbar = () => {
  const [isScrolled, setIsScrolled] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const { user } = useAuth();

  useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 10);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const navLinks = [
    { name: 'Features', href: '#features' },
    { name: 'Pricing', href: '#pricing' },
    { name: 'Installation', href: '#installation' },
  ];

  return (
    <nav className={`fixed top-0 left-0 right-0 z-50 transition-all duration-300 ${
      isScrolled ? 'bg-white/80 backdrop-blur-md border-b border-slate-200 py-3' : 'bg-transparent py-5'
    }`}>
      <div className="container mx-auto px-6">
        <div className="flex items-center justify-between">
          {/* Logo */}
          <Link to="/" className="flex items-center gap-2 group">
            <div className="bg-accent-orange p-1.5 rounded-lg text-white group-hover:scale-110 transition-transform">
              <Clock size={20} />
            </div>
            <span className="font-bold text-xl text-ink-900 tracking-tight">Schedule MCP</span>
          </Link>

          {/* Desktop Nav */}
          <div className="hidden md:flex items-center gap-8">
            <div className="flex items-center gap-6">
              {navLinks.map((link) => (
                <a
                  key={link.name}
                  href={link.href}
                  className="text-sm font-medium text-slate-600 hover:text-accent-orange transition-colors"
                >
                  {link.name}
                </a>
              ))}
            </div>
            <div className="h-4 w-px bg-slate-200"></div>
            <div className="flex items-center gap-4">
              {user ? (
                <Link
                  to="/dashboard"
                  className="text-sm font-bold text-ink-900 bg-white border border-slate-200 px-5 py-2 rounded-full hover:bg-slate-50 transition-all flex items-center gap-2 shadow-sm"
                >
                  Dashboard <ArrowRight size={16} />
                </Link>
              ) : (
                <>
                  <Link
                    to="/login"
                    className="text-sm font-medium text-slate-600 hover:text-ink-900 transition-colors"
                  >
                    Login
                  </Link>
                  <Link
                    to="/signup"
                    className="text-sm font-bold text-white bg-ink-900 px-6 py-2.5 rounded-full hover:bg-gray-800 transition-all shadow-md active:scale-95"
                  >
                    Sign Up
                  </Link>
                </>
              )}
            </div>
          </div>

          {/* Mobile Toggle */}
          <button
            className="md:hidden text-ink-900"
            onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
          >
            {isMobileMenuOpen ? <X /> : <Menu />}
          </button>
        </div>
      </div>

      {/* Mobile Menu */}
      {isMobileMenuOpen && (
        <div className="md:hidden absolute top-full left-0 right-0 bg-white border-b border-slate-200 shadow-xl animate-in slide-in-from-top duration-300">
          <div className="flex flex-col p-6 gap-4 text-center">
            {navLinks.map((link) => (
              <a
                key={link.name}
                href={link.href}
                className="text-lg font-medium text-slate-600"
                onClick={() => setIsMobileMenuOpen(false)}
              >
                {link.name}
              </a>
            ))}
            <div className="h-px w-full bg-slate-100 my-2"></div>
            {user ? (
              <Link
                to="/dashboard"
                className="bg-accent-orange text-white py-3 rounded-xl font-bold"
                onClick={() => setIsMobileMenuOpen(false)}
              >
                Dashboard
              </Link>
            ) : (
              <>
                <Link
                  to="/login"
                  className="text-slate-600 font-medium py-2"
                  onClick={() => setIsMobileMenuOpen(false)}
                >
                  Login
                </Link>
                <Link
                  to="/signup"
                  className="bg-ink-900 text-white py-3 rounded-xl font-bold"
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
