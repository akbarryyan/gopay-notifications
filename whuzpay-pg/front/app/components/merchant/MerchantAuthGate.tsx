"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import {
  getMerchantToken,
  setMerchantAuthNotice,
  setMerchantRedirectPath,
} from "@/lib/merchant-auth";

export default function MerchantAuthGate({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const token = getMerchantToken();
    if (!token) {
      setMerchantRedirectPath(pathname || "/dashboard");
      setMerchantAuthNotice("required");
      router.replace("/login");
      return;
    }
    // Reads localStorage, unavailable during SSR — `ready` must stay false
    // until this effect runs on the client, or server/client markup would
    // mismatch on first paint (see use-merchant-environment.ts for the
    // same reasoning).
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setReady(true);
  }, [router, pathname]);

  if (!ready) {
    return (
      <div className="admin-root flex min-h-screen items-center justify-center bg-[#eff4f8] text-[13px] text-[#8a97a8]">
        Checking session...
      </div>
    );
  }

  return <>{children}</>;
}
