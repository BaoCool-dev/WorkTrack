const numberFormat = new Intl.NumberFormat('vi-VN')

/** 5480 → "5.480" */
export function formatNumber(n: number): string {
  return numberFormat.format(n)
}
