import { useCallback, useEffect, useMemo, useState } from "react";

import { fetchCategories, fetchProducts, type Category, type ProductDetail } from "./api/catalog";
import { ApiError } from "./api/client";
import { fetchMeta, fetchReady, type MetaResponse, type ReadyResponse } from "./api/system";
import type { Order } from "./api/checkout";
import { AuthModal } from "./components/AuthModal";
import { CartDrawer } from "./components/CartDrawer";
import { CategoryFilter } from "./components/CategoryFilter";
import { CheckoutModal } from "./components/CheckoutModal";
import { Header } from "./components/Header";
import { HeroBanner } from "./components/HeroBanner";
import { PaymentModal } from "./components/PaymentModal";
import { ProductCard } from "./components/ProductCard";
import { ProductModal } from "./components/ProductModal";
import { AuthProvider } from "./context/AuthContext";
import { CartProvider, useCart } from "./context/CartContext";
import { ToastProvider } from "./context/ToastContext";
import { INITIAL_CATEGORIES, INITIAL_PRODUCTS } from "./data/initialProducts";

import "./styles/tokens.css";
import "./styles/neobrutalism.css";
import "./styles/diagnostic.css";

type LoadState = "memuat" | "siap" | "gagal";

interface Diagnostics {
  readonly meta: MetaResponse | null;
  readonly ready: ReadyResponse | null;
  readonly error: ApiError | null;
}

const EMPTY_DIAGNOSTICS: Diagnostics = { meta: null, ready: null, error: null };

function describeError(error: ApiError): string {
  const parts = [error.message];
  if (error.code) parts.push(`(${error.code})`);
  if (error.requestId) parts.push(`· request_id: ${error.requestId}`);
  return parts.join(" ");
}

