import { createContext, useContext } from "react";

export type ToastType = "success" | "error" | "info" | "warning";

export interface ToastItem {
  readonly id: string;
  readonly type: ToastType;
  readonly message: string;
  readonly title?: string | undefined;
  readonly duration: number;
}

export interface ToastOptions {
  type?: ToastType | undefined;
  message: string;
  title?: string | undefined;
  duration?: number | undefined;
}

export interface ToastContextType {
  toasts: readonly ToastItem[];
  showToast: (options: ToastOptions | string) => string;
  success: (message: string, title?: string, duration?: number) => string;
  error: (message: string, title?: string, duration?: number) => string;
  info: (message: string, title?: string, duration?: number) => string;
  warning: (message: string, title?: string, duration?: number) => string;
  dismissToast: (id: string) => void;
}

export const ToastContext = createContext<ToastContextType | undefined>(undefined);

const NOOP_TOAST: ToastContextType = {
  toasts: [],
  showToast: () => "",
  success: () => "",
  error: () => "",
  info: () => "",
  warning: () => "",
  dismissToast: () => {},
};

export function useToast(): ToastContextType {
  const context = useContext(ToastContext);
  return context ?? NOOP_TOAST;
}
