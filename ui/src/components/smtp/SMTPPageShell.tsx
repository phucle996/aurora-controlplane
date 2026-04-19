"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import PageBreadcrumb from "@/components/common/PageBreadCrumb";
import { tabs } from "@/components/smtp/data";

export function SMTPPageShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="space-y-6">
      <PageBreadcrumb pageTitle="SMTP" />
      <section className="border-b border-gray-200 dark:border-gray-800">
        <div className="flex flex-col gap-2 md:flex-row md:items-end md:gap-8">
          {tabs.map((tab) => {
            const active =
              tab.href === "/smtp"
                ? pathname === "/smtp"
                : pathname.startsWith(tab.href);

            return (
              <Link
                key={tab.key}
                href={tab.href}
                className={`group relative px-1 pb-4 text-left transition ${
                  active
                    ? "text-gray-900 dark:text-white"
                    : "text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-white/80"
                }`}
              >
                <span className="block text-sm font-semibold tracking-tight">
                  {tab.label}
                </span>
                <span className="mt-1 block text-xs text-gray-400 dark:text-gray-500">
                  {tab.caption}
                </span>
                <span
                  className={`absolute bottom-0 left-0 h-0.5 rounded-full bg-gray-900 transition-all dark:bg-white ${
                    active ? "w-full" : "w-0 group-hover:w-8"
                  }`}
                />
              </Link>
            );
          })}
        </div>
      </section>
      {children}
    </div>
  );
}
