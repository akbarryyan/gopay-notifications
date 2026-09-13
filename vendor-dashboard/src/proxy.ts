import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * Gerbang navigasi, bukan gerbang keamanan — sama persis pola dashboard
 * customer (lihat dashboard/src/proxy.ts). Nama cookie beda ("vendor_session"
 * vs "admin_session") supaya kedua dashboard tidak pernah saling
 * bertabrakan kalau kebetulan dibuka di browser yang sama.
 */
export function proxy(request: NextRequest) {
  const hasSession = request.cookies.has("vendor_session");
  const { pathname } = request.nextUrl;

  if (pathname === "/login") {
    if (hasSession) {
      return NextResponse.redirect(new URL("/", request.url));
    }
    return NextResponse.next();
  }

  if (!hasSession) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"],
};
