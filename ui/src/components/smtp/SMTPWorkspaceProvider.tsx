"use client";

import { createContext, useContext, useMemo, type ReactNode } from "react";
import { useWorkspaceScope } from "@/context/WorkspaceScopeContext";
import type { SMTPWorkspaceOption } from "@/components/smtp/types";

type SMTPWorkspaceContextValue = {
  workspace: SMTPWorkspaceOption | null;
  workspaces: SMTPWorkspaceOption[];
  workspaceID: string;
  isLoading: boolean;
  error: string;
  setWorkspaceID: (nextID: string) => void;
};

const SMTPWorkspaceContext = createContext<SMTPWorkspaceContextValue | null>(null);

export function SMTPWorkspaceProvider({ children }: { children: ReactNode }) {
  const scope = useWorkspaceScope();

  const value = useMemo<SMTPWorkspaceContextValue>(
    () => ({
      workspace: scope.workspace,
      workspaces: scope.workspaces,
      workspaceID: scope.workspaceID,
      isLoading: scope.isLoading,
      error: scope.error,
      setWorkspaceID: scope.setWorkspaceID,
    }),
    [scope],
  );

  return <SMTPWorkspaceContext.Provider value={value}>{children}</SMTPWorkspaceContext.Provider>;
}

export function useSMTPWorkspace() {
  const value = useContext(SMTPWorkspaceContext);
  if (value == null) {
    throw new Error("useSMTPWorkspace must be used within SMTPWorkspaceProvider");
  }
  return value;
}

