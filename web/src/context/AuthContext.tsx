import React, { createContext, useContext, useState, useEffect } from 'react';
import type { ResellerUser } from '../types';
import { api, getAuthToken, setAuthToken } from '../api/client';

interface AuthContextType {
  user: ResellerUser | null;
  token: string | null;
  isLoading: boolean;
  login: (tg_id: number, pass: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType>({
  user: null,
  token: null,
  isLoading: true,
  login: async () => {},
  logout: async () => {},
  refreshUser: async () => {},
});

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [token, setTokenState] = useState<string | null>(getAuthToken());
  const [user, setUser] = useState<ResellerUser | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const refreshUser = async () => {
    try {
      const u = await api.getMe();
      setUser(u);
    } catch {
      // If unauthorized, clear
      setUser(null);
      setTokenState(null);
      setAuthToken(null);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    const t = getAuthToken();
    if (t) {
      refreshUser();
    } else {
      setIsLoading(false);
    }
  }, []);

  const login = async (tg_id: number, pass: string) => {
    setIsLoading(true);
    try {
      const resp = await api.login({ tg_id, password: pass });
      setTokenState(resp.token);
      setUser(resp.user);
    } finally {
      setIsLoading(false);
    }
  };

  const logout = async () => {
    try {
      await api.logout();
    } finally {
      setAuthToken(null);
      setTokenState(null);
      setUser(null);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        isLoading,
        login,
        logout,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => useContext(AuthContext);
