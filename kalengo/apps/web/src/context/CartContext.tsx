import React, { createContext, useCallback, useEffect, useMemo, useState } from "react";
import {
  clearGuestCart,
  clearServerCart,
  fetchCart,
  loadGuestCart,
  mergeServerCart,
  removeServerCartItem,
  saveGuestCart,
  setServerCartItem,
  type CartIssue,
  type CartItem,
  type GuestCartItem,
} from "../api/cart";
import type { Product, ProductVariant } from "../api/catalog";
import { useAuth } from "./AuthContext";

export interface CartContextType {
  readonly items: readonly GuestCartItem[];
  readonly count: number;
  readonly subtotal: number;
  readonly isOpen: boolean;
  readonly isLoading: boolean;
  readonly issues: readonly CartIssue[];
  openCart: () => void;
  closeCart: () => void;
  toggleCart: () => void;
  addItem: (product: Product, variant: ProductVariant, quantity?: number) => Promise<void>;
  updateQuantity: (variantId: string, quantity: number) => Promise<void>;
  removeItem: (variantId: string) => Promise<void>;
  clearCart: () => Promise<void>;
}

export const CartContext = createContext<CartContextType | undefined>(undefined);

function mapServerItemToGuestItem(item: CartItem): GuestCartItem {
  return {
    variant_id: item.variant_id,
    product_id: item.product_id,
    product_slug: item.product_slug,
    product_name: item.product_name,
    sku: item.sku,
    volume_ml: item.volume_ml,
    pack_size: item.pack_size,
    quantity: item.quantity,
    unit_price: item.unit_price,
    line_total: item.line_total,
    mascot_svg_key: item.mascot_svg_key,
    max_per_order: item.max_per_order,
  };
}

