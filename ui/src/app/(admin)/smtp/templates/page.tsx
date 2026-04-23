import type { Metadata } from "next";
import React, { Suspense } from "react";
import { SMTPTemplatesPage } from "@/components/smtp/SMTPWorkspace";

export const metadata: Metadata = {
  title: "SMTP Templates | Aurora Control Plane",
  description: "SMTP template library",
};

export default function SMTPTemplatesRoutePage() {
  return (
    <Suspense fallback={null}>
      <SMTPTemplatesPage />
    </Suspense>
  );
}
