import type { Metadata } from "next";
import { redirect } from "next/navigation";

export const metadata: Metadata = {
  title: "New Gateway | SMTP | Aurora Control Plane",
  description: "Create a new SMTP gateway",
};

export default function NewSMTPLanePage() {
  redirect("/smtp/gateways/new");
}
