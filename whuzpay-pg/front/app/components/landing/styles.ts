/**
 * Shared class strings for the landing page.
 *
 * These exist because the previous version re-declared the same heading,
 * button, and card patterns in eight components with slightly different
 * sizes each time — that drift is what made the page read as inconsistent.
 * Import from here instead of retyping the classes.
 */

/** Focus indicator for controls on a light background. */
export const focusRing =
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-navy focus-visible:ring-offset-2 focus-visible:ring-offset-white";

/** Focus indicator for controls sitting on navy. */
export const focusRingDark =
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-yellow focus-visible:ring-offset-2 focus-visible:ring-offset-brand-navy";

/** Card / sub-section heading. Section titles come from <Section title>. */
export const cardHeading =
  "text-xl font-bold leading-snug tracking-tight text-brand-navy";

/** Default body copy on a light background. */
export const bodyText = "text-base leading-relaxed text-slate-600 sm:text-lg";

const buttonBase =
  "inline-flex items-center justify-center gap-2 rounded-lg px-6 py-3 text-sm font-bold uppercase tracking-wide transition-colors";

/** Yellow call-to-action. Works on both light and navy backgrounds. */
export const btnPrimary = `${buttonBase} bg-brand-yellow text-brand-navy-dark shadow-sm hover:bg-brand-yellow-dark`;

/** Outlined secondary action on a navy background. */
export const btnOnNavy = `${buttonBase} border border-white/40 text-white hover:bg-white/10`;

/** Outlined secondary action on a light background. */
export const btnOnLight = `${buttonBase} border border-slate-200 text-slate-700 hover:bg-slate-50`;
