import type { Metadata } from "next";
import React, { Suspense } from "react";
import { SMTPConsumersPage } from "@/components/smtp/SMTPWorkspace";

export const metadata: Metadata = {
  title: "SMTP Consumers | Aurora Control Plane",
  description: "SMTP consumer inventory",
};

export default function SMTPConsumersRoutePage() {
  return (
    <Suspense fallback={null}>
      <SMTPConsumersPage />
    </Suspense>
  );
}
