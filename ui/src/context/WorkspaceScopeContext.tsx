"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { listWorkspaceOptions } from "@/components/smtp/api";
import type { SMTPWorkspaceOption } from "@/components/smtp/types";

const scopeStorageKey = "workspace-scope:selected-id";
const backendWorkspaceCookieKey = "workspace_id";
const scopeCookieMaxAgeSeconds = 60 * 60 * 24 * 30;

type WorkspaceScopeContextValue = {
  workspace: SMTPWorkspaceOption | null;
  workspaces: SMTPWorkspaceOption[];
  workspaceID: string;
  isLoading: boolean;
  error: string;
  setWorkspaceID: (nextID: string) => void;
};

const WorkspaceScopeContext = createContext<WorkspaceScopeContextValue | null>(null);

export function WorkspaceScopeProvider({ children }: { children: ReactNode }) {
  const [workspaces, setWorkspaces] = useState<SMTPWorkspaceOption[]>([]);
  const [workspaceID, setWorkspaceID] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setIsLoading(true);
      setError("");
      try {
        const items = await listWorkspaceOptions();
        if (cancelled) {
          return;
        }
        setWorkspaces(items);

        const backendCookieWorkspaceID = readScopeCookieValue(backendWorkspaceCookieKey);
        const persistedID =
          typeof window === "undefined" ? "" : window.localStorage.getItem(scopeStorageKey) ?? "";
        const fallbackWorkspaceID =
          items.find((item) => item.status === "active")?.id ?? items[0]?.id ?? "";
        const resolvedWorkspaceID =
          items.find((item) => item.id === backendCookieWorkspaceID)?.id ??
          items.find((item) => item.id === persistedID)?.id ??
          fallbackWorkspaceID;

        setWorkspaceID(resolvedWorkspaceID);
        persistWorkspaceScope(items, resolvedWorkspaceID);
      } catch (err) {
        if (!cancelled) {
          setWorkspaces([]);
          setWorkspaceID("");
          setError(err instanceof Error ? err.message : "Failed to load workspace scope.");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  const value = useMemo<WorkspaceScopeContextValue>(() => {
    const workspace = workspaces.find((item) => item.id === workspaceID) ?? null;
    return {
      workspace,
      workspaces,
      workspaceID: workspace?.id ?? "",
      isLoading,
      error,
      setWorkspaceID(nextID: string) {
        const resolvedWorkspaceID = workspaces.find((item) => item.id === nextID)?.id ?? "";
        setWorkspaceID(resolvedWorkspaceID);
        persistWorkspaceScope(workspaces, resolvedWorkspaceID);
      },
    };
  }, [error, isLoading, workspaceID, workspaces]);

  return <WorkspaceScopeContext.Provider value={value}>{children}</WorkspaceScopeContext.Provider>;
}

export function useWorkspaceScope() {
  const value = useContext(WorkspaceScopeContext);
  if (value == null) {
    throw new Error("useWorkspaceScope must be used within WorkspaceScopeProvider");
  }
  return value;
}

function readScopeCookieValue(key: string) {
  if (typeof document === "undefined") {
    return "";
  }
  const entries = document.cookie.split(";").map((entry) => entry.trim());
  const pair = entries.find((entry) => entry.startsWith(`${key}=`));
  if (!pair) {
    return "";
  }
  const value = pair.split("=").slice(1).join("=");
  return decodeURIComponent(value);
}

function persistWorkspaceScope(workspaces: SMTPWorkspaceOption[], workspaceID: string) {
  if (typeof window !== "undefined") {
    if (workspaceID === "") {
      window.localStorage.removeItem(scopeStorageKey);
    } else {
      window.localStorage.setItem(scopeStorageKey, workspaceID);
    }
  }

  if (typeof document !== "undefined") {
    const selectedWorkspaceID = workspaces.find((item) => item.id === workspaceID)?.id ?? "";
    if (selectedWorkspaceID === "") {
      document.cookie = `${backendWorkspaceCookieKey}=; Max-Age=0; Path=/; SameSite=Lax`;
    } else {
      document.cookie = `${backendWorkspaceCookieKey}=${encodeURIComponent(
        selectedWorkspaceID,
      )}; Max-Age=${scopeCookieMaxAgeSeconds}; Path=/; SameSite=Lax`;
    }
  }
}