export function CartProvider({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth();
  const [items, setItems] = useState<readonly GuestCartItem[]>(() => loadGuestCart());
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [issues, setIssues] = useState<readonly CartIssue[]>([]);

  // Sinkronisasi saat status login berubah
  useEffect(() => {
    let active = true;

    async function syncCart() {
      if (!isAuthenticated) {
        // Mode tamu: muat dari localStorage
        const local = loadGuestCart();
        if (active) setItems(local);
        return;
      }

      setIsLoading(true);
      try {
        const guestItems = loadGuestCart();
        if (guestItems.length > 0) {
          const mergePayload = guestItems.map((item) => ({
            variant_id: item.variant_id,
            quantity: item.quantity,
          }));
          const mergeRes = await mergeServerCart(mergePayload);
          clearGuestCart();
          if (active) {
            setItems(mergeRes.cart.items.map(mapServerItemToGuestItem));
            setIssues(mergeRes.cart.issues);
          }
        } else {
          const serverCart = await fetchCart();
          if (active) {
            setItems(serverCart.items.map(mapServerItemToGuestItem));
            setIssues(serverCart.issues);
          }
        }
      } catch {
        // Fallback ke keranjang lokal jika server belum dapat dijangkau
        if (active) setItems(loadGuestCart());
      } finally {
        if (active) setIsLoading(false);
      }
    }

    void syncCart();

    return () => {
      active = false;
    };
  }, [isAuthenticated]);

  const openCart = useCallback(() => setIsOpen(true), []);
  const closeCart = useCallback(() => setIsOpen(false), []);
  const toggleCart = useCallback(() => setIsOpen((prev) => !prev), []);

  const addItem = useCallback(
    async (product: Product, variant: ProductVariant, quantity = 1) => {
      const maxLimit = variant.max_per_order ?? 24;

      if (isAuthenticated) {
        setIsLoading(true);
        try {
          const existing = items.find((i) => i.variant_id === variant.id);
          const nextQty = Math.min(maxLimit, (existing ? existing.quantity : 0) + quantity);
          const updated = await setServerCartItem(variant.id, nextQty);
          setItems(updated.items.map(mapServerItemToGuestItem));
          setIssues(updated.issues);
          setIsOpen(true);
          return;
        } catch {
          // Fallback lokal jika ada error
        } finally {
          setIsLoading(false);
        }
      }

      // Mode tamu (localStorage)
      setItems((prev) => {
        const existingIdx = prev.findIndex((i) => i.variant_id === variant.id);
        let next: GuestCartItem[];
        const current = existingIdx >= 0 ? prev[existingIdx] : undefined;
        if (current) {
          const newQty = Math.min(maxLimit, current.quantity + quantity);
          const updatedItem: GuestCartItem = {
            ...current,
            quantity: newQty,
            line_total: newQty * current.unit_price,
          };
          next = [...prev.slice(0, existingIdx), updatedItem, ...prev.slice(existingIdx + 1)];
        } else {
          const newItem: GuestCartItem = {
            variant_id: variant.id,
            product_id: product.id,
            product_slug: product.slug,
            product_name: product.name,
            sku: variant.sku,
            volume_ml: variant.volume_ml,
            pack_size: variant.pack_size,
            quantity: Math.min(maxLimit, quantity),
            unit_price: variant.price,
            line_total: Math.min(maxLimit, quantity) * variant.price,
            mascot_svg_key: product.mascot_svg_key,
            max_per_order: maxLimit,
          };
          next = [...prev, newItem];
        }
        saveGuestCart(next);
        return next;
      });
      setIsOpen(true);
    },
    [isAuthenticated, items],
  );

  const updateQuantity = useCallback(
    async (variantId: string, quantity: number) => {
      if (quantity <= 0) {
        return;
      }

      if (isAuthenticated) {
        setIsLoading(true);
        try {
          const updated = await setServerCartItem(variantId, quantity);
          setItems(updated.items.map(mapServerItemToGuestItem));
          setIssues(updated.issues);
          return;
        } catch {
          // Lanjut ke fallback lokal
        } finally {
          setIsLoading(false);
        }
      }

      setItems((prev) => {
        const next = prev.map((item) => {
          if (item.variant_id === variantId) {
            const capped = Math.min(item.max_per_order ?? 24, quantity);
            return {
              ...item,
              quantity: capped,
              line_total: capped * item.unit_price,
            };
          }
          return item;
        });
        saveGuestCart(next);
        return next;
      });
    },
    [isAuthenticated],
  );

  const removeItem = useCallback(
    async (variantId: string) => {
      if (isAuthenticated) {
        setIsLoading(true);
        try {
          const updated = await removeServerCartItem(variantId);
          setItems(updated.items.map(mapServerItemToGuestItem));
          setIssues(updated.issues);
          return;
        } catch {
          // Lanjut ke fallback lokal
        } finally {
          setIsLoading(false);
        }
      }

      setItems((prev) => {
        const next = prev.filter((item) => item.variant_id !== variantId);
        saveGuestCart(next);
        return next;
      });
    },
    [isAuthenticated],
  );

  const clearCart = useCallback(async () => {
    if (isAuthenticated) {
      setIsLoading(true);
      try {
        await clearServerCart();
      } catch {
        // Abaikan
      } finally {
        setIsLoading(false);
      }
    }
    clearGuestCart();
    setItems([]);
    setIssues([]);
  }, [isAuthenticated]);

  const count = useMemo(() => items.reduce((sum, item) => sum + item.quantity, 0), [items]);
  const subtotal = useMemo(() => items.reduce((sum, item) => sum + item.line_total, 0), [items]);

  const value = useMemo<CartContextType>(
    () => ({
      items,
      count,
      subtotal,
      isOpen,
      isLoading,
      issues,
      openCart,
      closeCart,
      toggleCart,
      addItem,
      updateQuantity,
      removeItem,
      clearCart,
    }),
    [items, count, subtotal, isOpen, isLoading, issues, openCart, closeCart, toggleCart, addItem, updateQuantity, removeItem, clearCart],
  );

  return <CartContext.Provider value={value}>{children}</CartContext.Provider>;
}

export { useCart } from "./useCart";
