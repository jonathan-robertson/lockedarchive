package storage

import (
	"errors"
	"sync"

	"github.com/puddingfactory/filecabinet/crypt"
)

type Entry struct {
	// Note from Amazon on naming:
	// Alphanumeric characters [0-9a-zA-Z]
	// Special characters !, -, _, ., *, ', (, and )
	// TODO: more notes are on that page... read them more and consider them

	// REVIEW: research and consider search values...
	// Tags could be recorded as metadata, comma-delimited... on Unmarshal, could have tagMap (map[string][]string == map[tag][]GUIDs)

	// TODO: 32 (?) chars of hex (?), incremented and inverted so that 00...01 becomes 10...00
	// NOTE: This is all you'll see in S3
	ID string

	// TODO: Adhere Windows' to standard... or S3's?
	// NOTE: Also in metadata..?
	Name string

	// TODO: Store this value in metadata? Or would it make more sense to store it as a prefix so we can do a lookup to get what's immediately inside a dir.
	// NOTE: Is not nested
	ParentID string

	// REVIEW: maybe should offload this to local FS instead (cache).
	Data []byte

	// REVIEW: Does a rune actually work here? Would take less steps to use string instead.
	EntryType rune

	// NOTE: The PUT request header is limited to 8 KB in size. Within the PUT request header, the user-defined metadata is limited to 2 KB in size. The size of user-defined metadata is measured by taking the sum of the number of bytes in the UTF-8 encoding of each key and value
	Metadata map[string]string
}

type entrymap struct {
	sync.RWMutex
	m map[string]Entry
}

type Cabinet struct {
	Name    string // aws bucket
	entries *entrymap
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

func MakeCabinet(name string) *Cabinet {
	return &Cabinet{
		Name: name,
		entries: &entrymap{
			m: make(map[string]Entry),
		},
	}
}

func generateNewID() string {
	return "" // TODO
}

// CreateEntry receives an Entry without an ID, assigns an ID, and Adds
func (cab *Cabinet) CreateEntry(e Entry) (Entry, error) {

	// Validate entry's fields
	if len(e.ID) != 0 { // Verify id is empty
		return e, errNotExpectingID
	}

	// TODO: Verify Name
	// TODO: Verify Metadata
	// TODO: Verify EntryType

	cab.entries.Lock()
	defer cab.entries.Unlock()

	// TODO: Verify parent exists
	if _, ok := cab.entries.m[e.ParentID]; !ok {
		return e, errParentDoesNotExist
	}

	var newID string
	for {
		newID = generateNewID()
		if _, ok := cab.entries.m[newID]; !ok {
			break
		}
	}

	e.ID = newID
	cab.entries.m[e.ID] = e

	// TODO: Upload object to storage provider?

	return e, nil
}

func (cab *Cabinet) AddEntry(e Entry) error {
	if len(e.ID) == 0 {
		return errNoID
	}

	cab.entries.Lock()
	defer cab.entries.Unlock()

	if _, ok := cab.entries.m[e.ID]; ok { // Expecting entry to not exist yet
		return errIdentifierInUse
	}

	cab.entries.m[e.ID] = e
	return nil
}

func (cab *Cabinet) UpdateEntry(e Entry) error {
	cab.entries.Lock()
	defer cab.entries.Unlock()

	if _, ok := cab.entries.m[e.ID]; !ok { // Expecting entry to exist already
		return errEntryNotPresent
	}

	cab.entries.m[e.ID] = e
	return nil
}

func (cab *Cabinet) DeleteEntry(id string) error {
	cab.entries.Lock()
	defer cab.entries.Unlock()

	delete(cab.entries.m, id)
	return nil
}

func (cab *Cabinet) GetEntry(id string) (Entry, error) {
	cab.entries.RLock()
	defer cab.entries.RUnlock()

	e, ok := cab.entries.m[id]
	if !ok {
		return e, errEntryNotPresent
	}

	return e, nil
}

func (e *Entry) EncryptData() (err error) {
	e.Data, err = crypt.Encrypt(e.Data)
	return
}

func (e *Entry) EncryptName() (err error) {
	e.Name, err = crypt.EncryptStringToHexString(e.Name)
	return
}

func (e *Entry) DecryptData() (err error) {
	e.Data, err = crypt.Decrypt(e.Data)
	return
}

func (e *Entry) DecryptName() (err error) {
	e.Name, err = crypt.DecryptHexStringToString(e.Name)
	return
}
