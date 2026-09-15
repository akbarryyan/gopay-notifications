import type { NextConfig } from "next";

// No remotePatterns: every image on the site is now local markup or SVG.
// Adding a host here re-opens the app to third-party image requests, so add
// one only when a real asset needs it.
//
// output: "standalone" -- deploy produksi butuh server Node yang berdiri
// sendiri di .next/standalone (lengkap dengan node_modules yang dipakai),
// pola sama persis dengan dashboard/ dan vendor-dashboard/ (lihat
// whuzpay-pg/deploy/README.md §U2/U3 analog "Frontend"). Tanpa ini,
// systemd unit whuzpay-pg-frontend.service (ExecStart node server.js)
// tidak akan menemukan server.js sama sekali.
const nextConfig: NextConfig = {
  output: "standalone",
};

export default nextConfig;
