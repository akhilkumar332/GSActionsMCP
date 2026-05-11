import DocumentationLayout from '../components/DocumentationLayout';
import { Terminal, Shield, Zap, Globe, Layers, Settings, Database, Code } from 'lucide-react';

const Overview = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">Overview</h1>
        <p className="text-xl text-slate-500 font-medium text-balance">Schedule MCP is a production-grade orchestration engine that brings persistence and reliability to the Model Context Protocol ecosystem.</p>
      </header>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-4">The Persistence Gap</h2>
        <p>
          Standard MCP implementations are inherently transient. Tools only exist while a session is active, 
          and there is no native way to trigger actions based on wall-clock time. If your LLM needs to 
          "remember" to perform a task in 4 hours, or every Monday at 9 AM, standard MCP fails.
        </p>
        <p className="mt-4 font-bold">Schedule MCP fills this gap by providing:</p>
        <ul className="list-disc pl-6 space-y-2 mt-4 text-slate-600">
          <li><strong>Autonomous Pipelines:</strong> Sequential task chaining where completion triggers the next action.</li>
          <li><strong>Secure Persistence:</strong> AES-256-GCM encrypted Global Secret Vault for centralized API key management.</li>
          <li><strong>Prompt Injection:</strong> Dynamic resolution of <code>{`{{secrets.NAME}}`}</code> and parent context injection.</li>
          <li><strong>Human-in-the-Loop:</strong> Real-time approval workflows for sensitive automated actions.</li>
          <li><strong>Durable State:</strong> Tasks survive server restarts and client disconnections.</li>
          <li><strong>Live Telemetry:</strong> Real-time status updates and log streaming powered by Redis.</li>
        </ul>
      </section>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 not-prose">
        <div className="p-8 rounded-3xl bg-blue-50 border border-blue-100 shadow-sm">
          <Database className="text-blue-600 mb-4" size={32} />
          <h3 className="font-bold text-xl text-blue-900 mb-2">Linear Scalability</h3>
          <p className="text-blue-700/70 text-sm">Distributed orchestration via Redis Pub/Sub ensures performance remains constant across nodes.</p>
        </div>
        <div className="p-8 rounded-3xl bg-indigo-50 border border-indigo-100 shadow-sm">
          <Layers className="text-indigo-600 mb-4" size={32} />
          <h3 className="font-bold text-xl text-indigo-900 mb-2">Agentic Chaining</h3>
          <p className="text-indigo-700/70 text-sm">Link tasks together to build complex, self-executing AI multi-step workflows.</p>
        </div>
      </div>
    </div>
  </DocumentationLayout>
);

const QuickStart = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">Quick Start</h1>
        <p className="text-xl text-slate-500 font-medium">Get your first persistent AI task running in under 5 minutes.</p>
      </header>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-6">1. Create an Account</h2>
        <p>Head over to the <a href="/signup">Sign Up</a> page. Every new account starts on the <strong>Free Tier</strong>, allowing up to 2 concurrent active tasks.</p>
      </section>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-6">2. Connect your Client</h2>
        <p className="mb-4">Install the global CLI client using <code>npx</code> and copy your API key from the Dashboard:</p>
        <div className="space-y-4">
          <pre className="p-6 rounded-2xl bg-ink-900 text-emerald-400 font-mono text-sm shadow-xl">
            $ npx @gsactions/mcp install --api-key YOUR_KEY
          </pre>
          <p className="text-sm text-slate-500 italic">Alternatively, manually configure your <code>mcp_config.json</code>:</p>
          <pre className="p-6 rounded-2xl bg-ink-900 text-emerald-400 font-mono text-sm shadow-xl">
{`{
  "mcpServers": {
    "schedule": {
      "command": "schedule-mcp",
      "args": ["run"],
      "env": { "X-API-KEY": "YOUR_KEY" }
    }
  }
}`}
          </pre>
        </div>
      </section>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-6">3. Schedule a Task</h2>
        <p>Use your LLM (Claude/Cursor) to create a task via the <code>create_task</code> tool:</p>
        <div className="bg-slate-100 p-6 rounded-2xl font-medium border border-slate-200">
          "Create a task named 'Check News' that runs every hour at minute 0 using cron '0 * * * *' and asks me 'What's happening in AI today?'"
        </div>
      </section>
    </div>
  </DocumentationLayout>
);

