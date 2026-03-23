package service

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/theencryptedafro/appwrap/internal/security"
)

// KeygenOpts configures key generation.
type KeygenOpts struct {
	OutputDir string
}

// KeygenResult holds generated key information.
type KeygenResult struct {
	Recipient string
	KeyFile   string
}

// GenerateKeys creates an Age keypair for encryption.
func (s *AppService) GenerateKeys(ctx context.Context, opts KeygenOpts) (*KeygenResult, error) {
	absDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("resolving output directory: %w", err)
	}

	recipient, err := security.GenerateKeyPair(absDir)
	if err != nil {
		return nil, fmt.Errorf("keygen failed: %w", err)
	}

	return &KeygenResult{
		Recipient: recipient,
		KeyFile:   filepath.Join(absDir, "appwrap-age-key.txt"),
	}, nil
}
