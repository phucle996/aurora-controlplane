import type { Metadata } from "next";
import React from "react";
import SMTPGeneralPage from "@/components/smtp/SMTPWorkspace";

export const metadata: Metadata = {
  title: "SMTP | Aurora Control Plane",
  description: "SMTP overview",
};

export default function SMTPPage() {
  return <SMTPGeneralPage />;
}
