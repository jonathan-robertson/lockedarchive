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
func Open(cabinet string) (cache Cache, err error) {
	cache = Cache{Cabinet: cabinet}
	if cache.db, err = sql.Open("sqlite3", cache.filename()); err != nil {
		if fileDoesNotExist(cache.filename()) { // handle if file simply doesn't exist
			if err = os.MkdirAll(filepath.Join(cacheRoot, cache.Cabinet), cacheMode); err == nil {
				if cache.db, err = sql.Open("sqlite3", cache.filename()); err != nil {
					_, err = cache.db.Exec(sqlCreateTables)
				}
			}
		}
	}

	if err = cache.db.Ping(); err != nil {
		fmt.Println("ERROR DURING PING", err)
	} else {
		fmt.Println("PING SUCCESS")
	}

	return
}

// Close shuts down the connection to the database
func (cache Cache) Close() error {
	return cache.db.Close()
}

// RecallEntry returns the entry (including its data file if cached)
func (cache Cache) RecallEntry(key string) (entry clob.Entry, err error) {
	if entry, err = cache.selectEntry(key); err == nil {
		if file, err := os.Open(filepath.Join(cacheRoot, cache.Cabinet, key)); err == nil {
			entry.Body = file // link file, ready for reading
		} else if os.IsNotExist(err) {
			err = nil // ignore error if the file just doesn't happen to exist
		}
	}
	return
}

// RememberEntry records the entry's file and metadata to cache
func (cache Cache) RememberEntry(entry clob.Entry) (err error) {
	if entry.Body != nil {
		defer entry.Body.Close() // be sure to close this even if err on upsertEntry
	}

	if err = cache.upsertEntry(entry); err == nil {
		if entry.Body == nil {
			err = deleteFileIfExists(filepath.Join(cacheRoot, cache.Cabinet, entry.Key))
		} else if cacheFile, err := os.Create(filepath.Join(cacheRoot, cache.Cabinet, entry.Key)); err == nil {
			defer cacheFile.Close()
			_, err = io.Copy(cacheFile, entry.Body)
		}
	}
	return
}

// ForgetEntry purges the entry's cache data and info from db
func (cache Cache) ForgetEntry(entry clob.Entry) (err error) {
	if err = deleteFileIfExists(filepath.Join(cacheRoot, cache.Cabinet, entry.Key)); err == nil {
		err = cache.deleteEntry(entry) // remove entry from db
	}
	return
}

// ContainsEntry returns if an entry exists at provided id
func (cache Cache) ContainsEntry(key string) (exists bool) {
	if err := cache.db.QueryRow(sqlSelectEntryExists, key).Scan(&exists); err != nil {
		log.Println(err)
	}
	return
}

// EnqueueJob queues a new job
func (cache Cache) EnqueueJob(key string, action int) (err error) {
	if !isValidAction(action) {
		return errInvalidAction
	}
	return cache.insertJob(key, action)
}

// DequeueJob is for fetching the contents of the next job in the queued
func (cache Cache) DequeueJob() (j Job, err error) {
	if j, err = cache.selectNextJob(); err == nil {
		err = cache.deleteJob(j)
	}
	return
}

func (cache Cache) filename() string {
	return filepath.Join(cacheRoot, cache.Cabinet, cacheFilename)
}

func (cache *Cache) selectEntry(key string) (entry clob.Entry, err error) {
	row := cache.db.QueryRow(sqlSelectEntryViaKey, key)

	var size, unixTimestamp int64
	err = row.Scan(
		&entry.Key,
		&entry.ParentKey,
		&entry.Name,
		&entry.Type,
		&size,
		&unixTimestamp,
	)
	if size != 0 {
		entry.Size = size
	}
	if unixTimestamp != 0 {
		entry.LastModified = time.Unix(unixTimestamp, 0)
	}
	return entry, err
}

func (cache Cache) upsertEntry(entry clob.Entry) (err error) {
	if err = cache.insertEntry(entry); err != nil {
		err = cache.updateEntry(entry)
	}
	return
}

func (cache Cache) updateEntry(entry clob.Entry) (err error) {
	// Try fetching existing entry
	if existingEntry, err := cache.selectEntry(entry.Key); err == nil {
		cols, vals := compareEntries(existingEntry, entry)
		for i, col := range cols {
			_, err = cache.db.Exec(fmt.Sprintf(sqlUpdateEntryViaKey, col), vals[i], entry.Key)
		}
	}
	return
}

func (cache Cache) insertEntry(entry clob.Entry) (err error) {
	var result sql.Result
	if entry.Body == nil {
		result, err = cache.db.Exec(
			sqlInsertEntryBase,
			entry.Key,
			entry.ParentKey,
			entry.Name,
			entry.Type,
		)
	} else {
		result, err = cache.db.Exec(
			sqlInsertEntryComplete,
			entry.Key,
			entry.ParentKey,
			entry.Name,
			entry.Type,
			entry.Size,
			entry.LastModified.Unix(),
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

func (cache Cache) deleteEntry(entry clob.Entry) (err error) {
	_, err = cache.db.Exec(sqlDeleteEntry, entry.Key)
	return
}

func (cache Cache) insertJob(key string, action int) (err error) {
	var result sql.Result
	if result, err = cache.db.Exec(sqlInsertJob, key, action); err == nil {
		var num int64
		if num, err = result.RowsAffected(); err == nil && num == 0 {
			err = errNoRowsChanged
		}
	}
	return
}

func (cache Cache) selectNextJob() (j Job, err error) {
	err = cache.db.QueryRow(sqlGetNextJob).Scan(&j.ID, &j.Key, &j.Action)
	return
}

func (cache Cache) deleteJob(j Job) (err error) {
	_, err = cache.db.Exec(sqlDeleteJob, j.ID)
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
