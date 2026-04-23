import type { Metadata } from "next";
import React, { Suspense } from "react";
import NewGatewayForm from "@/components/smtp/NewGatewayForm";

export const metadata: Metadata = {
  title: "New Gateway | SMTP | Aurora Control Plane",
  description: "Create a new SMTP gateway",
};

export default function NewSMTPGatewayPage() {
  return (
    <Suspense fallback={null}>
      <NewGatewayForm />
    </Suspense>
  );
}
