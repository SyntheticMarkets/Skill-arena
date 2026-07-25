"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { api, APIError } from "@/lib/api";
import type { AdminIdentity, SessionResponse } from "@/lib/types";

type AuthState = {
  admin: AdminIdentity | null;
  loading: boolean;
  enrollmentRequired: boolean;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
  can: (permission: string) => boolean;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [admin, setAdmin] = useState<AdminIdentity | null>(null);
  const [loading, setLoading] = useState(true);
  const [enrollmentRequired, setEnrollmentRequired] = useState(false);
  const router = useRouter();
  const pathname = usePathname();

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const session = await api<SessionResponse>("/api/v1/admin-crm/auth/session");
      setAdmin(session.admin);
      setEnrollmentRequired(session.mfaEnrollmentRequired);
    } catch (error) {
      if (error instanceof APIError && error.status === 401) {
        try {
          await api("/api/v1/admin-crm/auth/refresh", { method: "POST" });
          const session = await api<SessionResponse>("/api/v1/admin-crm/auth/session");
          setAdmin(session.admin);
          setEnrollmentRequired(session.mfaEnrollmentRequired);
        } catch {
          setAdmin(null);
          setEnrollmentRequired(false);
        }
      } else {
        setAdmin(null);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => void refresh(), 0);
    return () => window.clearTimeout(timer);
  }, [refresh]);

  useEffect(() => {
    if (loading) return;
    const publicRoute = pathname === "/login" || pathname === "/mfa";
    if (!admin && !publicRoute) router.replace("/login");
    if (admin && enrollmentRequired && pathname !== "/mfa/setup") router.replace("/mfa/setup");
    if (admin && !enrollmentRequired && publicRoute) router.replace("/");
  }, [admin, enrollmentRequired, loading, pathname, router]);

  const value = useMemo<AuthState>(() => ({
    admin,
    loading,
    enrollmentRequired,
    refresh,
    logout: async () => {
      try {
        await api("/api/v1/admin-crm/auth/logout", { method: "POST" });
      } finally {
        setAdmin(null);
        router.replace("/login");
      }
    },
    can: (permission) => Boolean(admin?.permissions.includes(permission))
  }), [admin, enrollmentRequired, loading, refresh, router]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
