import type { Metadata } from "next";
import AdminAuthGate from "../../components/admin/AdminAuthGate";
import AdminShell from "../../components/admin/AdminShell";

export const metadata: Metadata = {
  title: "Admin Dashboard | Lector",
  description: "Admin dashboard overview",
};

export default function AdminPanelLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AdminAuthGate>
      <AdminShell>{children}</AdminShell>
    </AdminAuthGate>
  );
}
