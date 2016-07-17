package storage

import (
	"errors"
	"sync"

	"github.com/puddingfactory/filecabinet/crypt"
)

type entry interface {
	DecryptName() error
	EncryptName() error
	id() string
}

type safeMap struct {
	sync.RWMutex
	m map[string]entry
}

type Cabinet struct {
	Name      string // aws bucket
	FileMap   *safeMap
	FolderMap *safeMap
}

type File struct {

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

	// TODO: Store this value in metadata
	// TODO: Verify how many characters can be in a metadata value
	// NOTE: can be "nested" - /Vital/Jonathan.
	Folder string

	// TODO: Build fetch func for downloading object's bytes and storing to Data if requested (don't 'load ahead')
	// NOTE: data that can be streamed to a file on system if download is opted for
	Data []byte

	// REF: http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingMetadata.html
	// NOTE: The PUT request header is limited to 8 KB in size. Within the PUT request header, the user-defined metadata is limited to 2 KB in size. The size of user-defined metadata is measured by taking the sum of the number of bytes in the UTF-8 encoding of each key and value
	Text string
}

var (
	_ entry = (*File)(nil)
	// _ entry = (*Folder)(nil)

	errIdentifierInUse = errors.New("ID in use")
	errEntryNotPresent = errors.New("No entry at provided ID")
)

func newSafeMap() *safeMap {
	return &safeMap{
		m: make(map[string]entry),
	}
}

func (s *safeMap) insert(e entry) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.m[e.id()]; ok { // Expecting entry to not exist yet
		return errIdentifierInUse
	}
	s.m[e.id()] = e

	return nil
}

func (s *safeMap) update(e entry) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.m[e.id()]; !ok { // Expecting entry to exist already
		return errEntryNotPresent
	}
	s.m[e.id()] = e

	return nil
}

func (s *safeMap) delete(id string) error {
	s.Lock()
	defer s.Unlock()

	delete(s.m, id)
	return nil
}

func (s *safeMap) get(id string) (entry, error) {
	s.RLock()
	defer s.RUnlock()

	e, ok := s.m[id]
	if !ok {
		return nil, errEntryNotPresent
	}

	return e, nil
}

func (f *File) id() string {
	return f.ID
}

func (f *File) EncryptData() (err error) {
	f.Data, err = crypt.Encrypt(f.Data)
	return
}

func (f *File) EncryptName() (err error) {
	f.Name, err = crypt.EncryptStringToHexString(f.Name)
	return
}

func (f *File) DecryptData() (err error) {
	f.Data, err = crypt.Decrypt(f.Data)
	return
}

func (f *File) DecryptName() (err error) {
	f.Name, err = crypt.DecryptHexStringToString(f.Name)
	return
}

// type Folder struct {
//     Parent string
//     Name string
// }

func NewCabinet(name string) *Cabinet {
	return &Cabinet{
		Name:      name,
		FileMap:   newSafeMap(),
		FolderMap: newSafeMap(),
	}
}
