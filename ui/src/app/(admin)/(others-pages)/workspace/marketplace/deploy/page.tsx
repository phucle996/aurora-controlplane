import { Suspense } from "react";
import { Metadata } from "next";
import MarketplaceDeployPage from "@/components/marketplace/MarketplaceDeployPage";

export const metadata: Metadata = {
  title: "Deploy Marketplace App | Aurora Control Plane",
  description: "Deploy a marketplace application into a workspace namespace",
};

export default function WorkspaceMarketplaceDeployRoutePage() {
  return (
    <Suspense fallback={null}>
      <MarketplaceDeployPage />
    </Suspense>
  );
}
