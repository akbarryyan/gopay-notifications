/**
 * Ekspor CSV sisi klien -- tidak ada endpoint backend khusus. Accounts
 * sudah diambil seutuhnya dalam satu array (lihat komentar di
 * accounts/page.tsx), jadi tinggal diserialisasi. Transactions memanggil
 * ulang endpoint dengan limit besar (lihat "Export CSV" di halaman itu)
 * supaya ekspor mengikuti filter yang sedang aktif, bukan cuma satu
 * halaman yang sedang tampil di layar.
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
  // "﻿" (BOM) supaya Excel yang buka file ini langsung mengenali UTF-8
  // -- tanpa ini karakter non-ASCII (nama bisnis dsb) bisa tampil rusak.
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
