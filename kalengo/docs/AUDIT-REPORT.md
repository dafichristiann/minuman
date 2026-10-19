# Laporan Audit Kode & Database

> **Gerbang (§13.3):** frontend produksi **tidak boleh dirilis** sebelum laporan
> ini bersih dari temuan **High** dan **Critical**.

Perbarui berkas ini setiap kali `make audit` dijalankan. Jangan menghapus baris
"—"; isi hasilnya, sekalipun gagal. Kegagalan yang tercatat adalah kemajuan;
kegagalan yang disembunyikan adalah utang.

---

## Identitas laporan

| Field | Nilai |
|---|---|
| Milestone | **M0 — Fondasi** |
| Tanggal | _belum dijalankan_ |
| Commit | _belum ada_ (`git rev-parse HEAD`) |
| Lingkungan | _belum ada_ (mesin pengembang, bukan CI) |
| Dijalankan oleh | _belum ada_ |
| Kesimpulan | ⛔ **BELUM DINILAI** — M0 belum pernah dikompilasi |

---

## Ringkasan temuan

| Severity | Terbuka | Diperbaiki |
|---|---|---|
| Critical | — | — |
| High | — | — |
| Medium | — | — |
| Low | — | — |
| Info | — | — |

---

## 1. Gerbang otomatis

Jalankan `make audit` (atau `make ci`), lalu isi kolom hasil.

| # | Langkah | Alat | Kriteria lulus | Hasil | Bukti |
|---|---|---|---|---|---|
| 1.1 | Analisis statis Go | `golangci-lint` | 0 temuan High | ⬜ belum | |
| 1.2 | Kerentanan dependensi Go | `govulncheck ./...` | 0 yang dapat dieksploitasi | ⬜ belum | |
| 1.3 | Kerentanan dependensi web | `npm audit --omit=dev` | 0 High | ⬜ belum | |
| 1.4 | Pemindaian rahasia | `gitleaks` | 0 temuan | ⬜ belum | |
| 1.5 | Validasi SQL | `sqlc vet`, `sqlc compile` | Lulus | ⬜ belum | |
| 1.6 | Format Go | `gofmt -l .` | Keluaran kosong | ⬜ belum | |
| 1.7 | Race detector | `go test -race ./...` | Lulus | ⬜ belum | |
| 1.8 | Cakupan uji backend | `go tool cover` | ≥ 80% (service ≥ 90%) | ⬜ belum | |
| 1.9 | Lint + typecheck web | `npm run lint && npm run typecheck` | Bersih | ⬜ belum | |
| 1.10 | Uji unit web | `npm run test -- --run` | Lulus | ⬜ belum | |
| 1.11 | Aksesibilitas otomatis | `npm run test:a11y` (axe) | 0 pelanggaran | ⬜ belum | |

---

## 2. Database

| # | Langkah | Cara | Kriteria lulus | Hasil | Bukti |
|---|---|---|---|---|---|
| 2.1 | Review skema | Checklist §13.4 | Semua butir ✅ | ⚠️ 1 butir tertunda (partisi) | [`DB-NOTES.md`](DB-NOTES.md) §2 |
| 2.2 | Query terberat | `pg_stat_statements` + `EXPLAIN (ANALYZE, BUFFERS)` untuk 15 query teratas | Tidak ada seq scan pada tabel besar di jalur utama | ⬜ belum | tempel keluaran `make audit-db` |
| 2.3 | Migrasi pada DB kosong | `make migrate-redo` | Idempoten, tanpa error | ⬜ belum | |
| 2.4 | Migrasi pada DB berisi | `up` setelah data ada | Tanpa kehilangan data | ⬜ belum | |
| 2.5 | Hak akses role | `psql` sebagai `kalengo_app` | DDL & mutasi tabel append-only ditolak | ⬜ belum | |
| 2.6 | Anti-oversell | Uji 200 pembeli merebut 50 stok | Tidak ada oversell; `stock_reserved ≤ stock_on_hand` selalu benar | ⬜ belum | `make audit-concurrency` (M2) |
| 2.7 | Idempotensi `POST /orders` | 10 kiriman paralel, satu `Idempotency-Key` | Tepat 1 pesanan | ⬜ belum | `make audit-idempotency` (M2) |
| 2.8 | Partisi & retensi | Reviu rencana | Rencana ada; implementasi terjadwal | ✅ rencana | [`DB-NOTES.md`](DB-NOTES.md) §5 |
| 2.9 | Pemulihan dari backup | Restore ke lingkungan bersih | Data konsisten; RTO ≤ 1 jam | ⬜ belum | M5 |

### Keluaran `make audit-db` (tempel di sini)

```
(belum dijalankan — butuh PostgreSQL hidup dan beban k6 lebih dulu)
```

---

## 3. Keamanan aplikasi (§13.1)

Belum dapat dinilai: sebagian besar kontrol bergantung pada autentikasi dan
alur pesanan (M1–M2). Ayam/telur ini disengaja — audit penuh dijalankan ulang di
M5.

