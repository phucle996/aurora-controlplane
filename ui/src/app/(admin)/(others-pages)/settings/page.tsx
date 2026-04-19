import { Metadata } from "next";
import { SettingsPageShell } from "@/components/settings/SettingsPageShell";
import { TwoFactorTab } from "@/components/settings/TwoFactorTab";

export const metadata: Metadata = {
  title: "Settings | Aurora Control Plane",
  description: "Account security settings",
};

export default function SettingsPage() {
  return (
    <SettingsPageShell>
      <TwoFactorTab />
    </SettingsPageShell>
  );
}
