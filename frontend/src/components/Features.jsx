import React from 'react';
import { Layers, Shield, Zap, Globe, RefreshCcw, Database } from 'lucide-react';

const Features = () => {
  const features = [
    {
      title: 'Persistent Scheduling',
      description: 'Tasks are stored in a multi-tenant PostgreSQL database with millisecond precision. Even if the entire worker cluster restarts, your schedules resume instantly.',
      icon: Database,
      color: 'bg-blue-100 text-blue-600',
    },
    {
      title: 'Reliable Sampling',
      description: 'Leveraging the Model Context Protocol to trigger LLM actions. We handle the physical connection state and SSE handshakes so you don\'t have to.',
      icon: Zap,
      color: 'bg-amber-100 text-amber-600',
    },
    {
      title: 'Multi-Node Resilience',
      description: 'Distributed worker nodes utilize Redis-backed locks to ensure no task is ever double-fired or lost. True stateless horizontal scaling.',
      icon: Globe,
      color: 'bg-emerald-100 text-emerald-600',
    },
    {
      title: 'Advanced RBAC',
      description: 'Full Role-Based Access Control out of the box. Manage User, Staff, and Admin permissions with secure session isolation.',
      icon: Shield,
      color: 'bg-purple-100 text-purple-600',
    },
    {
      title: 'Smart Retries',
      description: 'Built-in dead letter queue logic. Failed tasks are automatically retried with exponential backoff and owner notifications.',
      icon: RefreshCcw,
      color: 'bg-red-100 text-red-600',
    },
    {
      title: 'Cross-Task Context',
      description: 'Tasks can depend on the output of previous tasks. Create complex chains of AI actions with shared historical context.',
      icon: Layers,
      color: 'bg-indigo-100 text-indigo-600',
    },
  ];

  return (
    <section id="features" className="py-32 bg-white">
      <div className="container mx-auto px-6">
        <div className="max-w-3xl mx-auto text-center mb-24">
          <h2 className="text-4xl md:text-5xl font-bold text-ink-900 mb-6 tracking-tight leading-tight">
            Everything you need for <span className="text-accent-orange">industrial-grade</span> AI automation.
          </h2>
          <p className="text-xl text-slate-500 font-medium">
            We've solved the hard problems of state, time, and distributed consistency so you can focus on building agents.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-12">
          {features.map((f, index) => (
            <div 
              key={f.title} 
              className="group p-8 rounded-3xl border border-slate-100 hover:border-slate-200 hover:shadow-xl hover:shadow-slate-100/50 transition-all duration-300"
            >
              <div className={`w-14 h-14 rounded-2xl flex items-center justify-center mb-8 ${f.color} group-hover:scale-110 transition-transform`}>
                <f.icon size={28} />
              </div>
              <h3 className="text-2xl font-bold text-ink-900 mb-4 tracking-tight">{f.title}</h3>
              <p className="text-slate-500 leading-relaxed font-medium">
                {f.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default Features;
