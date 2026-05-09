import React, { createContext, useState, useContext, useEffect } from 'react';
import axios from 'axios';

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [csrfToken, setCsrfToken] = useState(null);

  const fetchCsrfToken = async () => {
    try {
      const res = await axios.get('/api/auth/csrf');
      if (res.data.success) {
        setCsrfToken(res.data.csrfToken);
        axios.defaults.headers.common['X-CSRF-Token'] = res.data.csrfToken;
        return res.data.csrfToken;
      }
    } catch (err) {
      console.error('Failed to fetch CSRF token', err);
    }
    return null;
  };

  const checkAuth = async () => {
    try {
      if (!csrfToken) {
        const token = await fetchCsrfToken();
        if (!token) return;
      }
      
      const res = await axios.get('/api/dashboard');
      if (res.data.success) {
        // Use functional update to prevent overwriting manual login
        setUser(prev => prev || res.data.data.user);
      }
    } catch (err) {
      if (err.response?.status === 401) {
        setUser(null);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // Setup interceptor for CSRF auto-refresh
    const interceptor = axios.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;
        if (error.response?.status === 403 && !originalRequest._retry) {
          originalRequest._retry = true;
          const newToken = await fetchCsrfToken();
          if (newToken) {
            originalRequest.headers['X-CSRF-Token'] = newToken;
            return axios(originalRequest);
          }
        }
        return Promise.reject(error);
      }
    );

    checkAuth();
    return () => axios.interceptors.response.eject(interceptor);
  }, []);

  const login = async (email, password) => {
    try {
      if (!csrfToken) await fetchCsrfToken();
      const res = await axios.post('/api/auth/login', { email, password });
      if (res.data.success) {
        setCsrfToken(res.data.csrfToken);
        axios.defaults.headers.common['X-CSRF-Token'] = res.data.csrfToken;
        setUser(res.data.data);
        return { success: true };
      }
      return { success: false, error: res.data.error };
    } catch (err) {
      return { success: false, error: err.response?.data?.error || 'Login failed' };
    }
  };

  const signup = async (email, password) => {
    try {
      if (!csrfToken) await fetchCsrfToken();
      const res = await axios.post('/api/auth/signup', { email, password });
      if (res.data.success) {
        setCsrfToken(res.data.csrfToken);
        axios.defaults.headers.common['X-CSRF-Token'] = res.data.csrfToken;
        return { success: true };
      }
      return { success: false, error: res.data.error };
    } catch (err) {
      return { success: false, error: err.response?.data?.error || 'Signup failed' };
    }
  };

  const logout = async () => {
    try {
      await axios.post('/api/auth/logout');
    } catch (err) {
      console.error('Logout error', err);
    } finally {
      setUser(null);
      setCsrfToken(null);
      delete axios.defaults.headers.common['X-CSRF-Token'];
    }
  };

  return (
    <AuthContext.Provider value={{ user, loading, login, signup, logout, checkAuth }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
