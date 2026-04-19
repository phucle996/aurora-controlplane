import { Metadata } from "next";
import NewWorkspaceNamespacePage from "@/components/workspace/NewWorkspaceNamespacePage";

export const metadata: Metadata = {
  title: "Add Namespace | Aurora Control Plane",
  description: "Create a workspace namespace",
};

export default function NewWorkspaceNamespaceRoutePage() {
  return <NewWorkspaceNamespacePage />;
}
