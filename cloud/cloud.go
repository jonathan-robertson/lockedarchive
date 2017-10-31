// Package cloud provides support for interacting with online storage
package cloud

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/jonathan-robertson/lockedarchive/stream"
)

// Client represents an object storage provider's service
// NOTE: Uploading and downloading bytes expected to interact with
// local file system (cache folder)
type Client interface {
	CreateArchive() error
	RemoveArchive() error

	List(chan Entry) error

	Upload(Entry) error
	Head(Entry) error
	Download(Entry) error
	Update(Entry) error
	Delete(Entry) error

	// TODO: add these in later
	// Restore(Entry, int) error
}

// Entry represents a standard object compatible with cloud operations
type Entry struct {
	Key string `json:"-"` // Key representing this Entry

	ParentKey    string    `json:"p"` // Key representing Entry containing this one
	Name         string    `json:"n"` // Name of this Entry
	IsDir        bool      `json:"d"` // Whether or not this Entry contains others
	Size         int64     `json:"s"` // Size of Entry's data
	LastModified time.Time `json:"m"` // Last time Entry was updated

	// TODO: add these in later
	// Tags []string
}

// Meta returns Entry's encrypted metadata
func (entry Entry) Meta(key *[stream.KeySize]byte) (encryptedMeta string, err error) {

	plaintext, err := json.Marshal(entry)
	if err != nil {
		return
	}

	ciphertext, err := stream.EncryptBytes(key, stream.GenerateNonce(), plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), err
}

// UpdateMeta reads in encrypted metadata and translates it to Entry's fields
// TODO: Update to no longer receive key - pull it from config
func (entry *Entry) UpdateMeta(encryptedMeta string, key *[stream.KeySize]byte) error {
	decoded, err := base64.StdEncoding.DecodeString(encryptedMeta)
	if err != nil {
		return err
	}

	plaintext, err := stream.DecryptBytes(key, decoded)
	if err != nil {
		return err
	}

	return json.Unmarshal(plaintext, entry)
}
