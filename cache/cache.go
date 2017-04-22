// Package cache is responsible for managing the local cache
package cache

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"os"

	"github.com/puddingfactory/lockedarchive/cryptopasta"
)

// TODO: init should reach out for the user's configuration to get key
var key *[32]byte // temporary key

// Write file to the cache; closes file
func Write(source, destination *os.File) (err error) {

	// Read all contents of file
	uncompressed, err := ioutil.ReadAll(source)
	if err != nil {
		return
	}

	// Compress
	compressedtext, err := compress(uncompressed)
	if err != nil {
		return
	}
	if err = source.Close(); err != nil {
		return
	}

	// Encrypt
	ciphertext, err := cryptopasta.Encrypt(compressedtext, key)
	if err != nil {
		return
	}

	// Write
	if _, err = destination.Write(ciphertext); err != nil {
		return
	}
	return destination.Close()
}

// Read file from the cache if it exists
func Read(name string) (plaintext []byte, err error) {

	// Get file if exists
	file, err := os.Open(name)
	if err != nil {
		return
	}

	// Read file contents
	ciphertext, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	// Close file
	if err = file.Close(); err != nil {
		return
	}

	// Decrypt
	compressed, err := cryptopasta.Decrypt(ciphertext, key)
	if err != nil {
		return
	}

	// Decompress
	plaintext, err = decompress(compressed)
	if err != nil {
		return
	}

	return
}

// compress responsible for compressing contents of a file
// REVIEW: update one day to support streaming
func compress(uncompressed []byte) (compressed []byte, err error) {

	// Compress file contents
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err = zw.Write(uncompressed); err != nil {
		return
	}

	// Close compression writer
	if err = zw.Close(); err != nil {
		return
	}

	compressed = buf.Bytes()
	return
}

// decompress responsible for decompressing contents
// REVIEW: update one day to support streaming
func decompress(compressed []byte) (decompressed []byte, err error) {

	// Create new decompression reader
	zr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return
	}

	// Read into bytes
	decompressed, err = ioutil.ReadAll(zr)
	if err != nil {
		return
	}

	// Close decompression reader
	err = zr.Close()
	return
}
