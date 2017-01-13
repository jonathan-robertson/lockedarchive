package localstorage

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver

	log "github.com/Sirupsen/logrus"

	"github.com/puddingfactory/filecabinet/clob"

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

	stmtUpdateEntryViaKey = `
	UPDATE entries
	SET %s = ?
	WHERE key = ?;`

	cacheFilename  = "cache.db"
	macDefaultPath = "Library/Caches/com.puddingfactory.filecabinet" // TODO: base dir off file system and offer user way to modify
	// winDefaultPath = "\\ProgramData\\puddingfactory\\filecabinet\\"
	// linDefaultPath = ""
)

var (
	cacheRoot             = "Cache/"
	cacheMode os.FileMode = 0700

	errNoRowsChanged = errors.New("no rows changed during operation")
	errNoEntry       = errors.New("no entry found at provided key")
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
	if db, err := c.open(); err == nil {
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
func (c Cache) RememberEntry(e clob.Entry) (err error) {
	if err = c.upsertEntry(e); err == nil && e.Body != nil {
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
	// Setup directory (if doesn't exist) and verify that cache can be opened
	if err = os.MkdirAll(filepath.Join(cacheRoot, c.Cabinet), cacheMode); err == nil {
		if db, err := c.open(); err == nil {
			defer db.Close()
			_, err = db.Exec(stmtCreateTables)
		}
	}
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

func (c Cache) upsertEntry(e clob.Entry) (err error) {
	if err = c.insertEntry(e); err != nil {
		err = c.updateEntry(e)
	}
	return
}

func (c Cache) updateEntry(e clob.Entry) (err error) {
	if db, err := c.open(); err == nil {
		defer db.Close()

		// Try fetching existing entry
		if existingEntry, exists, err := c.selectEntry(e.Key); exists && err == nil {
			cols, vals := compareEntries(existingEntry, e)
			for i, col := range cols {
				if _, err = db.Exec(fmt.Sprintf(stmtUpdateEntryViaKey, col), vals[i], e.Key); err != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		} else if !exists {
			return errNoEntry
		}
	}
	return
}

func (c Cache) insertEntry(e clob.Entry) (err error) {
	if db, err := c.open(); err == nil {
		defer db.Close()
		var result sql.Result
		if e.Body == nil {
			result, err = db.Exec(
				stmtInsertEntryBase,
				e.Key,
				e.ParentKey,
				e.Name,
				e.Type,
			)
		} else {
			result, err = db.Exec(
				stmtInsertEntryComplete,
				e.Key,
				e.ParentKey,
				e.Name,
				e.Type,
				e.Size,
				e.LastModified.Unix(),
			)
		}
		if err != nil {
			return err
		}

		var num int64
		if num, err = result.RowsAffected(); err == nil && num == 0 {
			err = errNoRowsChanged
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

func compareEntries(existing, new clob.Entry) (columns []string, values []interface{}) {
	if existing.ParentKey != new.ParentKey {
		columns = append(columns, "parent_key")
		values = append(values, new.ParentKey)
	}

	if existing.Name != new.Name {
		columns = append(columns, "name")
		values = append(values, new.Name)
	}

	if existing.Type != new.Type {
		columns = append(columns, "type")
		values = append(values, new.Type)
	}

	if existing.Size != new.Size {
		columns = append(columns, "size")
		values = append(values, new.Size)
	}

	if existing.LastModified != new.LastModified {
		columns = append(columns, "last_modified")
		values = append(values, new.LastModified)
	}

	return
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