const InstallationDocs = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">Installation</h1>
        <p className="text-xl text-slate-500 font-medium text-balance">Comprehensive guide for deploying Schedule MCP in your own environment.</p>
      </header>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-6 flex items-center gap-3">
          <Terminal size={24} className="text-accent-orange" /> Client-Side (Global Installer)
        </h2>
        <p className="mb-6">The fastest way to install the Schedule MCP client on any machine with Node.js installed.</p>
        <div className="space-y-4">
          <div className="p-4 bg-slate-100 rounded-xl font-mono text-sm">
            $ npx @gsactions/mcp install
          </div>
          <p className="text-sm text-slate-500">For non-NPM environments, use the standard shell installer:</p>
          <div className="p-4 bg-slate-100 rounded-xl font-mono text-sm">
            $ curl -sL https://github.com/akhilkumar332/schedule-mcp/install.sh | bash
          </div>
        </div>
      </section>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-6 flex items-center gap-3">
          <Globe size={24} className="text-blue-500" /> Self-Hosted Server (Docker)
        </h2>
        <p className="mb-6">Deploy your own private Schedule MCP server using Docker Compose.</p>
        <div className="space-y-4">
          <div className="p-4 bg-slate-100 rounded-xl font-mono text-sm">
            $ git clone https://github.com/akhilkumar332/schedule-mcp.git
          </div>
          <div className="p-4 bg-slate-100 rounded-xl font-mono text-sm">
            $ cd schedule-mcp && docker-compose up -d
          </div>
        </div>
      </section>

      <section className="bg-white p-8 rounded-3xl border border-slate-200 shadow-sm not-prose">
        <h3 className="text-xl font-bold mb-4 flex items-center gap-2">
          <Settings size={20} className="text-slate-400" /> Environment Configuration
        </h3>
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead>
              <tr className="border-b border-slate-100 text-slate-400 uppercase tracking-widest text-[10px]">
                <th className="py-3 px-2">Variable</th>
                <th className="py-3 px-2">Default</th>
                <th className="py-3 px-2">Purpose</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              <tr>
                <td className="py-3 px-2 font-mono text-accent-orange">DATABASE_URL</td>
                <td className="py-3 px-2 text-slate-400 italic">required</td>
                <td className="py-3 px-2 text-slate-600">PostgreSQL connection string</td>
              </tr>
              <tr>
                <td className="py-3 px-2 font-mono text-accent-orange">REDIS_URL</td>
                <td className="py-3 px-2 font-mono">localhost:6379</td>
                <td className="py-3 px-2 text-slate-600">Redis orchestration and Pub/Sub</td>
              </tr>
              <tr>
                <td className="py-3 px-2 font-mono text-accent-orange">ENCRYPTION_KEY</td>
                <td className="py-3 px-2 text-slate-400 italic">required</td>
                <td className="py-3 px-2 text-slate-600">64-character hex key for AES-256-GCM</td>
              </tr>
              <tr>
                <td className="py-3 px-2 font-mono text-accent-orange">STRIPE_API_KEY</td>
                <td className="py-3 px-2 text-slate-400 italic">optional</td>
                <td className="py-3 px-2 text-slate-600">Secret key for billing integration</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </DocumentationLayout>
);