function StorefrontContent() {
  const { addItem, count, openCart } = useCart();
  const [floatingCartBumping, setFloatingCartBumping] = useState<boolean>(false);

  useEffect(() => {
    if (count > 0) {
      setFloatingCartBumping(true);
      const timer = setTimeout(() => setFloatingCartBumping(false), 450);
      return () => clearTimeout(timer);
    }
    return undefined;
  }, [count]);

  // Data Katalog
  const [categories, setCategories] = useState<readonly Category[]>(INITIAL_CATEGORIES);
  const [products, setProducts] = useState<readonly ProductDetail[]>(INITIAL_PRODUCTS);
  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState<string>("");

  // Modals & Panels
  const [showDiagnostics, setShowDiagnostics] = useState<boolean>(true);
  const [activeProductDetail, setActiveProductDetail] = useState<ProductDetail | null>(null);
  const [isCheckoutOpen, setIsCheckoutOpen] = useState<boolean>(false);
  const [isAuthOpen, setIsAuthOpen] = useState<boolean>(false);
  const [authMode, setAuthMode] = useState<"login" | "register">("login");
  const [activePaymentOrder, setActivePaymentOrder] = useState<Order | null>(null);

  // Status Diagnostik M0
  const [diagState, setDiagState] = useState<LoadState>("memuat");
  const [diagData, setDiagData] = useState<Diagnostics>(EMPTY_DIAGNOSTICS);

  // Probe Diagnostik M0 (Pemeriksaan backend /readyz & /api/v1/meta)
  const loadDiagnostics = useCallback(async (signal: AbortSignal) => {
    setDiagState("memuat");
    try {
      const [meta, ready] = await Promise.all([fetchMeta(signal), fetchReady(signal)]);
      setDiagData({ meta, ready, error: null });
      setDiagState("siap");
    } catch (cause) {
      if (signal.aborted) return;
      const error =
        cause instanceof ApiError
          ? cause
          : new ApiError(0, {
              code: "API_SERVER_ERROR",
              message: "Kesalahan tak terduga saat menghubungi API.",
            });
      setDiagData({ meta: null, ready: null, error });
      setDiagState("gagal");
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void loadDiagnostics(controller.signal);
    return () => {
      controller.abort();
    };
  }, [loadDiagnostics]);

  const refreshDiagnostics = useCallback(() => {
    const controller = new AbortController();
    void loadDiagnostics(controller.signal);
  }, [loadDiagnostics]);

  // Muat data kategori dan produk dari API (fallback ke INITIAL_PRODUCTS jika offline)
  useEffect(() => {
    const controller = new AbortController();

    async function loadCatalog() {
      try {
        const [catRes, prodRes] = await Promise.all([
          fetchCategories(controller.signal).catch(() => INITIAL_CATEGORIES),
          fetchProducts({}, controller.signal).catch(() => null),
        ]);
        if (catRes && catRes.length > 0) {
          setCategories(catRes);
        }
        if (prodRes && prodRes.items && prodRes.items.length > 0) {
          // Gabungkan dengan detail awal agar varian dan gizi lengkap
          const merged: ProductDetail[] = INITIAL_PRODUCTS.map((initProd) => {
            const apiMatch = prodRes.items.find((p) => p.slug === initProd.slug || p.id === initProd.id);
            return apiMatch
              ? {
                  ...initProd,
                  ...apiMatch,
                  description: apiMatch.description || initProd.description,
                }
              : initProd;
          });
          setProducts(merged);
        }
      } catch {
        // Fallback aman
      }
    }

    void loadCatalog();

    return () => {
      controller.abort();
    };
  }, []);

  // Filter produk berdasarkan kategori dan kata kunci pencarian
  const filteredProducts = useMemo(() => {
    return products.filter((prod) => {
      const matchCat =
        selectedCategory === "all" ||
        prod.category?.slug === selectedCategory ||
        prod.category?.id === selectedCategory;

      if (!matchCat) return false;

      if (!searchQuery.trim()) return true;

      const q = searchQuery.toLowerCase();
      const matchName = prod.name.toLowerCase().includes(q);
      const matchFlavor = prod.flavor?.toLowerCase().includes(q) ?? false;
      const matchDesc = prod.description?.toLowerCase().includes(q) ?? false;

      return matchName || matchFlavor || matchDesc;
    });
  }, [products, selectedCategory, searchQuery]);

  return (
    <div className="kalengo-app">
      <Header
        showDiagnostics={showDiagnostics}
        onToggleDiagnostics={() => setShowDiagnostics((prev) => !prev)}
        onOpenAuth={() => {
          setAuthMode("login");
          setIsAuthOpen(true);
        }}
        activeCategory={selectedCategory}
        onSelectCategory={(slug) => {
          setSelectedCategory(slug);
          const el = document.getElementById("catalog-section");
          el?.scrollIntoView({ behavior: "smooth" });
        }}
      />

      <HeroBanner
        onShopNowClick={() => {
          const el = document.getElementById("catalog-section");
          el?.scrollIntoView({ behavior: "smooth" });
        }}
      />

      <main id="main-content" className="storefront-container">
        <section id="catalog-section" aria-labelledby="catalog-heading">
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: "1.5rem" }}>
            <h2 id="catalog-heading" style={{ fontFamily: "var(--font-display)", fontSize: "1.75rem", margin: 0 }}>
              Katalog Minuman Kaleng
            </h2>
            <span style={{ fontSize: "0.875rem", color: "#6a6a80", fontWeight: 700 }}>
              Menampilkan {filteredProducts.length} pilihan
            </span>
          </div>

          <CategoryFilter
            categories={categories}
            selectedCategory={selectedCategory}
            onSelectCategory={setSelectedCategory}
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
          />

          {filteredProducts.length === 0 ? (
            <div
              style={{
                backgroundColor: "#ffffff",
                border: "3px solid var(--ink, #1b1b2f)",
                borderRadius: "var(--radius-lg, 1.25rem)",
                padding: "3rem 1.5rem",
                textAlign: "center",
                boxShadow: "4px 4px 0 var(--ink, #1b1b2f)",
              }}
            >
              <span style={{ fontSize: "2.5rem" }} aria-hidden="true">🥤</span>
              <h3 style={{ fontFamily: "var(--font-display)", fontSize: "1.25rem", margin: "0.75rem 0 0.5rem" }}>
                Minuman tidak ditemukan
              </h3>
              <p style={{ color: "#6a6a80", margin: "0 0 1.25rem" }}>
                Tidak ada minuman yang cocok dengan pencarian &quot;{searchQuery}&quot;. Coba cari rasa lain!
              </p>
              <button
                type="button"
                className="btn btn--secondary btn--sm"
                onClick={() => {
                  setSearchQuery("");
                  setSelectedCategory("all");
                }}
              >
                Lihat Semua Minuman
              </button>
            </div>
          ) : (
            <div className="catalog-grid">
              {filteredProducts.map((product) => (
                <ProductCard
                  key={product.id}
                  product={product}
                  onViewDetail={(p) => setActiveProductDetail(p as ProductDetail)}
                  onAddToCart={(p, v) => void addItem(p, v, 1)}
                />
              ))}
            </div>
          )}
        </section>
      </main>

      {/* Floating Sticky Cart Button */}
      {count > 0 && (
        <button
          type="button"
          className={`floating-cart-trigger ${floatingCartBumping ? "cart-btn--wiggle" : ""}`}
          onClick={openCart}
          aria-label={`Buka Keranjang Belanja, total ${count} item`}
        >
          <span aria-hidden="true">🛒</span>
          <span>Keranjang</span>
          <span className={`floating-cart-count ${floatingCartBumping ? "cart-badge--bump" : ""}`}>
            {count}
          </span>
        </button>
      )}

      {/* Drawer & Modals */}
      <CartDrawer onProceedToCheckout={() => setIsCheckoutOpen(true)} />

      <ProductModal
        product={activeProductDetail}
        isOpen={activeProductDetail !== null}
        onClose={() => setActiveProductDetail(null)}
        onAddToCart={(p, v, q) => void addItem(p, v, q)}
      />

      <CheckoutModal
        isOpen={isCheckoutOpen}
        onClose={() => setIsCheckoutOpen(false)}
        onOrderCreated={(order) => {
          setActivePaymentOrder(order);
        }}
        onRequireAuth={() => {
          setAuthMode("login");
          setIsAuthOpen(true);
        }}
      />

      <PaymentModal
        order={activePaymentOrder}
        isOpen={activePaymentOrder !== null}
        onClose={() => setActivePaymentOrder(null)}
      />

      <AuthModal
        isOpen={isAuthOpen}
        initialMode={authMode}
        onClose={() => setIsAuthOpen(false)}
      />

      {/* Footer */}
      <footer className="site-footer">
        <div className="site-footer__container">
          <div className="site-footer__top">
            <div className="site-footer__brand">
              <div className="site-footer__logo">
                <span>🥫</span>
                <span>KALENGO</span>
              </div>
              <p className="site-footer__desc">
                Minuman kaleng klasik cita rasa otentik 1930s. Segar, nikmat, dan dingin sampai di depan pintumu di seluruh Indonesia.
              </p>
            </div>

            <div className="site-footer__badges">
              <span className="badge badge--lemon">100% GULA ALAMI</span>
              <span className="badge badge--mint">SNI & BPOM</span>
              <span className="badge badge--sky">KALENG RAMAH LINGKUNGAN</span>
            </div>
          </div>

          <div className="site-footer__bottom">
            <div>
              © 2026 KALENGO Indonesia. Hak Cipta Dilindungi Undang-Undang.
            </div>

            <button
              type="button"
              className="diag-toggle-btn"
              onClick={() => setShowDiagnostics((prev) => !prev)}
              aria-expanded={showDiagnostics}
            >
              ⚙️ {showDiagnostics ? "Sembunyikan Status Sistem M0" : "Status Sistem / Diagnostik M0"}
            </button>
          </div>
        </div>
      </footer>

      {/* Panel Diagnostik M0 (Tetap dapat diinspeksi teknis untuk audit & verifikasi API) */}
      {showDiagnostics && (
        <section className="diag" aria-labelledby="status-api-heading">
          <header className="diag__header">
            <p className="diag__eyebrow">M0 · Fondasi Sistem</p>
            <h1 id="status-api-heading" className="diag__title">
              Status Sistem KALENGO
            </h1>
            <p className="diag__lead">
              Panel teknis verifikasi konektivitas Web → API → PostgreSQL/Redis.
            </p>
          </header>

          <section className="diag__panel" aria-labelledby="status-api">
            <div className="diag__panel-head">
              <h2 id="status-api" className="diag__panel-title">
                Koneksi API
              </h2>
              <button
                type="button"
                className="diag__button"
                onClick={refreshDiagnostics}
                disabled={diagState === "memuat"}
              >
                {diagState === "memuat" ? "Memeriksa…" : "Periksa ulang"}
              </button>
            </div>

            <p role="status" aria-live="polite" className="diag__state">
              {diagState === "memuat" && "Sedang menghubungi API…"}
              {diagState === "siap" && "API terjangkau."}
              {diagState === "gagal" && "API tidak terjangkau."}
            </p>

            {diagState === "gagal" && diagData.error !== null && (
              <p className="diag__alert" role="alert">
                {describeError(diagData.error)}
              </p>
            )}

            {diagState === "siap" && diagData.meta !== null && (
              <dl className="diag__facts">
                <div className="diag__fact">
                  <dt>Nama aplikasi</dt>
                  <dd>{diagData.meta.app}</dd>
                </div>
                <div className="diag__fact">
                  <dt>Lingkungan</dt>
                  <dd>{diagData.meta.env}</dd>
                </div>
                <div className="diag__fact">
                  <dt>Versi</dt>
                  <dd>{diagData.meta.version}</dd>
                </div>
                <div className="diag__fact">
                  <dt>Penyedia pembayaran</dt>
                  <dd>{diagData.meta.payment_provider}</dd>
                </div>
                <div className="diag__fact">
                  <dt>Penyedia ongkir</dt>
                  <dd>{diagData.meta.shipping_provider}</dd>
                </div>
                <div className="diag__fact">
                  <dt>Mode notifikasi</dt>
                  <dd>{diagData.meta.notif_modes.join(", ")}</dd>
                </div>
              </dl>
            )}

            {diagState === "siap" && diagData.ready !== null && (
              <section className="diag__panel" aria-labelledby="status-dependensi" style={{ marginTop: "1rem" }}>
                <h2 id="status-dependensi" className="diag__panel-title">
                  Dependensi
                </h2>
                <p className="diag__state">
                  Status keseluruhan: <strong>{diagData.ready.status}</strong>
                </p>
                <table className="diag__table">
                  <caption className="sr-only">
                    Hasil pemeriksaan kesiapan setiap dependensi backend
                  </caption>
                  <thead>
                    <tr>
                      <th scope="col">Dependensi</th>
                      <th scope="col">Status</th>
                      <th scope="col">Latensi</th>
                      <th scope="col">Catatan</th>
                    </tr>
                  </thead>
                  <tbody>
                    {Object.entries(diagData.ready.checks).map(([name, check]) => (
                      <tr key={name}>
                        <th scope="row">{name}</th>
                        <td>{check.status}</td>
                        <td>{check.latency ?? "—"}</td>
                        <td>{check.error ?? "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </section>
            )}
          </section>
        </section>
      )}
    </div>
  );
}

export default function App() {
  return (
    <ToastProvider>
      <AuthProvider>
        <CartProvider>
          <StorefrontContent />
        </CartProvider>
      </AuthProvider>
    </ToastProvider>
  );
}
