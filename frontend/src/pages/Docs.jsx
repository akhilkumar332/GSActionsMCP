import React from 'react';
import DocumentationLayout from '../components/DocumentationLayout';

const DevOverview = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">Documentation Overview</h1>
        <p className="text-xl text-slate-500 font-medium">Welcome to the Schedule MCP developer documentation. Learn how to build persistent AI workflows with time-based triggers.</p>
      </header>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-4">What is Schedule MCP?</h2>
        <p>
          Schedule MCP is a high-performance orchestration layer designed to give Model Context Protocol (MCP) tools 
          the ability to handle long-running and recurring tasks. Unlike standard MCP implementations which are 
          session-based and transient, Schedule MCP provides a persistent state engine that ensures your tasks 
          are executed even if your local client or server restarts.
        </p>
      </section>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 not-prose">
        <div className="p-6 rounded-2xl bg-white border border-slate-200 shadow-sm">
          <h3 className="font-bold text-lg mb-2">Persistent Workflows</h3>
          <p className="text-slate-500 text-sm">PostgreSQL-backed task queues with ACID compliance.</p>
        </div>
        <div className="p-6 rounded-2xl bg-white border border-slate-200 shadow-sm">
          <h3 className="font-bold text-lg mb-2">Distributed Execution</h3>
          <p className="text-slate-500 text-sm">Scale horizontally across multiple nodes with Redis locking.</p>
        </div>
      </div>
    </div>
  </DocumentationLayout>
);

const ProtocolSpec = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">MCP Protocol Specification</h1>
        <p className="text-xl text-slate-500 font-medium">Technical deep-dive into how we implement and extend the Model Context Protocol.</p>
      </header>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-4">The Transport Layer</h2>
        <p>
          We use **Server-Sent Events (SSE)** for the primary communication channel. This allows our backend to 
          push triggers to your local client (Claude Desktop/Cursor) whenever a scheduled window is reached.
        </p>
        <pre className="p-4 rounded-xl font-mono text-sm overflow-x-auto">
{`GET /sse HTTP/1.1
Host: api.schedulemcp.com
X-API-Key: YOUR_SECURE_KEY`}
        </pre>
      </section>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-4">Sampling Flow</h2>
        <p>
          When a task is due, the server issues a <code>sampling/createMessage</code> request via the SSE bridge. 
          The client intercepts this, performs the LLM action, and posts the result back to the server.
        </p>
      </section>
    </div>
  </DocumentationLayout>
);

const DevGuide = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">Developer Guide</h1>
        <p className="text-xl text-slate-500 font-medium">Step-by-step instructions for building and extending the scheduler.</p>
      </header>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-4">Building a Tool</h2>
        <p>
          To add a new capability, you define it in <code>cmd/server/tools.go</code>. We provide a simplified 
          SDK wrapper that makes registration straightforward:
        </p>
        <pre className="p-4 rounded-xl font-mono text-sm overflow-x-auto">
{`s.AddTool(mcp.NewTool("my_action", 
  mcp.WithDescription("Does something cool"),
), handlerFunc)`}
        </pre>
      </section>

      <section className="bg-ink-900 p-8 rounded-3xl text-white not-prose">
        <h3 className="text-xl font-bold mb-4 flex items-center gap-2">
          <span className="text-accent-orange">💡</span> Pro Tip
        </h3>
        <p className="text-slate-400">
          Always use UTC for timestamps. Our scheduler logic (Pass 4) strictly enforces UTC to avoid 
          daylight savings issues across distributed node clusters.
        </p>
      </section>
    </div>
  </DocumentationLayout>
);

export { DevOverview, ProtocolSpec, DevGuide };
