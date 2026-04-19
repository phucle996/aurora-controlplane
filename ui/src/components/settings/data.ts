export type SettingsTab = {
  key: string;
  label: string;
  caption: string;
  href: string;
};

export const tabs: SettingsTab[] = [
  {
    key: "security",
    label: "Two-Factor Auth",
    caption: "Protect your account with authenticator-based verification.",
    href: "/settings",
  },
];
