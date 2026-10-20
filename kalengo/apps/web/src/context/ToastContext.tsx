import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  ToastContext,
  type ToastContextType,
  type ToastItem,
  type ToastOptions,
  type ToastType,
} from "./useToast";

const DEFAULT_DURATION = 4000;

interface ToastMiniMascotProps {
  type: ToastType;
}

const ToastMiniMascot: React.FC<ToastMiniMascotProps> = ({ type }) => {
  if (type === "success") {
    return (
      <svg
        viewBox="0 0 36 36"
        width="34"
        height="34"
        aria-hidden="true"
        style={{ flexShrink: 0 }}
      >
        <rect x="6" y="8" width="24" height="24" rx="5" fill="#ffd23f" stroke="#1b1b2f" strokeWidth="2.5" />
        <ellipse cx="18" cy="8" rx="10" ry="3" fill="#ffe9bd" stroke="#1b1b2f" strokeWidth="2" />
        {/* Left winking eye */}
        <path d="M 11 18 Q 14 15 17 18" fill="none" stroke="#1b1b2f" strokeWidth="2.5" strokeLinecap="round" />
        {/* Right pie-eye */}
        <circle cx="23" cy="18" r="2.5" fill="#1b1b2f" />
        {/* Smile */}
        <path d="M 13 23 Q 18 28 23 23" fill="none" stroke="#1b1b2f" strokeWidth="2.5" strokeLinecap="round" />
      </svg>
    );
  }

  if (type === "error") {
    return (
      <svg
        viewBox="0 0 36 36"
        width="34"
        height="34"
        aria-hidden="true"
        style={{ flexShrink: 0 }}
      >
        <rect x="6" y="8" width="24" height="24" rx="5" fill="#ff4b3e" stroke="#1b1b2f" strokeWidth="2.5" />
        <ellipse cx="18" cy="8" rx="10" ry="3" fill="#ffb4ab" stroke="#1b1b2f" strokeWidth="2" />
        {/* Shocked/X eyes */}
        <line x1="12" y1="16" x2="16" y2="20" stroke="#1b1b2f" strokeWidth="2.5" strokeLinecap="round" />
        <line x1="16" y1="16" x2="12" y2="20" stroke="#1b1b2f" strokeWidth="2.5" strokeLinecap="round" />
        <line x1="20" y1="16" x2="24" y2="20" stroke="#1b1b2f" strokeWidth="2.5" strokeLinecap="round" />
        <line x1="24" y1="16" x2="20" y2="20" stroke="#1b1b2f" strokeWidth="2.5" strokeLinecap="round" />
        {/* Open mouth */}
        <ellipse cx="18" cy="25" rx="3.5" ry="2.5" fill="#1b1b2f" />
      </svg>
    );
  }

  if (type === "warning") {
    return (
      <svg
        viewBox="0 0 36 36"
        width="34"
        height="34"
        aria-hidden="true"
        style={{ flexShrink: 0 }}
      >
        <rect x="6" y="8" width="24" height="24" rx="5" fill="#ffd23f" stroke="#1b1b2f" strokeWidth="2.5" />
        <ellipse cx="18" cy="8" rx="10" ry="3" fill="#ffe9bd" stroke="#1b1b2f" strokeWidth="2" />
        {/* Surprised round eyes */}
        <circle cx="13" cy="18" r="2.5" fill="#1b1b2f" />
        <circle cx="23" cy="18" r="2.5" fill="#1b1b2f" />
        {/* O-mouth */}
        <circle cx="18" cy="24" r="2" fill="#1b1b2f" />
      </svg>
    );
  }

  // Info
  return (
    <svg
      viewBox="0 0 36 36"
      width="34"
      height="34"
      aria-hidden="true"
      style={{ flexShrink: 0 }}
    >
      <rect x="6" y="8" width="24" height="24" rx="5" fill="#2ec4f1" stroke="#1b1b2f" strokeWidth="2.5" />
      <ellipse cx="18" cy="8" rx="10" ry="3" fill="#b9efff" stroke="#1b1b2f" strokeWidth="2" />
      {/* Friendly glance eyes */}
      <circle cx="14" cy="18" r="2.5" fill="#1b1b2f" />
      <circle cx="22" cy="18" r="2.5" fill="#1b1b2f" />
      {/* Friendly smile */}
      <path d="M 14 23 Q 18 26 22 23" fill="none" stroke="#1b1b2f" strokeWidth="2.5" strokeLinecap="round" />
    </svg>
  );
};

