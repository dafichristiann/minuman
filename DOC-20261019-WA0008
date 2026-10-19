# PRD — Website Jual Minuman Kaleng "KALENGO" (nama kerja)

> **Versi:** 1.0 · **Tanggal:** 19 Oktober 2026 · **Status:** Draft siap eksekusi
> **Stack:** React (Vite + TypeScript) · Go · PostgreSQL · Redis
> **Tema:** Animasi 2D (cartoon / rubber-hose modern) untuk logo & seluruh web
> **Prioritas khusus:** Sistem alert / pop-up / notifikasi yang sangat lengkap, backend + database rapi dan ter-audit, animasi ringan tapi "wah".

---

## 0. Cara Memakai Dokumen Ini

1. Baca bagian 1–4 untuk paham visi dan arah visual.
2. Bagian 5 (Pinterest) dikerjakan **sebelum** desain dimulai.
3. Bagian 8 (Sistem Notifikasi) adalah inti pembeda produk ini. Semua fitur lain wajib memakainya.
4. Bagian 11–13 (Arsitektur, Database, API) dikerjakan **backend-first**. Frontend baru disambung setelah backend lolos audit (bagian 14).
5. Bagian 17 berisi urutan kerja (milestone) dan Definition of Done.

Asumsi yang dipakai (ubah jika salah, lihat bagian 20):
- Produk dijual = **minuman kaleng non-alkohol** (soda, teh, kopi, jus, energy drink). Jika ada produk beralkohol, aktifkan modul *age gate* (bagian 9.8).
- Pasar = Indonesia (IDR, bahasa Indonesia, pembayaran lokal, ongkir lokal).
- Model = B2C, satu toko (bukan marketplace multi-seller).

---

## 1. Ringkasan Produk

### 1.1 Visi
Toko online minuman kaleng yang terasa seperti menonton kartun Sabtu pagi: setiap kaleng punya karakter, setiap aksi (tambah keranjang, bayar, kirim) mendapat respons visual dan pesan yang jelas, tanpa membuat halaman berat.

### 1.2 Masalah yang Diselesaikan
| Masalah | Solusi |
|---|---|
| Toko minuman online terasa generik dan membosankan | Identitas visual 2D yang konsisten dan punya karakter |
| User bingung status pesanan/pembayaran | Notifikasi multi-lapis (toast, banner, modal, in-app, email) di setiap titik penting |
| Stok sering habis saat checkout | Reservasi stok + peringatan stok real-time |
| Website beranimasi biasanya lambat | Anggaran animasi ketat: hanya `transform` dan `opacity`, tanpa Lottie besar |

### 1.3 Target Pengguna
- **Pembeli** usia 15–35: suka visual lucu, belanja lewat HP (mobile-first).
- **Admin toko**: kelola produk, stok, pesanan, voucher, lihat audit log.

### 1.4 Metrik Sukses
| Metrik | Target |
|---|---|
| Lighthouse Performance (mobile) | ≥ 90 |
| LCP | ≤ 2,5 detik (4G) |
| CLS | ≤ 0,05 |
| Bundle JS awal (gzip) | ≤ 180 KB |
| Total ukuran animasi/ilustrasi di halaman awal | ≤ 150 KB |
| Konversi keranjang → bayar | ≥ 35% |
| Error checkout karena stok | ≤ 1% |
| Cakupan test backend | ≥ 80% (service layer ≥ 90%) |
| Temuan audit severity High/Critical saat rilis | 0 |

---

## 2. Ruang Lingkup

### 2.1 Masuk Scope (MVP)
Katalog & pencarian, detail produk, keranjang, checkout, pembayaran (Midtrans/Xendit), ongkir (Biteship/RajaOngkir), voucher, akun & alamat, riwayat pesanan & pelacakan, wishlist, ulasan, notifikasi lengkap, panel admin, audit log.

### 2.2 Di Luar Scope MVP
Marketplace multi-seller, aplikasi mobile native, langganan berkala (bisa fase 2), program afiliasi, chat live.

---

## 3. Konsep Visual & Brand (Anti-"Slop")

### 3.1 Definisi "Slop" yang Dilarang
Hal-hal ini **tidak boleh** muncul di hasil akhir:
- Gradient ungu-biru generik, kartu kaca (glassmorphism) tanpa alasan.
- Ilustrasi AI mentah dengan jari/teks rusak, atau gambar stock yang tidak konsisten.
- Emoji sebagai pengganti ikon.
- Font default sistem sebagai font utama.
- Jarak, radius, dan tebal garis yang berbeda-beda tiap komponen.
- Animasi yang berjalan terus-menerus tanpa tujuan.
- Teks placeholder ("Lorem ipsum", "Product 1").
- Copywriting yang bisa ditempel ke toko mana pun.

### 3.2 Arah Gaya
**"Modern rubber-hose":** karakter kaleng berwajah dengan lengan/kaki lentur ala kartun 1930-an, tapi dengan warna flat modern, garis tebal seragam, dan bayangan keras (hard offset shadow). Bukan retro mentah, bukan flat korporat.

Prinsip:
1. **Satu tebal garis:** outline 3 px (desktop) / 2,5 px (mobile) di semua ilustrasi dan komponen.
2. **Satu bahasa bentuk:** sudut membulat besar, bentuk dasar (lingkaran, kapsul, silinder).
3. **Squash & stretch** sebagai satu-satunya "gaya gerak" utama.
4. **Hard shadow** offset 4 px 4 px tanpa blur.
5. Maksimal 5 warna aksen aktif per halaman.

### 3.3 Logo
- Nama kerja: **KALENGO**. Ganti jika sudah punya nama brand.
- Konsep: kaleng karakter (mata besar, mulut kecil, tutup tarik jadi "jambul") dengan wordmark huruf tebal membulat.
- **Deliverable logo (semua SVG, dibuat manual di Figma/Illustrator/Inkscape):**
  - `logo-full` (karakter + wordmark), `logo-mark` (karakter saja), `logo-wordmark` (teks saja)
  - Versi terang, gelap, dan 1 warna
  - Favicon 32/180/512 px, OG image 1200×630
- **Animasi logo (SVG + CSS, tanpa library):**
  | Momen | Animasi | Durasi |
  |---|---|---|
  | Load pertama | Kaleng jatuh, squash, memantul sekali, kedip | 900 ms |
  | Hover logo | Tutup tarik bergoyang, mata melirik | 400 ms |
  | Klik logo | Kaleng loncat kecil lalu kembali ke beranda | 350 ms |
  | Idle | Kedip mata tiap 6 detik (hanya jika tab aktif) | 150 ms |
- File animasi logo ≤ 8 KB (gzip). Dilarang pakai Lottie/GIF untuk logo.

### 3.4 Design Tokens
```css
:root {
  /* Warna */
  --cream:  #FFF4DC; /* latar utama */
  --ink:    #1B1B2F; /* outline & teks */
  --tomato: #FF4B3E; /* aksi utama, CTA */
  --sky:    #2EC4F1; /* info */
  --lemon:  #FFD23F; /* peringatan, highlight */
  --mint:   #3DDC97; /* sukses */
  --berry:  #D6336C; /* error / bahaya */
  --paper:  #FFFFFF;

  /* Garis & bayangan */
  --stroke: 3px;
  --shadow-hard: 4px 4px 0 var(--ink);
  --shadow-hard-sm: 2px 2px 0 var(--ink);

  /* Radius & jarak (skala 4 px) */
  --r-sm: 10px; --r-md: 16px; --r-lg: 24px; --r-pill: 999px;
  --space-1: 4px; --space-2: 8px; --space-3: 12px;
  --space-4: 16px; --space-6: 24px; --space-8: 32px;

  /* Gerak */
  --ease-pop: cubic-bezier(.34, 1.56, .64, 1); /* overshoot kecil */
  --ease-out: cubic-bezier(.22, 1, .36, 1);
  --dur-fast: 150ms; --dur-base: 250ms; --dur-slow: 450ms;
}
```
Kontras: semua teks wajib lolos WCAG AA (4,5:1). `--lemon` tidak boleh dipakai sebagai warna teks di atas `--cream`.

### 3.5 Tipografi
| Peran | Font (Google Fonts, self-host) | Catatan |
|---|---|---|
| Display / judul | **Bagel Fat One** atau **Chango** | Hanya H1–H2 dan harga besar |
| Body & UI | **Nunito** atau **DM Sans** | 400/600/800 |
| Angka harga | **Nunito 800** dengan `font-variant-numeric: tabular-nums` | |

Aturan: self-host `woff2`, subset Latin, `font-display: swap`, preload font display saja.

### 3.6 Ikon & Ilustrasi
- Set ikon tunggal bergaya outline tebal (custom atau Phosphor Bold disesuaikan stroke 2,5 px). Jangan campur set ikon.
- Setiap produk punya **ilustrasi kaleng sendiri** (SVG) dengan label & ekspresi karakter berbeda sesuai rasa. Foto produk asli ditambahkan di galeri detail, bukan pengganti ilustrasi.
- Maskot tambahan untuk empty state: keranjang kosong, pencarian tidak ketemu, 404, error 500, offline.

---

## 4. Referensi Pinterest (Wajib Sebelum Desain)

> Catatan: Pinterest dipakai sebagai **bahan referensi mood**, bukan untuk disalin. Gambar di Pinterest berhak cipta pihak lain. Jangan unduh lalu pakai sebagai aset final. Simpan sebagai papan inspirasi pribadi dan catat nama kreator untuk kredit atau, bila perlu, menghubungi mereka untuk lisensi/komisi.

### 4.1 Struktur Papan (Board)
| Board | Isi | Jumlah pin target |
|---|---|---|
| `01-logo-karakter` | Maskot kaleng, logo kartun, wordmark tebal | 25 |
| `02-warna-palet` | Palet cream + warna jenuh + outline gelap | 15 |
| `03-ilustrasi-kaleng` | Desain kemasan kaleng ilustratif | 30 |
| `04-ui-cartoon` | Web/app UI bergaya neo-brutalism lucu | 25 |
| `05-motion-2d` | Referensi gerak: squash & stretch, bounce, pop | 20 |
| `06-notifikasi-ui` | Toast, modal, badge, empty state lucu | 20 |
| `07-hindari` | Contoh yang terlalu generik (sebagai kontras) | 10 |

### 4.2 Kata Kunci Pencarian
Gunakan kombinasi ini di kotak pencarian Pinterest:

