import React, { useEffect, useRef } from "react";

export interface ConfirmModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: "danger" | "warning" | "primary";
  icon?: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export const ConfirmModal: React.FC<ConfirmModalProps> = ({
  isOpen,
  title,
  message,
  confirmText = "Ya, Lanjutkan",
  cancelText = "Batal",
  variant = "danger",
  icon,
  onConfirm,
  onCancel,
}) => {
  const confirmBtnRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (isOpen) {
      // Fokus ke tombol aksi saat terbuka
      confirmBtnRef.current?.focus();
    }
  }, [isOpen]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        onCancel();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onCancel]);

  if (!isOpen) return null;

  const defaultIcon =
    icon || (variant === "danger" ? "🗑️" : variant === "warning" ? "⚠️" : "✨");

  const headerBg =
    variant === "danger"
      ? "var(--tomato, #ff4b3e)"
      : variant === "warning"
        ? "var(--lemon, #ffd23f)"
        : "var(--mint, #3ddc97)";

  const headerColor =
    variant === "danger" ? "#ffffff" : "var(--ink, #1b1b2f)";

  const confirmBtnClass =
    variant === "danger"
      ? "btn--primary"
      : variant === "warning"
        ? "btn--lemon"
        : "btn--mint";

  return (
    <div
      className="modal-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel();
      }}
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="confirm-modal-title"
      aria-describedby="confirm-modal-desc"
    >
      <div
        className="modal-window modal-window--confirm"
        style={{ maxWidth: "440px" }}
      >
        <header
          className="modal-header"
          style={{
            backgroundColor: headerBg,
            color: headerColor,
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ fontSize: "1.5rem" }} aria-hidden="true">
              {defaultIcon}
            </span>
            <h2
              id="confirm-modal-title"
              className="modal-title"
              style={{ color: headerColor, fontSize: "1.25rem" }}
            >
              {title}
            </h2>
          </div>
          <button
            type="button"
            className="btn btn--sm btn--secondary"
            onClick={onCancel}
            aria-label="Tutup dialog konfirmasi"
            style={{
              padding: "0.25rem 0.5rem",
              minHeight: "32px",
              minWidth: "32px",
            }}
          >
            ✕
          </button>
        </header>

        <div className="modal-body" style={{ padding: "1.5rem 1.25rem" }}>
          <div
            style={{
              display: "flex",
              alignItems: "flex-start",
              gap: "1rem",
            }}
          >
            {/* Rubber-hose mascot alert icon */}
            <div
              style={{
                width: "48px",
                height: "48px",
                borderRadius: "50%",
                backgroundColor: headerBg,
                border: "2.5px solid var(--ink, #1b1b2f)",
                boxShadow: "2px 2px 0 var(--ink, #1b1b2f)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: "1.5rem",
                flexShrink: 0,
              }}
              aria-hidden="true"
            >
              {defaultIcon}
            </div>

            <p
              id="confirm-modal-desc"
              style={{
                fontFamily: "var(--font-body)",
                fontSize: "0.9375rem",
                color: "var(--ink, #1b1b2f)",
                lineHeight: 1.5,
                margin: 0,
                fontWeight: 600,
                whiteSpace: "pre-line",
              }}
            >
              {message}
            </p>
          </div>
        </div>

        <footer
          className="modal-footer"
          style={{
            justifyContent: "flex-end",
            gap: "0.75rem",
            padding: "1rem 1.25rem",
          }}
        >
          <button
            type="button"
            className="btn btn--sm btn--secondary"
            onClick={onCancel}
          >
            {cancelText}
          </button>
          <button
            ref={confirmBtnRef}
            type="button"
            className={`btn btn--sm ${confirmBtnClass}`}
            onClick={onConfirm}
          >
            {confirmText}
          </button>
        </footer>
      </div>
    </div>
  );
};
