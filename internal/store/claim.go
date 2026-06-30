package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Claim is the structured record of the currently claimed lap. It is persisted
// to .laps/claim as a single JSON object so that 'laps status' can surface when
// the lap was claimed.
//
// The zero value (Lap == "") is the canonical "no claim" sentinel: a missing,
// empty, or whitespace-only claim file reads back as Claim{}.
type Claim struct {
	Lap       string     `json:"lap"`
	File      string     `json:"file"`
	ClaimedAt *time.Time `json:"claimedAt,omitempty"`
}

// IsZero reports whether c represents the absence of a claim.
func (c Claim) IsZero() bool {
	return c.Lap == ""
}

// ErrMalformedClaim is returned by ReadClaim when the claim file begins as a
// JSON object but cannot be decoded into a Claim (bad syntax or wrong field
// types). It is distinct from a legacy bare-id file and from a missing file.
var ErrMalformedClaim = errors.New("malformed claim")

func ClaimPath(beadsDir string) string {
	return filepath.Join(beadsDir, "claim")
}

// ReadClaim reads the structured claim from beadsDir.
//
// Sentinel: a missing, empty, or whitespace-only claim file returns the zero
// Claim (no claim) and a nil error — never ErrMalformedClaim.
//
// Back-compat: a legacy bare-id file (non-JSON token) reads back as
// {lap: <id>, file: selectedFile, claimedAt: null}. The store layer cannot
// resolve which file the legacy id belonged to, so the caller passes the
// currently selected file.
//
// Forward-compat: unknown JSON fields are ignored.
//
// Any valid JSON value that is not an object claim, or any structured-looking
// invalid JSON, returns ErrMalformedClaim rather than being treated as legacy.
func ReadClaim(beadsDir, selectedFile string) (Claim, error) {
	path := ClaimPath(beadsDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Claim{}, nil
		}
		return Claim{}, err
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return Claim{}, nil
	}

	if json.Valid(trimmed) {
		if trimmed[0] != '{' {
			return Claim{}, fmt.Errorf("%w: expected JSON object", ErrMalformedClaim)
		}
		var c Claim
		if err := json.Unmarshal(trimmed, &c); err != nil {
			return Claim{}, fmt.Errorf("%w: %v", ErrMalformedClaim, err)
		}
		return c, nil
	}

	if trimmed[0] == '{' || trimmed[0] == '[' || trimmed[0] == '"' {
		return Claim{}, fmt.Errorf("%w: invalid JSON", ErrMalformedClaim)
	}

	// Legacy bare-id file: a non-JSON token. The id is the lap; the file is the
	// caller's currently selected file; there is no recorded claim time.
	return Claim{Lap: string(trimmed), File: selectedFile}, nil
}

// WriteClaim persists c to beadsDir as a structured claim.
//
// When the same lap is being re-claimed, an existing claimedAt is preserved so
// the recorded claim time reflects the first claim rather than each re-claim.
// If c.ClaimedAt is nil and no existing time can be preserved, the current UTC
// time is recorded.
func WriteClaim(beadsDir string, c Claim) error {
	path := ClaimPath(beadsDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// Preserve the original claim time on same-claim re-claim. A malformed or
	// legacy existing file simply yields no time to preserve.
	if existing, err := ReadClaim(beadsDir, c.File); err == nil &&
		existing.Lap == c.Lap && existing.File == c.File && existing.ClaimedAt != nil {
		c.ClaimedAt = existing.ClaimedAt
	} else if c.ClaimedAt == nil {
		now := time.Now().UTC()
		c.ClaimedAt = &now
	}

	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return SafeWriteFile(path, append(b, '\n'), 0o644)
}

func RemoveClaim(beadsDir string) error {
	path := ClaimPath(beadsDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
