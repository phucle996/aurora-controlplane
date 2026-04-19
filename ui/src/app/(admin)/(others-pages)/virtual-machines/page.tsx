import { Metadata } from "next";
import VirtualMachinesPage from "@/components/virtual-machines/VirtualMachinesPage";

export const metadata: Metadata = {
  title: "Virtual Machines | Aurora Control Plane",
  description: "Virtual machine inventory for the current user",
};

export default function VirtualMachinesRoutePage() {
  return <VirtualMachinesPage />;
}
