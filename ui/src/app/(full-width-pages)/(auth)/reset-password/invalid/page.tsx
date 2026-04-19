import InvalidResetTokenView from "@/components/auth/InvalidResetTokenView";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Invalid Reset Link | Aurora Control Plane",
  description: "Your reset token is invalid or expired.",
};

export default function InvalidResetTokenPage() {
  return <InvalidResetTokenView />;
}
