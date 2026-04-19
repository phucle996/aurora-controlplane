import NewPasswordForm from "@/components/auth/NewPasswordForm";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Set New Password | Aurora Control Plane",
  description: "Create a new password for your Aurora Control Plane account.",
};

export default function NewPasswordPage() {
  return <NewPasswordForm />;
}
