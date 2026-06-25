package grpcadapter

import (
	"errors"
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrToStatus(t *testing.T) {
	cases := map[kernel.Kind]codes.Code{
		kernel.KindInvalid:      codes.InvalidArgument,
		kernel.KindNotFound:     codes.NotFound,
		kernel.KindConflict:     codes.AlreadyExists,
		kernel.KindUnauthorized: codes.Unauthenticated,
		kernel.KindForbidden:    codes.PermissionDenied,
		kernel.KindInternal:     codes.Internal,
	}
	for kind, want := range cases {
		if got := status.Code(errToStatus(&kernel.Error{Kind: kind, Msg: "x"})); got != want {
			t.Errorf("kind %v -> %v, want %v", kind, got, want)
		}
	}
	if status.Code(errToStatus(errors.New("plain"))) != codes.Internal {
		t.Error("plain error should map to Internal")
	}
}
