import { Metadata } from "next";
import WorkspaceNamespacesPage from "@/components/workspace/WorkspaceNamespacesPage";

export const metadata: Metadata = {
  title: "Namespaces | Aurora Control Plane",
  description: "Workspace namespace catalog",
};

export default function WorkspaceNamespacesRoutePage() {
  return <WorkspaceNamespacesPage />;
}
