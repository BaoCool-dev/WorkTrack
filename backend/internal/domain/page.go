package domain

// B9: ?page=1&pageSize=20, tối đa 100.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	// MaxPage giữ offset (MaxPage*MaxPageSize) trong phạm vi int32 của PostgreSQL.
	MaxPage = 1_000_000
)

// PageRequest là yêu cầu phân trang đã chuẩn hóa.
type PageRequest struct {
	Page     int
	PageSize int
}

// NewPageRequest chuẩn hóa: page < 1 → 1; pageSize < 1 → 20; pageSize > 100 → 100.
func NewPageRequest(page, pageSize int) PageRequest {
	if page < 1 {
		page = 1
	}
	if page > MaxPage {
		page = MaxPage
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return PageRequest{Page: page, PageSize: pageSize}
}

// Offset là số dòng bỏ qua.
func (p PageRequest) Offset() int { return (p.Page - 1) * p.PageSize }

// Page là kết quả một trang: {items, page, pageSize, total}.
type Page[T any] struct {
	Items    []T
	Page     int
	PageSize int
	Total    int
}
