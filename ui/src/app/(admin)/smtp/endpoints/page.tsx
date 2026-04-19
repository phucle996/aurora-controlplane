import type { Metadata } from "next";
import React from "react";
import { SMTPEndpointsPage } from "@/components/smtp/SMTPWorkspace";

export const metadata: Metadata = {
  title: "SMTP Endpoints | Aurora Control Plane",
  description: "SMTP endpoint inventory and provisioning UI",
};

export default function DeliveryEndpointsRoutePage() {
  return <SMTPEndpointsPage />;
}
