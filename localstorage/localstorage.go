package localstorage

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver

	log "github.com/Sirupsen/logrus"

	"github.com/puddingfactory/filecabinet/clob"

	"io"

	"time"

	"github.com/mitchellh/go-homedir"
)

// Cache represents the data saved locally to the comptuer
type Cache struct {
	Cabinet string
}

const (
	stmtCreateTables = `
	CREATE TABLE entries(
        key TEXT NOT NULL PRIMARY KEY,
        parent_key TEXT NOT NULL,
        name TEXT NOT NULL,
        type CHAR(1) NOT NULL,
        size INTEGER,
        last_modified INTEGER
    );`

	stmtDeleteEntry = `
	DELETE FROM entries
	WHERE key = ?;`

	stmtInsertEntryBase = `
	INSERT INTO entries(key, parent_key, name, type)
	VALUES(?, ?, ?, ?);`

	stmtInsertEntryComplete = `
	INSERT INTO entries
	VALUES(?, ?, ?, ?, ?, ?);`

	stmtSelectEntryViaKey = `
	SELECT * FROM entries
	WHERE key = ?;`

	cacheFilename  = "cache.db"
	macDefaultPath = "Library/Caches/com.puddingfactory.filecabinet" // TODO: base dir off file system and offer user way to modify
	// winDefaultPath = "\\ProgramData\\puddingfactory\\filecabinet\\"
	// linDefaultPath = ""
)

var (
	cacheRoot             = "Cache/"
	cacheMode os.FileMode = 0700
)

// New opens or creates, opens, and returns the cache database
func New(cabinet string) (c Cache, err error) {
	c = Cache{Cabinet: cabinet}

	// Verify if cache.db already exists
	if fileDoesNotExist(c.filename()) {
		err = c.init()
		return
	}

	// Verify db can still be opened, then close it
	db, err := c.open()
	if err == nil {
		db.Close()
	}
	return
}

// RecallEntry returns the entry (including its data file if cached)
func (c Cache) RecallEntry(key string) (e clob.Entry, success bool, err error) {
	if e, success, err = c.selectEntry(key); err == nil && success {
		if file, err := os.Open(filepath.Join(cacheRoot, c.Cabinet, key)); err == nil {
			e.Body = file // link file, ready for reading
		} else if os.IsNotExist(err) {
			err = nil // ignore error if the file just doesn't happen to exist
		}
	}
	return
}

// RememberEntry records the entry's file and metadata to cache
// REVIEW: DOES NOT WORK IF ENTRY ALREADY EXISTS! (should approach as upsert instead)
func (c Cache) RememberEntry(e clob.Entry) (err error) {
	if err = c.insertEntry(e); err == nil && e.Body != nil {
		if cacheFile, err := os.Create(filepath.Join(cacheRoot, c.Cabinet, e.Key)); err == nil {
			defer e.Body.Close()
			defer cacheFile.Close()
			_, err = io.Copy(cacheFile, e.Body)
		}
	}
	return
}

// ForgetEntry purges the entry's cache data and info from db
func (c Cache) ForgetEntry(e clob.Entry) (err error) {
	if err = deleteFileIfExists(filepath.Join(cacheRoot, c.Cabinet, e.Key)); err == nil {
		err = c.deleteEntry(e) // remove entry from db
	}
	return
}

func (c Cache) init() (err error) {
	// Verify that cache can be opened

	if err = os.MkdirAll(filepath.Join(cacheRoot, c.Cabinet), cacheMode); err != nil {
		return
	}

	db, err := c.open()
	if err != nil {
		return
	}
	defer db.Close()

	_, err = db.Exec(stmtCreateTables)
	return
}

func (c Cache) filename() string {
	return filepath.Join(cacheRoot, c.Cabinet, cacheFilename)
}

func (c Cache) open() (*sql.DB, error) {
	return sql.Open("sqlite3", c.filename())
}

func (c Cache) selectEntry(key string) (e clob.Entry, success bool, err error) {
	if db, err := c.open(); err == nil {
		defer db.Close()
		row := db.QueryRow(stmtSelectEntryViaKey, key)

		var size, unixTimestamp int64
		err = row.Scan(
			&e.Key,
			&e.ParentKey,
			&e.Name,
			&e.Type,
			&size,
			&unixTimestamp,
		)
		if size != 0 {
			e.Size = size
		}
		if unixTimestamp != 0 {
			e.LastModified = time.Unix(unixTimestamp, 0)
		}
	}
	return e, e.Key == key, err
}

func (c Cache) insertEntry(e clob.Entry) (err error) {
	if db, err := c.open(); err == nil {
		defer db.Close()
		if e.Body == nil {
			_, err = db.Exec(
				stmtInsertEntryBase,
				e.Key,
				e.ParentKey,
				e.Name,
				e.Type,
			)
		} else {
			_, err = db.Exec(
				stmtInsertEntryComplete,
				e.Key,
				e.ParentKey,
				e.Name,
				e.Type,
				e.Size,
				e.LastModified.Unix(),
			)
		}
	}
	return err
}

func (c Cache) deleteEntry(e clob.Entry) (err error) {
	if db, err := c.open(); err == nil {
		defer db.Close()
		_, err = db.Exec(stmtDeleteEntry, e.Key)
	}
	return err
}

func init() {
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}

	switch runtime.GOOS {
	case "darwin":
		cacheRoot = filepath.Join(home, macDefaultPath)
	case "windows", "linux":
		log.Fatal(runtime.GOOS, "is not yet supported")
	default:
		log.Fatal(runtime.GOOS, "is not supported")
	}
}

func deleteFileIfExists(filename string) error {
	if fileDoesNotExist(filename) {
		return nil
	}
	return os.Remove(filename)
}

func fileDoesNotExist(filename string) bool {
	_, err := os.Stat(filename)
	return os.IsNotExist(err)
}
