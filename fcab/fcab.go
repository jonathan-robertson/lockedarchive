package fcab

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/puddingfactory/filecabinet/clob"
	"github.com/puddingfactory/filecabinet/crypt"
	"github.com/puddingfactory/filecabinet/localstorage"
)

// Client represents an interface other plugable systems where changes made to File Cabinet are also pushed
type Client interface {
	CreateCabinet() error
	DeleteCabinet() error
	List(string, chan clob.Entry) error
	Download(io.WriterAt, clob.Entry) error
	OpenDownstream(*clob.Entry) error
	Upload(clob.Entry) error
	Rename(clob.Entry, string) error
	Move(clob.Entry, string) error
	Update(clob.Entry) error
	Delete(clob.Entry) error
	Copy(clob.Entry, string) error
}

// Cabinet represents a collection of entries, symbolizing a cloud container/disk/bucket
type Cabinet struct {
	Name     string // aws bucket
	password string // encryption key used to safeguard online storage
	cache    localstorage.Cache
	client   Client
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
	errNoPlugins          = errors.New("at least 1 client is required to call Open")
)

// OpenCabinet returns a cabinet, if possible, complete with a loaded entries map; LOCKS
func OpenCabinet(name, pass string, client Client) (cab *Cabinet, err error) {
	if cache, err := localstorage.New(name); err == nil {
		cab = &Cabinet{
			Name:     name,
			cache:    cache,
			password: pass,
			client:   client,
		}

	}

	entries := make(chan clob.Entry)
	done := make(chan bool)
	go func() {
		defer close(done)
		for entry := range entries {
			if cacheErr := cab.cache.RememberEntry(entry); cacheErr != nil {
				log.Println(cacheErr) // TODO: use more permanent logging solution
			}
		}
	}()

	// REVIEW: maybe add logic here to choose between multiple plugins based on Listing/Get cost
	err = client.List("", entries)
	close(entries)  // indicate no new entries will be added
	<-done          // wait for mapping to complete
	return cab, err // return err if one exists
}

func (c Cabinet) monitorQueue(workerCount int) {
	jobs := make(chan localstorage.Job)
	for i := 0; i < workerCount; i++ {
		func() {
			for job := range jobs {
				switch job.Action {
				case localstorage.ActionDelete:
				case localstorage.ActionDownload:
				case localstorage.ActionList:
				case localstorage.ActionUpdate:
				case localstorage.ActionUpload:
				}
			}
		}()
	}

	for {
		if job, err := c.cache.DequeueJob(); err == nil {
			jobs <- job
		} else {
			time.Sleep(1 * time.Second) // sleep if no more jobs are queued
		}
	}

	// TODO: add mechanism to handle closure of cabinet
}

// assignKey generates and assigns a new, unused key to entry; ASSUMES LOCKED
func (c *Cabinet) assignKey(e clob.Entry) clob.Entry {
	newKey := rootKey
	for c.keyExists(newKey) {
		newKey = generateKey()
	}

	e.Key = newKey // set new, unused key to entry
	return e
}

// keyExists returns existence of key in entries or if key is the root key; ASSUMES R/LOCKED
func (c *Cabinet) keyExists(key string) (exists bool) {
	return key == rootKey || c.cache.ContainsEntry(key)
}

// upsert updates or inserts entry safely into cache
func (c *Cabinet) upsert(e clob.Entry) (clob.Entry, error) {

	// Verify parent exists
	if !c.keyExists(e.ParentKey) {
		return e, errParentDoesNotExist
	}

	// Generate new key if necessary and assign to
	if e.Key == "" {
		e = c.assignKey(e)
	}

	return e, c.cache.RememberEntry(e) // remember entry in cache
}

// QueueForUpload prepares the file/dir for upload
func (c *Cabinet) QueueForUpload(parentKey string, dirent *os.File) (e clob.Entry, err error) {
	defer dirent.Close()

	// Extract metadata
	if stats, err := dirent.Stat(); err == nil {
		var entryType rune
		if stats.IsDir() {
			entryType = typeDir
		} else {
			entryType = typeFile
		}

		e = clob.Entry{
			ParentKey:    parentKey,
			Name:         stats.Name(),
			Size:         stats.Size(),
			LastModified: stats.ModTime(),
			Type:         entryType,
		}
	} else {
		return e, err
	}

	// Encrypt and Cache body to prepare for upload
	if unsafeBytes, err := ioutil.ReadAll(dirent); err == nil {
		if encryptedBytes, err := crypt.Encrypt(unsafeBytes); err == nil {

			// TODO: update crypt to support streaming also

			pr, pw := io.Pipe()
			defer pw.Close()
			e.Body = pr

			gw := gzip.NewWriter(pw)
			defer gw.Close()

			go func() {
				if _, err := io.Copy(gw, bytes.NewReader(encryptedBytes)); err != nil {
					log.Println(err) // TODO: use something more permanent here
				}
			}()
		}
	}

	// Cache entry and data
	if e, err = c.upsert(e); err != nil {
		c.cache.EnqueueJob(e.Key, localstorage.ActionUpload) // queue upload job with cache
	}

	return
}

// func (cab Cabinet) pipeForUpload(readCloser io.ReadCloser, e clob.Entry) clob.Entry {
// 	crypt.
// }

// UploadEntry receives an Entry without key, assigns key, and updates map
func (c *Cabinet) UploadEntry(e clob.Entry) (clob.Entry, error) {

	// TODO: Verify Name
	// TODO: Verify EntryType
	// TODO: Verify Metadata

	// Update local map
	e, err := c.upsert(e)
	if err != nil {
		return e, err
	}

	return e, c.client.Upload(e) // REVIEW: retry logic to be handled in client?
}

// DeleteEntry removes an existing entry from the cabinet
func (c *Cabinet) DeleteEntry(e clob.Entry) error {

	// Remove from cache
	if err := c.cache.ForgetEntry(e); err != nil {
		log.Println(err) // TODO: use more permanent logging solution
	}

	// Delete from client
	return c.client.Delete(e)
}

// LookupEntry retrieves an existing entry from the cabinet
func (c *Cabinet) LookupEntry(key string) (e clob.Entry, err error) {
	if e, err = c.cache.RecallEntry(key); err == sql.ErrNoRows {
		// REVIEW: try fetching this key from client, then Remember it in cache and return?
	}
	return
}

func generateKey() (newKey string) {
	b := make([]byte, sizeOfKey)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)
}
