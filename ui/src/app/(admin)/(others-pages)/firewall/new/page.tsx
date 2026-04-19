import { Metadata } from "next";
import NewFirewallPage from "@/components/firewall/NewFirewallPage";

export const metadata: Metadata = {
  title: "Add Firewall | Aurora Control Plane",
  description: "Create a firewall with inbound and outbound rules",
};

export default function NewFirewallRoutePage() {
  return <NewFirewallPage />;
}
