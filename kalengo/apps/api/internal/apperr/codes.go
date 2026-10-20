package apperr

// ============================================================
// Registry kode error backend — sinkron dengan katalog §8.6.
//
// HANYA kode yang bisa dikembalikan API yang didaftarkan di sini. Notifikasi
// sukses/info/promo (M3) punya registry terpisah di internal/notify.
//
// Kode yang ditandai `// [TAMBAHAN]` BELUM ada di §8.6 dan harus ditambahkan ke
// notifications.catalog.ts frontend pada M6 agar satu sumber kebenaran tetap utuh.
// Daftar lengkapnya ada di docs/DEVIATIONS.md.
// ============================================================

// Autentikasi & akun.
const (
	CodeAuthEmailTaken      = "AUTH_EMAIL_TAKEN"
	CodeAuthLoginFail       = "AUTH_LOGIN_FAIL"
	CodeAuthLoginLocked     = "AUTH_LOGIN_LOCKED"
	CodeAuthVerifyExpired   = "AUTH_VERIFY_EXPIRED"
	CodeAuthSessionExpired  = "AUTH_SESSION_EXPIRED"
	CodeAuthPasswordWeak    = "AUTH_PASSWORD_WEAK"
	CodeAuthRefreshReuse    = "AUTH_REFRESH_REUSE"    // [TAMBAHAN]
	CodeAuthEmailUnverified = "AUTH_EMAIL_UNVERIFIED" // [TAMBAHAN]
	CodeAuthCSRFInvalid     = "AUTH_CSRF_INVALID"     // [TAMBAHAN]
	CodeAccDeleteConfirm    = "ACC_DELETE_CONFIRM"
)

// Katalog, keranjang, wishlist.
const (
	CodeCartQtyMax           = "CART_QTY_MAX"
	CodeCartPriceChanged     = "CART_PRICE_CHANGED"
	CodeCartItemUnavailable  = "CART_ITEM_UNAVAILABLE"
	CodeCartEmpty            = "CART_EMPTY" // [TAMBAHAN]
	CodeWishLoginReq         = "WISH_LOGIN_REQ"
	CodeStockOut             = "STOCK_OUT"
	CodeStockChangedCheckout = "STOCK_CHANGED_CHECKOUT"
)

// Voucher & promo.
const (
	CodeVoucherInvalid       = "VOUCHER_INVALID"
	CodeVoucherExpired       = "VOUCHER_EXPIRED"
	CodeVoucherMinSpend      = "VOUCHER_MIN_SPEND"
	CodeVoucherUsedUp        = "VOUCHER_USED_UP"
	CodeVoucherLimitUser     = "VOUCHER_LIMIT_USER"
	CodeVoucherNotApplicable = "VOUCHER_NOT_APPLICABLE" // [TAMBAHAN]
)

// Checkout & pembayaran.
const (
	CodeChkAddressReq       = "CHK_ADDRESS_REQ"
	CodeChkShippingFail     = "CHK_SHIPPING_FAIL"
	CodeChkAreaUnserved     = "CHK_AREA_UNSERVED"
	CodeChkTermsReq         = "CHK_TERMS_REQ"
	CodeChkDuplicateBlocked = "CHK_DUPLICATE_BLOCKED"
	CodeChkMinOrder         = "CHK_MIN_ORDER" // [TAMBAHAN]
	CodePayFailed           = "PAY_FAILED"
	CodePayExpired          = "PAY_EXPIRED"
	CodePayProviderError    = "PAY_PROVIDER_ERROR"            // [TAMBAHAN]
	CodePayAlreadyPaid      = "PAY_ALREADY_PAID"              // [TAMBAHAN]
	CodePayWebhookSignature = "PAY_WEBHOOK_SIGNATURE_INVALID" // [TAMBAHAN]
	CodePayAmountMismatch   = "PAY_AMOUNT_MISMATCH"           // [TAMBAHAN]
)

// Pesanan & ulasan.
const (
	CodeOrdCancelDenied      = "ORD_CANCEL_DENIED"
	CodeOrdNotFound          = "ORD_NOT_FOUND"          // [TAMBAHAN]
	CodeOrdInvalidTransition = "ORD_INVALID_TRANSITION" // [TAMBAHAN]
	CodeOrdRefundNotAllowed  = "ORD_REFUND_NOT_ALLOWED" // [TAMBAHAN]
	CodeRevNotEligible       = "REV_NOT_ELIGIBLE"
	CodeRevAlreadySubmitted  = "REV_ALREADY_SUBMITTED" // [TAMBAHAN]
)

