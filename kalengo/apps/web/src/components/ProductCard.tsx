import React, { useState } from "react";
import type { Product, ProductVariant } from "../api/catalog";
import { useToast } from "../context/useToast";
import { formatIdr } from "../utils/format";
import { MascotCan } from "./mascots/MascotCans";

export interface ProductCardProps {
  product: Product & { variants?: readonly ProductVariant[] };
  onViewDetail: (product: Product) => void;
  onAddToCart: (product: Product, variant: ProductVariant) => void;
}

export const ProductCard: React.FC<ProductCardProps> = ({
  product,
  onViewDetail,
  onAddToCart,
}) => {
  // Buat default varian jika varian belum ada di list item
  const defaultVariant: ProductVariant = {
    id: `${product.id}-var-1`,
    sku: `${product.slug}-1`,
    volume_ml: 330,
    pack_size: 1,
    price: product.price_from ?? 6500,
    weight_gram: 350,
    available: true,
  };

  const variants: readonly ProductVariant[] =
    product.variants && product.variants.length > 0
      ? product.variants
      : [defaultVariant];

  const firstVariant = variants[0] ?? defaultVariant;
  const [selectedVariantId, setSelectedVariantId] = useState<string>(firstVariant.id);
  const [isBouncing, setIsBouncing] = useState<boolean>(false);
  const toast = useToast();

  const currentVariant = variants.find((v) => v.id === selectedVariantId) ?? firstVariant;
  const isOutOfStock = currentVariant.available === false || currentVariant.stock_label === "habis";

  const getPackLabel = (packSize: number): string => {
    if (packSize === 1) return "1 Kaleng";
    if (packSize === 6) return "6-Pack";
    if (packSize === 12) return "12-Pack";
    if (packSize === 24) return "Dus (24)";
    return `${packSize} Pcs`;
  };

  const handleCardClick = (e: React.MouseEvent) => {
    // Jangan buka modal jika mengklik tombol varian atau tombol beli
    const target = e.target as HTMLElement;
    if (target.closest("button")) return;
    onViewDetail(product);
  };

  return (
    <article
      className="card-product"
      onClick={handleCardClick}
      style={{ cursor: "pointer" }}
      aria-label={`Produk ${product.name}`}
    >
      <div className="card-product__stage">
        {product.badges && product.badges.length > 0 && (
          <div className="card-product__badge-wrapper">
            {product.badges.map((b) => (
              <span
                key={b}
                className={`badge ${
                  b === "Baru"
                    ? "badge--mint"
                    : b === "Terlaris"
                      ? "badge--hot"
                      : "badge--lemon"
                }`}
              >
                {b}
              </span>
            ))}
          </div>
        )}

        <div className={`card-product__mascot ${isBouncing ? "can-anim--squish" : ""}`}>
          <MascotCan
            name={product.mascot_svg_key || "can-soda-lemon"}
            size={150}
            alt={`Ilustrasi Maskot ${product.name}`}
          />
        </div>
      </div>

      <div className="card-product__body">
        {product.flavor && (
          <p className="card-product__flavor-tag" style={{ margin: "0 0 0.25rem 0", color: "#6a6a80", fontSize: "0.8125rem", fontWeight: 700 }}>
            Rasa: {product.flavor}
          </p>
        )}

        <h3 className="card-product__title">
          <button
            type="button"
            className="link-title-btn"
            onClick={() => onViewDetail(product)}
            style={{
              background: "none",
              border: "none",
              padding: 0,
              font: "inherit",
              textAlign: "left",
              cursor: "pointer",
              color: "inherit",
            }}
          >
            {product.name}
          </button>
        </h3>

        <p className="card-product__desc">
          {product.description || "Minuman kaleng segar berkarakter dengan rasa otentik khas KALENGO."}
        </p>

        {/* Variant Pack Selector */}
        {variants.length > 1 && (
          <div className="card-product__variant-selector" style={{ marginBottom: "1rem" }}>
            <span style={{ fontSize: "0.75rem", fontWeight: 800, textTransform: "uppercase", display: "block", marginBottom: "0.35rem", color: "#4a4a63" }}>
              Pilih Kemasan:
            </span>
            <div className="filter-group" style={{ gap: "0.35rem" }}>
              {variants.map((v) => {
                const isSelected = v.id === currentVariant.id;
                const outOfStock = v.available === false || v.stock_label === "habis";
                return (
                  <button
                    key={v.id}
                    type="button"
                    disabled={outOfStock}
                    className={`filter-pill btn--sm ${isSelected ? "is-active" : ""}`}
                    style={{
                      padding: "0.25rem 0.6rem",
                      fontSize: "0.75rem",
                      opacity: outOfStock ? 0.45 : 1,
                      textDecoration: outOfStock ? "line-through" : "none",
                    }}
                    onClick={(e) => {
                      e.stopPropagation();
                      setSelectedVariantId(v.id);
                    }}
                    title={outOfStock ? "Stok habis" : `${getPackLabel(v.pack_size)} - ${formatIdr(v.price)}`}
                  >
                    {getPackLabel(v.pack_size)}
                  </button>
                );
              })}
            </div>
          </div>
        )}

        <div className="card-product__footer">
          <div>
            <div className="card-product__price">{formatIdr(currentVariant.price)}</div>
            {currentVariant.compare_at_price && (
              <span
                style={{
                  fontSize: "0.75rem",
                  color: "#999",
                  textDecoration: "line-through",
                  marginLeft: "0.25rem",
                  fontWeight: 700,
                }}
              >
                {formatIdr(currentVariant.compare_at_price)}
              </span>
            )}
          </div>

          <button
            type="button"
            className={`btn btn--sm ${isOutOfStock ? "btn--secondary" : "btn--primary"}`}
            disabled={isOutOfStock}
            onClick={(e) => {
              e.stopPropagation();
              setIsBouncing(true);
              setTimeout(() => setIsBouncing(false), 600);
              toast.success(
                `🥤 ${product.name} berhasil ditambahkan ke keranjang!`,
                "Keranjang Belanja",
              );
              onAddToCart(product, currentVariant);
            }}
            aria-label={`Tambah ${product.name} ${getPackLabel(currentVariant.pack_size)} ke keranjang`}
          >
            {isOutOfStock ? "Habis" : "+ Tambah"}
          </button>
        </div>
      </div>
    </article>
  );
};
