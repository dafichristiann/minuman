// Package http memasang seluruh rute dan rantai middleware (§10.2).
//
// Urutan middleware ditetapkan di sini dan TIDAK boleh diubah tanpa alasan jelas:
//
//	RequestID → AccessLog → RoutePattern → Recoverer → SecurityHeaders → CORS
//	          → [ /api/v1: Timeout → MaxBody → RateLimit ] → rute
//
// AccessLog harus mendahului RoutePattern karena AccessLog menyediakan wadah
// requestInfo yang diisi RoutePattern.
package http

import (
	"log/slog"
	nethttp "net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kalengo/api/internal/apperr"
	"github.com/kalengo/api/internal/config"
	"github.com/kalengo/api/internal/http/handler"
	"github.com/kalengo/api/internal/http/middleware"
	"github.com/kalengo/api/internal/platform/db"
	"github.com/kalengo/api/internal/platform/httpx"
	"github.com/kalengo/api/internal/platform/payment"
	"github.com/kalengo/api/internal/platform/ratelimit"
	"github.com/kalengo/api/internal/platform/redisclient"
	"github.com/kalengo/api/internal/platform/shipping"
	"github.com/kalengo/api/internal/repo"
	"github.com/kalengo/api/internal/service/account"
	"github.com/kalengo/api/internal/service/auth"
	"github.com/kalengo/api/internal/service/catalog"
	ordersvc "github.com/kalengo/api/internal/service/order"
	"github.com/kalengo/api/internal/service/shop"
)

// Deps adalah dependensi yang dibutuhkan router.
type Deps struct {
	Config  *config.Config
	Log     *slog.Logger
	DB      *db.Pool
	Redis   *redisclient.Client
	Limiter *ratelimit.Limiter
}

