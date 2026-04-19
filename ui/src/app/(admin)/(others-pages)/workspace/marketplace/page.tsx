import { Metadata } from "next";
import MarketplacePage from "@/components/marketplace/MarketplacePage";

export const metadata: Metadata = {
  title: "Workspace Marketplace | Aurora Control Plane",
  description: "One-click application deployment catalog for workspace namespaces",
};

export default function WorkspaceMarketplaceRoutePage() {
  return <MarketplacePage />;
}
