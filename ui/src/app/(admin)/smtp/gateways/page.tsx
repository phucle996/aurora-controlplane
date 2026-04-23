import type { Metadata } from "next";
import React, { Suspense } from "react";
import { SMTPGatewaysPage } from "@/components/smtp/SMTPWorkspace";

export const metadata: Metadata = {
  title: "SMTP Gateways | Aurora Control Plane",
  description: "SMTP gateway routing",
};

export default function SMTPGatewayRoutePage() {
  return (
    <Suspense fallback={null}>
      <SMTPGatewaysPage />
    </Suspense>
  );
}
