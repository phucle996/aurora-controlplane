import type { Metadata } from "next";
import { redirect } from "next/navigation";

export const metadata: Metadata = {
  title: "Gateway Detail | SMTP | Aurora Control Plane",
  description: "Inspect a delivery gateway",
};

export default function SMTPLaneDetailRoutePage() {
  redirect("/smtp/gateways/detail");
}
