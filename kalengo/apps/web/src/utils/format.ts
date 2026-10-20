/**
 * Format angka ke format Rupiah standar KALENGO (§11.1).
 * Contoh: 6500 -> "Rp6.500", 138000 -> "Rp138.000"
 */
export function formatIdr(amount: number): string {
  return `Rp${Math.round(amount).toLocaleString("id-ID")}`;
}
