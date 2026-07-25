"use client";

import { useCallback, useEffect, useState } from "react";
import { api, APIError } from "./api";

export function useResource<T>(path: string) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setData(await api<T>(path));
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "The operational service is unavailable.");
    } finally {
      setLoading(false);
    }
  }, [path]);

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 0);
    return () => window.clearTimeout(timer);
  }, [load]);

  return { data, setData, loading, error, reload: load };
}
