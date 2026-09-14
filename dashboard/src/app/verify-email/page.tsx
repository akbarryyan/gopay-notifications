"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { CheckCircle2, TriangleAlert } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";
import { AuthSidePanel } from "@/components/auth/side-panel";
import { verifyEmail } from "@/lib/api";
import { cn } from "@/lib/utils";

type Phase = "loading" | "done" | "invalid" | "missing";

/**
 * Token dari query string (?token=...), berbeda dari /reset-password yang
 * memakai fragment (#token=...). Verifikasi email BUKAN kunci akun --
 * dampak terburuk token ini bocor cuma menandai email terverifikasi lebih
 * cepat, jadi aman muncul di log akses.
 */
function VerifyEmailInner() {
  const params = useSearchParams();
  const [phase, setPhase] = useState<Phase>("loading");
  const [businessName, setBusinessName] = useState("");

  useEffect(() => {
    const token = params.get("token");
    if (!token) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setPhase("missing");
      return;
    }
    verifyEmail(token)
      .then((res) => {
        setBusinessName(res.business_name);
        setPhase("done");
      })
      .catch(() => {
        // Token tidak dikenal, kedaluwarsa, atau sudah dipakai -- backend
        // sengaja tidak membedakan ketiganya (invalid_token).
        setPhase("invalid");
      });
  }, [params]);

  return (
    <div className="flex min-h-screen">
      <AuthSidePanel
        eyebrow="Payment Bridge"
        title="Satu langkah lagi: pastikan email kamu benar."
        subtitle="Email yang terverifikasi memastikan pengingat dan link reset password sampai ke tempat yang tepat."
      />

      <div className="flex w-full flex-col items-center justify-center px-4 py-10 lg:w-1/2">
        <div className="w-full max-w-sm text-center">
          {phase === "loading" ? (
            <p className="text-sm text-slate-500">Memverifikasi email...</p>
          ) : phase === "done" ? (
            <>
              <span className="mb-4 inline-flex size-11 items-center justify-center rounded-xl bg-teal-50 text-teal-700 ring-1 ring-teal-600/20">
                <CheckCircle2 className="size-5" />
              </span>
              <h1 className="text-xl font-semibold text-slate-900">Email terverifikasi</h1>
              <p className="mt-2 text-sm text-slate-500">
                Email{" "}
                {businessName ? (
                  <span className="font-medium text-slate-900">({businessName})</span>
                ) : null}{" "}
                sudah dikonfirmasi. Pengingat dan notifikasi akan sampai ke alamat ini.
              </p>
              <Link
                href="/overview"
                className={cn(buttonVariants(), "mt-6 h-11 w-full rounded-lg")}
              >
                Buka dashboard
              </Link>
            </>
          ) : (
            <>
              <span className="mb-4 inline-flex size-11 items-center justify-center rounded-xl bg-amber-50 text-amber-700 ring-1 ring-amber-600/20">
                <TriangleAlert className="size-5" />
              </span>
              <h1 className="text-xl font-semibold text-slate-900">
                {phase === "missing" ? "Link tidak lengkap" : "Link tidak berlaku"}
              </h1>
              <p className="mt-2 text-sm text-slate-500">
                {phase === "missing"
                  ? "Buka link langsung dari email verifikasi, pastikan tersalin utuh."
                  : "Link ini sudah kedaluwarsa atau sudah pernah dipakai. Minta email verifikasi baru dari Settings."}
              </p>
              <Link
                href="/login"
                className={cn(
                  buttonVariants({ variant: "outline" }),
                  "mt-6 h-11 w-full rounded-lg",
                )}
              >
                Kembali ke halaman masuk
              </Link>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

// useSearchParams butuh Suspense boundary saat prerender (Next.js App
// Router) -- pola yang sama sudah dipakai di halaman lain yang membaca
// query string.
export default function VerifyEmailPage() {
  return (
    <Suspense fallback={null}>
      <VerifyEmailInner />
    </Suspense>
  );
}