// Sistem, jaringan, umum.
const (
	CodeAPIRateLimit           = "API_RATE_LIMIT"
	CodeAPIServerError         = "API_SERVER_ERROR"
	CodeAPIValidation          = "API_VALIDATION"
	CodeAPIForbidden           = "API_FORBIDDEN"
	CodeAPINotFound            = "API_NOT_FOUND"
	CodeAPIBadRequest          = "API_BAD_REQUEST"         // [TAMBAHAN]
	CodeAPIMethodNotAllowed    = "API_METHOD_NOT_ALLOWED"  // [TAMBAHAN]
	CodeAPIServiceUnavailable  = "API_SERVICE_UNAVAILABLE" // [TAMBAHAN]
	CodeAPIUnsupportedMedia    = "API_UNSUPPORTED_MEDIA"   // [TAMBAHAN]
	CodeAPITimeout             = "API_TIMEOUT"             // [TAMBAHAN]
	CodeAPIMaintenance         = "API_MAINTENANCE"
	CodeAPIIdempotencyConflict = "API_IDEMPOTENCY_CONFLICT" // [TAMBAHAN]
	CodeSysUploadTooBig        = "SYS_UPLOAD_TOO_BIG"
	CodeAgeGate                = "AGE_GATE"
)

// Admin.
const (
	CodeAdmPriceChangeConfirm = "ADM_PRICE_CHANGE_CONFIRM"
	CodeAdmAuditChainBroken   = "ADM_AUDIT_CHAIN_BROKEN" // [TAMBAHAN]
	CodeAdmStockReasonReq     = "ADM_STOCK_REASON_REQ"   // [TAMBAHAN]
)