| # | Kontrol | Kriteria lulus | M | Hasil |
|---|---|---|---|---|
| 3.1 | Validasi skema ketat, batas body 1 MB, verifikasi MIME upload | Body > batas ditolak `SYS_UPLOAD_TOO_BIG`; MIME diverifikasi dari isi, bukan header | M1/M4 | ⬜ |
| 3.2 | Semua query lewat sqlc/parameterized | 0 string concat SQL | M1 | ⬜ |
| 3.3 | Output JSON saja; tanpa `dangerouslySetInnerHTML` | Aturan ESLint aktif; 0 pelanggaran | M6 | 🟡 aturan sudah dipasang |
| 3.4 | HTTPS + HSTS | HSTS aktif di produksi | M5 | ⬜ |
| 3.5 | CSP ketat (tanpa `unsafe-inline`, nonce), `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`, `frame-ancestors 'none'` | Semua header ada; CSP lolos uji | M5 | 🟡 middleware ada, nonce belum |
| 3.6 | CORS whitelist eksplisit | Hanya origin yang terdaftar | M0 | 🟡 `CORS` middleware ada |
| 3.7 | Verifikasi signature webhook + cek `gross_amount` + cek status pesanan | Replay/signature salah/jumlah salah ditolak aman | M2 | ⬜ |
| 3.8 | Idempotency pembuatan pesanan & webhook | Lihat 2.7 | M2 | ⬜ |
| 3.9 | Harga/total selalu dihitung server | Body klien yang memuat harga diabaikan | M2 | ⬜ |
| 3.10 | Anti-abuse voucher (rate limit + kuota atomik) | `UPDATE ... WHERE usage_count < usage_limit` teruji | M2 | ⬜ |
| 3.11 | Rahasia lewat environment, tidak di repo | gitleaks 0 temuan | M0 | ⬜ |
| 3.12 | Kontainer non-root, image minimal, dependensi dipin | User non-root; tag dipin (bukan `latest`) | M5 | 🟡 ada `TODO(M0)` di `minio-init` |
| 3.13 | DB least-privilege; user migrasi terpisah; TLS | Lihat 2.5 | M0 | 🟡 belum diuji |
| 3.14 | Backup terenkripsi + uji restore terjadwal | Lihat 2.9 | M5 | ⬜ |

---

## 4. Audit trail aplikasi (§13.2)

| # | Kriteria | M | Hasil |
|---|---|---|---|
| 4.1 | Setiap aksi admin menghasilkan baris `audit_logs` | M4 | ⬜ |
| 4.2 | `before`/`after` tersimpan dengan data sensitif disamarkan | M4 | ⬜ |
| 4.3 | Rantai hash valid: `row_hash = SHA256(prev_hash \|\| canonical_json(row))` | M4 | 🟡 `internal/audit` + uji unit ada |
| 4.4 | Job harian memverifikasi rantai & memicu `ADM_AUDIT_CHAIN_BROKEN` jika rusak | M4 | ⬜ |
| 4.5 | Baris `UPDATE`/`DELETE` ditolak di level DB | M0 | ✅ trigger + `REVOKE` |
| 4.6 | Akses baca hanya admin | M4 | ⬜ |

---

## 5. Uji lain yang diwajibkan §13.3

| # | Langkah | M | Hasil |
|---|---|---|---|
| 5.1 | Uji webhook: replay, signature salah, jumlah salah, urutan terbalik | M2 | ⬜ |
| 5.2 | Uji otorisasi: matriks peran × endpoint, IDOR otomatis | M4 | ⬜ |
| 5.3 | DAST: OWASP ZAP baseline terhadap staging | M5 | ⬜ |
| 5.4 | Beban: k6 200 rps campuran 10 menit memenuhi SLO §7 | M5 | ⬜ (skrip ada: `scripts/k6/load-mixed.js`) |
| 5.5 | Fuzz ringan: `go test -fuzz` pada parser input (voucher, pencarian) | M2 | ⬜ |
| 5.6 | Chaos kecil: matikan Redis / lambatkan DB → degradasi anggun | M5 | ⬜ |
| 5.7 | Kontrak: respons API sesuai `openapi.yaml` (schemathesis) | M5 | ⬜ |
| 5.8 | E2E Playwright: daftar → belanja → voucher → bayar → notifikasi → selesai | M6 | ⬜ |
| 5.9 | Visual: snapshot komponen utama & notifikasi | M6 | ⬜ |
| 5.10 | Lighthouse CI: mobile ≥ 90, LCP ≤ 2.5 s, CLS ≤ 0.05 | M6 | ⬜ |
| 5.11 | Anggaran bundel: JS awal ≤ 180 KB gzip | M6 | 🟡 `scripts/check-budget.mjs` ada |
| 5.12 | Profil animasi: 4× CPU throttle tetap ≥ 55 fps | M6 | ⬜ |

Legenda: ✅ lulus · 🟡 sebagian · ⬜ belum dijalankan · ❌ gagal

---

## 6. Keputusan gerbang

| Pertanyaan | Jawaban |
|---|---|
| Ada temuan Critical/High terbuka? | **belum dinilai** |
| Frontend boleh disambung ke produksi? | **tidak** — M0 belum diverifikasi |
| Tindak lanjut | Jalankan langkah di [`M0-SETUP.md`](M0-SETUP.md) §5, isi tabel di atas, lalu nilai ulang |

---

## Cara mengisi laporan ini

```bash
# 1. Pastikan lingkungan hidup
docker compose up -d postgres redis

# 2. Jalankan seluruh gerbang otomatis
make audit

# 3. Jalankan beban lebih dulu, baru analisis query
make audit-load
make audit-db

# 4. Salin hasilnya ke tabel di atas; isi kolom Bukti dengan kutipan keluaran
```

Jangan menulis "OK" tanpa bukti. Kutipan keluaran, tautan CI, atau nama berkas
artefak — bukan ingatan.
