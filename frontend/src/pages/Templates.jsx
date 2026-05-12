import { useEffect, useState } from 'react';
import DashboardLayout from '../components/DashboardLayout';

const Templates = () => {
    const [templates, setTemplates] = useState([]);
    
    useEffect(() => {
        fetch('/api/v1/templates').then(res => res.json()).then(data => {
            if (Array.isArray(data)) setTemplates(data);
        }).catch(err => console.error(err));
    }, []);

    return (
        <DashboardLayout>
            <div className="p-8">
                <h1 className="text-2xl font-bold mb-4 text-white">Templates</h1>
                <div className="space-y-4">
                    {templates.map(t => <div key={t.id} className="p-4 border border-white/10 rounded text-white">{t.name}</div>)}
                </div>
            </div>
        </DashboardLayout>
    );
};
export default Templates;