**Logo & karakter**
- `rubber hose character logo`
- `cute can mascot illustration`
- `soda can character design flat`
- `retro cartoon mascot logo bold outline`
- `beverage brand mascot vector`

**Kemasan & ilustrasi**
- `illustrated soda can packaging design`
- `drink can label illustration fruit character`
- `energy drink can art playful`
- `canned tea packaging illustration`

**UI & web**
- `neo brutalism web design playful`
- `cartoon ecommerce website ui`
- `hard shadow ui design bold border`
- `playful landing page illustration beverage`
- `sticker style ui components`

**Motion**
- `squash and stretch animation reference`
- `2d bounce animation principles`
- `micro interaction cartoon button`
- `soda bubble animation loop`

**Notifikasi**
- `toast notification playful ui`
- `cute empty state illustration`
- `error 404 page illustration funny`
- `success modal confetti ui`

Tips: tambahkan kata `dribbble`, `behance`, atau `vector` untuk menyaring hasil berkualitas, dan gunakan fitur "More ideas" Pinterest dari pin terbaik untuk menemukan gaya serupa.

### 4.3 Proses Kurasi
1. Kumpulkan ≥ 150 pin awal ke board.
2. Pilih 10 pin terbaik per board → buat **moodboard tunggal** (1 frame Figma).
3. Tulis 5 kata sifat yang disepakati tim (contoh: *lucu, tebal, lincah, segar, jujur*).
4. Ekstrak: palet warna, tebal garis, bentuk mata karakter, gaya bayangan.
5. Setiap aset desain final diberi catatan "terinspirasi dari" di dokumen internal, tanpa menyalin komposisi.
6. **Gerbang:** moodboard + token desain disetujui sebelum desain layar dimulai.

---

## 5. Persona & User Story

### 5.1 Persona
- **Dinda (19, mahasiswi):** belanja lewat HP, ingin bundling hemat dan voucher, suka tampilan lucu.
- **Rafi (28, pekerja kantoran):** beli satu dus untuk stok kantor, peduli ongkir dan kecepatan kirim.
- **Bu Sari (admin toko):** butuh daftar pesanan jelas, peringatan stok menipis, dan jejak siapa mengubah apa.

### 5.2 User Story Inti
| ID | Sebagai | Saya ingin | Agar |
|---|---|---|---|
| US-01 | Pembeli | mencari dan memfilter minuman berdasarkan rasa, merek, harga | cepat menemukan yang cocok |
| US-02 | Pembeli | menambah ke keranjang dengan umpan balik jelas | yakin aksi berhasil |
| US-03 | Pembeli | diperingatkan jika stok tinggal sedikit atau habis | tidak kecewa saat bayar |
| US-04 | Pembeli | memakai voucher dan melihat potongan langsung | tahu total sebenarnya |
| US-05 | Pembeli | membayar dengan metode lokal | praktis |
| US-06 | Pembeli | mendapat notifikasi setiap perubahan status pesanan | tidak perlu menebak |
| US-07 | Pembeli | mengatur preferensi notifikasi | tidak dibanjiri hal yang tidak perlu |
| US-08 | Admin | menerima peringatan stok menipis dan pesanan baru | bisa bertindak cepat |
| US-09 | Admin | melihat audit log | tahu siapa mengubah harga/stok/status |
| US-10 | Pembeli | memakai situs dengan lancar di HP lama | tidak ada lag akibat animasi |

---

## 6. Kebutuhan Fungsional

### 6.1 Katalog & Pencarian
- Daftar produk dengan paginasi cursor, filter (kategori, merek, rentang harga, rasa, ketersediaan), urutan (terlaris, termurah, terbaru).
- Pencarian dengan toleransi typo (PostgreSQL `pg_trgm` + `tsvector`), saran otomatis (debounce 250 ms).
- Varian produk: ukuran kaleng (250/330/500 ml) dan paket isi (1, 6, 12, 24).

### 6.2 Detail Produk
- Galeri (ilustrasi + foto), deskripsi, nilai gizi, label (Baru, Terlaris, Rendah Gula), stok per varian, ulasan, produk terkait.
- Tombol "Tambah ke Keranjang" dengan animasi kaleng terbang ke ikon keranjang.

### 6.3 Keranjang
- Tersimpan di server untuk user login, di `localStorage` untuk tamu, digabung (merge) saat login.
- Validasi harga & stok ulang setiap kali keranjang dibuka (harga di server adalah sumber kebenaran).

### 6.4 Checkout
1. Alamat → 2. Pengiriman (ongkir real-time) → 3. Voucher → 4. Ringkasan → 5. Bayar.
- Stok **direservasi 15 menit** saat pesanan dibuat; dilepas otomatis jika tidak dibayar.
- Idempotency key untuk mencegah pesanan ganda saat klik berulang.

### 6.5 Pembayaran
- Midtrans Snap atau Xendit: VA bank, QRIS, e-wallet, kartu.
- Webhook terverifikasi tanda tangan; status pembayaran hanya berubah lewat webhook atau pengecekan server-ke-server.

### 6.6 Pesanan & Pelacakan
Status: `pending_payment → paid → processing → shipped → delivered → completed`, serta `cancelled`, `expired`, `refund_requested`, `refunded`. Setiap perubahan dicatat di riwayat dan memicu notifikasi.

### 6.7 Akun
Daftar, login, lupa/reset password, verifikasi email, kelola alamat, ubah password, hapus akun (anonimisasi data sesuai UU PDP).

### 6.8 Wishlist & Ulasan
Ulasan hanya dari pembeli yang pesanannya `completed` (1 ulasan per item per pesanan).

### 6.9 Panel Admin
Dashboard ringkas, CRUD produk/varian/kategori, manajemen stok (dengan alasan perubahan), daftar pesanan + ubah status + input resi, voucher, pengguna, **audit log** (hanya baca), pengaturan notifikasi toko.

---

## 7. Kebutuhan Non-Fungsional

| Area | Persyaratan |
|---|---|
| Performa | p95 API ≤ 300 ms (baca), ≤ 600 ms (tulis) pada 200 rps |
| Ketersediaan | 99,5% bulanan |
| Aksesibilitas | WCAG 2.2 AA, navigasi keyboard penuh, `prefers-reduced-motion` dihormati |
| Responsif | Mobile-first: 360 px sampai 1920 px |
| Browser | 2 versi terakhir Chrome, Safari, Firefox, Edge; iOS Safari 15+ |
| Bahasa | Indonesia (utama), siap i18n (Inggris fase 2) |
| Keamanan | Lihat bagian 14 |
| Privasi | Sesuai UU PDP: persetujuan eksplisit, hak hapus data, minimisasi data |
| Observabilitas | Log terstruktur, metrik Prometheus, tracing OpenTelemetry, error tracking (Sentry) |

---

## 8. SISTEM ALERT, POP-UP & NOTIFIKASI (Fokus Utama)

### 8.1 Prinsip
1. **Setiap aksi penting punya umpan balik.** Tidak ada aksi yang "diam".
2. **Pilih jenis yang tepat** (tabel 8.2). Jangan memakai modal untuk hal sepele.
3. **Banyak, tapi tidak mengganggu:** antrean, deduplikasi, batas tumpukan, dan preferensi pengguna.
4. **Satu sumber kebenaran:** semua notifikasi lewat satu `NotificationProvider` dan satu katalog kode (`notifications.catalog.ts`) yang disinkronkan dengan kode error backend.
5. **Bahasa kartun tapi jelas:** nada ramah, kalimat pendek, selalu ada langkah berikutnya.
6. **Aksesibel:** `role="status"` / `role="alert"`, `aria-live`, fokus dikelola pada modal.

### 8.2 Jenis Komponen
| Jenis | Kapan dipakai | Perilaku | Durasi | ARIA |
|---|---|---|---|---|
| **Toast** | Umpan balik ringan (berhasil, info) | Muncul pojok (atas-kanan desktop, atas-tengah mobile), bisa ditutup | 4 s (sukses), 6 s (info) | `role="status"` |
| **Toast peringatan** | Peringatan non-blokir | Sama, ikon kuning, getar kecil | 7 s | `role="alert"` |
| **Toast error** | Gagal aksi | Tidak hilang otomatis sampai ditutup untuk error penting | 8 s / manual | `role="alert"` |
| **Snackbar + Undo** | Aksi yang bisa dibatalkan (hapus item keranjang, hapus wishlist) | Tombol "Batalkan" | 6 s | `role="status"` |
| **Banner (sticky)** | Kondisi sistem/halaman (offline, promo, pembayaran menunggu) | Menempel di atas halaman | Sampai kondisi selesai / ditutup | `role="region"` |
| **Inline alert** | Validasi form, pesan di dalam konteks | Di bawah field/di atas form | Permanen sampai diperbaiki | `aria-describedby` |
| **Modal konfirmasi** | Aksi berisiko (batalkan pesanan, hapus akun, logout semua perangkat) | Fokus terkunci, Esc menutup, backdrop | Manual | `role="alertdialog"` |
| **Modal informasi** | Hal penting yang wajib dibaca (perubahan harga, age gate, sesi berakhir) | Satu tombol utama | Manual | `role="dialog"` |
| **Modal perayaan** | Momen senang (pesanan berhasil, voucher terpakai) | Konfetti ringan CSS | Manual / 5 s | `role="dialog"` |
| **Popover/Tooltip** | Penjelasan kecil (arti label, aturan voucher) | Hover/fokus | — | `role="tooltip"` |
| **Notification Center** | Riwayat semua notifikasi in-app | Panel lonceng dengan badge angka | Persisten (disimpan 90 hari) | `aria-live` pada badge |
| **Push/Browser Notification** | Pembaruan pesanan saat tab tidak aktif | Opt-in, via Web Push | — | — |
| **Email** | Transaksional & keamanan | Template bergaya brand | — | — |
| **Halaman status** | 404, 500, offline, maintenance | Maskot + tindakan | — | — |

### 8.3 Aturan Antrean & Anti-Spam
- Maksimal **3 toast tampil bersamaan**; sisanya masuk antrean (FIFO, prioritas: error > peringatan > sukses > info).
- **Deduplikasi:** notifikasi dengan `dedupeKey` sama dalam 3 detik digabung dan menampilkan hitungan ("×3").
- **Throttle:** maksimal 1 toast promo per 10 menit per sesi; promo tidak tampil saat checkout.
- Modal **tidak boleh bertumpuk**; modal baru masuk antrean, kecuali tipe `critical` (sesi berakhir, pembayaran gagal).
- Toast otomatis **jeda saat di-hover/fokus** dan saat tab tidak aktif.
- Pengguna bisa memilih: **Normal / Tenang** (hanya penting) / **Hening** (hanya error & keamanan) di Pengaturan.
- Bunyi opsional (default mati), getar halus di mobile (opsional, `navigator.vibrate`).

