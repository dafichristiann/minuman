import React, { createContext, useCallback, useEffect, useMemo, useState } from "react";
import { fetchMe, loginUser, logoutUser, registerUser, type LoginPayload, type RegisterPayload, type User } from "../api/auth";
import { ApiError } from "../api/client";

export interface AuthContextType {
  readonly user: User | null;
  readonly isLoading: boolean;
  readonly isAuthenticated: boolean;
  login: (payload: LoginPayload) => Promise<User>;
  register: (payload: RegisterPayload) => Promise<User>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const refreshUser = useCallback(async () => {
    try {
      const me = await fetchMe();
      setUser(me);
    } catch (err) {
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
        setUser(null);
      } else {
        // Jangan hapus user jika kesalahan koneksi offline
        setUser(null);
      }
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void refreshUser();
  }, [refreshUser]);

  const login = useCallback(async (payload: LoginPayload) => {
    const session = await loginUser(payload);
    setUser(session.user);
    return session.user;
  }, []);

  const register = useCallback(async (payload: RegisterPayload) => {
    const res = await registerUser(payload);
    // Setelah registrasi, coba login atau set user
    try {
      const session = await loginUser({ email: payload.email, password: payload.password });
      setUser(session.user);
      return session.user;
    } catch {
      setUser(res.user);
      return res.user;
    }
  }, []);

  const logout = useCallback(async () => {
    try {
      await logoutUser();
    } catch {
      // Abaikan jika logout jaringan gagal
    } finally {
      setUser(null);
    }
  }, []);

  const value = useMemo<AuthContextType>(
    () => ({
      user,
      isLoading,
      isAuthenticated: user !== null,
      login,
      register,
      logout,
      refreshUser,
    }),
    [user, isLoading, login, register, logout, refreshUser],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export { useAuth } from "./useAuth";
