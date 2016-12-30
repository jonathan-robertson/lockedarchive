package fcab

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/puddingfactory/filecabinet/clob"
)

// Plugin represents an interface other plugable systems where changes made to File Cabinet are also pushed
type Plugin interface {
	CreateCabinet() error
	DeleteCabinet() error
	List(string, chan clob.Entry) error
	Download(io.WriterAt, clob.Entry) error
	Upload(clob.Entry) error
	Rename(clob.Entry, string) error
	Move(clob.Entry, string) error
	Update(clob.Entry) error
	Delete(clob.Entry) error
	Copy(clob.Entry, string) error
}

// Cabinet represents a collection of entries, symbolizing a cloud container/disk/bucket
type Cabinet struct {
	Name    string                // aws bucket
	entries map[string]clob.Entry // map[key]clob.Entry
	plugin  Plugin
	sync.RWMutex
}

const (
	/* Following Linux standard
	-    Regular file
	b    Block special file
	c    Character special file
	d    Directory
	l    Symbolic link
	n    Network file
	p    FIFO
	s    Socket
	*/
	typeFile = '-'
	typeDir  = 'd'

	sizeOfKey = 16

	rootKey = "00000000000000000000000000000000"
)

var (
	errKeyInUse           = errors.New("key in use")
	errNoKey              = errors.New("no key is assigned to this entry")
	errNotExpectingKey    = errors.New("key detected on entry when not expecting one")
	errEntryDoesNotExist  = errors.New("no entry at provided key")
	errParentDoesNotExist = errors.New("parent key doesn't exist")
	errNoPlugins          = errors.New("at least 1 plugin is required to call Open")
)

// OpenCabinet returns a cabinet, if possible, complete with a loaded entries map; LOCKS
func OpenCabinet(name string, plugin Plugin) (*Cabinet, error) {
	cab := &Cabinet{
		Name:    name,
		entries: make(map[string]clob.Entry),
	}
	entries := make(chan clob.Entry)
	done := make(chan bool)

	go func() {
		cab.Lock()
		defer cab.Unlock()
		defer close(done)
		for entry := range entries {
			cab.entries[entry.Key] = entry
		}
	}()

	// REVIEW: maybe add logic here to choose between multiple plugins based on Listing/Get cost
	err := plugin.List("", entries)

	close(entries)  // indicate no new entries will be added
	<-done          // wait for mapping to complete
	return cab, err // return err if one exists
}

// assignKey generates and assigns a new, unused key to entry; ASSUMES LOCKED
func (cab *Cabinet) assignKey(e clob.Entry) clob.Entry {
	newKey := rootKey
	for cab.keyExists(newKey) {
		newKey = generateKey()
	}

	e.Key = newKey // set new, unused key to entry
	return e
}

// keyExists returns existence of key in entries or if key is the root key; ASSUMES R/LOCKED
func (cab *Cabinet) keyExists(key string) (exists bool) {
	_, ok := cab.entries[key]
	return ok || key == rootKey
}

// upsert updates or inserts entry safely into the map; LOCKS
func (cab *Cabinet) upsert(e clob.Entry) (clob.Entry, error) {
	cab.Lock()
	defer cab.Unlock()

	// Verify parent exists
	if !cab.keyExists(e.ParentKey) {
		return e, errParentDoesNotExist
	}

	// Generate new key if necessary and assign to
	if e.Key == "" {
		e = cab.assignKey(e)
	}

	// Assign entry to entries
	cab.entries[e.Key] = e

	return e, nil
}

// UploadEntry receives an Entry without key, assigns key, and updates map
func (cab *Cabinet) UploadEntry(e clob.Entry) (clob.Entry, error) {

	// TODO: Verify Name
	// TODO: Verify EntryType
	// TODO: Verify Metadata

	// Update local map
	e, err := cab.upsert(e)
	if err != nil {
		return e, err
	}

	return e, cab.plugin.Upload(e) // REVIEW: retry logic to be handled in plugin?
}

// DownloadEntry saves an entry to the local filesystem
func (cab *Cabinet) DownloadEntry(e clob.Entry, filename string) error {
	return cab.DownloadEntry(e, filename) // REVIEW: should have intermediate step for viewing file contents
}

// DeleteEntry removes an existing entry from the cabinet
func (cab *Cabinet) DeleteEntry(e clob.Entry) error {

	// Remove from local map
	cab.Lock()
	delete(cab.entries, e.Key)
	cab.Unlock()

	// Delete from plugin
	return cab.plugin.Delete(e)
}

// LookupEntry retrieves an existing entry from the cabinet
func (cab *Cabinet) LookupEntry(key string) (clob.Entry, error) {
	cab.RLock()
	defer cab.RUnlock()

	e, ok := cab.entries[key]
	if !ok {
		// REVIEW: try fetching this key from plugin?

		return e, errEntryDoesNotExist
	}

	return e, nil
}

func generateKey() (newKey string) {
	b := make([]byte, sizeOfKey)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)
}
