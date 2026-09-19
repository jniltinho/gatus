// Package adminbackup backs up and restores what was registered through the administration (fork): the managed
// endpoints, the managed status pages and the push keys created through the administration.
package adminbackup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"time"
)

const (
	// Format identifies a backup file
	Format = "gatus-admin-backup"

	// Version is the version of the backup file written by this version of Gatus, and the highest one it reads
	Version = 1

	// MaximumPlaintextBytes is the maximum size of a backup file without encryption
	MaximumPlaintextBytes = 2 << 20

	// MaximumEndpoints, MaximumStatusPages and MaximumPushKeys are the maximum numbers of items of a backup
	MaximumEndpoints   = 1000
	MaximumStatusPages = 200
	MaximumPushKeys    = 500
)

var (
	// ErrInvalidFile is returned when a backup file cannot be read
	ErrInvalidFile = errors.New("invalid backup file")

	// ErrTooLarge is returned when what was registered through the administration exceeds the limits of a backup
	ErrTooLarge = fmt.Errorf("the backup exceeds %d bytes or the limits of %d endpoints, %d status pages and %d push keys", MaximumPlaintextBytes, MaximumEndpoints, MaximumStatusPages, MaximumPushKeys)
)

// File is a backup file
type File struct {
	Format       string       `json:"format"`
	Version      int          `json:"version"`
	CreatedAt    time.Time    `json:"createdAt"`
	CreatedBy    string       `json:"createdBy"`
	GatusVersion string       `json:"gatusVersion,omitempty"`
	Endpoints    []Endpoint   `json:"endpoints"`
	StatusPages  []StatusPage `json:"statusPages"`
	PushKeys     []PushKey    `json:"pushKeys"`
}

// Endpoint is a managed endpoint of a backup, with its complete stored definition
type Endpoint struct {
	Key        string `json:"key"`
	Definition string `json:"definition"`
}

// StatusPage is a managed status page of a backup, with its complete stored definition
type StatusPage struct {
	Slug       string `json:"slug"`
	Definition string `json:"definition"`
}

// PushKey is a push key created through the administration, without its token. CreatedAt and CreatedBy are only
// informative: a restored push key is created by the author of the restore.
type PushKey struct {
	Name      string    `json:"name"`
	TokenHash string    `json:"tokenHash"`
	Hint      string    `json:"hint"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
}

func invalidFile(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidFile, fmt.Sprintf(format, args...))
}

// decodeStrict decodes a single JSON value, rejecting unknown fields and trailing data
func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("unexpected data after the JSON value")
	}
	return nil
}

// Decode reads a backup file without encryption: unknown fields, missing lists, duplicated items, an unknown format, a
// higher version and a file above the limits are rejected
func Decode(data []byte) (*File, error) {
	if len(data) > MaximumPlaintextBytes {
		return nil, invalidFile("the file exceeds %d bytes", MaximumPlaintextBytes)
	}
	var file File
	if err := decodeStrict(data, &file); err != nil {
		return nil, invalidFile("%s", err.Error())
	}
	if file.Format != Format {
		return nil, invalidFile("unknown format %q", file.Format)
	}
	if file.Version < 1 || file.Version > Version {
		return nil, invalidFile("unsupported version %d", file.Version)
	}
	if file.Endpoints == nil || file.StatusPages == nil || file.PushKeys == nil {
		return nil, invalidFile("the lists endpoints, statusPages and pushKeys are required")
	}
	if len(file.Endpoints) > MaximumEndpoints || len(file.StatusPages) > MaximumStatusPages || len(file.PushKeys) > MaximumPushKeys {
		return nil, invalidFile("the file exceeds the limits of %d endpoints, %d status pages and %d push keys", MaximumEndpoints, MaximumStatusPages, MaximumPushKeys)
	}
	if err := checkDuplicates(&file); err != nil {
		return nil, err
	}
	return &file, nil
}

func checkDuplicates(file *File) error {
	seen := make(map[string]bool)
	check := func(kind, value string) error {
		if len(value) == 0 {
			return invalidFile("a %s without identifier", kind)
		}
		if seen[kind+"\x00"+value] {
			return invalidFile("duplicated %s %q", kind, value)
		}
		seen[kind+"\x00"+value] = true
		return nil
	}
	for _, item := range file.Endpoints {
		if err := check("endpoint", item.Key); err != nil {
			return err
		}
	}
	for _, item := range file.StatusPages {
		if err := check("status page", item.Slug); err != nil {
			return err
		}
	}
	for _, item := range file.PushKeys {
		if err := check("push key", item.Name); err != nil {
			return err
		}
		if err := check("push key hash", item.TokenHash); err != nil {
			return err
		}
	}
	return nil
}

// Encode writes a backup file, indented to be readable
func Encode(file *File) ([]byte, error) {
	return json.MarshalIndent(file, "", "  ")
}

// gatusVersion returns the version of the module of the binary, if known
func gatusVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return ""
}
