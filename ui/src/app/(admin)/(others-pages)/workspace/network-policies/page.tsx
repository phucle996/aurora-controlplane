import { Suspense } from "react";
import { Metadata } from "next";
import WorkspaceNetworkPoliciesPage from "@/components/workspace/WorkspaceNetworkPoliciesPage";

export const metadata: Metadata = {
  title: "Network Policies | Aurora Control Plane",
  description: "Namespace-scoped network policy catalog",
};

export default function WorkspaceNetworkPoliciesRoutePage() {
  return (
    <Suspense fallback={null}>
      <WorkspaceNetworkPoliciesPage />
    </Suspense>
  );
}
