import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * Gerbang navigasi, bukan gerbang keamanan.
 *
 * Ini hanya memeriksa APAKAH cookie sesi ada, bukan apakah ia masih valid —
 * itu tidak bisa diverifikasi di sini tanpa memanggil backend pada setiap
 * navigasi. Backend tetap satu-satunya pihak yang memutuskan sah tidaknya
 * sesi lewat requireAdmin pada setiap panggilan API; ini murni mencegah
 * kedipan halaman kosong sebelum redirect ke /login.
 */
export function proxy(request: NextRequest) {
  const hasSession = request.cookies.has("admin_session");
  const { pathname } = request.nextUrl;

  if (pathname === "/login") {
    if (hasSession) {
      return NextResponse.redirect(new URL("/", request.url));
    }
    return NextResponse.next();
  }

  if (!hasSession) {
    const loginUrl = new URL("/login", request.url);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    // Semua path kecuali aset statis Next.js, favicon, dan API (backend
    // sendiri yang menegakkan auth untuk /api/*).
    "/((?!api|_next/static|_next/image|favicon.ico).*)",
  ],
};