### 8.4 Animasi Notifikasi
| Komponen | Masuk | Keluar |
|---|---|---|
| Toast | Slide 16 px + fade + overshoot kecil (`--ease-pop`, 280 ms) | Fade + geser 8 px (180 ms) |
| Modal | Backdrop fade 150 ms; kartu "pop" scale 0.92→1 (250 ms) | Scale 1→0.96 + fade (150 ms) |
| Ikon sukses | Centang tergambar (stroke-dashoffset, 400 ms) | — |
| Ikon error | Goyang horizontal 3× (300 ms) | — |
| Badge lonceng | Pop scale 1→1.25→1 saat angka berubah | — |
| Konfetti | 12–16 partikel CSS, 900 ms, hanya `transform` + `opacity` | — |
Semua animasi dimatikan/disederhanakan saat `prefers-reduced-motion: reduce` (hanya fade).

### 8.5 Struktur Data Notifikasi (Frontend)
```ts
type NotifKind = 'toast' | 'snackbar' | 'banner' | 'inline' | 'modal' | 'celebrate';
type Severity = 'success' | 'info' | 'warning' | 'error' | 'critical';

interface AppNotification {
  code: string;            // mis. "CART_ADDED" (cocok dengan katalog & kode error backend)
  kind: NotifKind;
  severity: Severity;
  title: string;
  message?: string;
  dedupeKey?: string;
  durationMs?: number;     // 0 = manual
  actions?: { label: string; onClick: () => void; variant?: 'primary' | 'ghost' }[];
  persist?: boolean;       // simpan ke Notification Center
  meta?: Record<string, unknown>;
}
```
Hook: `const notify = useNotify(); notify('CART_ADDED', { name, qty })` → pesan diambil dari katalog agar teks seragam dan mudah diubah.

### 8.6 Katalog Notifikasi Lengkap
Kolom **Tipe**: T=Toast, S=Snackbar+Undo, B=Banner, I=Inline, M=Modal, C=Celebrate, E=Email, P=Push, N=Notification Center.

#### A. Autentikasi & Akun
| Kode | Pemicu | Tipe | Severity | Teks (contoh) |
|---|---|---|---|---|
| AUTH_REGISTER_OK | Daftar berhasil | T, E | success | "Yeay, akun jadi! Cek email buat verifikasi ya." |
| AUTH_EMAIL_TAKEN | Email sudah terdaftar | I | error | "Email ini sudah terdaftar. Mau masuk saja?" |
| AUTH_VERIFY_SENT | Email verifikasi terkirim | T | info | "Email verifikasi meluncur. Cek inbox/spam." |
| AUTH_VERIFY_OK | Email terverifikasi | C | success | "Terverifikasi! Selamat belanja." |
| AUTH_VERIFY_EXPIRED | Tautan verifikasi kedaluwarsa | M | warning | "Tautan sudah kedaluwarsa. Kirim yang baru?" |
| AUTH_LOGIN_OK | Login berhasil | T | success | "Halo lagi, {nama}!" |
| AUTH_LOGIN_FAIL | Email/password salah | I | error | "Email atau password tidak cocok." (generik, tidak bocorkan mana yang salah) |
| AUTH_LOGIN_LOCKED | Terlalu banyak percobaan | M | critical | "Terlalu banyak percobaan. Coba lagi dalam {menit} menit." |
| AUTH_LOGIN_NEW_DEVICE | Login dari perangkat baru | E, N | warning | "Ada login baru dari {perangkat}, {kota}. Bukan kamu? Amankan akun." |
| AUTH_LOGOUT_OK | Logout | T | info | "Sampai jumpa! Kaleng-kalengmu kami jaga." |
| AUTH_LOGOUT_CONFIRM | Logout semua perangkat | M | warning | "Keluar dari semua perangkat?" |
| AUTH_SESSION_EXPIRING | 2 menit sebelum sesi habis | M | warning | "Sesi hampir habis. Tetap masuk?" |
| AUTH_SESSION_EXPIRED | Token habis & gagal refresh | M | critical | "Sesi berakhir. Masuk lagi ya, keranjangmu aman." |
| AUTH_PASSWORD_WEAK | Password lemah | I | warning | "Tambah panjang & angka biar lebih kuat." |
| AUTH_PASSWORD_RESET_SENT | Permintaan reset | T | info | "Jika email terdaftar, tautan reset sudah dikirim." |
| AUTH_PASSWORD_CHANGED | Password diubah | T, E | success | "Password berhasil diganti." |
| ACC_PROFILE_SAVED | Profil disimpan | T | success | "Profil diperbarui." |
| ACC_ADDRESS_ADDED | Alamat ditambah | T | success | "Alamat baru tersimpan." |
| ACC_ADDRESS_DELETE | Hapus alamat | S | info | "Alamat dihapus. [Batalkan]" |
| ACC_DELETE_CONFIRM | Hapus akun | M | critical | "Hapus akun permanen? Ketik HAPUS untuk lanjut." |
| ACC_DELETE_DONE | Akun dihapus | M | info | "Akunmu sudah dihapus. Terima kasih sudah mampir." |

#### B. Katalog, Keranjang & Wishlist
| Kode | Pemicu | Tipe | Severity | Teks |
|---|---|---|---|---|
| CART_ADDED | Tambah ke keranjang | T | success | "{produk} masuk keranjang!" + aksi "Lihat" |
| CART_QTY_MAX | Melebihi stok/batas | T | warning | "Maksimal {n} per pesanan untuk {produk}." |
| CART_REMOVED | Hapus item | S | info | "{produk} dihapus. [Batalkan]" |
| CART_CLEARED | Kosongkan keranjang | M | warning | "Kosongkan semua isi keranjang?" |
| CART_PRICE_CHANGED | Harga berubah sejak ditambahkan | M | warning | "Harga {produk} berubah dari {lama} jadi {baru}." |
| CART_ITEM_UNAVAILABLE | Produk tidak tersedia lagi | B | error | "{produk} sudah tidak tersedia dan dihapus dari keranjang." |
| CART_MERGED | Keranjang tamu digabung saat login | T | info | "Keranjangmu sebelumnya sudah digabung." |
| CART_FREE_SHIPPING_NEAR | Mendekati gratis ongkir | T | info | "Tambah {Rp} lagi, ongkir gratis!" |
| CART_FREE_SHIPPING_OK | Mencapai gratis ongkir | C | success | "Gratis ongkir terbuka!" |
| CART_SAVED_REMINDER | Keranjang lama tak disentuh | E, P | info | "Kaleng-kalengmu kangen. Lanjut belanja?" |
| STOCK_LOW | Stok ≤ 10 | T, I | warning | "Tinggal {n}! Jangan sampai kehabisan." |
| STOCK_LAST | Stok = 1 | I | warning | "Sisa 1 kaleng terakhir!" |
| STOCK_OUT | Stok 0 | I, B | error | "Habis. Kabari saya saat tersedia?" |
| STOCK_NOTIFY_ON | Daftar pengingat stok | T | success | "Oke, kami kabari saat {produk} restock." |
| STOCK_BACK | Produk restock | E, P, N | success | "{produk} sudah tersedia lagi!" |
| STOCK_CHANGED_CHECKOUT | Stok berubah saat checkout | M | error | "Stok {produk} tinggal {n}. Sesuaikan jumlahnya." |
| WISH_ADDED | Tambah wishlist | T | success | "Disimpan ke wishlist." |
| WISH_REMOVED | Hapus wishlist | S | info | "Dihapus dari wishlist. [Batalkan]" |
| WISH_LOGIN_REQ | Wishlist tanpa login | M | info | "Masuk dulu biar wishlist-mu tersimpan." |
| WISH_PRICE_DROP | Harga item wishlist turun | E, P, N | success | "Harga {produk} turun jadi {harga}!" |
| SEARCH_EMPTY | Tidak ada hasil | I | info | "Belum ketemu. Coba kata lain?" |
| FILTER_APPLIED | Filter diterapkan | T (ringan) | info | "{n} produk cocok." |

#### C. Voucher & Promo
| Kode | Pemicu | Tipe | Severity | Teks |
|---|---|---|---|---|
| VOUCHER_OK | Voucher berlaku | C/T | success | "Voucher dipakai! Hemat {Rp}." |
| VOUCHER_INVALID | Kode tidak ada | I | error | "Kode voucher tidak dikenali." |
| VOUCHER_EXPIRED | Kedaluwarsa | I | error | "Voucher sudah kedaluwarsa." |
| VOUCHER_MIN_SPEND | Belum memenuhi minimal | I | warning | "Belanja minimal {Rp} untuk voucher ini." |
| VOUCHER_USED_UP | Kuota habis | I | error | "Voucher sudah habis dipakai." |
| VOUCHER_LIMIT_USER | Batas per pengguna | I | error | "Kamu sudah memakai voucher ini." |
| VOUCHER_REMOVED | Voucher dilepas | T | info | "Voucher dilepas." |
| VOUCHER_EXPIRING | Voucher akan habis (<24 jam) | B, E, N | warning | "Voucher {kode} berakhir besok!" |
| PROMO_FLASH_START | Flash sale mulai | B, P | info | "Flash sale mulai! Berakhir dalam {hh:mm:ss}." |
| PROMO_FLASH_ENDING | 10 menit terakhir | T | warning | "10 menit lagi flash sale selesai." |
| PROMO_FLASH_END | Flash sale usai | T | info | "Flash sale selesai. Sampai promo berikutnya!" |

