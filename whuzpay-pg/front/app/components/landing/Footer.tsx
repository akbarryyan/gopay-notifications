import Container from "./Container";
import { focusRingDark } from "./styles";

const socials = [
  {
    label: "Instagram",
    path: "M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7Zm0 5.8a2.3 2.3 0 1 1 0-4.6 2.3 2.3 0 0 1 0 4.6ZM16 6.2a.9.9 0 1 0 0 1.8.9.9 0 0 0 0-1.8Z M8 4h8a4 4 0 0 1 4 4v8a4 4 0 0 1-4 4H8a4 4 0 0 1-4-4V8a4 4 0 0 1 4-4Zm0 1.5A2.5 2.5 0 0 0 5.5 8v8A2.5 2.5 0 0 0 8 18.5h8a2.5 2.5 0 0 0 2.5-2.5V8A2.5 2.5 0 0 0 16 5.5H8Z",
  },
  {
    label: "Facebook",
    path: "M13.5 9H15V6.5h-1.5C11.6 6.5 10.5 7.6 10.5 9v1.5H9V13h1.5v6H13v-6h1.8l.2-2.5H13V9c0-.3.2-.5.5-.5Z",
  },
  {
    label: "LinkedIn",
    path: "M6.9 8.6H4.2V19H6.9V8.6ZM5.6 4.5a1.6 1.6 0 1 0 0 3.2 1.6 1.6 0 0 0 0-3.2ZM19.8 19h-2.7v-5.3c0-1.3 0-3-1.8-3s-2 1.4-2 2.9V19H10.6V8.6h2.6v1.4h.1c.4-.7 1.3-1.5 2.7-1.5 2.9 0 3.8 1.9 3.8 4.4V19Z",
  },
];

export default function Footer() {
  return (
    <footer className="bg-brand-navy-dark pb-8 pt-16 text-white/70">
      <Container className="grid gap-10 sm:grid-cols-2 lg:grid-cols-4">
        <div className="flex flex-col gap-8">
          <div>
            <p className="text-xs font-bold uppercase tracking-wide text-white">
              Produk Kami
            </p>
            <ul className="mt-3 flex flex-col gap-2 text-sm">
              <li>
                <a href="#" className={`rounded-sm hover:text-white ${focusRingDark}`}>
                  Payment Gateway
                </a>
              </li>
              <li>
                <a href="#" className={`rounded-sm hover:text-white ${focusRingDark}`}>
                  Disbursement
                </a>
              </li>
            </ul>
          </div>

          <div>
            <p className="text-xs font-bold uppercase tracking-wide text-white">
              Perusahaan
            </p>
            <ul className="mt-3 flex flex-col gap-2 text-sm">
              <li>
                <a href="#" className={`rounded-sm hover:text-white ${focusRingDark}`}>
                  Tentang Kami
                </a>
              </li>
              <li>
                <a href="#" className={`rounded-sm hover:text-white ${focusRingDark}`}>
                  Karir
                </a>
              </li>
              <li>
                <a href="#" className={`rounded-sm hover:text-white ${focusRingDark}`}>
                  Kebijakan Privasi
                </a>
              </li>
            </ul>
          </div>

          <div>
            <p className="text-xs font-bold uppercase tracking-wide text-white">
              Follow Us
            </p>
            <div className="mt-3 flex gap-2">
              {socials.map((s) => (
                <a
                  key={s.label}
                  href="#"
                  aria-label={s.label}
                  className={`flex h-8 w-8 items-center justify-center rounded-full bg-white/10 text-white transition-colors hover:bg-white/20 ${focusRingDark}`}
                >
                  <svg viewBox="0 0 24 24" fill="currentColor" className="h-4 w-4">
                    <path d={s.path} />
                  </svg>
                </a>
              ))}
            </div>
          </div>
        </div>

        <div>
          <p className="text-xs font-bold uppercase tracking-wide text-white">
            Sertifikat
          </p>
          <ul className="mt-3 flex flex-col items-start gap-2">
            {[
              "Bank Indonesia",
              "PCI DSS Compliant",
              "KOMINFO",
              "ISO 27001:2013",
            ].map((label) => (
              <li
                key={label}
                className="rounded-md border border-white/15 bg-white/5 px-3 py-1.5 text-xs font-semibold text-white/80"
              >
                {label}
              </li>
            ))}
          </ul>
        </div>

        <div>
          <p className="text-xs font-bold uppercase tracking-wide text-white">
            Kantor Kami
          </p>
          <p className="mt-3 max-w-55 text-sm leading-relaxed">
            Graha Haidaya Unit R S T Jl. Raya Perjuangan No. 12, RT 01/RW
            02, Kel. Jaka Setia, Bekasi 17530
          </p>
        </div>

        <div>
          <p className="text-xs font-bold uppercase tracking-wide text-white">
            Hubungi Kami
          </p>
          <p className="mt-3 text-sm">Tel: +62 21 508 906 42</p>
          <p className="text-sm">WhatsApp: +62 838 8998 1421</p>
          <p className="mt-2 text-sm">Email: support@whuzpay.co.id</p>
        </div>
      </Container>

      <Container className="mt-12 border-t border-white/10 pt-6">
        <p className="text-xs text-white/40">
          &copy; 2020 &ndash; 2026 WhuzPay. All rights reserved.
        </p>
        <p className="mt-1 text-xs text-white/40">
          PT WhuzPay Networks. Terdaftar dan diawasi oleh Bank Indonesia,
          nomor izin 25/660/DKSP/56/B serta terdaftar sebagai Penyedia Jasa
          Sistem Pembayaran (PJSP) di Kementerian Komunikasi dan Informatika
          Republik Indonesia, nomor 00097210/DJAI/PSE/06/2021.
        </p>
      </Container>
    </footer>
  );
}
