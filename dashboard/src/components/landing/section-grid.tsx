/**
 * Garis grid tipis buat latar section putih -- dipakai di section yang
 * bg-nya polos putih supaya tidak terasa kosong, tanpa mengganggu
 * keterbacaan konten. Section pemanggil wajib `relative` dan konten
 * utamanya wajib punya class `relative` juga (posisi + urutan DOM yang
 * menentukan siapa di atas, bukan z-index eksplisit -- lihat hero.tsx
 * untuk pola yang sama).
 */
export function SectionGrid() {
  return (
    <div
      aria-hidden
      className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,rgb(15_23_42/0.05)_1px,transparent_1px),linear-gradient(to_bottom,rgb(15_23_42/0.05)_1px,transparent_1px)] bg-size-[40px_40px] mask-[linear-gradient(to_bottom,transparent,black_12%,black_88%,transparent)]"
    />
  );
}
