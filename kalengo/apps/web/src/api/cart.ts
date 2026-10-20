import { apiRequest } from "./client";

export interface CartIssue {
  readonly code: string;
  readonly message: string;
  readonly variant_id?: string;
  readonly product_name?: string;
  readonly old_price?: number;
  readonly new_price?: number;
  readonly max_available?: number;
}

export interface CartItem {
  readonly variant_id: string;
  readonly product_id: string;
  readonly product_slug: string;
  readonly product_name: string;
  readonly sku: string;
  readonly volume_ml: number;
  readonly pack_size: number;
  readonly quantity: number;
  readonly added_price: number;
  readonly unit_price: number;
  readonly line_total: number;
  readonly price_changed?: boolean | undefined;
  readonly available?: boolean | undefined;
  readonly is_available?: boolean | undefined;
  readonly max_per_order?: number | undefined;
  readonly mascot_svg_key?: string | null | undefined;
}

export interface Cart {
  readonly items: readonly CartItem[];
  readonly count: number;
  readonly subtotal: number;
  readonly currency: string;
  readonly issues: readonly CartIssue[];
}

export interface CartMergeSummary {
  readonly requested_count: number;
  readonly added_count: number;
  readonly updated_count: number;
  readonly skipped_count: number;
}

export interface CartMergeResponse {
  readonly cart: Cart;
  readonly merge: CartMergeSummary;
}

export interface GuestCartItem {
  readonly variant_id: string;
  readonly product_id: string;
  readonly product_slug: string;
  readonly product_name: string;
  readonly sku: string;
  readonly volume_ml: number;
  readonly pack_size: number;
  readonly quantity: number;
  readonly unit_price: number;
  readonly line_total: number;
  readonly mascot_svg_key?: string | null | undefined;
  readonly max_per_order?: number | undefined;
}

const GUEST_CART_KEY = "kalengo_guest_cart_v1";

export function loadGuestCart(): GuestCartItem[] {
  if (typeof window === "undefined" || !window.localStorage) return [];
  try {
    const raw = window.localStorage.getItem(GUEST_CART_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as GuestCartItem[]) : [];
  } catch {
    return [];
  }
}

export function saveGuestCart(items: readonly GuestCartItem[]): void {
  if (typeof window === "undefined" || !window.localStorage) return;
  try {
    window.localStorage.setItem(GUEST_CART_KEY, JSON.stringify(items));
  } catch {
    // Abaikan jika storage penuh atau kuota habis
  }
}

export function clearGuestCart(): void {
  if (typeof window === "undefined" || !window.localStorage) return;
  try {
    window.localStorage.removeItem(GUEST_CART_KEY);
  } catch {
    // Abaikan
  }
}

export async function fetchCart(signal?: AbortSignal): Promise<Cart> {
  return apiRequest<Cart>("/api/v1/cart", { signal });
}

export async function setServerCartItem(
  variantId: string,
  quantity: number,
  signal?: AbortSignal,
): Promise<Cart> {
  return apiRequest<Cart>(`/api/v1/cart/items/${encodeURIComponent(variantId)}`, {
    method: "PUT",
    body: { quantity },
    signal,
  });
}

export async function removeServerCartItem(
  variantId: string,
  signal?: AbortSignal,
): Promise<Cart> {
  return apiRequest<Cart>(`/api/v1/cart/items/${encodeURIComponent(variantId)}`, {
    method: "DELETE",
    signal,
  });
}

export async function clearServerCart(signal?: AbortSignal): Promise<void> {
  return apiRequest<void>("/api/v1/cart", {
    method: "DELETE",
    signal,
  });
}

export async function mergeServerCart(
  items: readonly { variant_id: string; quantity: number }[],
  signal?: AbortSignal,
): Promise<CartMergeResponse> {
  return apiRequest<CartMergeResponse>("/api/v1/cart/merge", {
    method: "POST",
    body: { items },
    signal,
  });
}

export async function validateServerCart(signal?: AbortSignal): Promise<Cart> {
  return apiRequest<Cart>("/api/v1/cart/validate", {
    method: "POST",
    signal,
  });
}
