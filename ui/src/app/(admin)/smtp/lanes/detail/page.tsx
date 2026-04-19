import type { Metadata } from "next";
import React, { Suspense } from "react";
import LaneDetailPage from "@/components/smtp/LaneDetailPage";

export const metadata: Metadata = {
  title: "Lane Detail | SMTP | Aurora Control Plane",
  description: "Inspect a delivery lane",
};

export default function SMTPLaneDetailRoutePage() {
  return (
    <Suspense fallback={null}>
      <LaneDetailPage />
    </Suspense>
  );
}
