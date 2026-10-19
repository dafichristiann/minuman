# Moodboard (§4) — Gerbang Sebelum Desain

> **PRD §4:** referensi Pinterest **wajib** sebelum desain dimulai.
> **PRD §4.3:** moodboard + token desain disetujui sebelum **satu pun** layar dirancang.

Dokumen ini adalah lembar kerja untuk gerbang tersebut. Isi kolomnya secara
manual — tautan pin, nama kreator, dan catatan.

> ⚠️ **Hak cipta.** Gambar di Pinterest milik kreatornya. Papan ini untuk
> referensi *mood*, **bukan** untuk disalin. Jangan mengunduh dan memakai pin
> sebagai aset final. Catat nama kreator untuk kredit; bila suatu gaya akan
> didekati terlalu dekat, hubungi kreatornya untuk lisensi atau komisi.

Status gerbang: ⬜ **belum dijalankan** — desain (M6) tidak boleh dimulai.

---

## 1. Struktur papan

| Board | Isi | Target pin | Terkumpul | 10 terpilih | Tautan papan |
|---|---|---|---|---|---|
| `01-logo-karakter` | Maskot kaleng, logo kartun, wordmark tebal | 25 | 0 | ⬜ | |
| `02-warna-palet` | Palet cream + warna jenuh + outline gelap | 15 | 0 | ⬜ | |
| `03-ilustrasi-kaleng` | Desain kemasan kaleng ilustratif | 30 | 0 | ⬜ | |
| `04-ui-cartoon` | Web/app UI neo-brutalism lucu | 25 | 0 | ⬜ | |
| `05-motion-2d` | Gerak: squash & stretch, bounce, pop | 20 | 0 | ⬜ | |
| `06-notifikasi-ui` | Toast, modal, badge, empty state lucu | 20 | 0 | ⬜ | |
| `07-hindari` | Contoh yang terlalu generik (sebagai kontras) | 10 | 0 | ⬜ | |
| **Total** | | **145** | **0** | | |

Target awal ≥ 150 pin untuk seluruh papan.

---

## 2. Kata kunci pencarian

Kombinasi siap tempel ke kotak pencarian Pinterest.

### Logo & karakter → `01-logo-karakter`
- `rubber hose character logo`
- `cute can mascot illustration`
- `soda can character design flat`
- `retro cartoon mascot logo bold outline`
- `beverage brand mascot vector`

### Kemasan & ilustrasi → `03-ilustrasi-kaleng`
- `illustrated soda can packaging design`
- `drink can label illustration fruit character`
- `energy drink can art playful`
- `canned tea packaging illustration`

### UI & web → `04-ui-cartoon`
- `neo brutalism web design playful`
- `cartoon ecommerce website ui`
- `hard shadow ui design bold border`
- `playful landing page illustration beverage`
- `sticker style ui components`

### Motion → `05-motion-2d`
- `squash and stretch animation reference`
- `2d bounce animation principles`
- `micro interaction cartoon button`
- `soda bubble animation loop`

### Notifikasi → `06-notifikasi-ui`
- `toast notification playful ui`
- `cute empty state illustration`
- `error 404 page illustration funny`
- `success modal confetti ui`

### Palet → `02-warna-palet`
- `retro cream color palette`
- `bold saturated color palette dark outline`
- `cartoon color palette modern`

### Yang harus dihindari → `07-hindari`
- `corporate gradient website`
- `glassmorphism dashboard`
- `generic minimal app ui`
- `stock photo ecommerce hero`

**Tips:** tambahkan kata `dribbble`, `behance`, atau `vector` untuk menyaring
hasil berkualitas, dan pakai fitur **More ideas** Pinterest dari pin terbaik
untuk menemukan gaya serupa.

---

## 3. Proses kurasi (§4.3) — checklist

- [ ] **1.** Kumpulkan ≥ 150 pin awal ke board (lihat tabel §1).
- [ ] **2.** Pilih 10 pin terbaik **per board** → susun menjadi **moodboard
      tunggal** (1 frame Figma). Tautkan di kolom "10 terpilih".
- [ ] **3.** Sepakati **5 kata sifat** untuk merek.
      Isi di sini: `_______`, `_______`, `_______`, `_______`, `_______`
      (contoh dari PRD: *lucu, tebal, lincah, segar, jujur*)
- [ ] **4.** Ekstrak atribut visual ke token — isi tabel §4.
- [ ] **5.** Setiap aset final diberi catatan **"terinspirasi dari"** (nama
      kreator + tautan) di dokumen ini.
- [ ] **6.** **GERBANG:** moodboard + token disetujui. Baru M6 dimulai.

### Papan `07-hindari` — mengapa

Papan ini bukan lelucon. Ia adalah pagar: bila sebuah layar mulai terlihat
seperti pin di sana, layar itu gagal. Rujuk juga larangan PRD §3:

- tidak ada gradien generik sebagai pengganti keputusan warna;
- tidak ada glassmorphism;
- tidak ada emoji sebagai ikon (ikon = SVG yang digambar);
- tidak ada Lottie/GIF untuk logo;
- tidak ada teks *placeholder*.

---

## 4. Ekstraksi token visual

Hasil langkah 4. Nilai ini mengisi ulang `apps/web/src/styles/tokens.css`
(lihat token dasar dari §3.4 yang sudah ada di sana).

| Atribut | Temuan dari moodboard | Token | Catatan |
|---|---|---|---|
| Warna dasar | | `--cream` | |
| Warna aksen 1 | | `--tomato` | |
| Warna aksen 2 | | `--berry` | |
| Warna dukung 1 | | `--sky` | |
| Warna dukung 2 | | `--lemon` | |
| Warna dukung 3 | | `--mint` | |
| Warna garis / teks | | `--ink` | |
| Tebal garis | | `--border-w` = ? px | |
| Radius sudut | | `--radius-*` | |
| Gaya bayangan | | `--shadow-hard` | contoh: `3px 3px 0` |
| Bentuk mata maskot | | — | |
| Proporsi kepala/badan | | — | |
| Ketebalan huruf display | | `--font-display` | |
| Karakter gerak | | `--ease-*` | |

---

## 5. Catatan kredit & inspirasi

Tempel di sini saat langkah 5 dijalankan.

| Aset final | Terinspirasi dari (kreator + tautan) | Lisensi / izin | Tanggal |
|---|---|---|---|
| | | | |

---

## 6. Gerbang

| Pertanyaan | Jawaban |
|---|---|
| ≥ 150 pin terkumpul? | ⬜ |
| 10 pin terbaik per board dipilih & disusun 1 frame? | ⬜ |
| 5 kata sifat disepakati? | ⬜ |
| Token visual diekstrak ke `tokens.css`? | ⬜ |
| Kredit & izin dicatat? | ⬜ |
| **Moodboard disetujui?** | ⬜ |
| **Desain layar (M6) boleh dimulai?** | **belum** |

Setelah seluruh centang ✅, baru `apps/web` diisi sistem desain penuh dan
halaman-halaman produk dibuat.
