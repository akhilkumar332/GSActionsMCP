import { useEffect, useState } from 'react';
import DashboardLayout from '../components/DashboardLayout';

const Workspaces = () => {
    const [workspaces, setWorkspaces] = useState([]);
    
    useEffect(() => {
        fetch('/api/v1/workspaces').then(res => res.json()).then(data => {
            if (Array.isArray(data)) setWorkspaces(data);
        }).catch(err => console.error(err));
    }, []);

    return (
        <DashboardLayout>
            <div className="p-8">
                <h1 className="text-2xl font-bold mb-4 text-white">Workspaces</h1>
                <div className="space-y-4">
                    {workspaces.map(w => <div key={w.id} className="p-4 border border-white/10 rounded text-white">{w.name}</div>)}
                </div>
            </div>
        </DashboardLayout>
    );
};
export default Workspaces;