#### D. Checkout & Pembayaran
| Kode | Pemicu | Tipe | Severity | Teks |
|---|---|---|---|---|
| CHK_ADDRESS_REQ | Alamat belum dipilih | I | error | "Pilih alamat pengiriman dulu." |
| CHK_SHIPPING_LOADING | Hitung ongkir | I | info | "Menghitung ongkir…" |
| CHK_SHIPPING_FAIL | Ongkir gagal dihitung | I | error | "Ongkir gagal dihitung. Coba lagi." + [Ulangi] |
| CHK_AREA_UNSERVED | Area tidak terjangkau | M | warning | "Belum bisa kirim ke area ini." |
| CHK_TERMS_REQ | Belum setuju syarat | I | error | "Centang persetujuan dulu." |
| CHK_ORDER_CREATING | Membuat pesanan | M (loading) | info | "Menyiapkan kaleng-kalengmu…" |
| CHK_ORDER_CREATED | Pesanan dibuat | T | success | "Pesanan #{no} dibuat. Selesaikan pembayaran." |
| CHK_DUPLICATE_BLOCKED | Klik ganda terdeteksi | T | info | "Pesanan sedang diproses, tunggu sebentar." |
| CHK_LEAVE_GUARD | Keluar di tengah checkout | M | warning | "Yakin keluar? Pesananmu belum selesai." |
| PAY_PENDING_BANNER | Ada pesanan menunggu bayar | B | warning | "Bayar sebelum {hh:mm}. Sisa {countdown}." |
| PAY_COUNTDOWN_5 | Sisa 5 menit | M | warning | "Tinggal 5 menit! Stok akan dilepas." |
| PAY_SUCCESS | Pembayaran diterima | C, E, P, N | success | "Pembayaran diterima! Kaleng segera dikemas." |
| PAY_FAILED | Pembayaran gagal | M | error | "Pembayaran gagal. Coba metode lain?" |
| PAY_EXPIRED | Waktu bayar habis | M, E, N | error | "Waktu pembayaran habis. Pesanan dibatalkan." |
| PAY_PROCESSING | Menunggu konfirmasi bank | B | info | "Pembayaran sedang diverifikasi…" |
| PAY_METHOD_COPY | Salin nomor VA | T | success | "Nomor VA tersalin." |
| PAY_REFUND_STARTED | Refund diproses | E, N | info | "Refund {Rp} sedang diproses." |
| PAY_REFUND_DONE | Refund selesai | E, N | success | "Refund sudah masuk ke {metode}." |

#### E. Pesanan & Pengiriman
| Kode | Pemicu | Tipe | Severity | Teks |
|---|---|---|---|---|
| ORD_PROCESSING | Dikemas | N, E | info | "Pesanan #{no} sedang dikemas." |
| ORD_SHIPPED | Dikirim | N, E, P | info | "Pesanan dikirim! Resi: {resi}." |
| ORD_OUT_FOR_DELIVERY | Kurir menuju lokasi | P, N | info | "Kurir hampir sampai!" |
| ORD_DELIVERED | Terkirim | C, E, N | success | "Kaleng sudah sampai! Selamat menikmati." |
| ORD_DELAYED | Keterlambatan | B, E, N | warning | "Pengiriman tertunda. Kami sedang cek ke kurir." |
| ORD_CANCEL_CONFIRM | Batalkan pesanan | M | warning | "Batalkan pesanan #{no}?" |
| ORD_CANCELLED | Dibatalkan | T, E | info | "Pesanan dibatalkan." |
| ORD_CANCEL_DENIED | Tidak bisa dibatalkan | M | error | "Pesanan sudah dikirim, tidak bisa dibatalkan." |
| ORD_REVIEW_REQ | Minta ulasan (H+2) | E, N | info | "Gimana rasanya? Tulis ulasanmu." |
| ORD_REORDER | Pesan ulang | T | success | "Item ditambahkan ke keranjang." |
| ORD_RETURN_REQ | Ajukan retur | M, T | info | "Pengajuan retur terkirim." |
| REV_SUBMITTED | Ulasan terkirim | C | success | "Terima kasih ulasannya!" |
| REV_NOT_ELIGIBLE | Belum boleh mengulas | T | warning | "Ulasan dibuka setelah pesanan selesai." |

#### F. Sistem, Jaringan & Umum
| Kode | Pemicu | Tipe | Severity | Teks |
|---|---|---|---|---|
| NET_OFFLINE | Koneksi hilang | B | warning | "Kamu offline. Beberapa fitur dimatikan." |
| NET_BACK_ONLINE | Koneksi pulih | T | success | "Kembali online!" |
| NET_SLOW | Respons lambat (>5 s) | T | info | "Koneksi agak lambat, sabar ya…" |
| API_RATE_LIMIT | 429 | T | warning | "Pelan-pelan dulu, coba lagi {detik} detik." |
| API_SERVER_ERROR | 5xx | T | error | "Server lagi batuk. Coba lagi sebentar." + [Ulangi] |
| API_VALIDATION | 422 | I | error | Pesan per field dari server |
| API_FORBIDDEN | 403 | M | error | "Kamu tidak punya akses ke halaman ini." |
| API_NOT_FOUND | 404 resource | I | info | "Yang kamu cari tidak ketemu." |
| SYS_MAINTENANCE_SOON | Maintenance terjadwal | B | info | "Pemeliharaan pukul {jam}. Selesaikan pesananmu dulu." |
| SYS_MAINTENANCE | Mode maintenance | Halaman | info | "Sedang dipoles. Segera kembali!" |
| SYS_NEW_VERSION | Versi baru tersedia | T | info | "Ada pembaruan. [Muat ulang]" |
| SYS_COPIED | Salin ke papan klip | T | success | "Tersalin!" |
| SYS_COOKIE_CONSENT | Kunjungan pertama | B | info | "Kami pakai cookie untuk pengalaman lebih baik." |
| SYS_PUSH_PERMISSION | Minta izin push | M | info | "Mau dikabari soal status pesanan?" |
| SYS_PUSH_DENIED | Izin push ditolak | T | info | "Tidak apa-apa. Kamu bisa ubah di Pengaturan." |
| SYS_UPLOAD_TOO_BIG | Berkas terlalu besar | I | error | "Maksimal {n} MB." |
| SYS_UNSAVED_CHANGES | Keluar form belum disimpan | M | warning | "Perubahan belum disimpan. Keluar?" |
| AGE_GATE | Produk berbatas umur (jika ada) | M | critical | "Produk ini khusus 21+. Konfirmasi usiamu." |

#### G. Admin
| Kode | Pemicu | Tipe | Severity | Teks |
|---|---|---|---|---|
| ADM_NEW_ORDER | Pesanan baru dibayar | T, P, N, suara opsional | info | "Pesanan baru #{no} — {Rp}" |
| ADM_STOCK_LOW | Stok di bawah ambang | B, N, E | warning | "{produk} tinggal {n}." |
| ADM_STOCK_ADJUSTED | Stok diubah manual | T | success | "Stok diperbarui. Alasan tercatat." |
| ADM_PRICE_CHANGE_CONFIRM | Ubah harga >20% | M | warning | "Harga berubah drastis. Yakin?" |
| ADM_DELETE_CONFIRM | Hapus produk | M | critical | "Hapus {produk}? Pesanan lama tetap aman." |
| ADM_STATUS_CHANGED | Ubah status pesanan | T | success | "Status diubah ke {status}." |
| ADM_WEBHOOK_FAIL | Webhook pembayaran gagal | B, E | error | "Webhook gagal diproses {n}× — cek segera." |
| ADM_AUDIT_EXPORT | Ekspor audit log | T | info | "Ekspor sedang disiapkan." |
| ADM_SUSPICIOUS | Aktivitas mencurigakan terdeteksi | B, E | critical | "Banyak percobaan login gagal dari {ip}." |

> Total katalog: **±115 notifikasi**. Semua teks disimpan di satu berkas, mudah diubah tanpa menyentuh logika.

### 8.7 Notifikasi Real-Time (Backend → Frontend)
- **Server-Sent Events (SSE)** di `GET /api/v1/notifications/stream` (autentikasi cookie). Alasan: satu arah, ringan, otomatis reconnect, lebih sederhana dari WebSocket.
- Redis Pub/Sub menyebarkan event ke semua instance Go.
- Event: `notification.created`, `order.status_changed`, `stock.changed`, `promo.started`.
- Fallback: polling tiap 30 detik jika SSE gagal 3×.
- Web Push (VAPID) untuk saat tab tertutup; email lewat antrean job (Asynq) dengan retry & dead-letter.

### 8.8 Kriteria Penerimaan Notifikasi
- [ ] Setiap kode di katalog punya test komponen (render + aksesibilitas `axe`).
- [ ] Tidak ada aksi mutasi (POST/PUT/PATCH/DELETE) tanpa umpan balik sukses **dan** gagal.
- [ ] Antrean menahan ≥ 20 notifikasi beruntun tanpa lag.
- [ ] Mode "Hening" benar-benar hanya menampilkan error & keamanan.
- [ ] Semua toast bisa ditutup dengan keyboard; modal mengembalikan fokus ke pemicu.
- [ ] Email transaksional lolos uji tampil di Gmail, Outlook, dan HP.

---

## 9. Animasi Halaman ("Tipis tapi Wah")

### 9.1 Anggaran Performa Animasi
| Aturan | Nilai |
|---|---|
| Properti yang boleh dianimasikan | `transform`, `opacity` (tidak ada `width/height/top/left/box-shadow`) |
| Animasi serentak maksimal di viewport | 6 |
| Durasi | 150–450 ms (kecuali loop dekoratif) |
| Loop dekoratif | Dijeda saat tab tidak aktif (`document.visibilityState`) dan di luar viewport (IntersectionObserver) |
| Library | CSS + SVG utama; `motion` (ex-Framer Motion, LazyMotion `domAnimation` ~5 KB) hanya untuk orkestrasi |
| Dilarang | Lottie besar, video latar, GIF, canvas penuh layar, parallax berat |
| Uji | Profil di HP kelas menengah (4× CPU throttle) tetap ≥ 55 fps |

### 9.2 Daftar Animasi
| # | Lokasi | Efek | Teknik |
|---|---|---|---|
| 1 | Pembuka situs | Logo jatuh + memantul, halaman fade-in berurutan (stagger 60 ms) | CSS keyframes |
| 2 | Hero | Kaleng maskot mengambang pelan (translateY ±6 px, 4 s), gelembung soda naik | CSS, `will-change: transform` terbatas |
| 3 | Kartu produk (hover/fokus) | Naik 4 px, miring 2°, bayangan keras bergeser; kaleng "bergoyang" | CSS transform |
| 4 | Tombol | Tekan = squash (scaleY .94) lalu pegas kembali | CSS `:active` |
| 5 | Tambah ke keranjang | Kaleng mini **terbang melengkung** ke ikon keranjang, ikon keranjang "bounce", badge pop | FLIP + WAAPI |
| 6 | Scroll reveal | Elemen naik 12 px + fade, sekali saja | IntersectionObserver + CSS |
| 7 | Transisi halaman | Fade + geser 8 px (200 ms) | React Router + motion |
| 8 | Skeleton | Shimmer halus (satu gradient bergeser dengan `transform`) | CSS |
| 9 | Filter/urutan | Kartu berpindah dengan layout animation | motion `layout` (terbatas ≤ 24 item) |
| 10 | Wishlist (hati) | Hati mengembang + percikan kecil | SVG + CSS |
| 11 | Stepper checkout | Kaleng kecil bergeser antar langkah | CSS transform |
| 12 | Sukses bayar | Konfetti ringan + maskot melompat | CSS, 16 partikel |
| 13 | Empty state | Maskot berkedip/berayun lambat | SVG + CSS |
| 14 | Countdown flash sale | Angka "flip" saat berubah | CSS |
| 15 | Kursor/scroll | **Tidak ada** efek kursor kustom (mencegah lag & isu aksesibilitas) | — |

