import type { Metadata } from "next";
import { Work_Sans, Geist_Mono, Bricolage_Grotesque, Plus_Jakarta_Sans } from "next/font/google";
import { Toaster } from "@/components/ui/toaster";
import "./globals.css";

const workSans = Work_Sans({
  variable: "--font-work-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

// Dipakai HANYA di halaman publik (landing "/", "/register") lewat utility
// arbitrary `font-(--font-lp-heading)`/`font-(--font-lp-body)` --
// sengaja tidak menyentuh --font-sans/--font-heading global di globals.css,
// supaya dashboard yang sudah login tetap memakai Work Sans seperti biasa.
const bricolageGrotesque = Bricolage_Grotesque({
  variable: "--font-lp-heading",
  subsets: ["latin"],
});

const plusJakartaSans = Plus_Jakarta_Sans({
  variable: "--font-lp-body",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Payment Bridge Dashboard",
  description: "Monitoring dan administrasi Payment Notification Bridge.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="id"
      className={`${workSans.variable} ${geistMono.variable} ${bricolageGrotesque.variable} ${plusJakartaSans.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col bg-background text-foreground">
        {children}
        <Toaster />
      </body>
    </html>
  );
}
