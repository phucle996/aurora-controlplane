import { Suspense } from "react";
import { Metadata } from "next";
import WorkspaceNetworkPolicyEditorPage from "@/components/workspace/WorkspaceNetworkPolicyEditorPage";

export const metadata: Metadata = {
  title: "Edit Network Policy | Aurora Control Plane",
  description: "Edit a namespace-scoped network policy",
};

export default function WorkspaceNetworkPolicyEditorRoutePage() {
  return (
    <Suspense fallback={null}>
      <WorkspaceNetworkPolicyEditorPage />
    </Suspense>
  );
}