### 9.3 Aksesibilitas Gerak
```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```
Tambahkan juga saklar "Kurangi animasi" di Pengaturan (menyimpan preferensi di akun).

### 9.4 Modul Khusus: Age Gate (opsional)
Aktif per produk lewat flag `requires_age_verification`. Modal menanyakan tanggal lahir, validasi di server, hasil disimpan di sesi/akun. Tidak ada klaim "otomatis lolos" di sisi klien.

---

## 10. Arsitektur Sistem

```
[ React SPA (Vite, TS) ] --HTTPS--> [ Nginx/Caddy ] --> [ Go API (chi) ]
        |                                                    |
        |<--------------- SSE / Web Push -------------------|
                                                              |--> PostgreSQL 16 (utama)
                                                              |--> Redis (cache, rate limit, pub/sub, antrean)
                                                              |--> Asynq Worker (email, expire pesanan, webhook retry)
                                                              |--> Midtrans/Xendit, Biteship/RajaOngkir, SMTP/SES, Object Storage (S3/MinIO)
```

### 10.1 Frontend
| Bidang | Pilihan |
|---|---|
| Build | Vite + React 18 + TypeScript (strict) |
| Routing | React Router 6 (lazy per rute) |
| State server | TanStack Query |
| State klien | Zustand (keranjang tamu, preferensi notifikasi) |
| Styling | Tailwind CSS + CSS variables dari token (komponen UI sendiri, bukan template) |
| Form | React Hook Form + Zod |
| Animasi | CSS + `motion` (LazyMotion) |
| Uji | Vitest, Testing Library, Playwright (E2E), axe (a11y) |
| Lint | ESLint, Prettier, `tsc --noEmit` |

### 10.2 Backend
| Bidang | Pilihan |
|---|---|
| Bahasa | Go 1.23+ |
| HTTP | `chi` + middleware (request ID, recover, timeout, CORS, secure headers, rate limit) |
| DB akses | `pgx/v5` + **`sqlc`** (query bertipe, tanpa ORM ajaib) |
| Migrasi | `golang-migrate` (up/down, dijalankan di CI) |
| Validasi | `go-playground/validator` + validasi domain di service |
| Auth | Access token JWT (15 menit) + refresh token acak, **disimpan hash di DB**, dikirim sebagai cookie `HttpOnly; Secure; SameSite=Lax`; rotasi refresh token + deteksi reuse |
| Password | **argon2id** (parameter tercatat di config) |
| Job | Asynq (Redis) |
| Log | `slog` JSON + request ID + user ID (tanpa PII sensitif) |
| Observabilitas | Prometheus metrics, OpenTelemetry, Sentry |
| Dokumentasi API | OpenAPI 3.1 (`openapi.yaml`) sebagai kontrak; klien TS dibangkitkan otomatis |
| Uji | `testing` + `testify`, `testcontainers-go` (Postgres & Redis asli), `httptest` |

### 10.3 Struktur Repositori (monorepo)
```
kalengo/
├─ apps/
│  ├─ web/                       # React
│  │  └─ src/
│  │     ├─ app/ (router, providers)
│  │     ├─ design/ (tokens.css, fonts, motion.ts)
│  │     ├─ components/ui/ (Button, Card, Modal, Toast, Badge, ...)
│  │     ├─ features/ (catalog, cart, checkout, orders, account, admin)
│  │     ├─ notifications/ (provider, queue, catalog.ts, center/, sse.ts)
│  │     ├─ assets/ (svg logo, maskot, kaleng)
│  │     └─ lib/ (api client, format, i18n)
│  └─ api/                       # Go
│     ├─ cmd/api/main.go
│     ├─ cmd/worker/main.go
│     ├─ internal/
│     │  ├─ config/  ├─ http/ (handler, middleware, dto)
│     │  ├─ domain/  ├─ service/ (logika bisnis)
│     │  ├─ repo/ (sqlc generated + wrapper)
│     │  ├─ auth/    ├─ payment/ ├─ shipping/
│     │  ├─ notify/ (SSE hub, push, email, templates)
│     │  ├─ jobs/    ├─ audit/   └─ platform/ (logger, redis, db)
│     ├─ db/migrations/  ├─ db/queries/ (*.sql untuk sqlc)
│     └─ openapi.yaml
├─ docker-compose.yml
├─ Makefile
└─ .github/workflows/ (ci.yml: lint, test, gosec, govulncheck, build)
```
Aturan lapisan: `handler → service → repo`. Handler tidak menyentuh SQL; service tidak tahu HTTP.

---

## 11. Desain Database (PostgreSQL 16)

### 11.1 Konvensi
- Primary key `uuid` (`gen_random_uuid()`), kecuali tabel referensi kecil.
- Uang disimpan sebagai **integer rupiah** (`bigint`), tidak pernah `float`.
- Semua tabel punya `created_at`, `updated_at` (trigger), tabel penting punya `deleted_at` (soft delete).
- Enum lewat tipe `ENUM` atau `CHECK`.
- Foreign key dengan `ON DELETE` eksplisit; indeks untuk setiap FK dan kolom filter.
- Ekstensi: `pgcrypto`, `pg_trgm`, `citext`.

