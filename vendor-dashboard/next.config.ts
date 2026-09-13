import type { NextConfig } from "next";

// Backend utama (Go) -- License Server yang dulu terpisah sudah dibongkar,
// sekarang endpoint vendor (/api/v1/vendor/*) hidup di binary yang sama
// dengan backend customer. Dev: Next.js me-rewrite ke sini. Produksi: Caddy
// yang merutekan /api/* langsung ke backend dan sisanya ke Next.js ini.
const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8090";

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${BACKEND_URL}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
