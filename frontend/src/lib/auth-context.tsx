import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { api, setToken, clearToken } from "./api";

interface AuthCtx {
  token: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  signup: (data: { first_name: string; last_name: string; email: string; password: string, country: string}) => Promise<void>;
  logout: () => void;
}

const Ctx = createContext<AuthCtx | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setTokenState] = useState<string | null>(null);

  useEffect(() => {
    setTokenState(localStorage.getItem("vuetube_token"));
  }, []);

  const login = async (email: string, password: string) => {
    const res = await api.login({ email, password });
    setToken(res.token);
    setTokenState(res.token);
  };
  const signup = async (data: { first_name: string; last_name: string; email: string; password: string, country: string;}) => {
    const res = await api.signup(data);
    setToken(res.token);
    setTokenState(res.token);
  };
  const logout = () => {
    clearToken();
    setTokenState(null);
  };

  return (
    <Ctx.Provider value={{ token, isAuthenticated: !!token, login, signup, logout }}>
      {children}
    </Ctx.Provider>
  );
}

export function useAuth() {
  const c = useContext(Ctx);
  if (!c) throw new Error("useAuth outside provider");
  return c;
}
