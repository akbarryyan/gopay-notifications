/**
 * Ekspor CSV sisi klien -- tidak ada endpoint CSV khusus di backend.
 * Daftar yang sudah diambil seutuhnya (mis. Devices) tinggal diserialisasi;
 * yang berpaginasi di server (Transactions, Events) diambil ulang lewat
 * fetchAllPages() supaya ekspor mengikuti filter yang aktif, bukan cuma
 * satu halaman yang sedang tampil di layar.
 */

export interface CsvColumn<T> {
  label: string;
  value: (row: T) => string | number;
}

function escapeCsvCell(value: string | number): string {
  const s = String(value);
  if (s.includes(",") || s.includes('"') || s.includes("\n")) {
    return `"${s.replace(/"/g, '""')}"`;
  }
  return s;
}

export function toCsv<T>(rows: T[], columns: CsvColumn<T>[]): string {
  const header = columns.map((c) => escapeCsvCell(c.label)).join(",");
  const lines = rows.map((row) => columns.map((c) => escapeCsvCell(c.value(row))).join(","));
  return [header, ...lines].join("\r\n");
}

export function downloadCsv(filename: string, csv: string) {
  // "﻿" (BOM) supaya Excel yang membuka file ini langsung mengenali
  // UTF-8 -- tanpa ini karakter non-ASCII (nama bisnis dsb) tampil rusak.
  const blob = new Blob(["﻿" + csv], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/**
 * Endpoint daftar di backend membatasi `limit` maksimal 1000 per
 * permintaan, jadi ekspor mengambil beberapa halaman berturut-turut alih-
 * alih satu permintaan raksasa -- menaikkan batas di server cuma
 * memindahkan masalahnya ke query yang lebih berat.
 */
export const EXPORT_PAGE_SIZE = 1000;

/**
 * Pagar pengaman supaya tombol ekspor tidak pernah menggantung browser
 * kalau datanya sudah sangat banyak. Pemanggil memberi tahu pengguna kalau
 * hasilnya terpotong (lihat `truncated`).
 */
export const EXPORT_MAX_PAGES = 10;

export async function fetchAllPages<T>(
  fetchPage: (limit: number, offset: number) => Promise<T[]>,
): Promise<{ rows: T[]; truncated: boolean }> {
  const rows: T[] = [];
  let truncated = false;
  for (let page = 0; page < EXPORT_MAX_PAGES; page++) {
    const batch = await fetchPage(EXPORT_PAGE_SIZE, page * EXPORT_PAGE_SIZE);
    rows.push(...batch);
    if (batch.length < EXPORT_PAGE_SIZE) break;
    if (page === EXPORT_MAX_PAGES - 1) truncated = true;
  }
  return { rows, truncated };
}
