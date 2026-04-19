import { Metadata } from "next";
import NewVirtualMachinePage from "@/components/virtual-machines/NewVirtualMachinePage";

export const metadata: Metadata = {
  title: "Create Virtual Machine | Aurora Control Plane",
  description: "Provision a new virtual machine",
};

export default function NewVirtualMachineRoutePage() {
  return <NewVirtualMachinePage />;
}
