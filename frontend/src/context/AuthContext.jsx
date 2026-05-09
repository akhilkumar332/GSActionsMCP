import { createContext, useState, useContext, useEffect, useCallback } from 'react';
import axios from 'axios';

const AuthContext = createContext(null);

// Ensure cookies are sent with every request
axios.defaults.withCredentials = true;

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [csrfToken, setCsrfToken] = useState(null);

  const fetchCsrfToken = useCallback(async () => {
    try {
      const res = await axios.get('/api/auth/csrf');
      if (res.data.success) {
        setCsrfToken(res.data.csrfToken);
        axios.defaults.headers.common['X-CSRF-Token'] = res.data.csrfToken;
        return res.data.csrfToken;
      }
    } catch {
      console.error('Failed to fetch CSRF token');
    }
    return null;
  }, []);

  const checkAuth = useCallback(async () => {
    try {
      const res = await axios.get('/api/dashboard');
      if (res.data.success) {
        setUser(res.data.data.user);
      }
    } catch (err) {
      if (err.response?.status === 401) {
        setUser(null);
      }
    } finally {
      setLoading(false);
    }
  }, []);

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

    const initAuth = async () => {
       await fetchCsrfToken();
       await checkAuth();
    };
    initAuth();

    return () => axios.interceptors.response.eject(interceptor);
  }, [checkAuth, fetchCsrfToken]);

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

// eslint-disable-next-line react-refresh/only-export-components
export const useAuth = () => useContext(AuthContext);
