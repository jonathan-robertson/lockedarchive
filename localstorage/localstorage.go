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
	db      *sql.DB // The returned DB is safe for concurrent use by multiple goroutines and maintains its own pool of idle connections. Thus, the Open function should be called just once. It is rarely necessary to close a DB.
}

// Job represents a queued action to take
type Job struct {
	ID     int
	Key    string
	Action int
}

const (
	// ActionList is a job action for List
	ActionList = iota

	// ActionUpload is a job action for Upload
	ActionUpload = iota

	// ActionDownload is a job action for Download
	ActionDownload = iota

	// ActionUpdate is a job action for Update
	ActionUpdate = iota

	// ActionDelete is a job action for Delete
	ActionDelete = iota

	sqlCreateTables = `
	CREATE TABLE entries(
		key TEXT NOT NULL PRIMARY KEY,
		parent_key TEXT NOT NULL,
		name TEXT NOT NULL,
		type CHAR(1) NOT NULL,
		size INTEGER,
		last_modified INTEGER
	);

	CREATE TABLE actions(
		id INTEGER PRIMARY KEY,
		name TEXT
	);
	INSERT INTO actions(name) VALUES('list');
	INSERT INTO actions(name) VALUES('upload');
	INSERT INTO actions(name) VALUES('download');
	INSERT INTO actions(name) VALUES('update');
	INSERT INTO actions(name) VALUES('delete');

	CREATE TABLE jobs(
		id INTEGER PRIMARY KEY,
		key TEXT NOT NULL,
		action INTEGER NOT NULL,
		FOREIGN KEY(key) REFERENCES entries(key),
		FOREIGN KEY(action) REFERENCES actions(id)
	);`

	sqlDeleteEntry = `
	DELETE FROM entries
	WHERE key = ?;`

	sqlInsertEntryBase = `
	INSERT INTO entries(key, parent_key, name, type)
	VALUES(?, ?, ?, ?);`

	sqlInsertEntryComplete = `
	INSERT INTO entries
	VALUES(?, ?, ?, ?, ?, ?);`

	sqlSelectEntryViaKey = `
	SELECT * FROM entries
	WHERE key = ?;`

	sqlUpdateEntryViaKey = `
	UPDATE entries
	SET %s = ?
	WHERE key = ?;`

	sqlSelectEntryExists = `
	SELECT EXISTS(1)
	FROM entries
	WHERE key = ?;`

	sqlInsertJob = `
	INSERT INTO jobs(key, action)
	VALUES(?, ?);`

	sqlDeleteJob = `
	DELETE FROM jobs
	WHERE id = ?;`

	sqlGetNextJob = `
	SELECT * FROM jobs
	LIMIT 1;`

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
	errInvalidAction = errors.New("invalid action")
)

// Open opens or creates, opens, and returns the cache database
func Open(cabinet string) (c Cache, err error) {
	c = Cache{Cabinet: cabinet}
	if c.db, err = sql.Open("sqlite3", c.filename()); err != nil {
		if fileDoesNotExist(c.filename()) { // handle if file simply doesn't exist
			if err = os.MkdirAll(filepath.Join(cacheRoot, c.Cabinet), cacheMode); err == nil {
				if c.db, err = sql.Open("sqlite3", c.filename()); err != nil {
					_, err = c.db.Exec(sqlCreateTables)
				}
			}
		}
	}
	return
}

// Close shuts down the connection to the database
func (c Cache) Close() error {
	return c.db.Close()
}

// RecallEntry returns the entry (including its data file if cached)
func (c Cache) RecallEntry(key string) (e clob.Entry, err error) {
	if e, err = c.selectEntry(key); err == nil {
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
	if e.Body != nil {
		defer e.Body.Close() // be sure to close this even if err on upsertEntry
	}

	if err = c.upsertEntry(e); err == nil {
		if e.Body == nil {
			err = deleteFileIfExists(filepath.Join(cacheRoot, c.Cabinet, e.Key))
		} else if cacheFile, err := os.Create(filepath.Join(cacheRoot, c.Cabinet, e.Key)); err == nil {
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

// ContainsEntry returns if an entry exists at provided id
func (c Cache) ContainsEntry(key string) (exists bool) {
	if err := c.db.QueryRow(sqlSelectEntryExists, key).Scan(&exists); err != nil {
		log.Println(err)
	}
	return
}

// EnqueueJob queues a new job
func (c Cache) EnqueueJob(key string, action int) (err error) {
	if !isValidAction(action) {
		return errInvalidAction
	}
	return c.insertJob(key, action)
}

// DequeueJob is for fetching the contents of the next job in the queued
func (c Cache) DequeueJob() (j Job, err error) {
	if j, err = c.selectNextJob(); err == nil {
		err = c.deleteJob(j)
	}
	return
}

func (c Cache) filename() string {
	return filepath.Join(cacheRoot, c.Cabinet, cacheFilename)
}

func (c Cache) selectEntry(key string) (e clob.Entry, err error) {
	row := c.db.QueryRow(sqlSelectEntryViaKey, key)

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
	return e, err
}

func (c Cache) upsertEntry(e clob.Entry) (err error) {
	if err = c.insertEntry(e); err != nil {
		err = c.updateEntry(e)
	}
	return
}

func (c Cache) updateEntry(e clob.Entry) (err error) {
	// Try fetching existing entry
	if existingEntry, err := c.selectEntry(e.Key); err == nil {
		cols, vals := compareEntries(existingEntry, e)
		for i, col := range cols {
			_, err = c.db.Exec(fmt.Sprintf(sqlUpdateEntryViaKey, col), vals[i], e.Key)
		}
	}
	return
}

func (c Cache) insertEntry(e clob.Entry) (err error) {
	var result sql.Result
	if e.Body == nil {
		result, err = c.db.Exec(
			sqlInsertEntryBase,
			e.Key,
			e.ParentKey,
			e.Name,
			e.Type,
		)
	} else {
		result, err = c.db.Exec(
			sqlInsertEntryComplete,
			e.Key,
			e.ParentKey,
			e.Name,
			e.Type,
			e.Size,
			e.LastModified.Unix(),
		)
	}
	if err == nil {
		var num int64
		if num, err = result.RowsAffected(); err == nil && num == 0 {
			err = errNoRowsChanged
		}
	}
	return
}

func (c Cache) deleteEntry(e clob.Entry) (err error) {
	_, err = c.db.Exec(sqlDeleteEntry, e.Key)
	return
}

func (c Cache) insertJob(key string, action int) (err error) {
	var result sql.Result
	if result, err = c.db.Exec(sqlInsertJob, key, action); err == nil {
		var num int64
		if num, err = result.RowsAffected(); err == nil && num == 0 {
			err = errNoRowsChanged
		}
	}
	return
}

func (c Cache) selectNextJob() (j Job, err error) {
	err = c.db.QueryRow(sqlGetNextJob).Scan(&j.ID, &j.Key, &j.Action)
	return
}

func (c Cache) deleteJob(j Job) (err error) {
	_, err = c.db.Exec(sqlDeleteJob, j.ID)
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

func isValidAction(action int) bool {
	switch action {
	case ActionList, ActionUpload, ActionDownload, ActionUpdate, ActionDelete:
		return true
	default:
		return false
	}
}
