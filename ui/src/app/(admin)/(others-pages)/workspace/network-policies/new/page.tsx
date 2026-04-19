import { Suspense } from "react";
import { Metadata } from "next";
import WorkspaceNetworkPolicyEditorPage from "@/components/workspace/WorkspaceNetworkPolicyEditorPage";

export const metadata: Metadata = {
  title: "New Network Policy | Aurora Control Plane",
  description: "Create a namespace-scoped network policy",
};

export default function NewWorkspaceNetworkPolicyRoutePage() {
  return (
    <Suspense fallback={null}>
      <WorkspaceNetworkPolicyEditorPage />
    </Suspense>
  );
}
