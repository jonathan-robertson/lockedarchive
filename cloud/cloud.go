// Package cloud provides support for interacting with online storage
package cloud

import (
	"time"
)

// Client represents an object storage provider's service
// NOTE: Uploading and downloading bytes expected to interact with
// local file system (cache folder)
type Client interface {
	CreateArchive() error
	RemoveArchive() error

	List(chan Entry) error

	Upload(Entry) error
	Download(Entry) error
	Update(Entry) error
	Delete(Entry) error

	// TODO: add these in later
	// Restore(Entry, int) error
}

// Entry represents a standard object compatible with cloud operations
type Entry struct {
	Key string // Key representing this Entry

	ParentKey    string    // Key representing Entry containing this one
	Name         string    // Name of this Entry
	IsDir        bool      // Whether or not this Entry contains others
	Size         int64     // Size of Entry's data
	LastModified time.Time // Last time Entry was updated

	// TODO: add these in later
	// Tags []string
}
