import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { getAuthStatus } from '../services/api';

interface AuthContextType {
  isAuthenticated: boolean;
  authEnabled: boolean;
  username: string | null;
  loading: boolean;
  logout: () => void;
  checkAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [authEnabled, setAuthEnabled] = useState(true);
  const [username, setUsername] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const checkAuth = useCallback(async () => {
    try {
      const { data } = await getAuthStatus();
      setAuthEnabled(data.authEnabled);

      if (!data.authEnabled) {
        // Auth is disabled, user is always authenticated
        setIsAuthenticated(true);
        setLoading(false);
        return;
      }

      // Check if token exists and is not expired
      const token = localStorage.getItem('token');
      const tokenExpires = localStorage.getItem('tokenExpires');
      const storedUsername = localStorage.getItem('username');

      if (token && tokenExpires) {
        const expiresAt = parseInt(tokenExpires, 10);
        if (Date.now() / 1000 < expiresAt) {
          setIsAuthenticated(true);
          setUsername(storedUsername);
        } else {
          // Token expired
          localStorage.removeItem('token');
          localStorage.removeItem('username');
          localStorage.removeItem('tokenExpires');
          setIsAuthenticated(false);
        }
      } else {
        setIsAuthenticated(false);
      }
    } catch (error) {
      console.error('Failed to check auth status:', error);
      // Assume auth is enabled if we can't check
      setAuthEnabled(true);
      setIsAuthenticated(false);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  const logout = useCallback(() => {
    localStorage.removeItem('token');
    localStorage.removeItem('username');
    localStorage.removeItem('tokenExpires');
    setIsAuthenticated(false);
    setUsername(null);
  }, []);

  return (
    <AuthContext.Provider value={{ 
      isAuthenticated, 
      authEnabled, 
      username, 
      loading, 
      logout,
      checkAuth 
    }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

