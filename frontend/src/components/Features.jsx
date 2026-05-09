import React from 'react';
import { Calendar, Shield, Cpu, Zap } from 'lucide-react';

const FeatureCard = ({ icon: Icon, title, description }) => (
  <div className="p-8 bg-white border border-gray-100 rounded-2xl shadow-sm hover:shadow-md transition-shadow group">
    <div className="w-12 h-12 mb-6 flex items-center justify-center bg-paper-50 rounded-xl group-hover:bg-accent-orange/10 transition-colors">
      <Icon className="w-6 h-6 text-ink-900 group-hover:text-accent-orange transition-colors" />
    </div>
    <h3 className="mb-3 text-xl font-bold text-ink-900">{title}</h3>
    <p className="text-gray-600 leading-relaxed font-medium">{description}</p>
  </div>
);

const Features = () => {
  const features = [
    {
      icon: Zap,
      title: "Persistent Scheduling",
      description: "Tasks are stored in SQLite and survive server restarts, node failures, and power outages. Reliability by design."
    },
    {
      icon: Cpu,
      title: "Reliable Sampling",
      description: "LLM human-in-the-loop triggers using the Model Context Protocol. Your agents can ask for permission or wait for inputs."
    },
    {
      icon: Shield,
      title: "RBAC & Security",
      description: "Multi-tenant security with granular User, Staff, and Admin roles. Control who schedules what across your organization."
    }
  ];

  return (
    <section className="py-24 bg-white" id="features">
      <div className="container px-4 mx-auto">
        <div className="max-w-2xl mx-auto mb-16 text-center">
          <h2 className="mb-4 text-4xl font-bold text-ink-900">Built for Reliability</h2>
          <p className="text-lg text-gray-600 font-medium leading-relaxed">
            Everything you need to build robust, time-aware LLM applications that never miss a beat.
          </p>
        </div>
        <div className="grid md:grid-cols-3 gap-8">
          {features.map((feature, index) => (
            <FeatureCard key={index} {...feature} />
          ))}
        </div>
      </div>
    </section>
  );
};

export default Features;
