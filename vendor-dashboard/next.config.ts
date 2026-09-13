import type { NextConfig } from "next";

// Alamat License Server (Go), hanya dipakai untuk proxy dev di mesin ini.
// Di produksi, Caddy yang merutekan /api/* langsung ke License Server dan
// sisanya ke Next.js ini — sama persis pola dashboard customer.
const LICENSE_SERVER_URL = process.env.LICENSE_SERVER_URL ?? "http://localhost:8095";

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${LICENSE_SERVER_URL}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