const CoreConcepts = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">Core Concepts</h1>
        <p className="text-xl text-slate-500 font-medium">Understand the mental model behind our scheduling engine.</p>
      </header>

      <div className="space-y-10">
        <div className="flex gap-8 group">
          <div className="flex-shrink-0 w-12 h-12 bg-white border border-slate-200 rounded-2xl flex items-center justify-center shadow-sm group-hover:bg-accent-orange group-hover:text-white transition-colors duration-500">
            <Zap size={24} />
          </div>
          <div>
            <h3 className="text-2xl font-bold text-ink-900 mb-2">The Sampling Bridge</h3>
            <p className="text-slate-600 leading-relaxed">
              Execution doesn't happen on our server. Instead, we use a **Pub/Sub bridge** to notify your 
              physical client session that a task is due. Your client then "samples" the LLM and 
              returns the output to us for logging and further scheduling.
            </p>
          </div>
        </div>

        <div className="flex gap-8 group">
          <div className="flex-shrink-0 w-12 h-12 bg-white border border-slate-200 rounded-2xl flex items-center justify-center shadow-sm group-hover:bg-indigo-600 group-hover:text-white transition-colors duration-500">
            <Layers size={24} />
          </div>
          <div>
            <h3 className="text-2xl font-bold text-ink-900 mb-2">Sequential Pipelines</h3>
            <p className="text-slate-600 leading-relaxed">
              Tasks can be chained together. When a parent task finishes, any dependent tasks flagged with 
              <code>trigger_on_completion</code> are fired immediately. The parent task's LLM output is 
              automatically injected into the child task's context, enabling complex multi-step workflows.
            </p>
          </div>
        </div>

        <div className="flex gap-8 group">
          <div className="flex-shrink-0 w-12 h-12 bg-white border border-slate-200 rounded-2xl flex items-center justify-center shadow-sm group-hover:bg-emerald-600 group-hover:text-white transition-colors duration-500">
            <Shield size={24} />
          </div>
          <div>
            <h3 className="text-2xl font-bold text-ink-900 mb-2">Secure Injection</h3>
            <p className="text-slate-600 leading-relaxed">
              Our <strong>Prompt Resolver</strong> securely handles credentials. When a task runs, any 
              <code>{`{{secrets.NAME}}`}</code> tags in your prompt are replaced with decrypted values 
              from the vault. This happens in-memory just milliseconds before the physical LLM call.
            </p>
          </div>
        </div>

        <div className="flex gap-8 group">
          <div className="flex-shrink-0 w-12 h-12 bg-white border border-slate-200 rounded-2xl flex items-center justify-center shadow-sm group-hover:bg-accent-orange group-hover:text-white transition-colors duration-500">
            <Database size={24} />
          </div>
          <div>
            <h3 className="text-2xl font-bold text-ink-900 mb-2">Transactional Locking</h3>
            <p className="text-slate-600 leading-relaxed">
              We use <code>FOR UPDATE SKIP LOCKED</code> at the database level. This allows multiple 
              parallel worker nodes to safely "claim" due tasks without ever double-triggering an 
              action, providing industry-standard consistency.
            </p>
          </div>
        </div>
      </div>
    </div>
  </DocumentationLayout>
);

