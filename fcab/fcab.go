package fcab

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"

	log "github.com/Sirupsen/logrus"

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

// Open returns a cabinet, if possible, complete with a loaded entries map
func Open(name string, plugins []Plugin) (*Cabinet, error) {
	if len(plugins) == 0 {
		return nil, errNoPlugins // don't proceed if we haven't provided at least 1 plugin
	}

	cab := &Cabinet{
		Name:    name,
		entries: make(map[string]clob.Entry),
	}
	entries := make(chan clob.Entry)
	done := make(chan bool)

	go func() {
		defer close(done)
		for entry := range entries {
			if err := cab.MapEntry(entry); err != nil {
				log.WithFields(log.Fields{"err": err}).Error("trouble adding entry to cabinet during list")
			}
		}
	}()

	err := plugins[0].List("", entries) // REVIEW: maybe add logic here to choose which plugin to run based on Listing/Get cost
	close(entries)                      // indicate no new entries will be added
	<-done                              // wait for mapping to complete
	return cab, err                     // return err if one exists
}

// MapEntry safely inserts an entry into the Cabinet's map
func (cab *Cabinet) MapEntry(e clob.Entry) error {
	if len(e.Key) == 0 {
		return errNoKey
	}

	cab.Lock()
	defer cab.Unlock()
	cab.entries[e.Key] = e
	return nil
}

// CreateEntry receives an Entry without key, assigns an key, and Adds
func (cab *Cabinet) CreateEntry(e clob.Entry) (clob.Entry, error) {

	// Validate entry's fields
	if len(e.Key) != 0 { // Verify key is empty
		return e, errNotExpectingKey
	}

	// TODO: Verify Name
	// TODO: Verify Metadata
	// TODO: Verify EntryType

	cab.Lock()
	defer cab.Unlock()

	// TODO: Verify parent exists
	if _, ok := cab.entries[e.ParentKey]; !ok {
		return e, errParentDoesNotExist
	}

	var newKey string
	for {
		newKey = generateNewID()
		if _, ok := cab.entries[newKey]; !ok {
			break
		}
	}

	e.Key = newKey
	cab.entries[e.Key] = e

	// Upload entry to each plugin
	for _, plugin := range cab.plugins {
		if err := plugin.Upload(e); err != nil {
			return e, err
		}
	}

	return e, nil
}

// UpdateEntry updates an existing entry in the Cabinet
func (cab *Cabinet) UpdateEntry(e clob.Entry) error {
	cab.Lock()
	defer cab.Unlock()

	if _, ok := cab.entries[e.Key]; !ok { // Expecting entry to exist already
		return errEntryDoesNotExist
	}

	// REVIEW: determine what changed and push that kind of change to plugins

	cab.entries[e.Key] = e
	return nil
}

// DeleteEntry removes an existing entry from the cabinet
func (cab *Cabinet) DeleteEntry(key string) error {
	e, err := cab.GetEntry(key)
	if err != nil {
		return err
	}

	cab.Lock()
	delete(cab.entries, key)
	cab.Unlock()

	for _, plugin := range cab.plugins {
		if err := plugin.Delete(e); err != nil {
			return err
		}
	}
	return nil
}

// GetEntry retrieves an existing entry from the cabinet
func (cab *Cabinet) GetEntry(key string) (clob.Entry, error) {
	cab.RLock()
	defer cab.RUnlock()

	e, ok := cab.entries[key]
	if !ok {
		return e, errEntryDoesNotExist
	}

	return e, nil
}

func generateNewID() (newID string) {
	b := make([]byte, sizeOfKey)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)
}
