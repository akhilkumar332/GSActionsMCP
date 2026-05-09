import { Layers, Shield, Zap, Cpu, Repeat } from 'lucide-react';
import { motion } from 'framer-motion';

const Features = () => {
  const features = [
    {
      title: 'Persistent Scheduling',
      description: 'Durable task queues with sub-second precision. Even if the entire worker cluster restarts, your schedules resume exactly where they left off.',
      icon: Clock,
      color: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
    },
    {
      title: 'Secure Secret Storage',
      description: 'AES-256-GCM encrypted persistence for your sensitive API keys and credentials. Data is decrypted only in-memory during execution.',
      icon: Shield,
      color: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
    },
    {
      title: 'Human-in-the-Loop',
      description: 'Optional manual approval workflows for sensitive tasks. Pause execution and approve actions directly from your live dashboard.',
      icon: Cpu,
      color: 'bg-purple-500/10 text-purple-400 border-purple-500/20',
    },
    {
      title: 'Real-time Live Dashboards',
      description: 'Monitor task execution and AI responses in real-time. Powered by Redis Pub/Sub and SSE for instant telemetry and status updates.',
      icon: Zap,
      color: 'bg-amber-500/10 text-accent-orange border-accent-orange/20',
    },
    {
      title: 'Auto-Recovery Engine',
      description: 'Built-in node reapers and dead letter queues. Failed tasks are automatically recovered, retried, or escalated based on custom policies.',
      icon: Repeat,
      color: 'bg-red-500/10 text-red-400 border-red-500/20',
    },
    {
      title: 'Contextual Chaining',
      description: 'Orchestrate multi-step AI actions where future tasks depend on the historical output of predecessors. Build truly complex AI workflows.',
      icon: Layers,
      color: 'bg-indigo-500/10 text-indigo-400 border-indigo-500/20',
    },
  ];

  return (
    <section id="features" className="py-40 bg-ai-black relative overflow-hidden">
      {/* Background Accents */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-full h-full pointer-events-none opacity-20">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-accent-orange/10 rounded-full blur-[120px]"></div>
        <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-blue-500/10 rounded-full blur-[120px]"></div>
      </div>

      <div className="container mx-auto px-6 relative z-10">
        <motion.div 
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="max-w-4xl mx-auto text-center mb-32"
        >
          <span className="text-accent-orange font-bold text-xs uppercase tracking-[0.4em] mb-6 inline-block">Capabilities</span>
          <h2 className="text-5xl md:text-7xl font-bold text-white mb-8 tracking-tighter leading-tight">
            Built for the next decade of <br />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-white to-slate-500">Autonomous Intelligence.</span>
          </h2>
          <p className="text-xl text-slate-400 font-medium max-w-2xl mx-auto">
            Industrial-grade orchestration for developers who demand 100% reliability from their AI automation layer.
          </p>
        </motion.div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
          {features.map((f, index) => (
            <motion.div 
              key={f.title}
              initial={{ opacity: 0, y: 30 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: index * 0.1, duration: 0.6 }}
              className={`group relative p-10 rounded-[2.5rem] border bg-white/5 backdrop-blur-sm ${f.color} hover:bg-white/10 transition-all duration-500`}
            >
              <div className="absolute inset-0 bg-gradient-to-br from-white/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500 rounded-[2.5rem]"></div>
              
              <div className="relative z-10">
                <div className="w-16 h-16 rounded-2xl flex items-center justify-center mb-10 bg-black/40 border border-white/5 group-hover:scale-110 group-hover:rotate-6 transition-all duration-500 shadow-2xl">
                  <f.icon size={32} />
                </div>
                <h3 className="text-2xl font-bold text-white mb-5 tracking-tight">{f.title}</h3>
                <p className="text-slate-400 leading-relaxed font-medium">
                  {f.description}
                </p>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
};

const Clock = ({ size, className }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
    <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
  </svg>
);

export default Features;
