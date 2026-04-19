import { Suspense } from "react";
import { Metadata } from "next";
import VirtualMachineDetailPage from "@/components/virtual-machines/VirtualMachineDetailPage";

export const metadata: Metadata = {
  title: "Virtual Machine Detail | Aurora Control Plane",
  description: "Virtual machine detail view",
};

export default function VirtualMachineDetailRoutePage() {
  return (
    <Suspense fallback={null}>
      <VirtualMachineDetailPage />
    </Suspense>
  );
}
