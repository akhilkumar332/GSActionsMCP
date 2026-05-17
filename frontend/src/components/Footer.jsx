import { Link } from 'react-router-dom';
import { Globe, Shield, Mail } from 'lucide-react';

const Footer = () => {
  return (
    <footer className="bg-ink-900 text-white pt-24 pb-12">
      <div className="container mx-auto px-6">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-12 mb-16">
          {/* Brand */}
          <div className="col-span-1 md:col-span-1">
            <Link to="/" className="flex items-center gap-3 mb-6 group">
              <img src="/logo-icon.svg" className="w-8 h-8 text-accent-orange" alt="Actionfy Logo" />
              <span className="font-bold text-xl tracking-tight text-white">Actionfy</span>
            </Link>
            <p className="text-slate-400 text-sm leading-relaxed mb-8">
              The industry standard for persistent task scheduling within the Model Context Protocol ecosystem. 
              Built for reliability, speed, and massive scale.
            </p>
            <div className="flex gap-4">
              <a href="#" className="p-2 bg-white/5 rounded-lg hover:bg-white/10 transition-colors">
                <Globe size={18} className="text-slate-300" />
              </a>
              <a href="#" className="p-2 bg-white/5 rounded-lg hover:bg-white/10 transition-colors">
                <Shield size={18} className="text-slate-300" />
              </a>
              <a href="#" className="p-2 bg-white/5 rounded-lg hover:bg-white/10 transition-colors">
                <Mail size={18} className="text-slate-300" />
              </a>
            </div>
          </div>

          {/* Product */}
          <div>
            <h4 className="font-bold text-white mb-6 uppercase text-xs tracking-widest">Product</h4>
            <ul className="space-y-4">
              <li><Link to="/#features" className="text-slate-400 hover:text-white transition-colors text-sm">Features</Link></li>
              <li><Link to="/#pricing" className="text-slate-400 hover:text-white transition-colors text-sm">Pricing</Link></li>
              <li><Link to="/#installation" className="text-slate-400 hover:text-white transition-colors text-sm">Installation</Link></li>
              <li><Link to="/dashboard" className="text-slate-400 hover:text-white transition-colors text-sm">Dashboard</Link></li>
              <li><a href="#" className="text-slate-400 hover:text-white transition-colors text-sm">SLA & Security</a></li>
            </ul>
          </div>

          {/* Resources */}
          <div>
            <h4 className="font-bold text-white mb-6 uppercase text-xs tracking-widest">Resources</h4>
            <ul className="space-y-4">
              <li><Link to="/docs/overview" className="text-slate-400 hover:text-white transition-colors text-sm">Documentation</Link></li>
              <li><Link to="/docs/protocol-spec" className="text-slate-400 hover:text-white transition-colors text-sm">MCP Protocol Spec</Link></li>
              <li><a href="#" className="text-slate-400 hover:text-white transition-colors text-sm">Blog</a></li>
              <li><Link to="/docs/api-reference" className="text-slate-400 hover:text-white transition-colors text-sm">Developer Guide</Link></li>
              <li><a href="#" className="text-slate-400 hover:text-white transition-colors text-sm">Status Page</a></li>
            </ul>
          </div>

          {/* Contact */}
          <div>
            <h4 className="font-bold text-white mb-6 uppercase text-xs tracking-widest">Newsletter</h4>
            <p className="text-slate-400 text-sm mb-6">Stay updated with the latest in AI automation and MCP tools.</p>
            <form className="flex gap-2">
              <input 
                type="email" 
                placeholder="you@email.com" 
                className="bg-white/5 border border-white/10 rounded-lg px-4 py-2 text-sm outline-none focus:border-accent-orange transition-colors w-full"
              />
              <button className="bg-accent-orange p-2 rounded-lg hover:bg-amber-700 transition-colors">
                <ArrowRight size={20} />
              </button>
            </form>
          </div>
        </div>

        <div className="pt-12 border-t border-white/5 flex flex-col md:flex-row justify-between items-center gap-6">
          <p className="text-slate-500 text-sm">
            &copy; {new Date().getFullYear()} Actionfy. All rights reserved.
          </p>
          <div className="flex gap-8">
            <a href="#" className="text-slate-500 hover:text-white text-xs transition-colors underline-offset-4 hover:underline">Privacy Policy</a>
            <a href="#" className="text-slate-500 hover:text-white text-xs transition-colors underline-offset-4 hover:underline">Terms of Service</a>
            <a href="#" className="text-slate-500 hover:text-white text-xs transition-colors underline-offset-4 hover:underline">Cookie Policy</a>
          </div>
        </div>
      </div>
    </footer>
  );
};

const ArrowRight = ({ size, className }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" className={className}>
    <path d="M5 12h14M12 5l7 7-7 7" />
  </svg>
);

export default Footer;
