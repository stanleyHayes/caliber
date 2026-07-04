package wiring

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// TestEachPageProcessesEveryRowThroughClamp guards the CAL-117 re-encryption
// data-loss defect: kernel.NewPage clamps the page size to MaxPageSize, so a
// lister delivers fewer rows per page than a naive page*pageSize count assumes.
// eachPage must process EVERY row (terminating on the DELIVERED count), or a key
// rotation would silently leave the tail of a large table sealed with the old key
// and then unreadable once that key is retired.
func TestEachPageProcessesEveryRowThroughClamp(t *testing.T) {
	// A lister that behaves like the real repos: it honors the page's clamped
	// offset/limit, so requesting a size above MaxPageSize still returns at most
	// MaxPageSize rows per page.
	newLister := func(total int) func(context.Context, kernel.Page) ([]int, int64, error) {
		data := make([]int, total)
		for i := range data {
			data[i] = i
		}
		return func(_ context.Context, p kernel.Page) ([]int, int64, error) {
			lo := min(p.Offset(), total)
			hi := min(lo+p.Limit(), total)
			return data[lo:hi], int64(total), nil
		}
	}

	// Sizes chosen around and above the MaxPageSize clamp: an exact multiple, a
	// partial last page, empty, and a single page.
	for _, total := range []int{0, 1, kernel.MaxPageSize, kernel.MaxPageSize + 1, 250, 3 * kernel.MaxPageSize} {
		var seen []int
		// Deliberately request DOUBLE the cap to exercise the clamp.
		err := eachPage(context.Background(), kernel.MaxPageSize*2, newLister(total),
			func(v int) error { seen = append(seen, v); return nil })
		require.NoError(t, err)
		require.Len(t, seen, total, "every row processed for total=%d despite the page-size clamp", total)
		for i := range total {
			assert.Equal(t, i, seen[i], "rows processed in order with no gaps or dups (total=%d)", total)
		}
	}
}
