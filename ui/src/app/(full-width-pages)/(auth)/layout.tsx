import GridShape from "@/components/common/GridShape";
import ThemeTogglerTwo from "@/components/common/ThemeTogglerTwo";

import { ThemeProvider } from "@/context/ThemeContext";
import Image from "next/image";
import Link from "next/link";
import React from "react";

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="relative z-1 overflow-x-hidden bg-white p-6 dark:bg-gray-900 sm:p-0">
      <ThemeProvider>
        <div className="relative flex min-h-screen w-full flex-col dark:bg-gray-900 sm:p-0 lg:flex-row">
          {children}
          <div className="hidden min-h-screen w-full overflow-hidden bg-[radial-gradient(circle_at_top,#465fff_0%,#1f245a_46%,#0f132f_100%)] lg:grid lg:w-1/2 lg:items-center">
            <div className="relative z-1 flex items-center justify-center">
              <GridShape />
              <div className="auth-orb absolute -left-20 top-16 h-64 w-64 rounded-full bg-white/10 blur-3xl" />
              <div className="auth-orb absolute bottom-12 right-10 h-56 w-56 rounded-full bg-brand-300/20 blur-3xl" />
              <div className="relative flex max-w-sm flex-col items-center px-8 text-center">
                <Link href="/" className="block mb-4">
                  <Image
                    width={231}
                    height={48}
                    src="./images/logo/auth-logo.svg"
                    alt="Logo"
                  />
                </Link>
                <span className="mb-4 rounded-full border border-white/15 bg-white/10 px-4 py-1 text-xs font-semibold uppercase tracking-[0.3em] text-white/70">
                  Aurora Control Plane
                </span>
                <h2 className="mb-4 text-4xl font-semibold text-white">
                  Secure access for your storage platform.
                </h2>
                <p className="text-center text-base leading-7 text-white/70">
                  Manage IAM, verify accounts, and operate infrastructure from a
                  single control plane with production-ready guardrails.
                </p>
              </div>
            </div>
          </div>
          <div className="fixed bottom-6 right-6 z-50 hidden sm:block">
            <ThemeTogglerTwo />
          </div>
        </div>
      </ThemeProvider>
    </div>
  );
}
