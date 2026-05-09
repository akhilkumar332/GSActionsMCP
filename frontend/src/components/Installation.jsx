import { useState } from 'react';
import { Terminal, Copy, Check, ExternalLink, Layout, Boxes } from 'lucide-react';
import { motion } from 'framer-motion';

const Installation = () => {
  const [copied, setCopied] = useState(null);

  const handleCopy = (text, id) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 2000);
  };

  const steps = [
    {
      title: 'Initialize Workspace',
      description: 'Sign up for a professional account and generate your cryptographically secure API key.',
      icon: Layout,
    },
    {
      title: 'Install CLI Client',
      description: 'Run the global installer with a single command to deploy the MCP bridge to your local machine.',
      icon: Terminal,
    },
    {
      title: 'Bridge the Session',
      description: 'Add the Schedule MCP engine to your local configuration via the Model Context Protocol bridge.',
      icon: Boxes,
    }
  ];

  const installCommand = 'npx @google-schedule-actions/mcp install';
  const configSnippet = `{
  "mcpServers": {
    "schedule-mcp": {
      "command": "schedule-mcp",
      "args": ["run"],
      "env": {
        "X-API-KEY": "YOUR_ENCRYPTED_KEY"
      }
    }
  }
}`;

  return (
    <section id="installation" className="py-40 bg-ai-black overflow-hidden relative">
      <div className="absolute inset-0 bg-gradient-to-b from-ai-black via-ai-grey to-ai-black"></div>
      
      <div className="container mx-auto px-6 relative z-10">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-32 items-center">
          <motion.div
            initial={{ opacity: 0, x: -50 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.8 }}
          >
            <span className="text-accent-orange font-bold text-xs uppercase tracking-[0.4em] mb-6 inline-block text-glow">Setup Guide</span>
            <h2 className="text-5xl md:text-7xl font-bold text-white mb-10 tracking-tighter">
              Deployment in <span className="text-accent-orange italic underline decoration-white/10 underline-offset-[12px]">Seconds.</span>
            </h2>
            <p className="text-xl text-slate-400 font-medium mb-16 leading-relaxed max-w-xl">
              Engineered for seamless integration with Claude Desktop, Cursor, and custom MCP clients.
            </p>

            <div className="space-y-12">
              {steps.map((step, idx) => (
                <motion.div 
                  key={idx}
                  initial={{ opacity: 0, y: 20 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true }}
                  transition={{ delay: idx * 0.2, duration: 0.5 }}
                  className="flex gap-8 group"
                >
                  <div className="flex-shrink-0 w-14 h-14 bg-white/5 border border-white/10 rounded-2xl flex items-center justify-center shadow-2xl group-hover:bg-accent-orange group-hover:text-white transition-all duration-500">
                    <step.icon size={28} />
                  </div>
                  <div>
                    <h4 className="text-xl font-bold text-white mb-3 tracking-tight">{step.title}</h4>
                    <p className="text-slate-500 font-medium leading-relaxed max-w-sm">{step.description}</p>
                    {step.title === 'Install CLI Client' && (
                       <div className="mt-4 bg-black/40 border border-white/5 rounded-xl p-4 flex items-center justify-between group/cmd">
                          <code className="text-emerald-400 text-xs">{installCommand}</code>
                          <button onClick={() => handleCopy(installCommand, 'install')} className="text-slate-600 hover:text-white">
                             {copied === 'install' ? <Check size={14} /> : <Copy size={14} />}
                          </button>
                       </div>
                    )}
                  </div>
                </motion.div>
              ))}
            </div>

            <div className="mt-16 flex flex-wrap gap-8 items-center">
              <a 
                href="/docs/quickstart" 
                className="group inline-flex items-center gap-2 text-sm font-bold text-accent-orange hover:text-white transition-colors"
              >
                Detailed Docs <ArrowRight size={16} className="group-hover:translate-x-1 transition-transform" />
              </a>
              <div className="h-1 w-1 rounded-full bg-slate-800"></div>
              <a 
                href="https://modelcontextprotocol.io" 
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-2 text-sm font-bold text-slate-500 hover:text-white transition-colors"
              >
                Protocol Spec <ExternalLink size={14} />
              </a>
            </div>
          </motion.div>

          <motion.div 
            initial={{ opacity: 0, scale: 0.9 }}
            whileInView={{ opacity: 1, scale: 1 }}
            viewport={{ once: true }}
            transition={{ duration: 0.8 }}
            className="relative"
          >
            <div className="absolute inset-0 bg-accent-orange/10 blur-[120px] rounded-full animate-pulse"></div>
            <div className="relative bg-black/60 backdrop-blur-3xl rounded-[3rem] border border-white/10 shadow-[0_40px_100px_rgba(0,0,0,0.8)] overflow-hidden">
              <div className="flex items-center justify-between px-10 py-6 border-b border-white/5 bg-white/5">
                <div className="flex gap-2">
                  <div className="w-3 h-3 rounded-full bg-red-500/20"></div>
                  <div className="w-3 h-3 rounded-full bg-amber-500/20"></div>
                  <div className="w-3 h-3 rounded-full bg-emerald-500/20"></div>
                </div>
                <button 
                  onClick={() => handleCopy(configSnippet, 'config')}
                  className="flex items-center gap-2 text-[10px] font-bold text-slate-500 hover:text-white uppercase tracking-widest transition-colors"
                >
                  {copied === 'config' ? (
                    <><Check size={14} className="text-emerald-500" /> Copied!</>
                  ) : (
                    <><Copy size={14} /> Copy Config</>
                  )}
                </button>
              </div>
              <div className="p-10 font-mono text-sm leading-relaxed overflow-x-auto min-h-[300px] flex items-center">
                <pre className="text-emerald-400/90 w-full">
                  {configSnippet}
                </pre>
              </div>
            </div>
          </motion.div>
        </div>
      </div>
    </section>
  );
};

const ArrowRight = ({ size, className }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" className={className}>
    <path d="M5 12h14M12 5l7 7-7 7" />
  </svg>
);

export default Installation;
