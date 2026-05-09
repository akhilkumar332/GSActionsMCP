import React from 'react';

const Hero = () => {
  return (
    <section className="relative pt-20 pb-32 overflow-hidden bg-paper-50">
      <div className="container px-4 mx-auto relative z-10">
        <div className="max-w-4xl mx-auto text-center">
          <span className="inline-block py-1 px-3 mb-4 text-xs font-semibold tracking-widest text-accent-orange uppercase bg-orange-50 rounded-full">
            Available Now
          </span>
          <h1 className="mb-8 text-6xl md:text-7xl font-bold font-sans text-ink-900 tracking-tight">
            The Persistent <span className="text-accent-orange italic">Scheduler</span> for MCP.
          </h1>
          <p className="mb-10 text-xl text-gray-600 font-medium leading-relaxed max-w-2xl mx-auto">
            Give your LLM tools the power of time. Schedule tasks, set reminders, and automate workflows that survive restarts and failures.
          </p>
          <div className="flex flex-wrap justify-center gap-4">
            <button className="px-8 py-4 text-white font-bold bg-ink-900 rounded-lg hover:bg-gray-800 transition-colors shadow-lg">
              Get Started
            </button>
            <button className="px-8 py-4 text-ink-900 font-bold bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors shadow-sm">
              View Documentation
            </button>
          </div>
        </div>
      </div>
      
      {/* Glassmorphism Background Elements */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-full h-full pointer-events-none overflow-hidden">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-accent-orange/5 rounded-full blur-3xl"></div>
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-ink-900/5 rounded-full blur-3xl"></div>
      </div>
    </section>
  );
};

export default Hero;
