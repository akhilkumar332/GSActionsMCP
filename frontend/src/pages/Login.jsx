import { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { useNavigate, Link } from 'react-router-dom';
import { Sparkles, ArrowRight } from 'lucide-react';
import { motion } from 'framer-motion';

const Login = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    const res = await login(email, password);
    if (res.success) {
      navigate('/dashboard');
    } else {
      setError(res.error || 'Invalid credentials');
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-ai-black relative overflow-hidden">
      {/* Background Decor */}
      <div className="absolute top-0 left-0 w-full h-full bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-20 pointer-events-none"></div>
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-accent-orange/10 rounded-full blur-[120px] pointer-events-none"></div>

      <motion.div 
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        className="relative z-10 bg-white/[0.03] backdrop-blur-2xl p-12 rounded-[3rem] border border-white/10 shadow-[0_40px_100px_rgba(0,0,0,0.8)] w-full max-w-lg"
      >
        <div className="flex flex-col items-center mb-12">
          <Link to="/" className="bg-accent-orange p-3 rounded-2xl text-white mb-8 shadow-[0_0_30px_rgba(217,119,6,0.3)] hover:rotate-[360deg] transition-transform duration-1000">
            <Sparkles size={32} />
          </Link>
          <h1 className="text-4xl font-black text-white tracking-tighter mb-2">Welcome Back.</h1>
          <p className="text-slate-500 font-bold uppercase tracking-widest text-[10px]">Access your neural orchestration layer</p>
        </div>

        {error && (
          <motion.div 
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            className="bg-red-500/10 text-red-400 p-4 rounded-2xl mb-8 text-xs font-bold border border-red-500/20 text-center uppercase tracking-widest"
          >
            {error}
          </motion.div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-2">
            <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] ml-2">Neural Identity</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full bg-black/40 px-6 py-5 rounded-2xl border border-white/10 text-white focus:border-accent-orange outline-none transition-all placeholder:text-slate-700"
              placeholder="id@network.com"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] ml-2">Access Key</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-black/40 px-6 py-5 rounded-2xl border border-white/10 text-white focus:border-accent-orange outline-none transition-all placeholder:text-slate-700"
              placeholder="••••••••"
              required
            />
          </div>
          
          <button
            type="submit"
            className="w-full bg-white text-ink-900 py-5 rounded-2xl font-black uppercase tracking-[0.2em] text-xs hover:bg-slate-100 transition-all shadow-[0_0_40px_rgba(255,255,255,0.1)] active:scale-[0.98] flex items-center justify-center gap-3"
          >
            Authenticate <ArrowRight size={16} />
          </button>
        </form>

        <p className="text-center mt-10 text-[10px] font-bold text-slate-600 uppercase tracking-widest">
          New to the network?{' '}
          <Link to="/signup" className="text-accent-orange hover:text-white transition-colors underline underline-offset-4">
            Request Access
          </Link>
        </p>
      </motion.div>
    </div>
  );
};

export default Login;
