package fcab

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"

	"github.com/puddingfactory/filecabinet/clob"
)

// Plugin represents an interface other plugable systems where changes made to File Cabinet are also pushed
type Plugin interface {
	CreateCabinet() error
	DeleteCabinet() error
	List() error
	Download(clob.Entry) error
	Create(clob.Entry) error
	Rename(clob.Entry, string) error
	Update(clob.Entry) error
	Delete(clob.Entry) error
	Copy(clob.Entry, clob.Entry) error
}

// Cabinet represents a collection of entries, symbolizing a cloud container/disk/bucket
type Cabinet struct {
	Name    string                // aws bucket
	entries map[string]clob.Entry // map[ID]clob.Entry
	plugins []Plugin
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

	sizeOfID = 32

	rootID = "00000000000000000000000000000000"
)

var (
	// _ entry = (*File)(nil)
	// _ entry = (*Folder)(nil)

	errIdentifierInUse    = errors.New("ID in use")
	errEntryNotPresent    = errors.New("No entry at provided ID")
	errNoID               = errors.New("No ID is assigned to this entry")
	errNotExpectingID     = errors.New("ID detected on entry when not expecting one")
	errParentDoesNotExist = errors.New("Parent doesn't exist")
)

// New returns a new cabinet struct
func New(name string) *Cabinet {
	return &Cabinet{
		Name:    name,
		entries: make(map[string]clob.Entry),
	}
}

// CreateEntry receives an Entry without an ID, assigns an ID, and Adds
func (cab *Cabinet) CreateEntry(e clob.Entry) (clob.Entry, error) {

	// Validate entry's fields
	if len(e.ID) != 0 { // Verify id is empty
		return e, errNotExpectingID
	}

	// TODO: Verify Name
	// TODO: Verify Metadata
	// TODO: Verify EntryType

	cab.Lock()
	defer cab.Unlock()

	// TODO: Verify parent exists
	if _, ok := cab.entries[e.ParentID]; !ok {
		return e, errParentDoesNotExist
	}

	var newID string
	for {
		newID = generateNewID()
		if _, ok := cab.entries[newID]; !ok {
			break
		}
	}

	e.ID = newID
	cab.entries[e.ID] = e

	// TODO: Upload object to storage provider?

	return e, nil
}

// AddEntry inserts an entry into the Cabinet
func (cab *Cabinet) AddEntry(e clob.Entry) error {
	if len(e.ID) == 0 {
		return errNoID
	}

	cab.Lock()
	defer cab.Unlock()

	if _, ok := cab.entries[e.ID]; ok { // Expecting entry to not exist yet
		return errIdentifierInUse
	}

	cab.entries[e.ID] = e
	return nil
}

// UpdateEntry updates an existing entry in the Cabinet
func (cab *Cabinet) UpdateEntry(e clob.Entry) error {
	cab.Lock()
	defer cab.Unlock()

	if _, ok := cab.entries[e.ID]; !ok { // Expecting entry to exist already
		return errEntryNotPresent
	}

	cab.entries[e.ID] = e
	return nil
}

// DeleteEntry removes an existing entry from the cabinet
func (cab *Cabinet) DeleteEntry(id string) error {
	cab.Lock()
	defer cab.Unlock()

	delete(cab.entries, id)
	return nil
}

// GetEntry retrieves an existing entry from the cabinet
func (cab *Cabinet) GetEntry(id string) (clob.Entry, error) {
	cab.RLock()
	defer cab.RUnlock()

	e, ok := cab.entries[id]
	if !ok {
		return e, errEntryNotPresent
	}

	return e, nil
}

func generateNewID() (newID string) {
	b := make([]byte, sizeOfID)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)
}
