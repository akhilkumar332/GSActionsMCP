import React, { useState } from 'react';
import { Terminal, Copy, Check, ExternalLink, Cpu, Layout } from 'lucide-react';

const Installation = () => {
  const [copied, setCopied] = useState(null);

  const handleCopy = (text, id) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 2000);
  };

  const steps = [
    {
      title: 'Step 1: Get your API Key',
      description: 'Sign up for a free account and copy your unique API key from the dashboard.',
      icon: Layout,
    },
    {
      title: 'Step 2: Add to MCP Config',
      description: 'Add the Schedule MCP server to your local Claude Desktop or Cursor configuration.',
      icon: Cpu,
    }
  ];

  const configSnippet = `{
  "mcpServers": {
    "schedule-mcp": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/inspector", "http://localhost:8080/sse"],
      "env": {
        "X-API-KEY": "YOUR_API_KEY_HERE"
      }
    }
  }
}`;

  return (
    <section id="installation" className="py-32 bg-paper-50">
      <div className="container mx-auto px-6">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-20 items-center">
          <div>
            <h2 className="text-4xl md:text-5xl font-bold text-ink-900 mb-8 tracking-tight">
              Get up and running in <span className="text-accent-orange italic">seconds</span>.
            </h2>
            <p className="text-xl text-slate-500 font-medium mb-12 leading-relaxed">
              Schedule MCP integrates seamlessly with your existing local LLM tools via the Model Context Protocol.
            </p>

            <div className="space-y-8">
              {steps.map((step, idx) => (
                <div key={idx} className="flex gap-6">
                  <div className="flex-shrink-0 w-12 h-12 bg-white border border-slate-200 rounded-xl flex items-center justify-center shadow-sm">
                    <step.icon size={24} className="text-accent-orange" />
                  </div>
                  <div>
                    <h4 className="text-lg font-bold text-ink-900 mb-2">{step.title}</h4>
                    <p className="text-slate-500 font-medium leading-relaxed">{step.description}</p>
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-12 flex flex-wrap gap-4">
              <a 
                href="#" 
                className="inline-flex items-center gap-2 text-sm font-bold text-accent-orange hover:underline"
              >
                View full documentation <ExternalLink size={14} />
              </a>
              <span className="text-slate-300">|</span>
              <a 
                href="https://modelcontextprotocol.io" 
                target="_blank"
                className="inline-flex items-center gap-2 text-sm font-bold text-slate-400 hover:text-ink-900 transition-colors"
              >
                Learn about MCP <ExternalLink size={14} />
              </a>
            </div>
          </div>

          <div className="relative">
            <div className="absolute inset-0 bg-accent-orange/5 blur-[100px] rounded-full"></div>
            <div className="relative bg-[#141413] rounded-3xl border border-white/10 shadow-2xl overflow-hidden">
              <div className="flex items-center justify-between px-6 py-4 border-b border-white/5 bg-white/5">
                <div className="flex gap-1.5">
                  <div className="w-3 h-3 rounded-full bg-slate-700"></div>
                  <div className="w-3 h-3 rounded-full bg-slate-700"></div>
                  <div className="w-3 h-3 rounded-full bg-slate-700"></div>
                </div>
                <button 
                  onClick={() => handleCopy(configSnippet, 'config')}
                  className="flex items-center gap-2 text-xs font-bold text-slate-400 hover:text-white transition-colors"
                >
                  {copied === 'config' ? (
                    <><Check size={14} className="text-emerald-500" /> Copied!</>
                  ) : (
                    <><Copy size={14} /> Copy Config</>
                  )}
                </button>
              </div>
              <div className="p-8 font-mono text-sm leading-relaxed overflow-x-auto">
                <pre className="text-emerald-400">
                  {configSnippet}
                </pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};

export default Installation;
