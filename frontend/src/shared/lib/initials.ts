/** Tên tiếng Việt: lấy chữ cái đầu của hai từ cuối ("Nguyễn Văn An" → "VA"). */
export function initials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean)
  const picked = words.length > 1 ? words.slice(-2) : words
  return picked.map((w) => w.charAt(0).toLocaleUpperCase('vi-VN')).join('')
}