const ApiReference = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">API Reference</h1>
        <p className="text-xl text-slate-500 font-medium text-balance">Technical documentation for the REST API and MCP Tools.</p>
      </header>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-6 flex items-center gap-3">
          <Code size={24} className="text-accent-orange" /> MCP Tools
        </h2>
        <div className="space-y-6">
          <div className="border border-slate-100 rounded-2xl overflow-hidden shadow-sm bg-white">
            <div className="px-6 py-4 bg-slate-50 border-b border-slate-100 flex items-center justify-between">
              <span className="font-mono font-bold text-ink-900">create_task</span>
              <span className="text-[10px] font-bold uppercase tracking-widest bg-emerald-100 text-emerald-700 px-2 py-0.5 rounded">Core Tool</span>
            </div>
            <div className="p-6 space-y-4">
              <p className="text-sm text-slate-600">Creates a new durable schedule entry.</p>
              <div className="grid grid-cols-2 gap-4 text-xs">
                <div className="space-y-1">
                  <p className="font-bold text-slate-400">Arguments</p>
                  <p className="font-mono text-slate-800">name, trigger_type, agent_prompt, secrets (optional), requires_approval (bool)</p>
                </div>
                <div className="space-y-1 text-right">
                  <p className="font-bold text-slate-400">Trigger Types</p>
                  <p className="font-mono text-slate-800">interval, cron, date</p>
                </div>
              </div>
            </div>
          </div>

          <div className="border border-slate-100 rounded-2xl overflow-hidden shadow-sm bg-white">
            <div className="px-6 py-4 bg-slate-50 border-b border-slate-100 flex items-center justify-between">
              <span className="font-mono font-bold text-ink-900">store_secret</span>
              <span className="text-[10px] font-bold uppercase tracking-widest bg-indigo-100 text-indigo-700 px-2 py-0.5 rounded">V3 Tool</span>
            </div>
            <div className="p-6">
              <p className="text-sm text-slate-600">Encrypts and stores a sensitive value in the Global Secret Vault.</p>
            </div>
          </div>

          <div className="border border-slate-100 rounded-2xl overflow-hidden shadow-sm bg-white">
            <div className="px-6 py-4 bg-slate-50 border-b border-slate-100 flex items-center justify-between">
              <span className="font-mono font-bold text-ink-900">list_secrets</span>
              <span className="text-[10px] font-bold uppercase tracking-widest bg-indigo-100 text-indigo-700 px-2 py-0.5 rounded">V3 Tool</span>
            </div>
            <div className="p-6">
              <p className="text-sm text-slate-600">Returns a Markdown table of names of stored secrets (never exposes values).</p>
            </div>
          </div>

          <div className="border border-slate-100 rounded-2xl overflow-hidden shadow-sm bg-white">
            <div className="px-6 py-4 bg-slate-50 border-b border-slate-100 flex items-center justify-between">
              <span className="font-mono font-bold text-ink-900">list_tasks</span>
              <span className="text-[10px] font-bold uppercase tracking-widest bg-emerald-100 text-emerald-700 px-2 py-0.5 rounded">Core Tool</span>
            </div>
            <div className="p-6">
              <p className="text-sm text-slate-600">Returns a beautiful Markdown table and raw JSON of all active and paused tasks.</p>
            </div>
          </div>
        </div>
      </section>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-6 flex items-center gap-3">
          <Globe size={24} className="text-blue-500" /> Outbound Webhooks
        </h2>
        <p className="mb-6">Integrate Schedule MCP with your external systems via event-driven callbacks.</p>
        <div className="bg-slate-50 p-8 rounded-3xl border border-slate-100 space-y-4">
           <h4 className="font-bold text-ink-900">Supported Events</h4>
           <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="bg-white p-4 rounded-xl border border-slate-200">
                 <p className="font-mono text-xs font-bold text-accent-orange">task_executed</p>
                 <p className="text-[11px] text-slate-500">Fires when a task completes successfully with LLM output.</p>
              </div>
              <div className="bg-white p-4 rounded-xl border border-slate-200">
                 <p className="font-mono text-xs font-bold text-red-500">task_failed</p>
                 <p className="text-[11px] text-slate-500">Fires when a task exceeds max retries or errors out.</p>
              </div>
           </div>
        </div>
      </section>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-6">REST Endpoints</h2>
        <div className="bg-ink-900 p-6 rounded-2xl space-y-4 font-mono text-xs">
          <div className="flex gap-4">
            <span className="text-blue-400 font-bold w-12">GET</span>
            <span className="text-slate-300">/api/dashboard</span>
            <span className="text-slate-500 ml-auto">Retrieve account stats</span>
          </div>
          <div className="flex gap-4 border-t border-white/5 pt-4">
            <span className="text-emerald-400 font-bold w-12">POST</span>
            <span className="text-slate-300">/api/auth/login</span>
            <span className="text-slate-500 ml-auto">Initiate browser session</span>
          </div>
        </div>
      </section>
    </div>
  </DocumentationLayout>
);

const WorkerArchitecture = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">Worker Architecture</h1>
        <p className="text-xl text-slate-500 font-medium">Inside the distributed execution engine.</p>
      </header>

      <section className="bg-white p-10 rounded-3xl border border-slate-200 shadow-sm relative overflow-hidden not-prose">
        <div className="absolute top-0 right-0 w-32 h-32 bg-accent-orange/5 rounded-full -translate-x-[-20%] -translate-y-[20%] blur-2xl"></div>
        <h2 className="text-2xl font-bold mb-8">The Lifecycle of a Task</h2>
        <div className="space-y-12 relative z-10">
          <div className="flex items-center gap-6">
            <div className="w-8 h-8 rounded-full bg-ink-900 text-white flex items-center justify-center font-bold">1</div>
            <div className="h-px flex-1 bg-slate-100"></div>
            <div className="text-sm font-bold text-slate-600">Claimed via SQL Lock</div>
          </div>
          <div className="flex items-center gap-6">
            <div className="w-8 h-8 rounded-full bg-ink-900 text-white flex items-center justify-center font-bold">2</div>
            <div className="h-px flex-1 bg-slate-100"></div>
            <div className="text-sm font-bold text-slate-600">Published to Redis Pub/Sub</div>
          </div>
          <div className="flex items-center gap-6">
            <div className="w-8 h-8 rounded-full bg-accent-orange text-white flex items-center justify-center font-bold">3</div>
            <div className="h-px flex-1 bg-slate-100"></div>
            <div className="text-sm font-bold text-slate-900 italic">Execution Node Triggers SSE</div>
          </div>
          <div className="flex items-center gap-6">
            <div className="w-8 h-8 rounded-full bg-ink-900 text-white flex items-center justify-center font-bold">4</div>
            <div className="h-px flex-1 bg-slate-100"></div>
            <div className="text-sm font-bold text-slate-600">Result Logged & Re-scheduled</div>
          </div>
        </div>
      </section>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-4">The Reaper Process</h2>
        <p>
          To ensure reliability, a separate <strong>Reaper</strong> process runs every 1 minute. 
          It scans the database for tasks that have been in the "processing" state for more than 5 minutes 
          (indicating a worker node failure) and resets them back to "active" for another node to pick up.
        </p>
      </section>
    </div>
  </DocumentationLayout>
);

