import VerifyEmailLink from "@/components/auth/VerifyEmailLink";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Verify Email | Aurora Control Plane",
  description: "Confirm your account verification link.",
};

export default function VerifyEmailPage() {
  return <VerifyEmailLink />;
}
