import { apiRequest } from "./client";

export interface Category {
  readonly id: string;
  readonly slug: string;
  readonly name: string;
  readonly parent_id?: string | null | undefined;
  readonly sort_order?: number | undefined;
}

export interface Brand {
  readonly id: string;
  readonly slug: string;
  readonly name: string;
}

export interface ProductVariant {
  readonly id: string;
  readonly sku: string;
  readonly volume_ml: number;
  readonly pack_size: number;
  readonly price: number;
  readonly compare_at_price?: number | null | undefined;
  readonly weight_gram: number;
  readonly max_per_order?: number | undefined;
  readonly stock_label?: string | undefined;
  readonly available?: boolean | undefined;
}

export interface NutritionFacts {
  readonly energi_kcal?: number | undefined;
  readonly gula_g?: number | undefined;
  readonly natrium_mg?: number | undefined;
  readonly kafein_mg?: number | undefined;
  readonly porsi_ml?: number | undefined;
  readonly [key: string]: unknown;
}

export interface ProductImage {
  readonly url: string;
  readonly alt?: string | undefined;
}

export interface Product {
  readonly id: string;
  readonly slug: string;
  readonly name: string;
  readonly flavor?: string | null | undefined;
  readonly brand?: Brand | null | undefined;
  readonly category?: Category | null | undefined;
  readonly mascot_svg_key?: string | null | undefined;
  readonly requires_age_verification?: boolean | undefined;
  readonly price_from?: number | undefined;
  readonly stock_label?: string | undefined;
  readonly badges?: readonly string[] | undefined;
  readonly description?: string | undefined;
}

export interface ProductDetail extends Product {
  readonly description: string;
  readonly nutrition?: NutritionFacts | undefined;
  readonly images?: readonly ProductImage[] | undefined;
  readonly variants: readonly ProductVariant[];
  readonly rating_avg?: number | null | undefined;
  readonly rating_count?: number | undefined;
}

export interface ProductListResponse {
  readonly items: readonly Product[];
  readonly has_more: boolean;
  readonly next_cursor: string | null;
}

export interface CategoriesResponse {
  readonly items: readonly Category[];
}

export interface BrandsResponse {
  readonly items: readonly Brand[];
}

export interface ProductQueryParams {
  readonly q?: string | undefined;
  readonly category?: string | undefined;
  readonly brand?: string | undefined;
  readonly min_price?: number | undefined;
  readonly max_price?: number | undefined;
  readonly in_stock?: boolean | undefined;
  readonly sort?: ("newest" | "price_asc" | "price_desc" | "name" | "popular") | undefined;
  readonly cursor?: string | undefined;
  readonly limit?: number | undefined;
}

export async function fetchProducts(
  params: ProductQueryParams = {},
  signal?: AbortSignal,
): Promise<ProductListResponse> {
  const query = new URLSearchParams();
  if (params.q) query.set("q", params.q);
  if (params.category) query.set("category", params.category);
  if (params.brand) query.set("brand", params.brand);
  if (params.min_price !== undefined) query.set("min_price", String(params.min_price));
  if (params.max_price !== undefined) query.set("max_price", String(params.max_price));
  if (params.in_stock !== undefined) query.set("in_stock", String(params.in_stock));
  if (params.sort) query.set("sort", params.sort);
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.limit !== undefined) query.set("limit", String(params.limit));

  const qs = query.toString();
  const path = qs ? `/api/v1/products?${qs}` : "/api/v1/products";
  return apiRequest<ProductListResponse>(path, { signal });
}

export async function fetchProductDetail(
  slugOrId: string,
  signal?: AbortSignal,
): Promise<ProductDetail> {
  return apiRequest<ProductDetail>(`/api/v1/products/${encodeURIComponent(slugOrId)}`, { signal });
}

export async function fetchCategories(signal?: AbortSignal): Promise<readonly Category[]> {
  const response = await apiRequest<CategoriesResponse>("/api/v1/categories", { signal });
  return response.items;
}

export async function fetchBrands(signal?: AbortSignal): Promise<readonly Brand[]> {
  const response = await apiRequest<BrandsResponse>("/api/v1/brands", { signal });
  return response.items;
}
