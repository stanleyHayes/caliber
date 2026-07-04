// Command reencrypt rewrites every PII-bearing row through the configured field
// cipher, re-sealing it with the current primary key (CAL-117). It is the
// operational tool for a field-encryption key rotation or first-time enablement:
// set CALIBER_FIELD_ENCRYPTION_KEY to the new key and, during a rotation,
// CALIBER_FIELD_ENCRYPTION_KEY_PREVIOUS to the retiring key(s), then run this to
// migrate stored rows before dropping the old key. It is idempotent and safe to
// re-run. See docs/runbooks/key-rotation.md.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"
	"github.com/xcreativs/caliber/internal/platform/wiring"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logging.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("reencrypt starting",
		"previous_keys", len(cfg.FieldEncryptionKeyPrevious))
	res, err := wiring.Reencrypt(ctx, cfg, log)
	if err != nil {
		return err
	}
	log.Info("reencrypt done",
		"candidates", res.Candidates, "profiles", res.Profiles,
		"interviews", res.Interviews, "matches", res.Matches)
	return nil
}
