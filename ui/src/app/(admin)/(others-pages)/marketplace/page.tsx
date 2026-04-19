import { Metadata } from "next";
import MarketplacePage from "@/components/marketplace/MarketplacePage";

export const metadata: Metadata = {
  title: "Marketplace | Aurora Control Plane",
  description: "Deploy platform applications on Kubernetes through the resource platform",
};

export default function MarketplaceRoutePage() {
  return <MarketplacePage />;
}
