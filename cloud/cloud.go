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

	Upload(string, string, *os.File) error
	Head(Entry) (string, error)
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
func (entry Entry) Meta(pc *secure.PassphraseContainer) (string, error) {
	plaintext, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}

	nonce, err := secure.GenerateNonce()
	if err != nil {
		return "", err
	}

	ciphertext, err := secure.EncryptWithSaltAndWipe(pc, nonce, plaintext)
	return base64.StdEncoding.EncodeToString(ciphertext), err
}

// UpdateMeta reads in encrypted metadata and translates it to Entry's fields
func (entry *Entry) UpdateMeta(pc *secure.PassphraseContainer, encryptedMeta string) error {
	sc, err := secure.DecryptWithSaltFromStringToSecret(pc, encryptedMeta)
	if err != nil {
		return err
	}
	defer sc.Destroy()

	return json.Unmarshal(sc.Buffer(), entry)
}

// TODO
// func (entry Entry) decryptKey() (secure.Key, error) {
// 	if len(entry.Key) == 0 {
// 		return nil, noEncryptionKey
// 	}

// 	// TODO: Get key/password from user to decrypt this key?
// 	// plainKey, err := secure.Decrypt(, entry.Key)

// }