### 11.2 Skema (DDL)
```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS citext;

-- Trigger updated_at
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END; $$ LANGUAGE plpgsql;

-- ===== ENUMS =====
CREATE TYPE user_role AS ENUM ('customer','admin','staff');
CREATE TYPE order_status AS ENUM
  ('pending_payment','paid','processing','shipped','delivered','completed',
   'cancelled','expired','refund_requested','refunded');
CREATE TYPE payment_status AS ENUM ('pending','paid','failed','expired','refunded','partial_refund');
CREATE TYPE voucher_type AS ENUM ('percent','fixed','free_shipping');
CREATE TYPE stock_reason AS ENUM
  ('restock','sale','reservation','release','adjustment','return','damage');
CREATE TYPE notif_channel AS ENUM ('in_app','email','push');

-- ===== USERS & AUTH =====
CREATE TABLE users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email         citext NOT NULL UNIQUE,
  password_hash text   NOT NULL,
  full_name     text   NOT NULL CHECK (char_length(full_name) BETWEEN 2 AND 100),
  phone         text,
  role          user_role NOT NULL DEFAULT 'customer',
  email_verified_at timestamptz,
  birth_date    date,
  failed_login_count int NOT NULL DEFAULT 0,
  locked_until  timestamptz,
  last_login_at timestamptz,
  deleted_at    timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_users_upd BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE refresh_tokens (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  family_id   uuid NOT NULL,                 -- untuk deteksi reuse
  token_hash  bytea NOT NULL UNIQUE,         -- SHA-256 token
  user_agent  text, ip inet,
  expires_at  timestamptz NOT NULL,
  revoked_at  timestamptz,
  replaced_by uuid REFERENCES refresh_tokens(id),
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_rt_user ON refresh_tokens(user_id);
CREATE INDEX idx_rt_family ON refresh_tokens(family_id);

CREATE TABLE one_time_tokens (          -- verifikasi email & reset password
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose text NOT NULL CHECK (purpose IN ('verify_email','reset_password')),
  token_hash bytea NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE addresses (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  label text NOT NULL, recipient text NOT NULL, phone text NOT NULL,
  province text NOT NULL, city text NOT NULL, district text NOT NULL,
  postal_code text NOT NULL, street text NOT NULL, notes text,
  area_id text,                           -- id area dari penyedia ongkir
  is_default boolean NOT NULL DEFAULT false,
  deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_addr_default ON addresses(user_id)
  WHERE is_default AND deleted_at IS NULL;

-- ===== KATALOG =====
CREATE TABLE categories (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug text NOT NULL UNIQUE, name text NOT NULL,
  parent_id uuid REFERENCES categories(id) ON DELETE SET NULL,
  sort_order int NOT NULL DEFAULT 0, is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE brands (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug text NOT NULL UNIQUE, name text NOT NULL
);

CREATE TABLE products (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug text NOT NULL UNIQUE,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  brand_id uuid REFERENCES brands(id),
  category_id uuid REFERENCES categories(id),
  flavor text,
  nutrition jsonb NOT NULL DEFAULT '{}',
  mascot_svg_key text,                      -- kunci aset ilustrasi kaleng
  requires_age_verification boolean NOT NULL DEFAULT false,
  is_active boolean NOT NULL DEFAULT true,
  search tsvector GENERATED ALWAYS AS
    (to_tsvector('simple', coalesce(name,'')||' '||coalesce(flavor,'')||' '||coalesce(description,''))) STORED,
  deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_products_search ON products USING gin(search);
CREATE INDEX idx_products_trgm ON products USING gin(name gin_trgm_ops);
CREATE INDEX idx_products_cat ON products(category_id) WHERE deleted_at IS NULL;

CREATE TABLE product_variants (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  sku text NOT NULL UNIQUE,
  volume_ml int NOT NULL CHECK (volume_ml > 0),
  pack_size int NOT NULL DEFAULT 1 CHECK (pack_size > 0),
  price bigint NOT NULL CHECK (price >= 0),
  compare_at_price bigint CHECK (compare_at_price IS NULL OR compare_at_price >= price),
  weight_gram int NOT NULL CHECK (weight_gram > 0),
  stock_on_hand int NOT NULL DEFAULT 0 CHECK (stock_on_hand >= 0),
  stock_reserved int NOT NULL DEFAULT 0 CHECK (stock_reserved >= 0),
  low_stock_threshold int NOT NULL DEFAULT 10,
  max_per_order int NOT NULL DEFAULT 24,
  is_active boolean NOT NULL DEFAULT true,
  version int NOT NULL DEFAULT 1,           -- optimistic locking
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (stock_reserved <= stock_on_hand)
);
CREATE INDEX idx_variants_product ON product_variants(product_id);

CREATE TABLE product_images (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  url text NOT NULL, alt text NOT NULL, sort_order int NOT NULL DEFAULT 0
);

CREATE TABLE stock_movements (              -- buku besar stok (append-only)
  id bigserial PRIMARY KEY,
  variant_id uuid NOT NULL REFERENCES product_variants(id),
  delta int NOT NULL,
  reason stock_reason NOT NULL,
  order_id uuid,                             -- FK ditambah setelah tabel orders
  actor_id uuid REFERENCES users(id),
  note text,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_stock_mv_variant ON stock_movements(variant_id, created_at DESC);

-- ===== KERANJANG, WISHLIST =====
CREATE TABLE carts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  updated_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE cart_items (
  cart_id uuid NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
  variant_id uuid NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
  quantity int NOT NULL CHECK (quantity BETWEEN 1 AND 99),
  added_price bigint NOT NULL,              -- untuk notifikasi perubahan harga
  PRIMARY KEY (cart_id, variant_id)
);

CREATE TABLE wishlist_items (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, product_id)
);

CREATE TABLE stock_alert_subscriptions (    -- "kabari saat restock"
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  variant_id uuid NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
  notified_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, variant_id)
);

-- ===== VOUCHER =====
CREATE TABLE vouchers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code citext NOT NULL UNIQUE,
  type voucher_type NOT NULL,
  value bigint NOT NULL CHECK (value > 0),   -- persen (1-100) atau rupiah
  max_discount bigint, min_spend bigint NOT NULL DEFAULT 0,
  usage_limit int, usage_count int NOT NULL DEFAULT 0,
  per_user_limit int NOT NULL DEFAULT 1,
  starts_at timestamptz NOT NULL, ends_at timestamptz NOT NULL,
  is_active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (ends_at > starts_at),
  CHECK (type <> 'percent' OR value BETWEEN 1 AND 100),
  CHECK (usage_limit IS NULL OR usage_count <= usage_limit)
);

-- ===== PESANAN =====
CREATE TABLE orders (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_no text NOT NULL UNIQUE,             -- mis. KLG-20261019-XXXX
  user_id uuid NOT NULL REFERENCES users(id),
  status order_status NOT NULL DEFAULT 'pending_payment',
  subtotal bigint NOT NULL CHECK (subtotal >= 0),
  discount bigint NOT NULL DEFAULT 0 CHECK (discount >= 0),
  shipping_cost bigint NOT NULL DEFAULT 0 CHECK (shipping_cost >= 0),
  total bigint NOT NULL CHECK (total >= 0),
  voucher_id uuid REFERENCES vouchers(id),
  address_snapshot jsonb NOT NULL,           -- salinan alamat saat pesan
  courier text, courier_service text,
  idempotency_key text NOT NULL,
  expires_at timestamptz NOT NULL,           -- batas bayar (reservasi stok)
  paid_at timestamptz, shipped_at timestamptz,
  delivered_at timestamptz, cancelled_at timestamptz,
  cancel_reason text,
  version int NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, idempotency_key),
  CHECK (total = subtotal - discount + shipping_cost)
);
CREATE INDEX idx_orders_user ON orders(user_id, created_at DESC);
CREATE INDEX idx_orders_status_exp ON orders(status, expires_at);

ALTER TABLE stock_movements
  ADD CONSTRAINT fk_sm_order FOREIGN KEY (order_id) REFERENCES orders(id);

CREATE TABLE order_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  variant_id uuid NOT NULL REFERENCES product_variants(id),
  product_name text NOT NULL, variant_label text NOT NULL, sku text NOT NULL,
  unit_price bigint NOT NULL CHECK (unit_price >= 0),
  quantity int NOT NULL CHECK (quantity > 0),
  line_total bigint NOT NULL CHECK (line_total = unit_price * quantity)
);
CREATE INDEX idx_oi_order ON order_items(order_id);

CREATE TABLE order_status_history (
  id bigserial PRIMARY KEY,
  order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  from_status order_status, to_status order_status NOT NULL,
  actor_id uuid REFERENCES users(id), note text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE voucher_redemptions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  voucher_id uuid NOT NULL REFERENCES vouchers(id),
  user_id uuid NOT NULL REFERENCES users(id),
  order_id uuid NOT NULL REFERENCES orders(id),
  amount bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (order_id)
);

-- ===== PEMBAYARAN & PENGIRIMAN =====
CREATE TABLE payments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id uuid NOT NULL REFERENCES orders(id),
  provider text NOT NULL, provider_ref text UNIQUE,
  method text, amount bigint NOT NULL CHECK (amount >= 0),
  status payment_status NOT NULL DEFAULT 'pending',
  paid_at timestamptz, raw_response jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_pay_order ON payments(order_id);

CREATE TABLE payment_webhook_events (       -- idempotensi webhook + jejak
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  provider text NOT NULL, event_id text NOT NULL,
  signature_valid boolean NOT NULL,
  payload jsonb NOT NULL,
  processed_at timestamptz, error text, attempts int NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (provider, event_id)
);

CREATE TABLE shipments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id uuid NOT NULL UNIQUE REFERENCES orders(id),
  courier text NOT NULL, tracking_no text,
  status text NOT NULL DEFAULT 'created',
  events jsonb NOT NULL DEFAULT '[]',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- ===== ULASAN =====
CREATE TABLE reviews (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  order_item_id uuid NOT NULL UNIQUE REFERENCES order_items(id),
  rating smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
  body text CHECK (char_length(body) <= 2000),
  is_visible boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- ===== NOTIFIKASI =====
CREATE TABLE notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code text NOT NULL,                        -- cocok dengan katalog 8.6
  severity text NOT NULL,
  title text NOT NULL, body text,
  data jsonb NOT NULL DEFAULT '{}',
  dedupe_key text,
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_notif_user_unread ON notifications(user_id, created_at DESC)
  WHERE read_at IS NULL;
CREATE UNIQUE INDEX uq_notif_dedupe ON notifications(user_id, dedupe_key)
  WHERE dedupe_key IS NOT NULL;

CREATE TABLE notification_preferences (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  channel notif_channel NOT NULL,
  category text NOT NULL,                    -- order, promo, stock, security, ...
  enabled boolean NOT NULL DEFAULT true,
  PRIMARY KEY (user_id, channel, category)
);
-- 'security' dan 'order' pada channel email/in_app tidak boleh dimatikan (dipaksa di service)

CREATE TABLE push_subscriptions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  endpoint text NOT NULL UNIQUE, p256dh text NOT NULL, auth text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE user_ui_settings (
  user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  notif_mode text NOT NULL DEFAULT 'normal' CHECK (notif_mode IN ('normal','calm','silent')),
  reduce_motion boolean NOT NULL DEFAULT false,
  sound_enabled boolean NOT NULL DEFAULT false
);

-- ===== AUDIT LOG (append-only) =====
CREATE TABLE audit_logs (
  id bigserial PRIMARY KEY,
  occurred_at timestamptz NOT NULL DEFAULT now(),
  actor_id uuid, actor_role text, ip inet, user_agent text, request_id text,
  action text NOT NULL,                      -- mis. product.price.update
  entity text NOT NULL, entity_id text,
  before jsonb, after jsonb,
  prev_hash bytea, row_hash bytea            -- rantai hash anti-manipulasi
);
CREATE INDEX idx_audit_entity ON audit_logs(entity, entity_id, occurred_at DESC);
CREATE INDEX idx_audit_actor ON audit_logs(actor_id, occurred_at DESC);

-- Larang UPDATE/DELETE pada audit_logs
CREATE OR REPLACE FUNCTION audit_immutable() RETURNS trigger AS $$
BEGIN RAISE EXCEPTION 'audit_logs is append-only'; END; $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_audit_no_mod BEFORE UPDATE OR DELETE ON audit_logs
  FOR EACH ROW EXECUTE FUNCTION audit_immutable();
```

### 11.3 Aturan Integritas Kritis
1. **Reservasi stok atomik** dalam satu transaksi:
```sql
-- reservasi (dijalankan di transaksi checkout, urutkan variant_id agar tidak deadlock)
UPDATE product_variants
   SET stock_reserved = stock_reserved + $qty, version = version + 1
 WHERE id = $variant_id
   AND is_active
   AND stock_on_hand - stock_reserved >= $qty;   -- 0 baris terpengaruh = stok kurang
```
2. **Pembayaran sukses:** `stock_on_hand -= qty` dan `stock_reserved -= qty`, catat `stock_movements(reason='sale')`.
3. **Kedaluwarsa/batal:** `stock_reserved -= qty`, catat `release`.
4. Perubahan status pesanan hanya lewat satu fungsi service yang memvalidasi **mesin status** (tabel transisi legal) dan menulis `order_status_history`.
5. Webhook diproses dengan `INSERT ... ON CONFLICT DO NOTHING` di `payment_webhook_events` agar idempoten.
6. Total pesanan dihitung di server dari harga DB; nilai dari klien diabaikan.
7. Cadangan: `pg_dump` harian + WAL archiving (PITR); uji restore tiap bulan.

### 11.4 Mesin Status Pesanan
| Dari | Ke yang diizinkan |
|---|---|
| pending_payment | paid, cancelled, expired |
| paid | processing, refund_requested, cancelled (oleh admin + refund) |
| processing | shipped, refund_requested |
| shipped | delivered |
| delivered | completed, refund_requested |
| refund_requested | refunded, (kembali ke status sebelumnya jika ditolak) |
| cancelled / expired / refunded / completed | — (final) |

---

## 12. Spesifikasi API (`/api/v1`)

Format error seragam (mengikuti RFC 9457 yang disederhanakan):
```json
{ "error": { "code": "CART_PRICE_CHANGED", "message": "Harga berubah", "fields": {"qty":"maks 24"}, "request_id": "..." } }
```
`error.code` **sama** dengan kode di katalog notifikasi frontend.

### 12.1 Autentikasi
| Method | Path | Auth | Keterangan |
|---|---|---|---|
| POST | /auth/register | — | Rate limit 5/jam/IP |
| POST | /auth/login | — | Lockout bertahap |
| POST | /auth/refresh | cookie | Rotasi + deteksi reuse |
| POST | /auth/logout | user | Cabut refresh token |
| POST | /auth/logout-all | user | Cabut semua keluarga token |
| POST | /auth/verify-email | — | Token sekali pakai |
| POST | /auth/forgot-password | — | Respons selalu sama |
| POST | /auth/reset-password | — | Token sekali pakai |
| GET | /me | user | Profil |
| PATCH | /me | user | Ubah profil |
| POST | /me/password | user | Ganti password (cabut sesi lain) |
| DELETE | /me | user | Hapus/anonimisasi akun |

