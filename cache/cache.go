package cache

import (
	"encoding"
	"time"

	"github.com/boltdb/bolt"
)

var (
	db      *bolt.DB
	bktName []byte
)

// Open initializes the database and is required before interacting with this package
func Open(databaseName, bucketName string) (err error) {
	db, err = bolt.Open(databaseName, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}

	bktName = []byte(bucketName)
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bktName))
		return err
	})
}

// Close shuts down this package and is required before another process can interact with the db
func Close() (err error) {
	return db.Close()
}

// Get retrieves object found at key
func Get(key string, v encoding.BinaryUnmarshaler) error {
	return db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktName)
		return v.UnmarshalBinary(bkt.Get([]byte(key)))
	})
}

// Put stores object in key
func Put(key string, v encoding.BinaryMarshaler) error {
	data, err := v.MarshalBinary()
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bktName).Put([]byte(key), data)
	})
}
