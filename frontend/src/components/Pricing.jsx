import { Check, Zap, Rocket, Shield } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { useNavigate } from 'react-router-dom';
import axios from 'axios';
import { motion } from 'framer-motion';

const Pricing = () => {
  const { user } = useAuth();
  const navigate = useNavigate();

  const handleUpgrade = async () => {
    if (!user) {
      navigate('/login');
      return;
    }
    try {
      const res = await axios.post('/api/billing/create-checkout-session');
      if (res.data.success && res.data.data.url) {
        window.location.assign(res.data.data.url);
      }
    } catch {
      alert('Failed to initiate upgrade');
    }
  };

  const plans = [
    {
      name: 'Sandbox',
      price: '$0',
      description: 'Ideal for rapid prototyping and individual discovery.',
      icon: Zap,
      features: [
        '2 concurrent task streams',
        'Standard delivery latency',
        '100 historical logs',
        'Community access',
      ],
      cta: user ? (user.tier === 'free' ? 'Active Plan' : 'Standard') : 'Start Free',
      active: user?.tier === 'free',
      highlight: false,
    },
    {
      name: 'Production',
      price: '$29',
      period: '/mo',
      description: 'For high-availability mission critical AI automation.',
      icon: Rocket,
      features: [
        '50 concurrent task streams',
        'Ultra-low latency priority',
        'Unlimited log persistence',
        'Direct engineer support',
        'Multi-region replication',
      ],
      cta: user?.tier === 'pro' ? 'Active Plan' : 'Scale to Pro',
      active: user?.tier === 'pro',
      highlight: true,
      onClick: handleUpgrade,
    },
    {
      name: 'Cluster',
      price: 'Custom',
      description: 'For industrial scale and multi-tenant deployments.',
      icon: Shield,
      features: [
        'Unlimited scaling',
        'Dedicated server clusters',
        'White-label dashboard',
        '99.99% uptime SLA',
        'On-premise deployment',
      ],
      cta: 'Contact Engineering',
      active: false,
      highlight: false,
    },
  ];

  return (
    <section id="pricing" className="py-40 bg-ai-black relative overflow-hidden">
      <div className="absolute top-0 left-0 w-full h-px bg-gradient-to-r from-transparent via-white/10 to-transparent"></div>
      
      <div className="container mx-auto px-6 relative z-10">
        <motion.div 
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="max-w-4xl mx-auto text-center mb-24"
        >
          <span className="text-accent-orange font-bold text-xs uppercase tracking-[0.4em] mb-6 inline-block text-glow">Monetization</span>
          <h2 className="text-5xl md:text-7xl font-bold text-white mb-8 tracking-tighter">
            Built for <span className="text-accent-orange">Scale.</span>
          </h2>
          <p className="text-xl text-slate-400 font-medium max-w-xl mx-auto">
            Choose the infrastructure that powers your next billion-dollar AI company.
          </p>
        </motion.div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-7xl mx-auto">
          {plans.map((plan, idx) => (
            <motion.div
              key={plan.name}
              initial={{ opacity: 0, y: 40 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: idx * 0.1, duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
              className={`group relative p-10 rounded-[2.5rem] border transition-all duration-500 flex flex-col ${
                plan.highlight
                  ? 'border-accent-orange bg-white/[0.03] shadow-[0_0_80px_rgba(217,119,6,0.15)] z-10'
                  : 'border-white/5 bg-white/[0.01] hover:bg-white/[0.03]'
              }`}
            >
              {plan.highlight && (
                <div className="absolute -top-4 left-1/2 -translate-x-1/2 bg-accent-orange text-white text-[10px] font-black px-4 py-1.5 rounded-full uppercase tracking-widest shadow-2xl z-20">
                  Recommended
                </div>
              )}

              <div className="mb-10 flex-1">
                <div className={`w-16 h-16 rounded-2xl flex items-center justify-center mb-8 bg-black/40 border border-white/5 group-hover:scale-110 transition-transform duration-500 ${
                  plan.highlight ? 'text-accent-orange shadow-[0_0_30px_rgba(217,119,6,0.2)]' : 'text-slate-400'
                }`}>
                  <plan.icon size={32} />
                </div>
                
                <h3 className="text-3xl font-bold text-white mb-3 tracking-tight">{plan.name}</h3>
                <div className="flex items-baseline gap-2 mb-6">
                  <span className="text-5xl font-bold text-white tracking-tighter">{plan.price}</span>
                  {plan.period && <span className="text-slate-500 font-bold text-sm tracking-widest uppercase">{plan.period}</span>}
                </div>
                <p className="text-slate-500 text-sm font-medium leading-relaxed mb-10">{plan.description}</p>
                
                <ul className="space-y-5">
                  {plan.features.map((feature) => (
                    <li key={feature} className="flex items-start gap-4 text-sm font-medium text-slate-400 group-hover:text-slate-300 transition-colors">
                      <Check size={18} className="text-accent-orange mt-0.5 flex-shrink-0" />
                      {feature}
                    </li>
                  ))}
                </ul>
              </div>

              <button
                disabled={plan.active}
                onClick={plan.onClick}
                className={`w-full py-5 rounded-2xl font-black text-sm uppercase tracking-widest transition-all active:scale-95 mt-auto shadow-2xl ${
                  plan.active
                    ? 'bg-white/5 text-slate-600 cursor-not-allowed border border-white/5'
                    : plan.highlight
                    ? 'bg-accent-orange text-white hover:bg-amber-700 shadow-orange-900/20'
                    : 'bg-white text-ink-900 hover:bg-slate-100'
                }`}
              >
                {plan.cta}
              </button>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default Pricing;
