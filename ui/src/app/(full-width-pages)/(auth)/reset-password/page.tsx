import ForgotPasswordForm from "@/components/auth/ForgotPasswordForm";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Forgot Password | Aurora Control Plane",
  description: "Request reset instructions for your Aurora Control Plane account.",
};

export default function ResetPasswordPage() {
  return <ForgotPasswordForm />;
}
