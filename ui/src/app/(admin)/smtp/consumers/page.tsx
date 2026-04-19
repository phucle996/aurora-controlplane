import type { Metadata } from "next";
import React from "react";
import { SMTPConsumersPage } from "@/components/smtp/SMTPWorkspace";

export const metadata: Metadata = {
  title: "SMTP Consumers | Aurora Control Plane",
  description: "SMTP consumer lanes",
};

export default function SMTPConsumersRoutePage() {
  return <SMTPConsumersPage />;
}
