import { useEffect, useState, useCallback, useRef } from 'react';
import ReactFlow, { 
  addEdge, 
  Background, 
  Controls, 
  MiniMap, 
  useNodesState, 
  useEdgesState,
  MarkerType
} from 'reactflow';
import dagre from 'dagre';
import 'reactflow/dist/style.css';
import DashboardLayout from '../components/DashboardLayout';
import TaskWizard from '../components/TaskWizard';
import { motion, AnimatePresence } from 'framer-motion';
import axios from 'axios';
import { Save, RefreshCw, Layers, X, Trash2, Play, Pause, FastForward, Rewind, Activity } from 'lucide-react';
import DecisionNode from '../components/DecisionNode';
import ManualRouteModal from '../components/ManualRouteModal';

const nodeTypes = {
  decision: DecisionNode,
};

const WorkflowCanvas = () => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [selectedTask, setSelectedTask] = useState(null);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isManualRouteOpen, setIsManualRouteOpen] = useState(false);
  const [rawTasks, setRawTasks] = useState([]);
  
  // Playback states
  const [playbackMode, setPlaybackMode] = useState(false);
  const [executions, setExecutions] = useState([]);
  const [selectedExecutionId, setSelectedExecutionId] = useState('');
  const [traces, setTraces] = useState([]);
  const [currentTraceIndex, setCurrentTraceIndex] = useState(-1);
  const [isPlaying, setIsPlaying] = useState(false);
  const playbackTimerRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);
  
  const sseRef = useRef(null);
  const isMountedRef = useRef(true);

  useEffect(() => {
    isMountedRef.current = true;
    return () => { isMountedRef.current = false; };
  }, []);

  const mapTasksToFlow = useCallback((tasksList) => {
    if (!isMountedRef.current) return;
    // Map tasks to nodes
    const newNodes = tasksList.map((task, index) => {
      let position = { x: index * 250, y: 100 };
      
      if (task.ui_coordinates) {
        try {
          if (typeof task.ui_coordinates === 'string') {
            position = JSON.parse(atob(task.ui_coordinates));
          } else {
            position = task.ui_coordinates;
          }
        } catch (e) {
          console.warn("Failed to parse coordinates for task", task.id, e);
        }
      }

      const isProcessing = task.status === 'processing';
      const isRouter = task.task_type === 'decision_router' || task.task_type === 'swarm_router';

      return {
        id: task.id,
        position,
        type: isRouter ? 'decision' : undefined,
        data: { 
          task,
          label: isRouter ? undefined : (
            <div className={`flex flex-col items-center gap-1 transition-all duration-500 ${isProcessing ? 'scale-110' : ''}`}>
              <div className="text-[10px] font-black uppercase tracking-widest text-slate-500">{task.trigger_type}</div>
              <div className="font-bold text-white text-xs">{task.name}</div>
              <div className={`text-[8px] font-black uppercase px-2 py-0.5 rounded ${
                task.status === 'active' ? 'bg-emerald-500/20 text-emerald-400' : 
                task.status === 'processing' ? 'bg-amber-500/20 text-amber-400 animate-pulse' :
                'bg-slate-500/20 text-slate-400'
              }`}>
                {task.status}
              </div>
              {isProcessing && (
                <div className="absolute -inset-4 bg-amber-500/10 rounded-[1.5rem] -z-10 animate-ping duration-[2000ms]" />
              )}
            </div>
          )
        },
        style: isRouter ? undefined : {
          background: isProcessing ? 'rgba(217, 119, 6, 0.15)' : 'rgba(15, 23, 42, 0.8)',
          color: '#fff',
          border: isProcessing ? '2px solid rgba(217, 119, 6, 0.5)' : '1px solid rgba(255, 255, 255, 0.1)',
          borderRadius: '1rem',
          padding: '1rem',
          width: 180,
          backdropFilter: 'blur(12px)',
          boxShadow: isProcessing ? '0 0 20px rgba(217, 119, 6, 0.3)' : 'none',
        },
      };
    });

    // Map dependencies to edges
    const newEdges = tasksList
      .filter(task => task.depends_on_task_id)
      .map(task => {
        const sourceTask = tasksList.find(t => t.id === task.depends_on_task_id);
        const isRouterSource = sourceTask?.task_type === 'decision_router' || sourceTask?.task_type === 'swarm_router';
        
        let label = task.trigger_on_completion ? 'triggers' : 'depends';
        let branchCond = null;
        
        if (task.branch_condition) {
          const rawCond = task.branch_condition;
          if (typeof rawCond === 'object') {
            branchCond = rawCond;
          } else {
            const strCond = String(rawCond);
            try {
              const decoded = atob(strCond);
              if (decoded.startsWith('{') || decoded.startsWith('[')) {
                branchCond = JSON.parse(decoded);
              }
            } catch { /* ignore */ }

            if (!branchCond) {
              try {
                branchCond = JSON.parse(strCond);
              } catch { /* ignore */ }
            }
          }
        }

        if (isRouterSource) {
          label = branchCond?.key || branchCond?.value || 'branch';
        } else if (branchCond?.value) {
          label = `if: ${branchCond.value}`;
        }

        return {
          id: `e-${task.depends_on_task_id}-${task.id}`,
          source: task.depends_on_task_id,
          target: task.id,
          animated: task.trigger_on_completion || task.status === 'processing' || isRouterSource,
          label: label,
          labelStyle: { fill: isRouterSource ? '#818cf8' : '#94a3b8', fontWeight: 800, fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.1em' },
          labelBgStyle: { fill: 'rgba(15, 23, 42, 0.8)', fillOpacity: 0.8 },
          labelBgPadding: [4, 2],
          labelBgBorderRadius: 4,
          style: { stroke: isRouterSource ? '#6366f1' : (task.trigger_on_completion ? '#f59e0b' : '#3b82f6'), strokeWidth: isRouterSource ? 3 : 2 },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: isRouterSource ? '#6366f1' : (task.trigger_on_completion ? '#f59e0b' : '#3b82f6'),
          },
        };
      });

    setNodes(newNodes);
    setEdges(newEdges);
  }, [setNodes, setEdges]);


  const fetchTasks = useCallback(async () => {
    try {
      const res = await axios.get('/api/v1/tasks');
      if (res.data.success) {
        const tasksData = res.data.data || [];
        setRawTasks(tasksData);
        mapTasksToFlow(tasksData);
      }
    } catch (err) {
      console.error('Failed to fetch tasks', err);
    } finally {
      setLoading(false);
    }
  }, [mapTasksToFlow]);

  const fetchExecutions = useCallback(async (taskId) => {
    try {
      const res = await axios.get(`/api/v1/tasks/${taskId}/executions`);
      if (res.data.success) {
        setExecutions(res.data.data || []);
        if (res.data.data?.length > 0) {
          setSelectedExecutionId(res.data.data[0].id);
        }
      }
    } catch (err) {
      console.error('Failed to fetch executions', err);
    }
  }, []);

  const fetchTraces = useCallback(async (taskId, executionId) => {
    try {
      const res = await axios.get(`/api/v1/tasks/${taskId}/traces/${executionId}`);
      if (res.data.success) {
        setTraces(res.data.data || []);
        setCurrentTraceIndex(0);
      }
    } catch (err) {
      console.error('Failed to fetch traces', err);
    }
  }, []);

  useEffect(() => {
    const loadExecutions = async () => {
      if (playbackMode && selectedTask) {
        await fetchExecutions(selectedTask.id);
      } else {
        setExecutions([]);
        setTraces([]);
        setCurrentTraceIndex(-1);
        setIsPlaying(false);
      }
    };
    loadExecutions();
  }, [playbackMode, selectedTask, fetchExecutions]);

  useEffect(() => {
    const loadTraces = async () => {
      if (selectedExecutionId && selectedTask) {
        await fetchTraces(selectedTask.id, selectedExecutionId);
      }
    };
    loadTraces();
  }, [selectedExecutionId, selectedTask, fetchTraces]);

  // Handle Playback Animation
  useEffect(() => {
    if (isPlaying && traces.length > 0) {
      playbackTimerRef.current = setInterval(() => {
        setCurrentTraceIndex(prev => {
          if (prev >= traces.length - 1) {
            setIsPlaying(false);
            return prev;
          }
          return prev + 1;
        });
      }, 1000);
    } else {
      clearInterval(playbackTimerRef.current);
    }
    return () => clearInterval(playbackTimerRef.current);
  }, [isPlaying, traces]);

  // Update visual nodes based on playback
  useEffect(() => {
    if (playbackMode && currentTraceIndex >= 0 && traces[currentTraceIndex]) {
      const activeStepName = traces[currentTraceIndex].step_name;
      
      setNodes(prev => prev.map(node => {
        const isActive = node.data.task.name === activeStepName || 
                        (currentTraceIndex === 0 && node.id === selectedTask?.id);
        
        return {
          ...node,
          style: {
            ...node.style,
            border: isActive ? '3px solid #f59e0b' : '1px solid rgba(255, 255, 255, 0.1)',
            boxShadow: isActive ? '0 0 25px rgba(245, 158, 11, 0.4)' : 'none',
            transform: isActive ? 'scale(1.05)' : 'scale(1)',
            transition: 'all 0.3s ease'
          }
        };
      }));
    } else if (!playbackMode) {
        // Reset styles when leaving playback mode
        // eslint-disable-next-line react-hooks/set-state-in-effect
        fetchTasks();
    }
  }, [playbackMode, currentTraceIndex, traces, selectedTask, fetchTasks, setNodes]);

  const updateTaskStatusLocally = useCallback((taskId, status) => {
    if (!isMountedRef.current) return;
    setNodes(prev => prev.map(node => {
        if (node.id === taskId) {
            const updatedTask = { ...node.data.task, status };
            const isProcessing = status === 'processing';
            return {
                ...node,
                data: {
                    ...node.data,
                    task: updatedTask,
                    label: (
                        <div className={`flex flex-col items-center gap-1 transition-all duration-500 ${isProcessing ? 'scale-110' : ''}`}>
                          <div className="text-[10px] font-black uppercase tracking-widest text-slate-500">{updatedTask.trigger_type}</div>
                          <div className="font-bold text-white text-xs">{updatedTask.name}</div>
                          <div className={`text-[8px] font-black uppercase px-2 py-0.5 rounded ${
                            updatedTask.status === 'active' ? 'bg-emerald-500/20 text-emerald-400' : 
                            updatedTask.status === 'processing' ? 'bg-amber-500/20 text-amber-400 animate-pulse' :
                            'bg-slate-500/20 text-slate-400'
                          }`}>
                            {updatedTask.status}
                          </div>
                          {isProcessing && (
                            <div className="absolute -inset-4 bg-amber-500/10 rounded-[1.5rem] -z-10 animate-ping duration-[2000ms]" />
                          )}
                        </div>
                    )
                },
                style: {
                    ...node.style,
                    background: isProcessing ? 'rgba(217, 119, 6, 0.15)' : 'rgba(15, 23, 42, 0.8)',
                    border: isProcessing ? '2px solid rgba(217, 119, 6, 0.5)' : '1px solid rgba(255, 255, 255, 0.1)',
                    boxShadow: isProcessing ? '0 0 20px rgba(217, 119, 6, 0.3)' : 'none',
                }
            };
        }
        return node;
    }));

    setEdges(prev => prev.map(edge => {
        if (edge.target === taskId || edge.source === taskId) {
            return {
                ...edge,
                animated: edge.animated || status === 'processing'
            };
        }
        return edge;
    }));
  }, [setNodes, setEdges]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchTasks();

    // SSE Setup
    const setupSSE = () => {
      if (sseRef.current) sseRef.current.close();
      
      const sse = new EventSource('/api/v1/events');
      sseRef.current = sse;

      sse.onmessage = (e) => {
        try {
          const event = JSON.parse(e.data);
          if (event.event_type === 'task_status_changed') {
            const payload = JSON.parse(event.payload);
            updateTaskStatusLocally(payload.task_id, payload.status);
          }
        } catch (err) {
          console.error("Failed to parse SSE event", err);
        }
      };

      sse.onerror = () => {
        console.warn("SSE Connection lost. Reconnecting in 5s...");
        sse.close();
        // Store timeout in ref so it can be cleared
        reconnectTimeoutRef.current = setTimeout(setupSSE, 5000);
      };
    };

    setupSSE();

    return () => {
      if (sseRef.current) sseRef.current.close();
      if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
    };
  }, [fetchTasks, updateTaskStatusLocally]);

  const onLayout = useCallback(() => {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setDefaultEdgeLabel(() => ({}));
    dagreGraph.setGraph({ rankdir: 'LR' });

    nodes.forEach((node) => {
      dagreGraph.setNode(node.id, { width: 180, height: 100 });
    });

    edges.forEach((edge) => {
      dagreGraph.setEdge(edge.source, edge.target);
    });

    dagre.layout(dagreGraph);

    const newNodes = nodes.map((node) => {
      const nodeWithPosition = dagreGraph.node(node.id);
      return {
        ...node,
        position: {
          x: nodeWithPosition.x - 180 / 2,
          y: nodeWithPosition.y - 100 / 2,
        },
      };
    });

    setNodes(newNodes);
  }, [nodes, edges, setNodes]);

  const onConnect = useCallback(async (params) => {
    const { source, target } = params;
    try {
      const res = await axios.post(`/api/v1/tasks/${target}/link`, {
        depends_on_task_id: source,
        trigger_on_completion: true
      });
      if (res.data.success) {
        setEdges((eds) => addEdge({ ...params, animated: true, style: { stroke: '#f59e0b' } }, eds));
        // Refresh tasks to get updated dependency state
        fetchTasks();
      }
    } catch (err) {
      console.error("Failed to link tasks", err);
      alert("Failed to link tasks: " + (err.response?.data?.error || err.message));
    }
  }, [setEdges, fetchTasks]);

  const onNodesDelete = useCallback(async (deletedNodes) => {
    for (const node of deletedNodes) {
      try {
        await axios.delete(`/api/v1/tasks/${node.id}`);
      } catch (err) {
        console.error(`Failed to delete task ${node.id}`, err);
      }
    }
    fetchTasks();
  }, [fetchTasks]);

  const onEdgesDelete = useCallback(async (deletedEdges) => {
    for (const edge of deletedEdges) {
      try {
        // Remove dependency by setting depends_on_task_id to null
        await axios.post(`/api/v1/tasks/${edge.target}/link`, {
          depends_on_task_id: null
        });
      } catch (err) {
        console.error(`Failed to remove dependency for task ${edge.target}`, err);
      }
    }
    fetchTasks();
  }, [fetchTasks]);

  const onNodeClick = useCallback((event, node) => {
    const task = node.data.task;
    setSelectedTask(task);
    
    if (task.task_type === 'decision_router' && task.last_approval_status === 'needs_routing') {
      setIsManualRouteOpen(true);
    } else {
      setIsSidebarOpen(true);
    }
  }, []);

  const handleDeleteTask = useCallback(async (taskId) => {
    if (!window.confirm('Are you sure you want to delete this task?')) return;
    
    try {
      const res = await axios.delete(`/api/v1/tasks/${taskId}`);
      if (res.data.success) {
        setIsSidebarOpen(false);
        fetchTasks();
      }
    } catch (err) {
      console.error("Failed to delete task", err);
      alert("Failed to delete task: " + (err.response?.data?.error || err.message));
    }
  }, [fetchTasks]);

  const saveLayout = async () => {
    setSaving(true);
    try {
      const promises = nodes.map(node => {
        return axios.patch(`/api/v1/tasks/${node.id}`, {
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
            onClick={onLayout}
            className="bg-white/5 text-white px-6 py-4 rounded-2xl text-xs font-black uppercase tracking-widest border border-white/10 hover:bg-white/10 transition-all flex items-center gap-2"
          >
            <Activity size={16} className="text-indigo-400" />
            Auto-Layout
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

      <div className="relative h-[calc(100vh-300px)] w-full bg-slate-900/50 backdrop-blur-xl border border-white/10 rounded-[2.5rem] overflow-hidden shadow-[0_40px_100px_rgba(0,0,0,0.5)]">
        {loading && nodes.length === 0 ? (
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
            nodeTypes={nodeTypes}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            onNodesDelete={onNodesDelete}
            onEdgesDelete={onEdgesDelete}
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

        <ManualRouteModal 
          isOpen={isManualRouteOpen}
          onClose={() => setIsManualRouteOpen(false)}
          task={selectedTask}
          tasks={rawTasks}
          onRouted={fetchTasks}
        />

        {/* Sidebar for editing */}
        <AnimatePresence>
          {isSidebarOpen && (
            <>
              <motion.div 
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                onClick={() => setIsSidebarOpen(false)}
                className="absolute inset-0 bg-black/40 backdrop-blur-sm z-40"
              />
              <motion.div 
                initial={{ x: '100%' }}
                animate={{ x: 0 }}
                exit={{ x: '100%' }}
                transition={{ type: 'spring', damping: 25, stiffness: 200 }}
                className="absolute right-0 top-0 h-full w-full max-w-md bg-zinc-900 border-l border-white/10 z-50 shadow-2xl flex flex-col"
              >
                <div className="p-6 border-b border-white/5 flex items-center justify-between">
                  <div>
                    <h3 className="text-lg font-black text-white uppercase tracking-tighter">Task Inspector</h3>
                    <p className="text-[10px] text-slate-500 font-black uppercase tracking-widest">Configuration & Logic</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <button 
                      onClick={() => setPlaybackMode(!playbackMode)}
                      className={`p-2 rounded-xl transition-all ${playbackMode ? 'bg-accent-orange text-white shadow-lg shadow-orange-500/20' : 'hover:bg-white/5 text-slate-500'}`}
                      title={playbackMode ? "Exit Playback" : "Visual Playback"}
                    >
                      <Activity size={20} />
                    </button>
                    <button 
                      onClick={() => handleDeleteTask(selectedTask.id)}
                      className="p-2 hover:bg-red-500/10 rounded-xl text-red-500 transition-colors"
                      title="Delete Task"
                    >
                      <Trash2 size={20} />
                    </button>
                    <button 
                      onClick={() => setIsSidebarOpen(false)}
                      className="p-2 hover:bg-white/5 rounded-xl text-slate-500 hover:text-white transition-colors"
                    >
                      <X size={20} />
                    </button>
                  </div>
                </div>
                
                <div className="flex-1 overflow-y-auto relative custom-scrollbar">
                   {playbackMode ? (
                     <div className="p-6 space-y-8">
                       <div className="space-y-4">
                         <label className="text-[10px] font-black uppercase tracking-widest text-slate-500">Select Execution</label>
                         <select 
                           value={selectedExecutionId}
                           onChange={(e) => setSelectedExecutionId(e.target.value)}
                           className="w-full bg-white/5 border border-white/10 rounded-xl p-3 text-xs text-white focus:outline-none focus:border-accent-orange"
                         >
                           {executions.map(ex => (
                             <option key={ex.id} value={ex.id} className="bg-zinc-900">
                               {new Date(ex.started_at).toLocaleString()} ({ex.status})
                             </option>
                           ))}
                         </select>
                       </div>

                       {traces.length > 0 ? (
                         <div className="space-y-6">
                           <div className="bg-white/5 border border-white/10 rounded-[1.5rem] p-6 space-y-6">
                             <div className="flex items-center justify-between">
                               <div className="text-[10px] font-black uppercase tracking-widest text-accent-orange">Playback Controls</div>
                               <div className="text-[10px] font-black text-slate-500">{currentTraceIndex + 1} / {traces.length}</div>
                             </div>
                             
                             <div className="flex items-center justify-center gap-4">
                               <button 
                                 onClick={() => setCurrentTraceIndex(prev => Math.max(0, prev - 1))}
                                 className="p-3 bg-white/5 rounded-full text-white hover:bg-white/10 transition-colors"
                               >
                                 <Rewind size={20} />
                               </button>
                               <button 
                                 onClick={() => setIsPlaying(!isPlaying)}
                                 className="p-5 bg-accent-orange rounded-full text-white hover:scale-110 transition-transform shadow-lg shadow-orange-500/20"
                               >
                                 {isPlaying ? <Pause size={24} /> : <Play size={24} />}
                               </button>
                               <button 
                                 onClick={() => setCurrentTraceIndex(prev => Math.min(traces.length - 1, prev + 1))}
                                 className="p-3 bg-white/5 rounded-full text-white hover:bg-white/10 transition-colors"
                               >
                                 <FastForward size={20} />
                               </button>
                             </div>

                             <div className="space-y-2">
                               <input 
                                 type="range" 
                                 min="0" 
                                 max={Math.max(0, traces.length - 1)} 
                                 value={currentTraceIndex}
                                 onChange={(e) => setCurrentTraceIndex(parseInt(e.target.value))}
                                 className="w-full h-1 bg-white/10 rounded-lg appearance-none cursor-pointer accent-accent-orange"
                               />
                             </div>
                           </div>

                           <div className="space-y-4">
                             <div className="text-[10px] font-black uppercase tracking-widest text-slate-500">Step Details</div>
                             <div className="bg-white/5 border border-white/10 rounded-[1.5rem] p-6 space-y-4">
                               <div>
                                 <div className="text-[8px] font-black uppercase text-slate-500 mb-1">Step Name</div>
                                 <div className="text-sm font-bold text-white">{traces[currentTraceIndex].step_name}</div>
                               </div>
                               <div>
                                 <div className="text-[8px] font-black uppercase text-slate-500 mb-1">Input</div>
                                 <pre className="text-[10px] bg-black/40 p-3 rounded-lg text-emerald-400 overflow-x-auto">
                                   {traces[currentTraceIndex].input_data ? JSON.stringify(JSON.parse(traces[currentTraceIndex].input_data), null, 2) : 'null'}
                                 </pre>
                               </div>
                               <div>
                                 <div className="text-[8px] font-black uppercase text-slate-500 mb-1">Output</div>
                                 <pre className="text-[10px] bg-black/40 p-3 rounded-lg text-amber-400 overflow-x-auto">
                                   {traces[currentTraceIndex].output_data ? JSON.stringify(JSON.parse(traces[currentTraceIndex].output_data), null, 2) : 'null'}
                                 </pre>
                               </div>
                               <div className="flex items-center justify-between pt-2 border-t border-white/5">
                                 <div className="text-[8px] font-black uppercase text-slate-500">Duration</div>
                                 <div className="text-[10px] font-mono text-white">
                                   {((new Date(traces[currentTraceIndex].end_time) - new Date(traces[currentTraceIndex].start_time)) / 1000).toFixed(2)}s
                                 </div>
                               </div>
                             </div>
                           </div>
                         </div>
                       ) : (
                         <div className="flex flex-col items-center justify-center py-12 text-center space-y-4">
                           <Activity size={32} className="text-slate-700 animate-pulse" />
                           <div className="text-[10px] font-black uppercase tracking-widest text-slate-500">No traces available for this execution</div>
                         </div>
                       )}
                     </div>
                   ) : (
                     <TaskWizard 
                        isOpen={isSidebarOpen} 
                        onClose={() => setIsSidebarOpen(false)} 
                        initialData={selectedTask}
                        onTaskCreated={() => {
                          fetchTasks();
                          setIsSidebarOpen(false);
                        }}
                     />
                   )}
                </div>
              </motion.div>
            </>
          )}
        </AnimatePresence>
      </div>
    </DashboardLayout>
  );
};

export default WorkflowCanvas;
