import type { SMTPTab } from "@/components/smtp/types";

export const tabs: SMTPTab[] = [
  {
    key: "general",
    label: "General",
    caption: "Workspace overview and runtime health",
    href: "/smtp",
  },
  {
    key: "consumers",
    label: "Consumers",
    caption: "Broker ingress and worker intake",
    href: "/smtp/consumers",
  },
  {
    key: "templates",
    label: "Templates",
    caption: "Rendered mail content and policies",
    href: "/smtp/templates",
  },
  {
    key: "gateways",
    label: "Gateways",
    caption: "Routing, shard state, and endpoint bindings",
    href: "/smtp/gateways",
  },
  {
    key: "endpoints",
    label: "Endpoints",
    caption: "SMTP relays and delivery capacity",
    href: "/smtp/endpoints",
  },
];
