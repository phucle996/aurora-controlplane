import { Metadata } from "next";
import WorkspacePage from "@/components/workspace/WorkspacePage";

export const metadata: Metadata = {
  title: "My Workspace | Aurora Control Plane",
  description: "Unified inventory for namespaced and global user resources",
};

export default function WorkspaceRoutePage() {
  return <WorkspacePage />;
}
