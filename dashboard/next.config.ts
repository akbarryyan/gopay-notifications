import type { NextConfig } from "next";

// Alamat backend Go, hanya dipakai untuk proxy dev di mesin ini. Di produksi,
// Caddy yang merutekan /api/* langsung ke backend dan sisanya ke Next.js —
// rewrite di bawah ini tidak pernah tersentuh saat berjalan di belakang Caddy.
const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8090";

const nextConfig: NextConfig = {
  // Menghasilkan .next/standalone: server Node minimal + subset node_modules
  // yang benar-benar dipakai, tanpa perlu `npm install` di VPS sama sekali.
  // Krusial untuk model self-hosted ini — tiap customer akan melakukan
  // deploy yang sama, jadi makin sederhana langkahnya makin baik.
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
