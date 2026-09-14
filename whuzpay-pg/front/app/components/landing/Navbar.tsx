"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import Container from "./Container";
import { focusRingDark } from "./styles";

const navItems = [
  { label: "Beranda", href: "#beranda" },
  { label: "Produk", href: "#produk" },
  { label: "Solusi", href: "#solusi" },
  { label: "Integrasi", href: "#integrasi" },
  { label: "Bantuan", href: "#faq" },
];

const sectionIds = navItems.map((item) => item.href.slice(1));

function UserIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-4 w-4">
      <circle cx="12" cy="8" r="3.5" />
      <path d="M4.5 20c1.4-3.4 4.4-5.5 7.5-5.5s6.1 2.1 7.5 5.5" strokeLinecap="round" />
    </svg>
  );
}

export default function Navbar() {
  const [mobileOpen, setMobileOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const [activeId, setActiveId] = useState(sectionIds[0]);

  useEffect(() => {
    function onScroll() {
      setScrolled(window.scrollY > 24);
    }
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  useEffect(() => {
    const sections = sectionIds
      .map((id) => document.getElementById(id))
      .filter((el): el is HTMLElement => el !== null);
    if (sections.length === 0) return;

    // The band between 35% and 45% down the viewport decides which section
    // is "current" — a single line rather than "whatever is on screen", so
    // the highlight never flickers between two sections at once.
    const observer = new IntersectionObserver(
      (entries) => {
        const entering = entries
          .filter((entry) => entry.isIntersecting)
          .sort(
            (a, b) => a.boundingClientRect.top - b.boundingClientRect.top,
          )[0];
        if (entering) setActiveId(entering.target.id);
      },
      { rootMargin: "-35% 0px -55% 0px", threshold: 0 },
    );

    sections.forEach((section) => observer.observe(section));
    return () => observer.disconnect();
  }, []);

  return (
    <header
      className={`fixed inset-x-0 top-0 z-50 transition-colors duration-300 ${
        scrolled || mobileOpen
          ? "bg-brand-navy-dark shadow-lg"
          : "bg-transparent"
      }`}
    >
      <Container className="flex h-20 items-center justify-between">
        <Link
          href="/"
          className={`rounded-md text-2xl font-bold lowercase italic tracking-tight text-white ${focusRingDark}`}
        >
          whuzpay
        </Link>

        <nav className="hidden items-center gap-8 lg:flex">
          {navItems.map((item) => (
            <a
              key={item.label}
              href={item.href}
              aria-current={activeId === item.href.slice(1) ? "true" : undefined}
              className={`relative rounded-md pb-1 text-sm font-medium transition-colors ${focusRingDark} ${
                activeId === item.href.slice(1)
                  ? "text-brand-yellow"
                  : "text-white/80 hover:text-white"
              }`}
            >
              {item.label}
              <span
                className={`absolute -bottom-1 left-0 h-0.5 bg-brand-yellow transition-all duration-300 ${
                  activeId === item.href.slice(1) ? "w-full" : "w-0"
                }`}
              />
            </a>
          ))}
        </nav>

        <div className="hidden items-center gap-5 lg:flex">
          <Link
            href="/login"
            className={`flex items-center gap-2 rounded-md text-sm font-medium text-white/90 hover:text-white ${focusRingDark}`}
          >
            Login
            <span className="flex h-7 w-7 items-center justify-center rounded-full bg-white/15">
              <UserIcon />
            </span>
          </Link>
          <Link
            href="/register"
            className={`rounded-lg bg-brand-yellow px-4 py-2 text-sm font-bold text-brand-navy-dark transition-colors hover:bg-brand-yellow-dark ${focusRingDark}`}
          >
            Coba Sandbox
          </Link>
        </div>

        <button
          onClick={() => setMobileOpen((v) => !v)}
          className={`flex h-10 w-10 items-center justify-center rounded-md text-white lg:hidden ${focusRingDark}`}
          aria-label={mobileOpen ? "Tutup menu" : "Buka menu"}
          aria-expanded={mobileOpen}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} className="h-6 w-6">
            {mobileOpen ? (
              <path d="M6 6l12 12M6 18L18 6" strokeLinecap="round" />
            ) : (
              <path d="M4 7h16M4 12h16M4 17h16" strokeLinecap="round" />
            )}
          </svg>
        </button>
      </Container>

      {mobileOpen && (
        <div className="border-t border-white/10 bg-brand-navy-dark px-6 pb-6 pt-4 lg:hidden">
          <nav className="flex flex-col">
            {navItems.map((item) => (
              <a
                key={item.label}
                href={item.href}
                onClick={() => setMobileOpen(false)}
                className={`rounded-md py-3 text-base font-medium ${focusRingDark} ${
                  activeId === item.href.slice(1)
                    ? "text-brand-yellow"
                    : "text-white/80"
                }`}
              >
                {item.label}
              </a>
            ))}
            <Link
              href="/login"
              onClick={() => setMobileOpen(false)}
              className={`rounded-md py-3 text-base font-medium text-white/80 ${focusRingDark}`}
            >
              Login
            </Link>
            <Link
              href="/register"
              onClick={() => setMobileOpen(false)}
              className={`mt-3 rounded-lg bg-brand-yellow px-4 py-3 text-center text-sm font-bold uppercase tracking-wide text-brand-navy-dark ${focusRingDark}`}
            >
              Coba Sandbox
            </Link>
          </nav>
        </div>
      )}
    </header>
  );
}
