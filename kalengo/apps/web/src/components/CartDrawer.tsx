import React, { useEffect, useState } from "react";
import { useCart } from "../context/CartContext";
import { useToast } from "../context/useToast";
import { formatIdr } from "../utils/format";
import { ConfirmModal } from "./ConfirmModal";
import { CanLiteZero, MascotCan } from "./mascots/MascotCans";

export interface CartDrawerProps {
  onProceedToCheckout: () => void;
}

export const CartDrawer: React.FC<CartDrawerProps> = ({ onProceedToCheckout }) => {
  const {
    items,
    count,
    subtotal,
    isOpen,
    closeCart,
    updateQuantity,
    removeItem,
    clearCart,
    issues,
  } = useCart();
  const toast = useToast();

  const [itemToDelete, setItemToDelete] = useState<{ variantId: string; name: string } | null>(null);
  const [showClearConfirm, setShowClearConfirm] = useState<boolean>(false);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        closeCart();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, closeCart]);

  if (!isOpen) return null;

  return (
    <>
      <div
        className="drawer-backdrop"
        onClick={closeCart}
        aria-hidden="true"
      />

      <aside
        className="drawer-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="drawer-cart-title"
      >
        <header className="drawer-header">
          <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ fontSize: "1.5rem" }} aria-hidden="true">🛒</span>
            <h2 id="drawer-cart-title" style={{ margin: 0, fontSize: "1.25rem", fontFamily: "var(--font-display)" }}>
              Keranjang ({count})
            </h2>
          </div>
          <button
            type="button"
            className="btn btn--sm btn--secondary"
            onClick={closeCart}
            aria-label="Tutup keranjang"
            style={{ width: "36px", height: "36px", padding: 0 }}
          >
            ✕
          </button>
        </header>

        {issues && issues.length > 0 && (
          <div
            style={{
              padding: "0.75rem 1rem",
              backgroundColor: "var(--lemon, #ffd23f)",
              borderBottom: "2px solid #1b1b2f",
              fontSize: "0.8125rem",
              fontWeight: 700,
            }}
          >
            {issues.map((iss, idx) => (
              <div key={idx}>⚠️ {iss.message}</div>
            ))}
          </div>
        )}

        <div style={{ flex: 1, overflowY: "auto", padding: "1.25rem" }}>
          {items.length === 0 ? (
            <div
              style={{
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                textAlign: "center",
                padding: "2rem 1rem",
                gap: "1rem",
              }}
            >
              <CanLiteZero size={150} alt="Maskot Keranjang Kosong" />
              <h3 style={{ margin: 0, fontFamily: "var(--font-display)", fontSize: "1.25rem" }}>
                Keranjangmu Masih Kosong!
              </h3>
              <p style={{ margin: 0, color: "#6a6a80", fontSize: "0.875rem" }}>
                Yuk pilih minuman kaleng favoritmu dan nikmati kesegaran nostalgia klasik sekarang!
              </p>
              <button
                type="button"
                className="btn btn--primary btn--pill"
                onClick={closeCart}
              >
                Mulai Belanja 🥤
              </button>
            </div>
          ) : (
            <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <span style={{ fontSize: "0.75rem", fontWeight: 800, textTransform: "uppercase", color: "#6a6a80" }}>
                  Daftar Pesanan ({items.length} item)
                </span>
                <button
                  type="button"
                  onClick={() => setShowClearConfirm(true)}
                  style={{
                    background: "none",
                    border: "none",
                    color: "var(--tomato, #ff4b3e)",
                    fontSize: "0.75rem",
                    fontWeight: 800,
                    cursor: "pointer",
                    textDecoration: "underline",
                  }}
                >
                  Kosongkan
                </button>
              </div>

              {items.map((item) => (
                <div
                  key={item.variant_id}
                  style={{
                    display: "flex",
                    gap: "0.75rem",
                    padding: "0.875rem",
                    backgroundColor: "#ffffff",
                    border: "2.5px solid var(--ink, #1b1b2f)",
                    borderRadius: "var(--radius-md, 0.875rem)",
                    boxShadow: "3px 3px 0 var(--ink, #1b1b2f)",
                    alignItems: "center",
                  }}
                >
                  <div
                    style={{
                      width: "60px",
                      height: "60px",
                      backgroundColor: "var(--cream, #fff4dc)",
                      border: "2px solid var(--ink, #1b1b2f)",
                      borderRadius: "0.5rem",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      flexShrink: 0,
                    }}
                  >
                    <MascotCan
                      name={item.mascot_svg_key || "can-soda-lemon"}
                      size={50}
                    />
                  </div>

                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div
                      style={{
                        fontFamily: "var(--font-display)",
                        fontSize: "0.9375rem",
                        color: "var(--ink, #1b1b2f)",
                        whiteSpace: "nowrap",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                      }}
                    >
                      {item.product_name}
                    </div>
                    <div style={{ fontSize: "0.75rem", color: "#6a6a80", fontWeight: 700 }}>
                      {item.pack_size === 1 ? "1 Kaleng" : `${item.pack_size}-Pack`} · {formatIdr(item.unit_price)}
                    </div>
                    <div style={{ fontSize: "0.875rem", fontWeight: 800, color: "var(--ink, #1b1b2f)", marginTop: "0.25rem" }}>
                      {formatIdr(item.line_total)}
                    </div>
                  </div>

                  <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-end", gap: "0.35rem" }}>
                    <div style={{ display: "inline-flex", alignItems: "center", gap: "0.35rem" }}>
                      <button
                        type="button"
                        className="btn btn--sm btn--secondary"
                        style={{ width: "28px", height: "28px", minWidth: "28px", minHeight: "28px", padding: 0 }}
                        onClick={() => {
                          if (item.quantity <= 1) {
                            setItemToDelete({ variantId: item.variant_id, name: item.product_name });
                          } else {
                            const newQty = item.quantity - 1;
                            void updateQuantity(item.variant_id, newQty);
                            toast.info(`Jumlah ${item.product_name} diubah menjadi ${newQty}`, "Keranjang");
                          }
                        }}
                        aria-label={`Kurangi jumlah ${item.product_name}`}
                      >
                        -
                      </button>
                      <span style={{ fontFamily: "var(--font-mono)", fontWeight: 800, fontSize: "0.875rem", minWidth: "20px", textAlign: "center" }}>
                        {item.quantity}
                      </span>
                      <button
                        type="button"
                        className="btn btn--sm btn--secondary"
                        style={{ width: "28px", height: "28px", minWidth: "28px", minHeight: "28px", padding: 0 }}
                        onClick={() => {
                          const newQty = item.quantity + 1;
                          void updateQuantity(item.variant_id, newQty);
                          toast.info(`Jumlah ${item.product_name} diubah menjadi ${newQty}`, "Keranjang");
                        }}
                        aria-label={`Tambah jumlah ${item.product_name}`}
                      >
                        +
                      </button>
                    </div>

                    <button
                      type="button"
                      onClick={() => setItemToDelete({ variantId: item.variant_id, name: item.product_name })}
                      style={{
                        background: "none",
                        border: "none",
                        color: "#999",
                        fontSize: "0.7rem",
                        cursor: "pointer",
                        fontWeight: 700,
                      }}
                      aria-label={`Hapus ${item.product_name} dari keranjang`}
                    >
                      Hapus
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {items.length > 0 && (
          <footer
            style={{
              padding: "1.25rem",
              backgroundColor: "var(--cream, #fff4dc)",
              borderTop: "3px solid var(--ink, #1b1b2f)",
            }}
          >
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
              <div>
                <div style={{ fontSize: "0.75rem", color: "#6a6a80", fontWeight: 700 }}>Total Belanja</div>
                <div style={{ fontSize: "1.5rem", fontWeight: 900, color: "var(--ink, #1b1b2f)", fontFamily: "var(--font-body)" }}>
                  {formatIdr(subtotal)}
                </div>
              </div>
              <span className="badge badge--lemon">Termasuk PPN</span>
            </div>

            <button
              type="button"
              className="btn btn--primary"
              style={{ width: "100%", padding: "0.875rem 1rem", fontSize: "1.125rem" }}
              onClick={() => {
                closeCart();
                onProceedToCheckout();
              }}
            >
              Lanjut ke Pembayaran 👉
            </button>
          </footer>
        )}
      </aside>

      <ConfirmModal
        isOpen={itemToDelete !== null}
        title="Hapus Minuman?"
        message={`Hapus minuman ini dari keranjang? (${itemToDelete?.name ?? ""})`}
        confirmText="Hapus"
        cancelText="Batal"
        variant="danger"
        icon="🗑️"
        onConfirm={() => {
          if (itemToDelete) {
            void removeItem(itemToDelete.variantId);
            toast.info(`"${itemToDelete.name}" telah dihapus dari keranjang.`, "Keranjang");
            setItemToDelete(null);
          }
        }}
        onCancel={() => setItemToDelete(null)}
      />

      <ConfirmModal
        isOpen={showClearConfirm}
        title="Kosongkan Keranjang?"
        message="Kosongkan seluruh keranjang?"
        confirmText="Kosongkan"
        cancelText="Batal"
        variant="danger"
        icon="🥫"
        onConfirm={() => {
          void clearCart();
          toast.info("Seluruh keranjang berhasil dikosongkan.", "Keranjang");
          setShowClearConfirm(false);
        }}
        onCancel={() => setShowClearConfirm(false)}
      />
    </>
  );
};
