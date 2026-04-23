import type { Metadata } from "next";
import React, { Suspense } from "react";
import NewConsumerForm from "@/components/smtp/NewConsumerForm";

export const metadata: Metadata = {
  title: "New Consumer | SMTP | Aurora Control Plane",
  description: "SMTP consumer creation UI preview",
};

export default function NewSMTPConsumerPage() {
  return (
    <Suspense fallback={null}>
      <NewConsumerForm />
    </Suspense>
  );
}
