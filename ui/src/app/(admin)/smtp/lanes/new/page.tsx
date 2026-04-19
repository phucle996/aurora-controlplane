import type { Metadata } from "next";
import React from "react";
import NewLaneForm from "@/components/smtp/NewLaneForm";

export const metadata: Metadata = {
  title: "New Lane | SMTP | Aurora Control Plane",
  description: "Create a new SMTP delivery lane",
};

export default function NewSMTPLanePage() {
  return <NewLaneForm />;
}
