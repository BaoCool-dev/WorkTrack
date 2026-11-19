/** Ghép class Tailwind có điều kiện; bỏ qua giá trị rỗng/false. */
export function cn(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(' ')
}