### 12.2 Katalog
| Method | Path | Keterangan |
|---|---|---|
| GET | /products | Filter, sort, cursor pagination |
| GET | /products/{slug} | Detail + varian + stok publik (angka dibulatkan: "tersisa ≤10") |
| GET | /products/suggest?q= | Saran pencarian |
| GET | /categories, /brands | Daftar |
| GET | /products/{id}/reviews | Ulasan |

### 12.3 Keranjang, Wishlist
| Method | Path |
|---|---|
| GET/DELETE | /cart |
| PUT | /cart/items/{variantId} (set qty) |
| DELETE | /cart/items/{variantId} |
| POST | /cart/merge |
| POST | /cart/validate (cek harga & stok) |
| GET/PUT/DELETE | /wishlist/{productId} |
| POST/DELETE | /stock-alerts/{variantId} |

### 12.4 Checkout & Pesanan
| Method | Path | Keterangan |
|---|---|---|
| POST | /shipping/rates | Hitung ongkir (cache 5 menit) |
| POST | /vouchers/validate | Cek voucher pada keranjang |
| POST | /orders | Header `Idempotency-Key` wajib; reservasi stok |
| GET | /orders, /orders/{id} | Milik sendiri saja (cek kepemilikan di query) |
| POST | /orders/{id}/cancel | Hanya status yang diizinkan |
| POST | /orders/{id}/pay | Buat/ambil token pembayaran |
| POST | /orders/{id}/refund-request | |
| POST | /orders/{id}/reorder | |
| POST | /orders/{id}/items/{itemId}/review | Hanya jika completed |

### 12.5 Pembayaran & Webhook
| Method | Path | Keterangan |
|---|---|---|
| POST | /webhooks/midtrans | Verifikasi signature, idempoten, tanpa auth user |
| POST | /webhooks/shipping | Verifikasi signature |

### 12.6 Notifikasi
| Method | Path |
|---|---|
| GET | /notifications?cursor= |
| POST | /notifications/{id}/read, /notifications/read-all |
| GET | /notifications/stream (SSE) |
| GET/PUT | /notification-preferences |
| POST/DELETE | /push-subscriptions |
| GET/PUT | /me/ui-settings |

### 12.7 Admin (`role in (admin, staff)`, RBAC per endpoint)
| Method | Path | Catatan |
|---|---|---|
| CRUD | /admin/products, /admin/variants, /admin/categories | Setiap perubahan → audit log |
| POST | /admin/variants/{id}/stock | Wajib `reason` + `note` |
| GET | /admin/orders | Filter status/tanggal |
| POST | /admin/orders/{id}/status | Lewat mesin status |
| POST | /admin/orders/{id}/shipment | Input resi |
| CRUD | /admin/vouchers | |
| GET | /admin/audit-logs | Hanya baca, filter aktor/entitas/tanggal, ekspor CSV |
| GET | /admin/dashboard | Ringkasan |
| GET | /admin/users | Tanpa data sensitif berlebihan |

### 12.8 Rate Limit (Redis, sliding window)
| Endpoint | Batas |
|---|---|
| Umum (per IP) | 120 req/menit |
| `/auth/login` | 10/menit/IP + 5 gagal/15 menit/akun |
| `/auth/register`, `/forgot-password` | 5/jam/IP |
| `/vouchers/validate` | 20/menit/user (cegah tebak kode) |
| `/orders` | 10/menit/user |

---

## 13. Keamanan & **AUDIT Backend + Database**

### 13.1 Daftar Periksa Keamanan (OWASP ASVS L2 sebagai acuan)
**Autentikasi & sesi**
- [ ] argon2id, tanpa pembatasan karakter password, minimum 10 karakter
- [ ] Refresh token di-hash di DB, rotasi, reuse = cabut seluruh keluarga + notifikasi `AUTH_LOGIN_NEW_DEVICE`
- [ ] Cookie `HttpOnly; Secure; SameSite=Lax`; token CSRF (double-submit) untuk metode mutasi
- [ ] Pesan login generik; waktu respons konstan (hindari user enumeration)
- [ ] Lockout bertahap + CAPTCHA (Turnstile) setelah 5 gagal

**Otorisasi**
- [ ] RBAC di middleware **dan** pengecekan kepemilikan di setiap query (`WHERE user_id = $1`) untuk mencegah IDOR/BOLA
- [ ] Admin endpoint terpisah, uji negatif untuk tiap peran

**Input & output**
- [ ] Validasi skema ketat, batas ukuran body (1 MB), batas upload & verifikasi MIME sebenarnya
- [ ] Semua query lewat sqlc/parameterized; **dilarang** string concat SQL
- [ ] Output JSON saja; tidak ada HTML dari input pengguna; frontend tidak memakai `dangerouslySetInnerHTML`

**Transport & header**
- [ ] HTTPS wajib + HSTS
- [ ] CSP ketat (tanpa `unsafe-inline`, gunakan nonce), `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`, `frame-ancestors 'none'`
- [ ] CORS whitelist domain eksplisit

**Pembayaran & bisnis**
- [ ] Verifikasi signature webhook, cek `gross_amount` = total pesanan, cek order ada & status sesuai
- [ ] Idempotency untuk pembuatan pesanan dan webhook
- [ ] Harga/total selalu dihitung server
- [ ] Anti-abuse voucher (rate limit + kuota atomik: `UPDATE ... WHERE usage_count < usage_limit`)

**Rahasia & infrastruktur**
- [ ] Rahasia lewat environment/secret manager, tidak ada di repo (cek `gitleaks` di CI)
- [ ] Kontainer non-root, image minimal (distroless), dependensi dipin
- [ ] Database: user aplikasi **tanpa** hak DDL/superuser; user migrasi terpisah; koneksi TLS
- [ ] Backup terenkripsi, uji restore terjadwal

### 13.2 Audit Trail Aplikasi
- Middleware/service mencatat ke `audit_logs`: login admin, perubahan produk/harga/stok/voucher, perubahan status pesanan, perubahan peran, ekspor data, penghapusan akun.
- Simpan `before`/`after` (data sensitif disamarkan).
- **Rantai hash:** `row_hash = SHA256(prev_hash || canonical_json(row))`; job harian memverifikasi integritas rantai dan memicu `ADM_SUSPICIOUS` jika rusak.
- Retensi minimal 1 tahun; akses baca hanya admin.

### 13.3 Audit Kode & Database (Dikerjakan Sebelum Frontend Disambung)
| Langkah | Alat / Cara | Kriteria Lulus |
|---|---|---|
| Analisis statis Go | `golangci-lint` (errcheck, staticcheck, gosec, bodyclose, sqlclosecheck) | 0 temuan High |
| Kerentanan dependensi | `govulncheck ./...`, `npm audit --omit=dev` | 0 yang dapat dieksploitasi |
| Pemindaian rahasia | `gitleaks` | 0 temuan |
| Validasi SQL | `sqlc vet`, `sqlc compile` | Lulus |
| Review skema | Checklist 13.4 + `pg_stat_statements` + `EXPLAIN (ANALYZE, BUFFERS)` untuk 15 query terberat | Tidak ada seq scan pada tabel besar untuk jalur utama |
| Migrasi | Jalankan up → down → up pada DB kosong & DB berisi | Idempoten, tanpa error |
| Uji konkurensi stok | `k6`/test Go: 200 pembeli serentak berebut 50 stok | Tidak ada oversell, `stock_reserved ≤ stock_on_hand` selalu |
| Uji idempotensi | Kirim `POST /orders` 10× paralel dengan kunci sama | 1 pesanan |
| Uji webhook | Replay, signature salah, jumlah salah, urutan terbalik | Status benar & aman |
| Uji otorisasi | Matriks peran × endpoint, uji IDOR otomatis | Semua akses ilegal ditolak (403/404) |
| DAST | OWASP ZAP baseline terhadap staging | 0 High/Medium belum ditangani |
| Beban | `k6`: 200 rps campuran selama 10 menit | Memenuhi SLO bagian 7 |
| Race detector | `go test -race ./...` | Lulus |
| Fuzz ringan | `go test -fuzz` pada parser input (voucher, pencarian) | Tanpa panic |
| Chaos kecil | Matikan Redis / lambatkan DB | Degradasi anggun, notifikasi `NET_*`/`API_*` muncul benar |
| Audit trail | Uji bahwa setiap aksi admin menghasilkan baris audit & rantai hash valid | 100% |
| Pemulihan | Restore backup ke lingkungan bersih | Data konsisten, RTO ≤ 1 jam |

Hasil audit ditulis di `docs/AUDIT-REPORT.md` (temuan, severity, status perbaikan, bukti). **Gerbang:** frontend produksi tidak boleh dirilis sebelum laporan ini bersih dari High/Critical.

### 13.4 Checklist Review Skema
- [ ] Setiap FK terindeks; tidak ada kolom `text` bebas tanpa batas pada input publik
- [ ] `CHECK` constraint untuk angka (harga ≥ 0, stok ≥ 0, total konsisten)
- [ ] Soft delete konsisten dan indeks parsial menghormatinya
- [ ] Tidak ada `float` untuk uang
- [ ] Snapshot data pesanan (alamat, nama produk, harga) agar riwayat tidak berubah
- [ ] Tabel besar (notifications, audit_logs, stock_movements) punya rencana partisi bulanan/retensi
- [ ] Semua timestamp `timestamptz` (UTC)
- [ ] Peran DB least-privilege terdokumentasi

---

## 14. Rencana Pengujian

| Level | Cakupan | Alat |
|---|---|---|
| Unit (Go) | Service: harga, voucher, mesin status, reservasi | `testing`, `testify` |
| Integrasi (Go) | Repo + API dengan Postgres/Redis asli | `testcontainers-go` |
| Kontrak | Respons API = `openapi.yaml` | `schemathesis` / `oapi-codegen` |
| Unit UI | Komponen, katalog notifikasi, antrean | Vitest + Testing Library |
| A11y | Semua halaman & modal | `axe` + uji keyboard manual |
| E2E | Daftar → belanja → voucher → bayar (sandbox) → notifikasi → pesanan selesai | Playwright |
| Visual | Snapshot komponen utama + notifikasi | Playwright screenshot |
| Performa FE | Lighthouse CI, profil 4× CPU throttle | Lighthouse CI |
| Beban BE | Skenario 12.8 | k6 |

