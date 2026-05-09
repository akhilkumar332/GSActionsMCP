import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import Landing from './pages/Landing';
import Login from './pages/Login';
import Signup from './pages/Signup';
import Dashboard from './pages/Dashboard';
import Monitor from './pages/Monitor';
import AdminUsers from './pages/AdminUsers';
import { DevOverview, ProtocolSpec, DevGuide } from './pages/Docs';

const ProtectedRoute = ({ children, roles }) => {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[#faf9f5]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-[#d97706]"></div>
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
          <Route path="/docs/overview" element={<DevOverview />} />
          <Route path="/docs/protocol-spec" element={<ProtocolSpec />} />
          <Route path="/docs/quickstart" element={<DevGuide />} />
          <Route path="/docs/concepts" element={<DevOverview />} />
          <Route path="/docs/api-reference" element={<DevGuide />} />
          <Route path="/docs/architecture" element={<DevOverview />} />
          <Route path="/docs/installation" element={<DevOverview />} />
          <Route path="/docs/security" element={<ProtocolSpec />} />

          <Route 
            path="/dashboard" 
            element={
              <ProtectedRoute roles={['user', 'staff', 'admin']}>
                <Dashboard />
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

          <Route path="*" element={<Navigate to="/" />} />
        </Routes>
      </Router>
    </AuthProvider>
  );
}

export default App;
