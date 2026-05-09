import { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { useNavigate, Link } from 'react-router-dom';
import { UserPlus, ArrowRight } from 'lucide-react';
import { motion } from 'framer-motion';

const Signup = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const { signup } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    const res = await signup(email, password);
    if (res.success) {
      navigate('/login?message=Account+created+successfully');
    } else {
      setError(res.error || 'Failed to create account');
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-ai-black relative overflow-hidden">
      {/* Background Decor */}
      <div className="absolute top-0 left-0 w-full h-full bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-20 pointer-events-none"></div>
      <div className="absolute top-0 right-0 w-[500px] h-[500px] bg-accent-orange/5 rounded-full blur-[120px] pointer-events-none translate-x-1/2 -translate-y-1/2"></div>

      <motion.div 
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        className="relative z-10 bg-white/[0.03] backdrop-blur-2xl p-12 rounded-[3rem] border border-white/10 shadow-[0_40px_100px_rgba(0,0,0,0.8)] w-full max-w-lg"
      >
        <div className="flex flex-col items-center mb-12">
          <Link to="/" className="bg-accent-orange/10 border border-accent-orange/20 p-3 rounded-2xl text-accent-orange mb-8 shadow-2xl hover:scale-110 transition-transform">
            <UserPlus size={32} />
          </Link>
          <h1 className="text-4xl font-black text-white tracking-tighter mb-2 text-center">Join the Network.</h1>
          <p className="text-slate-500 font-bold uppercase tracking-widest text-[10px]">Initialize your persistent AI workspace</p>
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
              className="w-full bg-black/40 px-6 py-5 rounded-2xl border border-white/10 text-white focus:border-accent-orange outline-none transition-all placeholder:text-slate-700 shadow-inner"
              placeholder="id@network.com"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="block text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] ml-2">Create Key</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-black/40 px-6 py-5 rounded-2xl border border-white/10 text-white focus:border-accent-orange outline-none transition-all placeholder:text-slate-700 shadow-inner"
              placeholder="••••••••"
              required
            />
          </div>
          
          <button
            type="submit"
            className="w-full bg-accent-orange text-white py-5 rounded-2xl font-black uppercase tracking-[0.2em] text-xs hover:bg-amber-700 transition-all shadow-[0_20px_50px_rgba(217,119,6,0.3)] active:scale-[0.98] flex items-center justify-center gap-3"
          >
            Create Identity <ArrowRight size={16} />
          </button>
        </form>

        <p className="text-center mt-10 text-[10px] font-bold text-slate-600 uppercase tracking-widest">
          Already registered?{' '}
          <Link to="/login" className="text-white hover:text-accent-orange transition-colors underline underline-offset-4">
            Authenticate
          </Link>
        </p>
      </motion.div>
    </div>
  );
};

export default Signup;
