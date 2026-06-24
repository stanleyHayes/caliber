package kernel

// Pagination defaults applied across every collection query.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Page is a normalized, 1-based pagination request.
type Page struct {
	Number int
	Size   int
}

// NewPage clamps raw inputs into a valid Page (number >= 1, 1 <= size <= max).
func NewPage(number, size int) Page {
	if number < 1 {
		number = 1
	}
	switch {
	case size < 1:
		size = DefaultPageSize
	case size > MaxPageSize:
		size = MaxPageSize
	}
	return Page{Number: number, Size: size}
}

// Offset is the SQL offset for this page.
func (p Page) Offset() int { return (p.Number - 1) * p.Size }

// Limit is the SQL limit for this page.
func (p Page) Limit() int { return p.Size }

// TotalPages computes the number of pages for a total item count and page size.
func TotalPages(total int64, size int) int {
	if size <= 0 || total <= 0 {
		return 0
	}
	return int((total + int64(size) - 1) / int64(size))
}
