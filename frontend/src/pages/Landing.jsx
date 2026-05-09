import React from 'react';
import Hero from '../components/Hero';
import Features from '../components/Features';
import Installation from '../components/Installation';

const Landing = () => {
  return (
    <main className="min-h-screen">
      <Hero />
      <Features />
      <Installation />
      
      {/* Footer */}
      <footer className="py-12 bg-white border-t border-gray-100">
        <div className="container px-4 mx-auto text-center">
          <p className="text-gray-500 font-medium">
            &copy; {new Date().getFullYear()} Schedule MCP. Built with reliability and speed.
          </p>
        </div>
      </footer>
    </main>
  );
};

export default Landing;