func init() {
	register(
		// ---------- A. Autentikasi & akun ----------
		Definition{Code: CodeAuthEmailTaken, Status: 409, Severity: SeverityError, Category: CategoryAuth,
			Message: "Email ini sudah terdaftar. Mau masuk saja?"},
		Definition{Code: CodeAuthLoginFail, Status: 401, Severity: SeverityError, Category: CategoryAuth,
			// Sengaja generik: tidak membocorkan mana yang salah (§13.1).
			Message: "Email atau password tidak cocok."},
		Definition{Code: CodeAuthLoginLocked, Status: 423, Severity: SeverityCritical, Category: CategoryAuth, Persist: true,
			Message: "Terlalu banyak percobaan. Coba lagi dalam {menit} menit."},
		Definition{Code: CodeAuthVerifyExpired, Status: 410, Severity: SeverityWarning, Category: CategoryAuth,
			Message: "Tautan sudah kedaluwarsa. Kirim yang baru?"},
		Definition{Code: CodeAuthSessionExpired, Status: 401, Severity: SeverityCritical, Category: CategoryAuth, Persist: true,
			Message: "Sesi berakhir. Masuk lagi ya, keranjangmu aman."},
		Definition{Code: CodeAuthPasswordWeak, Status: 422, Severity: SeverityWarning, Category: CategoryAuth,
			Message: "Tambah panjang & angka biar lebih kuat."},
		Definition{Code: CodeAuthRefreshReuse, Status: 401, Severity: SeverityCritical, Category: CategoryAuth, Persist: true,
			Message: "Sesi ini tidak valid lagi. Demi keamanan, masuk ulang ya."},
		Definition{Code: CodeAuthEmailUnverified, Status: 403, Severity: SeverityWarning, Category: CategoryAuth, Persist: true,
			Message: "Verifikasi email dulu ya sebelum lanjut."},
		Definition{Code: CodeAuthCSRFInvalid, Status: 403, Severity: SeverityError, Category: CategoryAuth,
			Message: "Permintaan ditolak karena token keamanan tidak cocok. Muat ulang halaman."},
		Definition{Code: CodeAccDeleteConfirm, Status: 409, Severity: SeverityCritical, Category: CategoryAccount,
			Message: "Hapus akun permanen? Ketik HAPUS untuk lanjut."},

		// ---------- B. Katalog, keranjang, wishlist ----------
		Definition{Code: CodeCartQtyMax, Status: 409, Severity: SeverityWarning, Category: CategoryCatalog,
			Message: "Maksimal {n} per pesanan untuk {produk}."},
		Definition{Code: CodeCartPriceChanged, Status: 409, Severity: SeverityWarning, Category: CategoryCatalog,
			Message: "Harga {produk} berubah dari {lama} jadi {baru}."},
		Definition{Code: CodeCartItemUnavailable, Status: 409, Severity: SeverityError, Category: CategoryCatalog,
			Message: "{produk} sudah tidak tersedia dan dihapus dari keranjang."},
		Definition{Code: CodeCartEmpty, Status: 409, Severity: SeverityWarning, Category: CategoryCatalog,
			Message: "Keranjangmu masih kosong. Pilih kaleng dulu ya."},
		Definition{Code: CodeWishLoginReq, Status: 401, Severity: SeverityInfo, Category: CategoryCatalog,
			Message: "Masuk dulu biar wishlist-mu tersimpan."},
		Definition{Code: CodeStockOut, Status: 409, Severity: SeverityError, Category: CategoryCatalog,
			Message: "Habis. Kabari saya saat tersedia?"},
		Definition{Code: CodeStockChangedCheckout, Status: 409, Severity: SeverityError, Category: CategoryCatalog, Persist: true,
			Message: "Stok {produk} tinggal {n}. Sesuaikan jumlahnya."},

		// ---------- C. Voucher & promo ----------
		Definition{Code: CodeVoucherInvalid, Status: 422, Severity: SeverityError, Category: CategoryVoucher,
			Message: "Kode voucher tidak dikenali."},
		Definition{Code: CodeVoucherExpired, Status: 410, Severity: SeverityError, Category: CategoryVoucher,
			Message: "Voucher sudah kedaluwarsa."},
		Definition{Code: CodeVoucherMinSpend, Status: 422, Severity: SeverityWarning, Category: CategoryVoucher,
			Message: "Belanja minimal {Rp} untuk voucher ini."},
		Definition{Code: CodeVoucherUsedUp, Status: 409, Severity: SeverityError, Category: CategoryVoucher,
			Message: "Voucher sudah habis dipakai."},
		Definition{Code: CodeVoucherLimitUser, Status: 409, Severity: SeverityError, Category: CategoryVoucher,
			Message: "Kamu sudah memakai voucher ini."},
		Definition{Code: CodeVoucherNotApplicable, Status: 422, Severity: SeverityWarning, Category: CategoryVoucher,
			Message: "Voucher {kode} tidak berlaku untuk item di keranjangmu."},

		// ---------- D. Checkout & pembayaran ----------
		Definition{Code: CodeChkAddressReq, Status: 422, Severity: SeverityError, Category: CategoryCheckout,
			Message: "Pilih alamat pengiriman dulu."},
		Definition{Code: CodeChkShippingFail, Status: 502, Severity: SeverityError, Category: CategoryCheckout,
			Message: "Ongkir gagal dihitung. Coba lagi ya."},
		Definition{Code: CodeChkAreaUnserved, Status: 422, Severity: SeverityWarning, Category: CategoryCheckout,
			Message: "Belum bisa kirim ke area ini."},
		Definition{Code: CodeChkTermsReq, Status: 422, Severity: SeverityError, Category: CategoryCheckout,
			Message: "Centang persetujuan dulu."},
		Definition{Code: CodeChkDuplicateBlocked, Status: 409, Severity: SeverityInfo, Category: CategoryCheckout,
			Message: "Pesanan sedang diproses, tunggu sebentar."},
		Definition{Code: CodeChkMinOrder, Status: 422, Severity: SeverityWarning, Category: CategoryCheckout,
			Message: "Minimal belanja {Rp} untuk bisa checkout."},
		Definition{Code: CodePayFailed, Status: 402, Severity: SeverityError, Category: CategoryPayment, Persist: true,
			Message: "Pembayaran gagal. Coba metode lain?"},
		Definition{Code: CodePayExpired, Status: 410, Severity: SeverityError, Category: CategoryPayment, Persist: true,
			Message: "Waktu pembayaran habis. Pesanan dibatalkan."},
		Definition{Code: CodePayProviderError, Status: 502, Severity: SeverityError, Category: CategoryPayment,
			Message: "Pembayaran sedang tidak bisa diproses. Coba beberapa saat lagi."},
		Definition{Code: CodePayAlreadyPaid, Status: 409, Severity: SeverityInfo, Category: CategoryPayment,
			Message: "Pesanan ini sudah dibayar."},
		Definition{Code: CodePayWebhookSignature, Status: 401, Severity: SeverityCritical,
			Category: CategoryPayment, Persist: true,
			Message: "Pemberitahuan pembayaran ditolak: tanda tangan tidak sah."},
		Definition{Code: CodePayAmountMismatch, Status: 409, Severity: SeverityCritical,
			Category: CategoryPayment, Persist: true,
			Message: "Pemberitahuan pembayaran ditolak: nominal tidak cocok dengan pesanan."},

		// ---------- E. Pesanan & ulasan ----------
		Definition{Code: CodeOrdCancelDenied, Status: 409, Severity: SeverityError, Category: CategoryOrder,
			Message: "Pesanan sudah dikirim, tidak bisa dibatalkan."},
		Definition{Code: CodeOrdNotFound, Status: 404, Severity: SeverityInfo, Category: CategoryOrder,
			Message: "Pesanan tidak ditemukan."},
		Definition{Code: CodeOrdInvalidTransition, Status: 409, Severity: SeverityError, Category: CategoryOrder,
			Message: "Status pesanan tidak bisa diubah ke {status}."},
		Definition{Code: CodeOrdRefundNotAllowed, Status: 409, Severity: SeverityWarning, Category: CategoryOrder,
			Message: "Pesanan ini belum bisa diajukan refund."},
		Definition{Code: CodeRevNotEligible, Status: 403, Severity: SeverityWarning, Category: CategoryOrder,
			Message: "Ulasan dibuka setelah pesanan selesai."},
		Definition{Code: CodeRevAlreadySubmitted, Status: 409, Severity: SeverityWarning, Category: CategoryOrder,
			Message: "Kamu sudah mengulas item ini."},

		// ---------- F. Sistem, jaringan & umum ----------
		Definition{Code: CodeAPIRateLimit, Status: 429, Severity: SeverityWarning, Category: CategorySystem,
			Message: "Pelan-pelan dulu, coba lagi {detik} detik."},
		Definition{Code: CodeAPIServerError, Status: 500, Severity: SeverityError, Category: CategorySystem, Persist: true,
			Message: "Server lagi batuk. Coba lagi sebentar ya."},
		Definition{Code: CodeAPIValidation, Status: 422, Severity: SeverityError, Category: CategorySystem,
			Message: "Ada isian yang perlu diperbaiki."},
		Definition{Code: CodeAPIForbidden, Status: 403, Severity: SeverityError, Category: CategorySystem, Persist: true,
			Message: "Kamu tidak punya akses ke halaman ini."},
		Definition{Code: CodeAPINotFound, Status: 404, Severity: SeverityInfo, Category: CategorySystem,
			Message: "Yang kamu cari tidak ketemu."},
		Definition{Code: CodeAPIBadRequest, Status: 400, Severity: SeverityError, Category: CategorySystem,
			Message: "Permintaan tidak bisa diproses."},
		Definition{Code: CodeAPIMethodNotAllowed, Status: 405, Severity: SeverityError, Category: CategorySystem,
			Message: "Metode permintaan tidak didukung untuk alamat ini."},
		Definition{Code: CodeAPIServiceUnavailable, Status: 503, Severity: SeverityWarning, Category: CategorySystem,
			Message: "Layanan sedang tidak tersedia. Coba lagi sebentar."},
		Definition{Code: CodeAPIUnsupportedMedia, Status: 415, Severity: SeverityError, Category: CategorySystem,
			Message: "Format berkas tidak didukung."},
		Definition{Code: CodeAPITimeout, Status: 504, Severity: SeverityError, Category: CategorySystem,
			Message: "Server kehabisan waktu. Coba lagi."},
		Definition{Code: CodeAPIMaintenance, Status: 503, Severity: SeverityInfo, Category: CategorySystem,
			Message: "Sedang dipoles. Segera kembali!"},
		Definition{Code: CodeAPIIdempotencyConflict, Status: 409, Severity: SeverityWarning, Category: CategorySystem,
			Message: "Permintaan dengan kunci sama sedang diproses. Tunggu sebentar."},
		Definition{Code: CodeSysUploadTooBig, Status: 413, Severity: SeverityError, Category: CategorySystem,
			Message: "Maksimal {n} MB."},
		Definition{Code: CodeAgeGate, Status: 403, Severity: SeverityCritical, Category: CategorySystem, Persist: true,
			Message: "Produk ini khusus 21+. Konfirmasi usiamu."},

		// ---------- G. Admin ----------
		Definition{Code: CodeAdmPriceChangeConfirm, Status: 409, Severity: SeverityWarning, Category: CategoryAdmin,
			Message: "Harga berubah drastis. Yakin?"},
		Definition{Code: CodeAdmAuditChainBroken, Status: 500, Severity: SeverityCritical, Category: CategoryAdmin, Persist: true,
			Message: "Rantai audit log rusak. Segera periksa."},
		Definition{Code: CodeAdmStockReasonReq, Status: 422, Severity: SeverityError, Category: CategoryAdmin,
			Message: "Alasan perubahan stok wajib diisi."},
	)
}
