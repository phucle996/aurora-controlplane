import type { Metadata } from "next";
import React from "react";
import { SMTPLanesPage } from "@/components/smtp/SMTPWorkspace";

export const metadata: Metadata = {
  title: "SMTP Lane | Aurora Control Plane",
  description: "SMTP lane routing",
};

export default function SMTPLaneRoutePage() {
  return <SMTPLanesPage />;
}
