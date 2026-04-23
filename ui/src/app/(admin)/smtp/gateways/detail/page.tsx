import type { Metadata } from "next";
import React, { Suspense } from "react";
import GatewayDetailPage from "@/components/smtp/GatewayDetailPage";

export const metadata: Metadata = {
  title: "Gateway Detail | SMTP | Aurora Control Plane",
  description: "Inspect a delivery gateway",
};

export default function SMTPGatewayDetailRoutePage() {
  return (
    <Suspense fallback={null}>
      <GatewayDetailPage />
    </Suspense>
  );
}