// NewRouter menyusun router lengkap aplikasi.
func NewRouter(d Deps) *chi.Mux {
	cfg := d.Config
	trustProxy := cfg.HTTP.TrustProxyHeaders

	// ---------- lapisan data & layanan ----------
	// Store dibangun dari *pgxpool.Pool (bukan *db.Pool) agar tipe yang
	// memenuhi repo.DBTX jelas dan tidak bergantung pada metode yang kebetulan
	// dipromosikan.
	store := repo.New(d.DB.Pool)

	authSvc := auth.New(auth.Options{
		Config:          cfg,
		Users:           store,
		Sessions:        store,
		Mailer:          auth.LogMailer{Log: d.Log},
		ExposeDevTokens: !cfg.IsProduction(),
	})
	accountSvc := account.New(store, store, store)
	catalogSvc := catalog.New(store)
	shopSvc := shop.New(store)

	// ---------- penyedia M2 ----------
	//
	// M2 memakai penyedia PALSU yang berjalan di dalam proses: tidak ada
	// panggilan jaringan ke Midtrans/Biteship. Adapter sungguhan ditambahkan pada
	// M5 dengan antarmuka yang sama (lihat internal/platform/{payment,shipping}).
	payProvider := payment.Provider(&payment.Fake{ServerKey: cfg.Payment.MidtransServerKey})
	shipProvider := shipping.Provider(&shipping.Fake{})

	orderSvc := ordersvc.New(ordersvc.Deps{
		Config:   cfg,
		Store:    store,
		Payments: payProvider,
		Shipping: shipProvider,
		Log:      d.Log,
	})

	// ---------- handler ----------
	sys := handler.NewSystem(cfg, d.DB, d.Redis)
	authH := handler.NewAuth(cfg, authSvc)
	meH := handler.NewMe(cfg, accountSvc)
	addrH := handler.NewAddress(cfg, accountSvc)
	catH := handler.NewCatalog(cfg, catalogSvc)
	cartH := handler.NewCart(cfg, shopSvc)
	orderH := handler.NewOrder(cfg, orderSvc)
	checkoutH := handler.NewCheckout(cfg, orderSvc)
	webhookH := handler.NewWebhook(orderSvc)

	// ---------- middleware bergantung sesi ----------
	requireAuth := middleware.RequireAuth(cfg, authSvc.Tokens(), store)
	csrf := middleware.CSRF(cfg)

	r := chi.NewRouter()

	// ---------- rantai global ----------
	r.Use(middleware.RequestID)
	r.Use(middleware.AccessLog(d.Log, trustProxy))
	r.Use(middleware.RoutePattern)
	r.Use(middleware.Recoverer(d.Log))
	r.Use(middleware.SecurityHeaders(cfg.IsProduction()))
	r.Use(middleware.CORS(cfg.App.WebOrigin))

	// ---------- respons standar untuk rute tak dikenal ----------
	r.NotFound(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		httpx.Error(w, r, apperr.New(apperr.CodeAPINotFound, nil))
	})
	r.MethodNotAllowed(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		httpx.Error(w, r, apperr.New(apperr.CodeAPIMethodNotAllowed, nil))
	})

	// ---------- endpoint operasional (di luar kontrak §12) ----------
	r.Get("/healthz", sys.Live)
	r.Get("/readyz", sys.Ready)
	if cfg.Obs.MetricsEnabled {
		// Metrik Prometheus (§7). Di produksi endpoint ini sebaiknya hanya
		// dijangkau dari jaringan internal / jaringan monitoring.
		r.Handle("/metrics", promhttp.Handler())
	}

	// ---------- API publik ----------
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(chimw.Timeout(cfg.HTTP.RequestTimeout))
		r.Use(middleware.MaxBody(cfg.HTTP.MaxBodyBytes))
		r.Use(middleware.RateLimit(
			d.Limiter,
			middleware.KeyByIP(trustProxy),
			cfg.RateLimit.GeneralPerMinute,
			time.Minute,
			true, // lewati streaming SSE
		))

		// Meta & sinkronisasi katalog notifikasi
		r.Get("/meta", sys.Meta)
		r.Get("/meta/error-codes", sys.ErrorCodes)

		// ---------------------------------------------------------------
		// Rute M1
		// ---------------------------------------------------------------

		// Batas laju khusus endpoint sensitif (§12.8). Endpoint yang bisa
		// dipakai menebak email atau menebak password memakai anggaran per jam
		// yang ketat, terpisah dari batas umum per menit.
		registerLimit := middleware.RateLimit(
			d.Limiter, middleware.KeyByIP(trustProxy),
			cfg.RateLimit.RegisterPerHour, time.Hour, false,
		)
		loginLimit := middleware.RateLimit(
			d.Limiter, middleware.KeyByIP(trustProxy),
			cfg.RateLimit.LoginPerMinute, time.Minute, false,
		)
		sensitiveLimit := middleware.RateLimit(
			d.Limiter, middleware.KeyByIP(trustProxy),
			cfg.RateLimit.RegisterPerHour, time.Hour, false,
		)

		r.Route("/auth", func(r chi.Router) {
			r.With(registerLimit).Post("/register", authH.Register)
			r.With(loginLimit).Post("/login", authH.Login)
			r.With(sensitiveLimit).Post("/refresh", authH.Refresh)

			r.With(sensitiveLimit).Post("/verify-email", authH.VerifyEmail)
			r.With(sensitiveLimit).Post("/verify-email/resend", authH.ResendVerification)
			r.With(sensitiveLimit).Post("/forgot-password", authH.ForgotPassword)
			r.With(sensitiveLimit).Post("/reset-password", authH.ResetPassword)

			// Logout hanya butuh refresh token (access token boleh kedaluwarsa),
			// tetapi tetap wajib menyertakan token CSRF.
			r.With(csrf).Post("/logout", authH.Logout)

			r.Group(func(r chi.Router) {
				r.Use(requireAuth)
				r.Use(csrf)
				r.Post("/logout-all", authH.LogoutAll)
			})
		})

		// --- katalog publik (§12.2) ---
		//
		// Catatan chi: satu posisi segmen hanya boleh punya SATU nama parameter,
		// sehingga detail (/products/{slug}) dan ulasan (/products/{id}/reviews)
		// memakai nama generik `product`. Handler detail memperlakukannya
		// sebagai slug, handler ulasan mem-parse-nya sebagai UUID.
		r.Get("/products", catH.List)
		r.Get("/products/suggest", catH.Suggest)
		r.Get("/products/{product}", catH.Detail)
		r.Get("/products/{product}/reviews", catH.Reviews)
		r.Get("/categories", catH.Categories)
		r.Get("/brands", catH.Brands)

		// --- area terautentikasi (§12.1 & §12.3) ---
		r.Group(func(r chi.Router) {
			r.Use(requireAuth)
			r.Use(middleware.NoStore)
			r.Use(csrf)

			r.Get("/me", meH.Get)
			r.Patch("/me", meH.Patch)
			r.Delete("/me", meH.Delete)
			r.Post("/me/password", authH.ChangePassword)

			r.Get("/me/addresses", addrH.List)
			r.Post("/me/addresses", addrH.Create)
			r.Patch("/me/addresses/{id}", addrH.Update)
			r.Delete("/me/addresses/{id}", addrH.Delete)
			r.Post("/me/addresses/{id}/default", addrH.SetDefault)

			r.Get("/cart", cartH.Get)
			r.Delete("/cart", cartH.Clear)
			r.Put("/cart/items/{variantId}", cartH.SetItem)
			r.Delete("/cart/items/{variantId}", cartH.DeleteItem)
			r.Post("/cart/merge", cartH.Merge)
			r.Post("/cart/validate", cartH.Validate)

			r.Get("/wishlist", cartH.ListWishlist)
			r.Get("/wishlist/{productId}", cartH.GetWishlist)
			r.Put("/wishlist/{productId}", cartH.AddWishlist)
			r.Delete("/wishlist/{productId}", cartH.RemoveWishlist)

			r.Post("/stock-alerts/{variantId}", cartH.SubscribeStock)
			r.Delete("/stock-alerts/{variantId}", cartH.UnsubscribeStock)

			// --- checkout & pesanan (§12.4) ---
			r.Post("/shipping/rates", checkoutH.ShippingRates)
			r.Post("/vouchers/validate", checkoutH.ValidateVoucher)

			r.Post("/orders", orderH.Create)
			r.Get("/orders", orderH.List)
			r.Get("/orders/{id}", orderH.Detail)
			r.Post("/orders/{id}/cancel", orderH.Cancel)
			r.Post("/orders/{id}/pay", orderH.Pay)
		})

		// --- webhook penyedia (§12.5) ---
		//
		// SENGAJA di luar grup terautentikasi: pemanggilnya penyedia pembayaran,
		// bukan pengguna. Tanpa RequireAuth dan tanpa CSRF — keamanannya
		// bertumpu pada verifikasi tanda tangan di lapisan service.
		r.Post("/webhooks/midtrans", webhookH.Midtrans)
		r.Post("/webhooks/shipping", webhookH.Shipping)
		// ---------------------------------------------------------------

		// ---------------------------------------------------------------
		// Rute milestone berikutnya:
		//   M2: /shipping/rates, /vouchers/validate, /orders, /webhooks/*
		//   M3: /notifications, /notification-preferences, /push-subscriptions,
		//       /me/ui-settings
		//   M4: /admin/*
		//
		// PENTING (M3): GET /notifications/stream adalah SSE berumur panjang dan
		// TIDAK boleh berada di grup ini karena chimw.Timeout akan memutus
		// streaming. Daftarkan grup terpisah di bawah, di luar Route("/api/v1"),
		// dengan rute absolut:
		//
		//   r.Group(func(r chi.Router) {
		//       r.Use(middleware.RequireAuth(...))
		//       r.Get("/api/v1/notifications/stream", sseHandler)
		//   })
		// ---------------------------------------------------------------
	})

	return r
}
