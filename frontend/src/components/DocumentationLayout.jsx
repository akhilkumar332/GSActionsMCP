import React from 'react';
import { NavLink } from 'react-router-dom';
import { Book, Code, Terminal, FileJson, Info, Zap, Shield, Workflow } from 'lucide-react';

const DocumentationLayout = ({ children }) => {
  const sections = [
    {
      title: 'Getting Started',
      links: [
        { name: 'Overview', path: '/docs/overview', icon: Info },
        { name: 'Quick Start', path: '/docs/quickstart', icon: Zap },
        { name: 'Installation', path: '/docs/installation', icon: Terminal },
      ]
    },
    {
      title: 'Developer Guide',
      links: [
        { name: 'Core Concepts', path: '/docs/concepts', icon: Book },
        { name: 'API Reference', path: '/docs/api-reference', icon: Code },
        { name: 'Worker Architecture', path: '/docs/architecture', icon: Workflow },
      ]
    },
    {
      title: 'MCP Protocol',
      links: [
        { name: 'Protocol Spec', path: '/docs/protocol-spec', icon: FileJson },
        { name: 'Auth & Security', path: '/docs/security', icon: Shield },
      ]
    }
  ];

  return (
    <div className="flex min-h-screen bg-[#faf9f5]">
      {/* Docs Sidebar */}
      <aside className="w-80 border-r border-slate-200 bg-white hidden lg:block sticky top-0 h-screen overflow-y-auto">
        <div className="p-8">
          <NavLink to="/" className="flex items-center gap-2 mb-10 group">
            <div className="bg-accent-orange p-1 rounded text-white group-hover:scale-110 transition-transform">
              <Zap size={18} fill="currentColor" />
            </div>
            <span className="font-bold text-lg tracking-tight text-ink-900">Schedule MCP</span>
          </NavLink>

          <nav className="space-y-8">
            {sections.map((section) => (
              <div key={section.title}>
                <h4 className="text-xs font-bold text-slate-400 uppercase tracking-widest mb-4">{section.title}</h4>
                <ul className="space-y-1">
                  {section.links.map((link) => (
                    <li key={link.path}>
                      <NavLink
                        to={link.path}
                        className={({ isActive }) =>
                          `flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all ${
                            isActive
                              ? 'bg-accent-orange/10 text-accent-orange shadow-sm'
                              : 'text-slate-600 hover:text-ink-900 hover:bg-slate-50'
                          }`
                        }
                      >
                        <link.icon size={16} />
                        {link.name}
                      </NavLink>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </nav>
        </div>
      </aside>

      {/* Docs Content */}
      <main className="flex-1 p-8 md:p-16 lg:p-24 overflow-y-auto">
        <div className="max-w-3xl mx-auto prose prose-slate prose-headings:font-bold prose-headings:tracking-tight prose-a:text-accent-orange prose-pre:bg-ink-900 prose-pre:border prose-pre:border-white/10 prose-code:text-accent-orange">
          {children}
        </div>
      </main>
    </div>
  );
};

export default DocumentationLayout;
