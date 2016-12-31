package localstorage

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"

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
        key STRING NOT NULL PRIMARY KEY,
        parent_key STRING NOT NULL,
        name STRING NOT NULL,
        type CHARACTER(1) NOT NULL,
        size UNSIGNED BIG INT,
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

	cacheFilename  = "cache.db"
	macDefaultPath = "Library/Caches/com.puddingfactory.filecabinet" // TODO: base dir off file system and offer user way to modify
	// winDefaultPath = "\\ProgramData\\puddingfactory\\filecabinet\\"
	// linDefaultPath = ""
)

var (
	cacheRoot = "Cache/"
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

// ForgetEntry purges the entry's cache data and info from db
func (c Cache) ForgetEntry(e clob.Entry) (err error) {
	if err = deleteFileIfExists(filepath.Join(cacheRoot, c.Cabinet, e.Key)); err == nil {
		err = c.deleteEntry(e) // remove entry from db
	}
	return
}

func (c Cache) init() (err error) {
	// Verify that cache can be opened

	if err = os.MkdirAll(filepath.Join(cacheRoot, c.Cabinet), 0700); err != nil {
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
