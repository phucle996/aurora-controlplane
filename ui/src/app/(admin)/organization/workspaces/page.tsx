import { Metadata } from "next";
import WorkspaceManagementPage from "@/components/organization/WorkspaceManagementPage";

export const metadata: Metadata = {
  title: "Workspace | Aurora Control Plane",
  description: "Manage organization workspaces",
};

export default function OrganizationWorkspacesRoutePage() {
  return <WorkspaceManagementPage />;
}
