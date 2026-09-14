import type { NextConfig } from "next";

// No remotePatterns: every image on the site is now local markup or SVG.
// Adding a host here re-opens the app to third-party image requests, so add
// one only when a real asset needs it.
const nextConfig: NextConfig = {};

export default nextConfig;
