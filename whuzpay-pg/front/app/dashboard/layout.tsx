import type { Metadata } from "next";
import MerchantAuthGate from "../components/merchant/MerchantAuthGate";
import MerchantShell from "../components/merchant/MerchantShell";

export const metadata: Metadata = {
  title: "Merchant Dashboard | Lector",
  description: "Merchant payment dashboard",
};

export default function MerchantDashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <MerchantAuthGate>
      <MerchantShell>{children}</MerchantShell>
    </MerchantAuthGate>
  );
}
