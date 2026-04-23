import type { Metadata } from "next";
import React, { Suspense } from "react";
import NewTemplateForm from "@/components/smtp/NewTemplateForm";

export const metadata: Metadata = {
  title: "New Template | SMTP | Aurora Control Plane",
  description: "SMTP template creation UI preview",
};

export default function NewSMTPTemplatePage() {
  return (
    <Suspense fallback={null}>
      <NewTemplateForm />
    </Suspense>
  );
}
