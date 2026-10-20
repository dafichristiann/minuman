import React, { useEffect, useState } from "react";
import type { Product, ProductDetail, ProductVariant } from "../api/catalog";
import { useToast } from "../context/useToast";
import { formatIdr } from "../utils/format";
import { MascotCan } from "./mascots/MascotCans";

export interface ProductModalProps {
  product: (Product | ProductDetail) | null;
  isOpen: boolean;
  onClose: () => void;
  onAddToCart: (product: Product, variant: ProductVariant, quantity: number) => void;
}

export const ProductModal: React.FC<ProductModalProps> = ({
  product,
  isOpen,
  onClose,
  onAddToCart,
}) => {
  const detail = product as ProductDetail | null;

  const defaultVariant: ProductVariant = {
    id: `${product?.id || "temp"}-var-1`,
    sku: `${product?.slug || "temp"}-1`,
    volume_ml: 330,
    pack_size: 1,
    price: product?.price_from ?? 6500,
    weight_gram: 350,
    available: true,
  };

  const variants: readonly ProductVariant[] =
    detail?.variants && detail.variants.length > 0
      ? detail.variants
      : [defaultVariant];

  const firstVariant = variants[0] ?? defaultVariant;
  const [selectedVariantId, setSelectedVariantId] = useState<string>(firstVariant.id);
  const [quantity, setQuantity] = useState<number>(1);
  const toast = useToast();

  const firstVariantId = firstVariant.id;

  // Reset saat ganti produk
  useEffect(() => {
    setSelectedVariantId(firstVariantId);
    setQuantity(1);
  }, [product?.id, firstVariantId]);

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen || !product) return null;

  const currentVariant = variants.find((v) => v.id === selectedVariantId) ?? firstVariant;
  const maxLimit = currentVariant.max_per_order ?? 24;
  const isOutOfStock = currentVariant.available === false || currentVariant.stock_label === "habis";

  const getPackLabel = (packSize: number): string => {
    if (packSize === 1) return "1 Kaleng";
    if (packSize === 6) return "6-Pack Hemat";
    if (packSize === 12) return "12-Pack Kotak";
    if (packSize === 24) return "Dus Karton (24)";
    return `${packSize} Pcs`;
  };

  const handleIncrement = () => {
    setQuantity((q) => Math.min(maxLimit, q + 1));
  };

  const handleDecrement = () => {
    setQuantity((q) => Math.max(1, q - 1));
  };

  const handleAddToCart = () => {
    if (isOutOfStock) return;
    toast.success(
      `🥤 ${quantity}x ${product.name} berhasil ditambahkan ke keranjang!`,
      "Keranjang Belanja",
    );
    onAddToCart(product, currentVariant, quantity);
    onClose();
  };

  const nutrition = detail?.nutrition;

  return (
    <div
      className="modal-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="product-modal-title"
    >
      <div className="modal-window">
        <header className="modal-header">
          <div>
            <span className="badge badge--mint" style={{ marginBottom: "0.25rem" }}>
              {product.category?.name || "Minuman Kaleng"}
            </span>
            <h2 id="product-modal-title" className="modal-title">
              {product.name}
            </h2>
          </div>
          <button
            type="button"
            className="btn btn--sm btn--secondary"
            onClick={onClose}
            aria-label="Tutup jendela detail produk"
            style={{ padding: "0.4rem 0.75rem" }}
          >
            ✕
          </button>
        </header>

        <div className="modal-body" style={{ display: "flex", flexDirection: "column", gap: "1.25rem" }}>
          {/* Stage Maskot */}
          <div
            style={{
              backgroundColor: "var(--cream, #fff4dc)",
              border: "3px solid var(--ink, #1b1b2f)",
              borderRadius: "var(--radius-md, 0.875rem)",
              padding: "1.5rem",
              display: "flex",
              justifyContent: "center",
              alignItems: "center",
            }}
          >
            <MascotCan
              name={product.mascot_svg_key || "can-soda-lemon"}
              size={180}
              alt={`Maskot ${product.name}`}
            />
          </div>

          {/* Deskripsi & Info Rasa */}
          <div>
            {product.flavor && (
              <p style={{ margin: "0 0 0.5rem 0", fontWeight: 800, color: "var(--ink, #1b1b2f)" }}>
                Varian Rasa: <span className="highlight-text">{product.flavor}</span>
              </p>
            )}
            <p style={{ color: "#4a4a63", lineHeight: 1.6, margin: 0 }}>
              {product.description ||
                "Minuman kaleng klasik racikan segar dengan gelembung pas dan rasa alami tanpa pemanis buatan berlebih."}
            </p>
          </div>

          {/* Informasi Nilai Gizi */}
          {nutrition && (
            <div
              style={{
                backgroundColor: "#fffaf0",
                border: "2px dashed var(--ink, #1b1b2f)",
                borderRadius: "var(--radius-md, 0.875rem)",
                padding: "1rem",
              }}
            >
              <h3 style={{ margin: "0 0 0.75rem 0", fontSize: "0.9375rem", fontWeight: 800 }}>
                📋 Nilai Gizi (per porsi {nutrition.porsi_ml ? `${nutrition.porsi_ml} ml` : "kaleng"})
              </h3>
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "repeat(auto-fit, minmax(90px, 1fr))",
                  gap: "0.5rem",
                }}
              >
                <div style={{ backgroundColor: "#ffffff", padding: "0.5rem", border: "1.5px solid #1b1b2f", borderRadius: "0.5rem", textAlign: "center" }}>
                  <div style={{ fontSize: "0.75rem", color: "#666", fontWeight: 700 }}>Energi</div>
                  <div style={{ fontSize: "1.125rem", fontWeight: 900 }}>{nutrition.energi_kcal ?? "-"} kcal</div>
                </div>
                <div style={{ backgroundColor: "#ffffff", padding: "0.5rem", border: "1.5px solid #1b1b2f", borderRadius: "0.5rem", textAlign: "center" }}>
                  <div style={{ fontSize: "0.75rem", color: "#666", fontWeight: 700 }}>Gula</div>
                  <div style={{ fontSize: "1.125rem", fontWeight: 900 }}>{nutrition.gula_g ?? "-"} g</div>
                </div>
                <div style={{ backgroundColor: "#ffffff", padding: "0.5rem", border: "1.5px solid #1b1b2f", borderRadius: "0.5rem", textAlign: "center" }}>
                  <div style={{ fontSize: "0.75rem", color: "#666", fontWeight: 700 }}>Natrium</div>
                  <div style={{ fontSize: "1.125rem", fontWeight: 900 }}>{nutrition.natrium_mg ?? "-"} mg</div>
                </div>
                {nutrition.kafein_mg !== undefined && (
                  <div style={{ backgroundColor: "#ffffff", padding: "0.5rem", border: "1.5px solid #1b1b2f", borderRadius: "0.5rem", textAlign: "center" }}>
                    <div style={{ fontSize: "0.75rem", color: "#666", fontWeight: 700 }}>Kafein</div>
                    <div style={{ fontSize: "1.125rem", fontWeight: 900 }}>{nutrition.kafein_mg} mg</div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Pemilihan Varian Kemasan */}
          <div>
            <label style={{ display: "block", fontWeight: 800, marginBottom: "0.5rem", fontSize: "0.875rem" }}>
              Pilihan Kemasan:
            </label>
            <div className="filter-group" style={{ gap: "0.5rem" }}>
              {variants.map((v) => {
                const isSelected = v.id === currentVariant.id;
                const outOfStock = v.available === false || v.stock_label === "habis";
                return (
                  <button
                    key={v.id}
                    type="button"
                    disabled={outOfStock}
                    className={`filter-pill ${isSelected ? "is-active" : ""}`}
                    onClick={() => {
                      setSelectedVariantId(v.id);
                      setQuantity(1);
                    }}
                    style={{
                      opacity: outOfStock ? 0.45 : 1,
                      textDecoration: outOfStock ? "line-through" : "none",
                    }}
                  >
                    {getPackLabel(v.pack_size)} — {formatIdr(v.price)}
                  </button>
                );
              })}
            </div>
          </div>

          {/* Stepper Jumlah */}
          {!isOutOfStock && (
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", paddingTop: "0.5rem" }}>
              <span style={{ fontWeight: 800 }}>Jumlah Pembelian:</span>
              <div style={{ display: "inline-flex", alignItems: "center", gap: "0.5rem" }}>
                <button
                  type="button"
                  className="btn btn--sm btn--secondary"
                  onClick={handleDecrement}
                  disabled={quantity <= 1}
                  aria-label="Kurangi jumlah"
                  style={{ width: "36px", height: "36px", padding: 0 }}
                >
                  -
                </button>
                <span
                  style={{
                    fontFamily: "var(--font-mono, monospace)",
                    fontSize: "1.125rem",
                    fontWeight: 900,
                    minWidth: "32px",
                    textAlign: "center",
                  }}
                >
                  {quantity}
                </span>
                <button
                  type="button"
                  className="btn btn--sm btn--secondary"
                  onClick={handleIncrement}
                  disabled={quantity >= maxLimit}
                  aria-label="Tambah jumlah"
                  style={{ width: "36px", height: "36px", padding: 0 }}
                >
                  +
                </button>
              </div>
            </div>
          )}
        </div>

        <footer className="modal-footer" style={{ justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <div style={{ fontSize: "0.75rem", color: "#6a6a80", fontWeight: 700 }}>Total Harga</div>
            <div style={{ fontSize: "1.375rem", fontWeight: 900, color: "var(--ink, #1b1b2f)" }}>
              {formatIdr(currentVariant.price * quantity)}
            </div>
          </div>

          <button
            type="button"
            className={`btn ${isOutOfStock ? "btn--secondary" : "btn--primary"}`}
            disabled={isOutOfStock}
            onClick={handleAddToCart}
          >
            {isOutOfStock ? "Stok Habis" : "Tambah ke Keranjang 🛒"}
          </button>
        </footer>
      </div>
    </div>
  );
};
