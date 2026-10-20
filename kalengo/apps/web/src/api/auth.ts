import { apiRequest } from "./client";

export interface User {
  readonly id: string;
  readonly email: string;
  readonly full_name: string;
  readonly phone?: string | null | undefined;
  readonly role: "customer" | "admin" | "staff";
  readonly email_verified_at?: string | null | undefined;
  readonly created_at?: string | undefined;
}

export interface LoginPayload {
  readonly email: string;
  readonly password: string;
}

export interface RegisterPayload {
  readonly email: string;
  readonly password: string;
  readonly full_name: string;
  readonly phone?: string | null | undefined;
  readonly birth_date?: string | null | undefined;
  readonly accept_terms: true;
}

export interface SessionResponse {
  readonly user: User;
  readonly csrf_token?: string | undefined;
}

export interface Address {
  readonly id: string;
  readonly label: string;
  readonly recipient: string;
  readonly phone: string;
  readonly province: string;
  readonly city: string;
  readonly district: string;
  readonly postal_code: string;
  readonly street: string;
  readonly notes?: string | null | undefined;
  readonly area_id?: string | null | undefined;
  readonly is_default: boolean;
}

export type AddressInput = Omit<Address, "id">;

export interface AddressesResponse {
  readonly items: readonly Address[];
}

export async function loginUser(
  payload: LoginPayload,
  signal?: AbortSignal,
): Promise<SessionResponse> {
  return apiRequest<SessionResponse>("/api/v1/auth/login", {
    method: "POST",
    body: payload,
    signal,
  });
}

export async function registerUser(
  payload: RegisterPayload,
  signal?: AbortSignal,
): Promise<{ user: User }> {
  return apiRequest<{ user: User }>("/api/v1/auth/register", {
    method: "POST",
    body: payload,
    signal,
  });
}

export async function fetchMe(signal?: AbortSignal): Promise<User> {
  return apiRequest<User>("/api/v1/me", { signal });
}

export async function logoutUser(signal?: AbortSignal): Promise<void> {
  return apiRequest<void>("/api/v1/auth/logout", {
    method: "POST",
    signal,
  });
}

export async function fetchAddresses(signal?: AbortSignal): Promise<readonly Address[]> {
  const response = await apiRequest<AddressesResponse>("/api/v1/me/addresses", { signal });
  return response.items;
}

export async function createAddress(
  payload: AddressInput,
  signal?: AbortSignal,
): Promise<Address> {
  return apiRequest<Address>("/api/v1/me/addresses", {
    method: "POST",
    body: payload,
    signal,
  });
}
