import type { Metadata } from "next";
import { redirect } from "next/navigation";

export const metadata: Metadata = {
  title: "SMTP Gateways | Aurora Control Plane",
  description: "SMTP gateway routing",
};

export default function SMTPLaneRoutePage() {
  redirect("/smtp/gateways");
}
