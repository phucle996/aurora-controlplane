import { Metadata } from "next";
import OrganizationPage from "@/components/organization/OrganizationPage";

export const metadata: Metadata = {
  title: "Organization | Aurora Control Plane",
  description: "Create and manage organizations",
};

export default function OrganizationRoutePage() {
  return <OrganizationPage />;
}
