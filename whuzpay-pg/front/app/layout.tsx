import type { Metadata } from "next";
import { Inter, Geist } from "next/font/google";
import ToasterProvider from "./components/providers/ToasterProvider";
import "./globals.css";
import { cn } from "@/lib/utils";

const geist = Geist({subsets:['latin'],variable:'--font-sans'});

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
});

export const metadata: Metadata = {
  title: "WhuzPay - Solusi Pembayaran untuk Pertumbuhan Bisnis Anda",
  description:
    "Terima pembayaran online dan offline melalui satu infrastruktur yang aman dan terpercaya.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={cn("h-full", "antialiased", inter.variable, "font-sans", geist.variable)}>
      <body className="min-h-full flex flex-col font-sans">
        {children}
        <ToasterProvider />
      </body>
    </html>
  );
}
