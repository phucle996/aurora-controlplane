import ConfirmForgotPasswordLink from "@/components/auth/ConfirmForgotPasswordLink";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Confirm Reset Link | Aurora Control Plane",
  description: "Validate your forgot-password reset link.",
};

export default function ConfirmResetPasswordPage() {
  return <ConfirmForgotPasswordLink />;
}
