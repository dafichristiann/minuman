import { apiRequest } from "./client";

export interface ShippingRate {
  readonly courier: string;
  readonly service: string;
  readonly description?: string | null | undefined;
  readonly cost: number;
  readonly etd: string;
  readonly cache_ttl_seconds?: number | undefined;
}

export interface ShippingRatesResponse {
  readonly items: readonly ShippingRate[];
}

export interface ShippingRatesParams {
  readonly address_id: string;
  readonly courier?: string | undefined;
}

export interface VoucherResult {
  readonly code: string;
  readonly type: "percent" | "fixed" | "free_shipping";
  readonly discount: number;
  readonly free_shipping?: boolean | undefined;
  readonly message?: string | undefined;
}

export interface OrderItem {
  readonly id?: string | undefined;
  readonly variant_id: string;
  readonly product_name: string;
  readonly sku?: string | undefined;
  readonly pack_size?: number | undefined;
  readonly quantity: number;
  readonly unit_price?: number | undefined;
  readonly line_total: number;
}

export type OrderStatus =
  | "pending_payment"
  | "paid"
  | "processing"
  | "shipped"
  | "delivered"
  | "completed"
  | "cancelled"
  | "expired"
  | "refund_requested"
  | "refunded";

export interface Order {
  readonly id: string;
  readonly order_no: string;
  readonly status: OrderStatus;
  readonly subtotal: number;
  readonly discount: number;
  readonly shipping_cost: number;
  readonly total: number;
  readonly courier?: string | null | undefined;
  readonly courier_service?: string | null | undefined;
  readonly expires_at: string;
  readonly created_at: string;
  readonly paid_at?: string | null | undefined;
  readonly shipped_at?: string | null | undefined;
  readonly delivered_at?: string | null | undefined;
  readonly items?: readonly OrderItem[] | undefined;
}

export interface OrderDetail extends Order {
  readonly address_snapshot?: Record<string, unknown> | undefined;
  readonly payment?: {
    readonly method?: string | null | undefined;
    readonly paid_at?: string | null | undefined;
    readonly amount?: number | undefined;
  } | undefined;
}

export interface CreateOrderPayload {
  readonly address_id: string;
  readonly courier: string;
  readonly courier_service: string;
  readonly voucher_code?: string | null | undefined;
  readonly notes?: string | null | undefined;
  readonly accept_terms: true;
}

export interface PaymentSession {
  readonly order_no: string;
  readonly total: number;
  readonly provider: string;
  readonly token: string;
  readonly redirect_url?: string | undefined;
  readonly expires_at: string;
}

export function generateIdempotencyKey(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `klg-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

export async function fetchShippingRates(
  params: ShippingRatesParams,
  signal?: AbortSignal,
): Promise<readonly ShippingRate[]> {
  const response = await apiRequest<ShippingRatesResponse>("/api/v1/shipping/rates", {
    method: "POST",
    body: params,
    signal,
  });
  return response.items;
}

export async function validateVoucher(
  code: string,
  signal?: AbortSignal,
): Promise<VoucherResult> {
  return apiRequest<VoucherResult>("/api/v1/vouchers/validate", {
    method: "POST",
    body: { code },
    signal,
  });
}

export async function createOrder(
  payload: CreateOrderPayload,
  idempotencyKey?: string,
  signal?: AbortSignal,
): Promise<Order> {
  const key = idempotencyKey || generateIdempotencyKey();
  return apiRequest<Order>("/api/v1/orders", {
    method: "POST",
    body: payload,
    idempotencyKey: key,
    signal,
  });
}

export async function createPaymentSession(
  orderId: string,
  signal?: AbortSignal,
): Promise<PaymentSession> {
  return apiRequest<PaymentSession>(`/api/v1/orders/${encodeURIComponent(orderId)}/pay`, {
    method: "POST",
    signal,
  });
}

export async function fetchOrderDetail(
  orderId: string,
  signal?: AbortSignal,
): Promise<OrderDetail> {
  return apiRequest<OrderDetail>(`/api/v1/orders/${encodeURIComponent(orderId)}`, {
    signal,
  });
}
