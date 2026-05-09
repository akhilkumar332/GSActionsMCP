import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import Landing from './pages/Landing';
import Login from './pages/Login';
import Signup from './pages/Signup';
import Dashboard from './pages/Dashboard';
import Vault from './pages/Vault';
import Monitor from './pages/Monitor';
import AdminUsers from './pages/AdminUsers';
import AdminSEO from './pages/AdminSEO';
import { 
  Overview, 
  QuickStart, 
  InstallationDocs, 
  CoreConcepts, 
  ApiReference, 
  WorkerArchitecture, 
  ProtocolSpecDoc, 
  SecurityDocs 
} from './pages/Docs';

const ProtectedRoute = ({ children, roles }) => {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-ai-black text-white">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-orange"></div>
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" />;
  }

  if (roles && !roles.includes(user.role)) {
    return <Navigate to="/dashboard" />;
  }

  return children;
};

function App() {
  return (
    <AuthProvider>
      <Router>
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/login" element={<Login />} />
          <Route path="/signup" element={<Signup />} />
          
          {/* Documentation Routes */}
          <Route path="/docs/overview" element={<Overview />} />
          <Route path="/docs/quickstart" element={<QuickStart />} />
          <Route path="/docs/installation" element={<InstallationDocs />} />
          <Route path="/docs/concepts" element={<CoreConcepts />} />
          <Route path="/docs/api-reference" element={<ApiReference />} />
          <Route path="/docs/architecture" element={<WorkerArchitecture />} />
          <Route path="/docs/protocol-spec" element={<ProtocolSpecDoc />} />
          <Route path="/docs/security" element={<SecurityDocs />} />

          <Route 
            path="/dashboard" 
            element={
              <ProtectedRoute roles={['user', 'staff', 'admin']}>
                <Dashboard />
              </ProtectedRoute>
            } 
          />

          <Route 
            path="/vault" 
            element={
              <ProtectedRoute roles={['user', 'staff', 'admin']}>
                <Vault />
              </ProtectedRoute>
            } 
          />
          
          <Route 
            path="/monitor" 
            element={
              <ProtectedRoute roles={['staff', 'admin']}>
                <Monitor />
              </ProtectedRoute>
            } 
          />
          
          <Route 
            path="/admin/users" 
            element={
              <ProtectedRoute roles={['admin']}>
                <AdminUsers />
              </ProtectedRoute>
            } 
          />

          <Route 
            path="/admin/seo" 
            element={
              <ProtectedRoute roles={['admin']}>
                <AdminSEO />
              </ProtectedRoute>
            } 
          />

          <Route path="*" element={<Navigate to="/" />} />
        </Routes>
      </Router>
    </AuthProvider>
  );
}

export default App;
