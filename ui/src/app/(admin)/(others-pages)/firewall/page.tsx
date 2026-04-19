import { Metadata } from "next";
import FirewallPage from "@/components/firewall/FirewallPage";

export const metadata: Metadata = {
  title: "Firewalls | Aurora Control Plane",
  description: "Firewall inventory for computing resources",
};

export default function FirewallRoutePage() {
  return <FirewallPage />;
}
