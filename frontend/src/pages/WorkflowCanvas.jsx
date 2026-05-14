import { useEffect, useState, useCallback } from 'react';
import ReactFlow, { 
  addEdge, 
  Background, 
  Controls, 
  MiniMap, 
  useNodesState, 
  useEdgesState,
  MarkerType
} from 'reactflow';
import 'reactflow/dist/style.css';
import DashboardLayout from '../components/DashboardLayout';
import { motion } from 'framer-motion';
import axios from 'axios';
import { Save, RefreshCw, Layers } from 'lucide-react';

const WorkflowCanvas = () => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const fetchTasks = useCallback(async () => {
    setLoading(true);
    try {
      const res = await axios.get('/api/tasks');
      if (res.data.success) {
        const tasks = res.data.data || [];
        
        // Map tasks to nodes
        const newNodes = tasks.map((task, index) => {
          let position = { x: index * 250, y: 100 };
          
          if (task.ui_coordinates) {
            try {
              // Handle both base64 string (if Go returns []byte) and object (if already parsed)
              if (typeof task.ui_coordinates === 'string') {
                position = JSON.parse(atob(task.ui_coordinates));
              } else {
                position = task.ui_coordinates;
              }
            } catch (e) {
              console.warn("Failed to parse coordinates for task", task.id, e);
            }
          }

          return {
            id: task.id,
            position,
            data: { 
              label: (
                <div className="flex flex-col items-center gap-1">
                  <div className="text-[10px] font-black uppercase tracking-widest text-slate-500">{task.trigger_type}</div>
                  <div className="font-bold text-white text-xs">{task.name}</div>
                  <div className={`text-[8px] font-black uppercase px-2 py-0.5 rounded ${
                    task.status === 'active' ? 'bg-emerald-500/20 text-emerald-400' : 'bg-slate-500/20 text-slate-400'
                  }`}>
                    {task.status}
                  </div>
                </div>
              )
            },
            style: {
              background: 'rgba(15, 23, 42, 0.8)',
              color: '#fff',
              border: '1px solid rgba(255, 255, 255, 0.1)',
              borderRadius: '1rem',
              padding: '1rem',
              width: 180,
              backdropFilter: 'blur(12px)',
            },
          };
        });

        // Map dependencies to edges
        const newEdges = tasks
          .filter(task => task.depends_on_task_id)
          .map(task => ({
            id: `e-${task.depends_on_task_id}-${task.id}`,
            source: task.depends_on_task_id,
            target: task.id,
            animated: task.trigger_on_completion,
            label: task.trigger_on_completion ? 'triggers' : 'depends',
            labelStyle: { fill: '#94a3b8', fontWeight: 700, fontSize: 8, textTransform: 'uppercase', letterSpacing: '0.1em' },
            labelBgStyle: { fill: 'transparent' },
            style: { stroke: task.trigger_on_completion ? '#f59e0b' : '#3b82f6' },
            markerEnd: {
              type: MarkerType.ArrowClosed,
              color: task.trigger_on_completion ? '#f59e0b' : '#3b82f6',
            },
          }));

        setNodes(newNodes);
        setEdges(newEdges);
      }
    } catch (err) {
      console.error('Failed to fetch tasks', err);
    } finally {
      setLoading(false);
    }
  }, [setNodes, setEdges]);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  const onConnect = useCallback((params) => setEdges((eds) => addEdge(params, eds)), [setEdges]);

  const saveLayout = async () => {
    setSaving(true);
    try {
      // For each node, update its task's ui_coordinates
      const promises = nodes.map(node => {
        return axios.patch(`/api/tasks/${node.id}`, {
          ui_coordinates: node.position
        });
      });
      await Promise.all(promises);
      alert('Layout saved successfully!');
    } catch (err) {
      console.error('Failed to save layout', err);
      alert('Failed to save layout');
    } finally {
      setSaving(false);
    }
  };

  return (
    <DashboardLayout>
      <header className="mb-12 flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div>
          <motion.h1 
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            className="text-4xl font-black text-white tracking-tight mb-2 flex items-center gap-4"
          >
            <Layers className="text-accent-orange" size={32} />
            Workflow Canvas
          </motion.h1>
          <p className="text-slate-400 font-medium tracking-wide uppercase text-[10px] tracking-[0.2em]">Visual orchestration and flow design</p>
        </div>
        <div className="flex gap-4">
          <button 
            onClick={fetchTasks}
            disabled={loading}
            className="p-4 bg-white/5 text-white rounded-2xl border border-white/10 hover:bg-white/10 transition-colors disabled:opacity-50"
            title="Refresh Tasks"
          >
            <RefreshCw size={20} className={loading ? 'animate-spin' : ''} />
          </button>
          <button 
            onClick={saveLayout}
            disabled={saving || loading || nodes.length === 0}
            className="bg-accent-orange text-white px-8 py-4 rounded-2xl text-xs font-black uppercase tracking-widest shadow-[0_10px_30px_rgba(217,119,6,0.3)] hover:scale-105 transition-transform flex items-center gap-2 disabled:opacity-50"
          >
            {saving ? <RefreshCw size={16} className="animate-spin" /> : <Save size={16} />}
            Save Layout
          </button>
        </div>
      </header>

      <div className="h-[calc(100vh-300px)] w-full bg-slate-900/50 backdrop-blur-xl border border-white/10 rounded-[2.5rem] overflow-hidden shadow-[0_40px_100px_rgba(0,0,0,0.5)]">
        {loading ? (
          <div className="h-full w-full flex items-center justify-center">
             <div className="flex flex-col items-center gap-4">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-accent-orange"></div>
                <div className="text-slate-500 font-black uppercase tracking-widest text-[10px]">Loading Workspace...</div>
             </div>
          </div>
        ) : nodes.length === 0 ? (
          <div className="h-full w-full flex items-center justify-center">
             <div className="text-center">
                <Layers size={48} className="text-slate-700 mx-auto mb-4" />
                <div className="text-slate-500 font-black uppercase tracking-widest text-xs">No tasks found in this workspace</div>
             </div>
          </div>
        ) : (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            colorMode="dark"
            fitView
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
        )}
      </div>
    </DashboardLayout>
  );
};

export default WorkflowCanvas;
