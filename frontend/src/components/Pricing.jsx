import React from 'react';
import { Check, Zap, Rocket, Shield } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import axios from 'axios';

const Pricing = () => {
  const { user } = useAuth();

  const handleUpgrade = async () => {
    if (!user) {
      window.location.href = '/login';
      return;
    }
    try {
      const res = await axios.post('/api/billing/create-checkout-session');
      if (res.data.success && res.data.data.url) {
        window.location.href = res.data.data.url;
      }
    } catch (err) {
      console.error('Upgrade error', err);
      alert('Failed to initiate upgrade');
    }
  };

  const plans = [
    {
      name: 'Free',
      price: '$0',
      description: 'Perfect for individual developers and testing.',
      icon: Zap,
      features: [
        'Up to 2 concurrent tasks',
        'Standard execution priority',
        'Standard retry policy',
        'Community support',
        'Basic monitoring',
      ],
      cta: user ? 'Current Plan' : 'Get Started',
      active: user?.tier === 'free',
      highlight: false,
    },
    {
      name: 'Pro',
      price: '$20',
      period: '/mo',
      description: 'For power users and small teams.',
      icon: Rocket,
      features: [
        'Up to 20 concurrent tasks',
        'High execution priority',
        'Advanced retry logic',
        'Priority email support',
        'Full execution logs',
        'Multi-node resilience',
      ],
      cta: user?.tier === 'pro' ? 'Current Plan' : 'Upgrade to Pro',
      active: user?.tier === 'pro',
      highlight: true,
      onClick: handleUpgrade,
    },
    {
      name: 'Enterprise',
      price: 'Custom',
      description: 'For massive scale and mission-critical workflows.',
      icon: Shield,
      features: [
        'Unlimited concurrent tasks',
        'Dedicated worker nodes',
        'SLA guarantees',
        '24/7 Phone & Slack support',
        'Custom integrations',
        'On-premise deployment options',
      ],
      cta: 'Contact Sales',
      active: false,
      highlight: false,
    },
  ];

  return (
    <section id="pricing" className="py-32 bg-white">
      <div className="container mx-auto px-6">
        <div className="max-w-3xl mx-auto text-center mb-20">
          <h2 className="text-4xl md:text-5xl font-bold text-ink-900 mb-6 tracking-tight">
            Simple, transparent <span className="text-accent-orange italic">pricing</span>.
          </h2>
          <p className="text-xl text-slate-500 font-medium">
            Choose the plan that fits your workflow. Scale up as your automation needs grow.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-6xl mx-auto">
          {plans.map((plan) => (
            <div
              key={plan.name}
              className={`relative p-8 rounded-3xl border transition-all duration-300 ${
                plan.highlight
                  ? 'border-accent-orange shadow-xl scale-105 z-10 bg-white'
                  : 'border-slate-100 hover:border-slate-200 hover:shadow-lg bg-paper-50/50'
              }`}
            >
              {plan.highlight && (
                <div className="absolute top-0 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-accent-orange text-white text-xs font-bold px-4 py-1 rounded-full uppercase tracking-widest shadow-lg">
                  Most Popular
                </div>
              )}

              <div className="mb-8">
                <div className={`w-12 h-12 rounded-2xl flex items-center justify-center mb-6 ${
                  plan.highlight ? 'bg-accent-orange text-white' : 'bg-slate-100 text-slate-600'
                }`}>
                  <plan.icon size={24} />
                </div>
                <h3 className="text-2xl font-bold text-ink-900 mb-2">{plan.name}</h3>
                <div className="flex items-baseline gap-1 mb-4">
                  <span className="text-4xl font-bold text-ink-900">{plan.price}</span>
                  {plan.period && <span className="text-slate-500 font-medium">{plan.period}</span>}
                </div>
                <p className="text-slate-500 text-sm leading-relaxed">{plan.description}</p>
              </div>

              <ul className="space-y-4 mb-10">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-3 text-sm text-slate-600">
                    <div className="mt-0.5 text-accent-orange">
                      <Check size={16} />
                    </div>
                    {feature}
                  </li>
                ))}
              </ul>

              <button
                disabled={plan.active}
                onClick={plan.onClick}
                className={`w-full py-4 rounded-2xl font-bold transition-all active:scale-95 ${
                  plan.active
                    ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                    : plan.highlight
                    ? 'bg-accent-orange text-white hover:bg-amber-700 shadow-md shadow-amber-200'
                    : 'bg-ink-900 text-white hover:bg-gray-800'
                }`}
              >
                {plan.cta}
              </button>
            </div>
          ))}
        </div>

        <div className="mt-20 text-center">
          <p className="text-slate-500 text-sm">
            All plans include SSL encryption, multi-node availability, and open-source SDK access.
            <br />
            Need a custom plan? <a href="#" className="text-accent-orange font-bold hover:underline">Chat with us</a>.
          </p>
        </div>
      </div>
    </section>
  );
};

export default Pricing;
