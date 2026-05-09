import { useEffect } from 'react';
import Navbar from '../components/Navbar';
import Hero from '../components/Hero';
import Features from '../components/Features';
import Pricing from '../components/Pricing';
import Installation from '../components/Installation';
import Footer from '../components/Footer';

const Landing = () => {
  useEffect(() => {
    document.body.classList.add('dark-theme');
    return () => document.body.classList.remove('dark-theme');
  }, []);

  return (
    <div className="min-h-screen selection:bg-accent-orange selection:text-white bg-ai-black">
      <Navbar />
      <main>
        <Hero />
        <Features />
        <Installation />
        <Pricing />
      </main>
      <Footer />
    </div>
  );
};

export default Landing;