const ToastCard: React.FC<{
  toast: ToastItem;
  onDismiss: (id: string) => void;
}> = ({ toast, onDismiss }) => {
  const badgeColors: Record<ToastType, { bg: string; text: string; label: string }> = {
    success: { bg: "var(--mint, #3ddc97)", text: "var(--ink, #1b1b2f)", label: "SUKSES" },
    error: { bg: "var(--tomato, #ff4b3e)", text: "#ffffff", label: "GAGAL" },
    warning: { bg: "var(--lemon, #ffd23f)", text: "var(--ink, #1b1b2f)", label: "PERHATIAN" },
    info: { bg: "var(--sky, #2ec4f1)", text: "var(--ink, #1b1b2f)", label: "INFO" },
  };

  const badge = badgeColors[toast.type];

  return (
    <div
      className={`kalengo-toast kalengo-toast--${toast.type}`}
      role={toast.type === "error" ? "alert" : "status"}
      aria-live={toast.type === "error" ? "assertive" : "polite"}
      style={{
        animationDuration: `${toast.duration}ms`,
      }}
    >
      <div className="kalengo-toast__content">
        <ToastMiniMascot type={toast.type} />
        <div className="kalengo-toast__body">
          <div className="kalengo-toast__header">
            <span
              className="badge"
              style={{
                backgroundColor: badge.bg,
                color: badge.text,
                fontSize: "0.6875rem",
                padding: "0.15rem 0.45rem",
                border: "2px solid var(--ink, #1b1b2f)",
                fontWeight: 900,
              }}
            >
              {badge.label}
            </span>
            {toast.title && (
              <strong className="kalengo-toast__title">{toast.title}</strong>
            )}
          </div>
          <p className="kalengo-toast__message">{toast.message}</p>
        </div>

        <button
          type="button"
          className="kalengo-toast__close"
          onClick={() => onDismiss(toast.id)}
          aria-label="Tutup notifikasi"
        >
          ✕
        </button>
      </div>

      {toast.duration > 0 && (
        <div
          className="kalengo-toast__progress-bar"
          style={{
            animationDuration: `${toast.duration}ms`,
            backgroundColor: badge.bg,
          }}
        />
      )}
    </div>
  );
};

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<readonly ToastItem[]>([]);
  const timeoutsRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  const dismissToast = useCallback((id: string) => {
    const existing = timeoutsRef.current.get(id);
    if (existing) {
      clearTimeout(existing);
      timeoutsRef.current.delete(id);
    }
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const showToast = useCallback(
    (options: ToastOptions | string): string => {
      const opts: ToastOptions = typeof options === "string" ? { message: options } : options;
      const id = `toast-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`;
      const duration = opts.duration ?? DEFAULT_DURATION;
      const type = opts.type ?? "info";

      const newItem: ToastItem = {
        id,
        type,
        message: opts.message,
        title: opts.title,
        duration,
      };

      setToasts((prev) => [newItem, ...prev.slice(0, 4)]); // max 5 toasts concurrent

      if (duration > 0) {
        const timeout = setTimeout(() => {
          dismissToast(id);
        }, duration);
        timeoutsRef.current.set(id, timeout);
      }

      return id;
    },
    [dismissToast],
  );

  const success = useCallback(
    (message: string, title?: string, duration?: number) =>
      showToast({ type: "success", message, title, duration }),
    [showToast],
  );

  const error = useCallback(
    (message: string, title?: string, duration?: number) =>
      showToast({ type: "error", message, title, duration }),
    [showToast],
  );

  const info = useCallback(
    (message: string, title?: string, duration?: number) =>
      showToast({ type: "info", message, title, duration }),
    [showToast],
  );

  const warning = useCallback(
    (message: string, title?: string, duration?: number) =>
      showToast({ type: "warning", message, title, duration }),
    [showToast],
  );

  // Bersihkan timeout saat unmount
  useEffect(() => {
    const current = timeoutsRef.current;
    return () => {
      current.forEach((t) => clearTimeout(t));
      current.clear();
    };
  }, []);

  const contextValue = useMemo<ToastContextType>(
    () => ({
      toasts,
      showToast,
      success,
      error,
      info,
      warning,
      dismissToast,
    }),
    [toasts, showToast, success, error, info, warning, dismissToast],
  );

  return (
    <ToastContext.Provider value={contextValue}>
      {children}
      <div
        className="kalengo-toast-container"
        role="region"
        aria-label="Pemberitahuan Sistem"
      >
        {toasts.map((toast) => (
          <ToastCard key={toast.id} toast={toast} onDismiss={dismissToast} />
        ))}
      </div>
    </ToastContext.Provider>
  );
};

export { useToast } from "./useToast";
export type { ToastContextType, ToastItem, ToastOptions, ToastType } from "./useToast";
