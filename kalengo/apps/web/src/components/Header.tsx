import React, { useEffect, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { useCart } from "../context/CartContext";
import { useToast } from "../context/useToast";
import { ConfirmModal } from "./ConfirmModal";

export interface HeaderProps {
  onToggleDiagnostics: () => void;
  showDiagnostics: boolean;
  onOpenAuth: () => void;
  onSelectCategory?: (categorySlug: string) => void;
  activeCategory?: string;
}

export const Header: React.FC<HeaderProps> = ({
  onToggleDiagnostics,
  showDiagnostics,
  onOpenAuth,
  onSelectCategory,
  activeCategory = "all",
}) => {
  const { user, isAuthenticated, logout } = useAuth();
  const { count, openCart } = useCart();
  const toast = useToast();

  const [showLogoutConfirm, setShowLogoutConfirm] = useState(false);
  const [cartBumping, setCartBumping] = useState(false);

  useEffect(() => {
    if (count > 0) {
      setCartBumping(true);
      const timer = setTimeout(() => setCartBumping(false), 450);
      return () => clearTimeout(timer);
    }
    return undefined;
  }, [count]);

  const categories = [
    { slug: "all", label: "Semua Minuman" },
    { slug: "soda", label: "Soda Segar" },
    { slug: "kopi-susu", label: "Kopi Susu" },
    { slug: "teh", label: "Teh Asli" },
    { slug: "rendah-gula", label: "Nol / Rendah Gula" },
  ];

  return (
    <header className="site-header" role="banner">
      <div className="site-header__topbar">
        <div className="site-header__container">
          <div className="site-header__left">
            <a href="#main-content" className="site-logo" aria-label="KALENGO Beranda">
              <span className="site-logo__can" aria-hidden="true">🥫</span>
              <span className="site-logo__text">KALENGO</span>
              <span className="badge badge--lemon" style={{ marginLeft: "0.5rem" }}>
                1930s CAN STORE
              </span>
            </a>
          </div>

          <div className="site-header__actions">
            <button
              type="button"
              className={`btn btn--sm ${showDiagnostics ? "btn--lemon" : "btn--secondary"}`}
              onClick={onToggleDiagnostics}
              title="Periksa koneksi backend M0 dan status database"
              aria-pressed={showDiagnostics}
            >
              <span aria-hidden="true">🛠️</span>
              <span className="hide-on-mobile">Diagnostik M0</span>
            </button>

            {isAuthenticated ? (
              <div className="user-dropdown-container">
                <button
                  type="button"
                  className="btn btn--sm btn--secondary"
                  onClick={() => setShowLogoutConfirm(true)}
                  title={`Masuk sebagai ${user?.full_name || user?.email}`}
                >
                  <span aria-hidden="true">👤</span>
                  <span className="hide-on-mobile">{user?.full_name?.split(" ")[0] || "Akun"}</span>
                  <span className="badge badge--mint" style={{ fontSize: "0.65rem", padding: "0.15rem 0.4rem" }}>
                    Keluar
                  </span>
                </button>
              </div>
            ) : (
              <button
                type="button"
                className="btn btn--sm btn--lemon"
                onClick={onOpenAuth}
                aria-label="Masuk atau Daftar Akun"
              >
                <span aria-hidden="true">👋</span>
                <span>Masuk</span>
              </button>
            )}

            <button
              type="button"
              className="btn btn--sm btn--primary site-header__cart-btn"
              onClick={openCart}
              aria-label={`Buka Keranjang Belanja, berisi ${count} item`}
            >
              <span aria-hidden="true">🛒</span>
              <span>Keranjang</span>
              <span
                className={`site-header__cart-badge ${cartBumping ? "cart-badge--bump" : ""}`}
                aria-hidden="true"
              >
                {count}
              </span>
            </button>
          </div>
        </div>
      </div>

      <nav className="site-header__nav" aria-label="Kategori Utama">
        <div className="site-header__container site-header__nav-scroll">
          {categories.map((cat) => {
            const isActive = activeCategory === cat.slug;
            return (
              <button
                key={cat.slug}
                type="button"
                className={`header-nav-link ${isActive ? "is-active" : ""}`}
                onClick={() => onSelectCategory?.(cat.slug)}
              >
                {cat.label}
              </button>
            );
          })}
        </div>
      </nav>

      <ConfirmModal
        isOpen={showLogoutConfirm}
        title="Keluar Akun?"
        message="Yakin ingin keluar akun?"
        confirmText="Ya, Keluar"
        cancelText="Batal"
        variant="warning"
        icon="🚪"
        onConfirm={() => {
          void logout();
          setShowLogoutConfirm(false);
          toast.info("Anda telah keluar dari akun KALENGO.", "Sampai Jumpa!");
        }}
        onCancel={() => setShowLogoutConfirm(false)}
      />
    </header>
  );
};
