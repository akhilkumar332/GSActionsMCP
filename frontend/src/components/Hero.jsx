import { Link } from 'react-router-dom';
import { Terminal, ArrowRight, ShieldCheck, Clock, Sparkles } from 'lucide-react';
import { motion } from 'framer-motion';
import Scene3D from './Scene3D';

const Hero = () => {
  return (
    <section className="relative min-h-screen flex items-center justify-center overflow-hidden bg-ai-black pt-20">
      {/* Dynamic Grid Background */}
      <div className="absolute inset-0 bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-20 brightness-100 pointer-events-none"></div>
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-ai-black to-ai-black"></div>
      
      <div className="container mx-auto px-6 relative z-20">
        <div className="flex flex-col items-center text-center">
          
          {/* AI Badge */}
          <motion.div 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8 }}
            className="inline-flex items-center gap-2 py-2 px-5 mb-12 text-[10px] font-bold tracking-[0.2em] text-accent-orange uppercase bg-white/5 border border-white/10 rounded-full backdrop-blur-md glow-text"
          >
            <Sparkles size={14} />
            The Future of Persistent AI
          </motion.div>

          {/* Epic Headline */}
          <motion.h1 
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 1, ease: [0.16, 1, 0.3, 1] }}
            className="mb-8 text-7xl md:text-9xl font-bold font-sans text-white tracking-tighter leading-[0.85] max-w-6xl"
          >
            Orchestrate <br />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-accent-orange to-amber-200">Intelligence.</span>
          </motion.h1>

          {/* Visionary Subheadline */}
          <motion.p 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.4, duration: 1 }}
            className="mb-14 text-xl md:text-2xl text-slate-400 font-medium leading-relaxed max-w-3xl text-balance"
          >
            A high-performance state engine for Model Context Protocol. <br />
            Durable scheduling, cross-node consistency, and autonomous AI workflows.
          </motion.p>

          {/* Premium CTAs */}
          <motion.div 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.6, duration: 0.8 }}
            className="flex flex-wrap justify-center gap-6 mb-24"
          >
            <Link 
              to="/signup" 
              className="group relative px-12 py-5 text-ink-900 font-bold bg-white rounded-2xl hover:bg-slate-100 transition-all shadow-[0_0_40px_rgba(255,255,255,0.15)] flex items-center gap-3 active:scale-95 overflow-hidden"
            >
              <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-1000"></div>
              Launch Engine <ArrowRight size={20} className="group-hover:translate-x-1 transition-transform" />
            </Link>
            <a 
              href="#installation" 
              className="px-12 py-5 text-white font-bold bg-white/5 border border-white/10 rounded-2xl hover:bg-white/10 transition-all flex items-center gap-3 backdrop-blur-xl active:scale-95 shadow-2xl"
            >
              <Terminal size={20} className="text-accent-orange" /> Developer Guide
            </a>
          </motion.div>
        </div>

        {/* Floating 3D Component */}
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-full h-full -z-10 opacity-60">
           <Scene3D />
        </div>
      </div>

      {/* Trust & Scale Row */}
      <div className="absolute bottom-10 left-0 right-0 z-30">
        <div className="container mx-auto px-6">
          <div className="flex flex-wrap justify-center items-center gap-12 md:gap-24 opacity-30 grayscale hover:opacity-100 transition-all duration-700">
             <div className="flex items-center gap-3 font-bold text-xs tracking-widest text-white uppercase">
               <ShieldCheck size={18} className="text-accent-orange" /> ACID State
             </div>
             <div className="flex items-center gap-3 font-bold text-xs tracking-widest text-white uppercase">
               <Clock size={18} className="text-accent-orange" /> Real-Time SSE
             </div>
             <div className="flex items-center gap-3 font-bold text-xs tracking-widest text-white uppercase">
               <Sparkles size={18} className="text-accent-orange" /> Model Agnostic
             </div>
          </div>
        </div>
      </div>

      {/* Side Glows */}
      <div className="absolute top-1/4 -left-64 w-[600px] h-[600px] bg-accent-orange/10 rounded-full blur-[160px] pointer-events-none"></div>
      <div className="absolute bottom-1/4 -right-64 w-[600px] h-[600px] bg-accent-orange/5 rounded-full blur-[160px] pointer-events-none"></div>
    </section>
  );
};

export default Hero;
