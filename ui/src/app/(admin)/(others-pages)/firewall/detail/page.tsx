import { Suspense } from "react";
import { Metadata } from "next";
import FirewallDetailPage from "@/components/firewall/FirewallDetailPage";

export const metadata: Metadata = {
  title: "Firewall Detail | Aurora Control Plane",
  description: "Firewall detail view with inbound and outbound rules",
};

export default function FirewallDetailRoutePage() {
  return (
    <Suspense fallback={null}>
      <FirewallDetailPage />
    </Suspense>
  );
}
