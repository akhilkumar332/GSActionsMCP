import { useCallback } from 'react';
import ReactFlow, { addEdge, Background, Controls, MiniMap, useNodesState, useEdgesState } from 'reactflow';
import 'reactflow/dist/style.css';
import DashboardLayout from '../components/DashboardLayout';
import { motion } from 'framer-motion';

const initialNodes = [
  { id: '1', position: { x: 0, y: 0 }, data: { label: 'Trigger' } },
  { id: '2', position: { x: 0, y: 100 }, data: { label: 'Task' } },
];
const initialEdges = [{ id: 'e1-2', source: '1', target: '2' }];

const WorkflowCanvas = () => {
  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  const onConnect = useCallback((params) => setEdges((eds) => addEdge(params, eds)), [setEdges]);

  return (
    <DashboardLayout>
      <header className="mb-12">
        <motion.h1 
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl font-black text-white tracking-tight mb-2"
        >
          Workflow Canvas
        </motion.h1>
        <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em]">Visual orchestration and flow design</p>
      </header>

      <div className="h-[calc(100vh-300px)] w-full bg-slate-900/50 backdrop-blur-xl border border-white/10 rounded-[2.5rem] overflow-hidden shadow-[0_40px_100px_rgba(0,0,0,0.5)]">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          colorMode="dark"
        >
          <Controls />
          <MiniMap 
            style={{
              backgroundColor: 'rgba(15, 23, 42, 0.8)',
              borderRadius: '1rem',
              border: '1px solid rgba(255, 255, 255, 0.1)',
            }}
            maskColor="rgba(0, 0, 0, 0.1)"
            nodeColor="rgba(217, 119, 6, 0.5)"
          />
          <Background variant="dots" gap={12} size={1} color="rgba(255, 255, 255, 0.1)" />
        </ReactFlow>
      </div>
    </DashboardLayout>
  );
};

export default WorkflowCanvas;
