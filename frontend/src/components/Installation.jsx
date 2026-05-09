import React, { useState } from 'react';
import { Terminal, Copy, Check } from 'lucide-react';

const CodeSnippet = ({ code }) => {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative group">
      <pre className="p-6 bg-ink-900 text-paper-50 rounded-xl overflow-x-auto font-mono text-sm leading-relaxed">
        <code>{code}</code>
      </pre>
      <button 
        onClick={copyToClipboard}
        className="absolute top-4 right-4 p-2 bg-white/10 hover:bg-white/20 rounded-lg transition-colors"
      >
        {copied ? <Check className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4 text-paper-50" />}
      </button>
    </div>
  );
};

const Installation = () => {
  const steps = [
    {
      title: "Install the MCP Server",
      description: "Clone the repository and build the server binary.",
      code: "git clone https://github.com/example/schedule-mcp\ncd schedule-mcp\ngo build -o server ./cmd/server"
    },
    {
      title: "Configure Claude Desktop",
      description: "Add the scheduler to your Claude config file.",
      code: "{\n  \"mcpServers\": {\n    \"scheduler\": {\n      \"command\": \"/path/to/schedule-mcp/server\",\n      \"args\": [\"-db\", \"./tasks.db\"]\n    }\n  }\n}"
    }
  ];

  return (
    <section className="py-24 bg-paper-50" id="installation">
      <div className="container px-4 mx-auto">
        <div className="max-w-3xl mx-auto">
          <h2 className="mb-12 text-4xl font-bold text-ink-900 text-center">Get Started in Minutes</h2>
          <div className="space-y-12">
            {steps.map((step, index) => (
              <div key={index} className="relative pl-12">
                <div className="absolute left-0 top-0 w-8 h-8 flex items-center justify-center bg-ink-900 text-white rounded-full font-bold text-sm">
                  {index + 1}
                </div>
                <h3 className="mb-4 text-xl font-bold text-ink-900">{step.title}</h3>
                <p className="mb-6 text-gray-600 font-medium leading-relaxed">{step.description}</p>
                <CodeSnippet code={step.code} />
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
};

export default Installation;
