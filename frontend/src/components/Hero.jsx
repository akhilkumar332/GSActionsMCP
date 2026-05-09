import React from 'react';
import { Link } from 'react-router-dom';
import { Play, Terminal, ArrowRight, ShieldCheck } from 'lucide-react';

const Hero = () => {
  return (
    <section className="relative pt-32 pb-20 overflow-hidden bg-paper-50">
      <div className="container mx-auto px-6 relative z-10">
        <div className="flex flex-col items-center text-center max-w-5xl mx-auto">
          {/* Badge */}
          <div className="inline-flex items-center gap-2 py-1.5 px-4 mb-10 text-xs font-bold tracking-widest text-accent-orange uppercase bg-orange-100/50 border border-orange-200 rounded-full animate-in fade-in slide-in-from-bottom-2 duration-700">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-accent-orange opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-accent-orange"></span>
            </span>
            Enterprise-Grade Scheduling for MCP
          </div>

          {/* Headline */}
          <h1 className="mb-8 text-6xl md:text-8xl font-bold font-sans text-ink-900 tracking-tighter leading-[0.9] animate-in fade-in slide-in-from-bottom-4 duration-1000">
            The power of <span className="text-accent-orange italic">time</span> for your LLM tools.
          </h1>

          {/* Subheadline */}
          <p className="mb-12 text-xl md:text-2xl text-slate-600 font-medium leading-relaxed max-w-3xl animate-in fade-in slide-in-from-bottom-6 duration-1000 delay-200">
            Schedule tasks, set reminders, and orchestrate complex AI workflows that survive restarts, node failures, and network drops.
          </p>

          {/* CTAs */}
          <div className="flex flex-wrap justify-center gap-6 mb-20 animate-in fade-in slide-in-from-bottom-8 duration-1000 delay-300">
            <Link 
              to="/signup" 
              className="group px-10 py-5 text-white font-bold bg-ink-900 rounded-2xl hover:bg-gray-800 transition-all shadow-xl shadow-gray-200 flex items-center gap-3 active:scale-95"
            >
              Get Started Free <ArrowRight size={20} className="group-hover:translate-x-1 transition-transform" />
            </Link>
            <a 
              href="#installation" 
              className="px-10 py-5 text-ink-900 font-bold bg-white border border-slate-200 rounded-2xl hover:bg-slate-50 transition-all flex items-center gap-3 shadow-sm active:scale-95"
            >
              <Terminal size={20} /> View Setup
            </a>
          </div>

          {/* Visual Element: Bento/Terminal Mockup */}
          <div className="w-full max-w-6xl mx-auto relative group animate-in fade-in zoom-in duration-1000 delay-500">
            <div className="absolute inset-0 bg-accent-orange/10 blur-[120px] rounded-full group-hover:bg-accent-orange/20 transition-colors duration-700"></div>
            <div className="relative bg-[#141413] rounded-3xl border border-white/10 shadow-2xl overflow-hidden aspect-video md:aspect-[21/9] flex flex-col">
              <div className="flex items-center gap-1.5 px-6 py-4 border-b border-white/5 bg-white/5">
                <div className="w-3 h-3 rounded-full bg-red-500/50"></div>
                <div className="w-3 h-3 rounded-full bg-amber-500/50"></div>
                <div className="w-3 h-3 rounded-full bg-emerald-500/50"></div>
                <div className="ml-4 text-xs font-mono text-slate-500 tracking-widest">mcp_scheduler --status=active</div>
              </div>
              <div className="flex-1 p-8 font-mono text-sm text-slate-400 overflow-hidden">
                <div className="space-y-1">
                  <div className="flex gap-4">
                    <span className="text-emerald-500">[08:00:00]</span>
                    <span>Triggering "Daily Market Analysis" (task_id: 8f2a...)</span>
                  </div>
                  <div className="flex gap-4">
                    <span className="text-blue-500">[08:00:02]</span>
                    <span>Sampling LLM via Claude Desktop session...</span>
                  </div>
                  <div className="flex gap-4">
                    <span className="text-blue-500">[08:00:15]</span>
                    <span>LLM Response received: "Market trend is bullish based on current indicators..."</span>
                  </div>
                  <div className="flex gap-4">
                    <span className="text-emerald-500">[08:00:16]</span>
                    <span>Task success. Next run scheduled for tomorrow.</span>
                  </div>
                  <div className="flex gap-4 mt-4">
                    <span className="text-amber-500">[09:30:00]</span>
                    <span>Retrying "System Backup" (Attempt 2/3)...</span>
                  </div>
                  <div className="flex gap-4 opacity-50 italic">
                    <span>... monitoring active nodes ...</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Decorative Blur */}
      <div className="absolute top-0 right-0 w-[500px] h-[500px] bg-accent-orange/5 rounded-full blur-[120px] translate-x-1/2 -translate-y-1/2"></div>
      <div className="absolute bottom-0 left-0 w-[400px] h-[400px] bg-ink-900/5 rounded-full blur-[100px] -translate-x-1/2 translate-y-1/2"></div>
    </section>
  );
};

export default Hero;
