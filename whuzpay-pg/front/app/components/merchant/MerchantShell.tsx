"use client";

import { useState } from "react";
import MerchantHeader from "./MerchantHeader";
import MerchantSidebar from "./MerchantSidebar";

export default function MerchantShell({
  children,
}: {
  children: React.ReactNode;
}) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  return (
    <div className="admin-root flex min-h-screen bg-[#eff4f8] text-[#1f2a37]">
      <MerchantSidebar collapsed={sidebarCollapsed} />
      <div className="flex min-w-0 flex-1 flex-col">
        <MerchantHeader
          sidebarCollapsed={sidebarCollapsed}
          onMenuClick={() => setSidebarCollapsed((v) => !v)}
        />
        <main className="flex-1 p-4 sm:p-5 lg:p-6">{children}</main>
      </div>
    </div>
  );
}