const ProtocolSpecDoc = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">Protocol Specification</h1>
        <p className="text-xl text-slate-500 font-medium">Deep technical details on the Schedule MCP implementation of the protocol.</p>
      </header>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-4">The SSE Transport</h2>
        <p>
          We implement the **SSE Transport** as defined in the MCP base spec, but with a persistent connection model. 
          Every client connection is assigned a unique internal ID and mapped to a User in Redis.
        </p>
      </section>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-4">Remote Sampling (createMessage)</h2>
        <p>
          This is our core innovation. When the server decides a task is due, it crafts a <code>sampling/createMessage</code> 
          JSON-RPC notification.
        </p>
        <pre className="p-6 rounded-2xl bg-ink-900 text-emerald-400 font-mono text-sm overflow-x-auto">
{`{
  "jsonrpc": "2.0",
  "method": "sampling/createMessage",
  "params": {
    "messages": [{ "role": "user", "content": { "type": "text", "text": "..." } }],
    "maxTokens": 1000
  }
}`}
        </pre>
      </section>
    </div>
  </DocumentationLayout>
);

const SecurityDocs = () => (
  <DocumentationLayout>
    <div className="space-y-12">
      <header>
        <h1 className="text-4xl font-extrabold text-ink-900 mb-4 tracking-tighter">Auth & Security</h1>
        <p className="text-xl text-slate-500 font-medium">How we protect your credentials and AI tasks.</p>
      </header>

      <section className="grid grid-cols-1 md:grid-cols-2 gap-8 not-prose">
        <div className="p-8 rounded-3xl bg-white border border-slate-200 shadow-sm">
          <Shield size={32} className="text-emerald-500 mb-4" />
          <h3 className="font-bold text-xl mb-2">Zero-Trust Vault</h3>
          <p className="text-slate-500 text-sm">Task secrets are encrypted using AES-256-GCM at rest. We never store plain-text credentials; they are decrypted only in-memory during task execution.</p>
        </div>
        <div className="p-8 rounded-3xl bg-white border border-slate-200 shadow-sm">
          <Zap size={32} className="text-amber-500 mb-4" />
          <h3 className="font-bold text-xl mb-2">CSRF Protection</h3>
          <p className="text-slate-500 text-sm">Every mutation request is protected by double-submit cookie tokens and strictly validated Origins, mitigating cross-site scripting risks.</p>
        </div>
      </section>

      <section>
        <h2 className="text-2xl font-bold text-ink-900 mb-4">Identity & Session Isolation</h2>
        <p>
          We use **Database-backed Sessions** instead of simple JWTs. This allows for instant session revocation 
          (e.g., when a user logs out or rotates an API key). Session cookies are set with <code>HttpOnly</code>, 
          <code>Secure</code>, and <code>SameSite=Lax</code> flags.
        </p>
      </section>

      <section className="p-8 bg-red-50 border border-red-100 rounded-3xl not-prose">
        <h3 className="text-red-900 font-bold flex items-center gap-2 mb-4">
          <ShieldCheck size={20} /> Data Privacy
        </h3>
        <p className="text-red-700 text-sm">
          Schedule MCP never logs the actual contents of your API keys or passwords. Even our Staff monitoring 
          views (Pass 5) only display masked identifiers and high-level execution metadata.
        </p>
      </section>
    </div>
  </DocumentationLayout>
);

// Minimal Crown icon helper
const ShieldCheck = ({ size, className }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" className={className}>
    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10" />
    <path d="m9 12 2 2 4-4" />
  </svg>
);

export { 
  Overview, 
  QuickStart, 
  InstallationDocs, 
  CoreConcepts, 
  ApiReference, 
  WorkerArchitecture, 
  ProtocolSpecDoc, 
  SecurityDocs 
};
