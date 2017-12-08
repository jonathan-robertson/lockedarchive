// Package cloud provides support for interacting with online storage
package cloud

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/jonathan-robertson/lockedarchive/secure"
)

var (
	errNoEncryptionKey = errors.New("no encryption key to decrypt for entry")
)

// Client represents an object storage provider's service
// NOTE: Uploading and downloading bytes expected to interact with
// local file system (cache folder)
type Client interface {
	CreateArchive() error
	RemoveArchive() error

	List(chan Entry) error

	Upload(Entry, *os.File) error
	Head(Entry) error
	Download(Entry) (io.ReadCloser, error)
	Update(Entry) error
	Delete(Entry) error

	// TODO: add these in later
	// Restore(Entry, int) error
}

// Entry represents a standard object compatible with cloud operations
type Entry struct {
	ID string `json:"-"` // ID representing this Entry

	Key          string      `json:"k"` // Encrypted encryption key used to encrypt/decrypt this data
	ParentID     string      `json:"p"` // ID representing Entry containing this one
	Name         string      `json:"n"` // Name of this Entry
	IsDir        bool        `json:"d"` // Whether or not this Entry contains others
	Size         int64       `json:"s"` // Size of Entry's data
	LastModified time.Time   `json:"m"` // Last time Entry was updated
	Mode         os.FileMode `json:"f"` // File Mode

	// TODO: add these in later
	// Tags []string
}

// Meta returns Entry's encrypted metadata
func (entry Entry) Meta(kc *secure.KeyContainer) (encryptedMeta string, err error) {

	// TODO: update to decrypt entry.Key with incoming key

	plaintext, err := json.Marshal(entry)
	if err != nil {
		return
	}

	nonce, err := secure.GenerateNonce()
	if err != nil {
		return
	}

	ciphertext := secure.EncryptAndWipe(kc, nonce, plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), err
}

// UpdateMeta reads in encrypted metadata and translates it to Entry's fields
// TODO: Update to no longer receive key - pull it from config
func (entry *Entry) UpdateMeta(encryptedMeta string, kc *secure.KeyContainer) error {

	// TODO: update to decrypt entry.Key with incoming key

	decoded, err := base64.StdEncoding.DecodeString(encryptedMeta)
	if err != nil {
		return err
	}

	plaintext, err := secure.Decrypt(kc, decoded)
	if err != nil {
		return err
	}

	return json.Unmarshal(plaintext, entry)
}

// TODO
// func (entry Entry) decryptKey() (secure.Key, error) {
// 	if len(entry.Key) == 0 {
// 		return nil, noEncryptionKey
// 	}

// 	// TODO: Get key/password from user to decrypt this key?
// 	// plainKey, err := secure.Decrypt(, entry.Key)

// }