Skenario E2E wajib memverifikasi **muncul-nya notifikasi yang tepat** di setiap langkah (rujuk kode katalog 8.6).

---

## 15. Deployment & Operasional

- **Lingkungan:** local (docker-compose), staging, production.
- **docker-compose (dev):** `postgres`, `redis`, `mailhog`, `minio`, `api`, `worker`, `web`.
- **CI (GitHub Actions):** lint → test → gosec/govulncheck/gitleaks → build image → migrasi di staging → E2E → deploy.
- **CD:** deploy bertahap (blue/green atau rolling), migrasi dijalankan sebelum rilis dengan pola *expand → migrate → contract*.
- **Monitoring:** dashboard (RPS, p95, error rate, antrean job, SSE connections), alert ke Telegram/Slack/email.
- **CDN** untuk aset statis, cache `immutable` untuk berkas ber-hash.
- **Pembersihan terjadwal:** expire pesanan (tiap menit), hapus token kedaluwarsa (harian), arsip notifikasi >90 hari, verifikasi rantai audit (harian).

---

## 16. Struktur Halaman (Sitemap)

```
/                       Beranda (hero maskot, produk unggulan, promo)
/produk                 Katalog + filter
/produk/:slug           Detail produk
/keranjang              Keranjang
/checkout               Checkout bertahap
/pesanan                Riwayat pesanan
/pesanan/:id            Detail + pelacakan
/wishlist               Wishlist
/akun                   Profil, alamat, keamanan, notifikasi & tampilan
/notifikasi             Pusat notifikasi lengkap
/masuk  /daftar  /lupa-password  /verifikasi-email
/bantuan  /syarat  /privasi
/404  /500  /offline  /maintenance
/admin/*                Dashboard, produk, stok, pesanan, voucher, pengguna, audit log
```

---

## 17. Roadmap & Milestone

Urutan sengaja **backend-first**, sesuai permintaan: backend dan database selesai & ter-audit dulu, baru frontend disambung penuh.

| Fase | Isi | Hasil / Gerbang |
|---|---|---|
| **M0 — Fondasi** (minggu 1) | Repo, CI, docker-compose, konfigurasi, logger, skema DB v1, migrasi, OpenAPI awal | Lingkungan jalan; migrasi up/down lulus |
| **M0.5 — Pinterest & Brand** (paralel M0–M1) | Moodboard, token desain, logo SVG + animasi, maskot | Moodboard & token disetujui |
| **M1 — Backend inti** (minggu 2–3) | Auth, akun, alamat, katalog, pencarian, keranjang, wishlist | Uji unit+integrasi lulus |
| **M2 — Transaksi** (minggu 4–5) | Voucher, ongkir, pesanan + reservasi stok, pembayaran + webhook, job expire, mesin status | Uji konkurensi & idempotensi lulus |
| **M3 — Notifikasi backend** (minggu 5–6) | Tabel & layanan notifikasi, SSE, email, Web Push, preferensi, templat | Semua kode katalog bisa dipicu dari backend |
| **M4 — Admin & Audit trail** (minggu 6–7) | Panel API admin, audit log berantai, dashboard | Matriks otorisasi lulus |
| **M5 — AUDIT** (minggu 7–8) | Seluruh tabel 13.3, perbaikan temuan, `AUDIT-REPORT.md` | **0 High/Critical** |
| **M6 — Frontend inti** (minggu 8–10) | Design system, NotificationProvider + seluruh komponen, halaman publik, keranjang, checkout, akun | Katalog notifikasi 100% terhubung |
| **M7 — Animasi & polish** (minggu 10–11) | Animasi 9.2, transisi, empty state, mode reduce-motion | Anggaran performa 9.1 terpenuhi |
| **M8 — Admin UI & E2E** (minggu 11–12) | UI admin, Playwright E2E, a11y, Lighthouse | Semua kriteria 1.4 terpenuhi |
| **M9 — Beta & rilis** (minggu 13) | Uji pengguna 10–20 orang, perbaikan, rilis bertahap | Go-live |

### Definition of Done (per fitur)
1. Kode lolos lint + test + review.
2. Ada test unit/integrasi dan, bila relevan, E2E.
3. Semua status (loading, kosong, sukses, error, offline) punya tampilan **dan** notifikasi yang sesuai.
4. A11y diperiksa (keyboard, pembaca layar, kontras, reduced motion).
5. Audit log tercatat untuk aksi sensitif.
6. Dokumentasi OpenAPI & katalog notifikasi diperbarui.

---

## 18. Risiko & Mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Animasi membuat halaman lambat | Konversi turun | Anggaran 9.1, uji CPU throttle, reduce-motion |
| Terlalu banyak notifikasi mengganggu | User kesal | Antrean, dedupe, mode Tenang/Hening, throttle promo |
| Oversell stok | Pesanan batal, citra buruk | Reservasi atomik + uji konkurensi |
| Webhook ganda/terlambat | Status salah | Idempotensi + rekonsiliasi job tiap 5 menit |
| Aset Pinterest dipakai langsung | Masalah hak cipta | Hanya referensi; buat aset orisinal |
| Ilustrasi tidak konsisten | Terlihat "slop" | Satu aturan garis/warna, review desain tiap kaleng baru |
| Kebocoran data | Hukum & reputasi | Bagian 13, minimisasi data, enkripsi, UU PDP |
| Ongkos kurir berubah | Ongkir salah | Cache pendek + hitung ulang di checkout |

---

## 19. Konten & Copywriting

Nada suara: **ramah, jenaka, singkat, tidak norak.** Maksimal satu candaan per layar; pesan error dan keamanan tidak boleh bercanda berlebihan.
Aturan: kalimat aktif, sebut langkah berikutnya, hindari singkatan membingungkan, angka rupiah selalu `Rp12.500` (pemisah titik).

Contoh pasangan baik vs buruk:
- Buruk: "Error 500. Terjadi kesalahan."
- Baik: "Server lagi batuk. Coba lagi sebentar ya." [Ulangi]

---

## 20. Pertanyaan Terbuka (Mohon Dikonfirmasi)

1. Nama brand final dan apakah sudah ada logo/maskot sendiri?
2. Apakah menjual minuman **beralkohol**? (menentukan age gate dan regulasi)
3. Payment gateway pilihan: Midtrans atau Xendit?
4. Penyedia ongkir: Biteship atau RajaOngkir? Apakah ada gudang lebih dari satu?
5. Hosting yang dipakai (VPS, cloud, atau PaaS)?
6. Perlu bahasa Inggris sejak awal?
7. Siapa yang menggambar ilustrasi kaleng: ilustrator internal/freelancer, atau dibuat dari awal oleh tim desain?
8. Perkiraan jumlah SKU dan trafik harian (menentukan skala infrastruktur)?

---

## 21. Lampiran

### A. Contoh Keyframe Ringan
```css
/* Logo jatuh + pantul */
@keyframes can-drop {
  0%   { transform: translateY(-120px) scale(1, 1); opacity: 0; }
  55%  { transform: translateY(0) scale(1.12, .86); opacity: 1; }
  72%  { transform: translateY(-14px) scale(.96, 1.06); }
  100% { transform: translateY(0) scale(1, 1); }
}
.logo-mark { animation: can-drop 900ms var(--ease-out) both; }

/* Tombol: squash saat ditekan */
.btn { transition: transform var(--dur-fast) var(--ease-pop), box-shadow var(--dur-fast); }
.btn:hover  { transform: translate(-2px,-2px); box-shadow: 6px 6px 0 var(--ink); }
.btn:active { transform: translate(2px,2px) scale(1, .94); box-shadow: 0 0 0 var(--ink); }

/* Toast masuk */
@keyframes toast-in {
  from { transform: translateY(-16px) scale(.96); opacity: 0; }
  to   { transform: translateY(0) scale(1); opacity: 1; }
}
.toast { animation: toast-in 280ms var(--ease-pop) both; }
```

### B. Kerangka Hook Notifikasi
```ts
// notifications/useNotify.ts
export function useNotify() {
  const { enqueue } = useNotificationStore();
  return (code: keyof typeof CATALOG, vars: Record<string, string | number> = {}) => {
    const def = CATALOG[code];
    enqueue({
      ...def,
      code,
      title: render(def.title, vars),
      message: def.message && render(def.message, vars),
      dedupeKey: def.dedupeKey?.(vars) ?? code,
    });
  };
}
```

### C. Kerangka Transaksi Checkout (Go, ringkas)
```go
func (s *OrderService) Create(ctx context.Context, in CreateOrderInput) (*Order, error) {
    return s.tx.WithTx(ctx, func(q *repo.Queries) (*Order, error) {
        // 1. idempotency: kembalikan pesanan lama bila key sama
        if o, err := q.GetOrderByIdemKey(ctx, in.UserID, in.IdemKey); err == nil { return o, nil }
        // 2. ambil keranjang & harga TERBARU dari DB (abaikan harga klien)
        items, _ := q.LoadCartWithPrices(ctx, in.UserID)
        // 3. urutkan variant_id, reservasi stok atomik (UPDATE ... WHERE available >= qty)
        for _, it := range sortByVariant(items) {
            if n, _ := q.ReserveStock(ctx, it.VariantID, it.Qty); n == 0 {
                return nil, apperr.New("STOCK_CHANGED_CHECKOUT", it.Name)
            }
        }
        // 4. voucher: klaim kuota atomik, hitung diskon di server
        // 5. buat order + items + status history + stock_movements + audit
        // 6. jadwalkan job expire (expires_at = now+15m); publikasikan notifikasi CHK_ORDER_CREATED
        return order, nil
    })
}
```

### D. Daftar Aset yang Harus Dibuat
- [ ] Logo (3 varian × 3 warna) + favicon + OG image
- [ ] Maskot utama (5 ekspresi: senang, kaget, sedih, tidur, rayakan)
- [ ] Ilustrasi kaleng per rasa (minimal 8 untuk peluncuran)
- [ ] Ilustrasi empty state (keranjang, cari, 404, 500, offline, maintenance, notifikasi kosong)
- [ ] Set ikon (±40 ikon, stroke seragam)
- [ ] Ikon notifikasi animasi (sukses, info, peringatan, error)
- [ ] Template email (verifikasi, reset, pesanan dibuat, dibayar, dikirim, sampai, refund, login baru)
- [ ] Bunyi notifikasi opsional (2 file kecil <10 KB)

---

*Akhir dokumen. Setelah pertanyaan di bagian 20 dijawab, dokumen ini siap dipecah menjadi tiket kerja per milestone.*
