// Package cache is responsible for managing the local cache
package cache

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/jonathan-robertson/lockedarchive/stream"
)

// Encode compresses and encrypts a file at provided path, writing it to the cache
func Encode(source string, key [stream.KeySize]byte) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(src.Name() + ".la")
	if err != nil {
		return err
	}
	defer dst.Close()

	pr, pw := io.Pipe()
	defer pw.Close()
	ctx, cancel := context.WithCancel(context.Background())

	// Compress data
	var compressionErr error
	go func() {
		if _, compressionErr = stream.Compress(src, pw); compressionErr != nil {
			cancel()
		}
		if compressionErr = pw.Close(); compressionErr != nil {
			cancel()
		}
	}()

	// Encrypt data
	if _, err = stream.Encrypt(ctx, key, pr, dst); err != nil {
		if err == context.Canceled {
			return compressionErr
		}
		return err
	}

	return dst.Sync()
}

// Decode decrypts and decompresses a file at provided path
func Decode(source string, key [stream.KeySize]byte) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(strings.TrimSuffix(src.Name(), ".la"))
	if err != nil {
		return err
	}
	defer dst.Close()

	pr, pw := io.Pipe()
	defer pw.Close()
	ctx, cancel := context.WithCancel(context.Background())

	// var decompressionErr error
	// go func() {
	// 	_, decompressionErr = stream.Decompress(pr, dst)
	// 	cancel()
	// }()

	// if err := stream.Decrypt(ctx, key, src, pw); err != nil {
	// 	if err == context.Canceled {
	// 		return decompressionErr
	// 	}
	// 	return err
	// }
	// if err := pw.Close(); err != nil {
	// 	return err
	// }

	// <-ctx.Done()
	// return dst.Sync()

	// Decrypt data
	var decryptionErr error
	go func() {
		if _, decryptionErr = stream.Decrypt(ctx, key, src, pw); decryptionErr != nil {
			cancel() // TODO: THIS DOESN'T DO REALLY ANYTHING
		}
		if decryptionErr = pw.Close(); decryptionErr != nil {
			cancel() // TODO: THIS DOESN'T DO REALLY ANYTHING
		}
	}()

	// Decompress data
	if _, err := stream.Decompress(pr, dst); err != nil {
		return err
	}

	return dst.Sync()
}

// // TODO: init should reach out for the user's configuration to get key
// var key *[32]byte // temporary key

// // Write compresses and encrypts file, writing it to the cache; closes file
// func Write(source, destination *os.File) (err error) {

// 	// Read all contents of file
// 	uncompressed, err := ioutil.ReadAll(source)
// 	if err != nil {
// 		return
// 	}

// 	// Compress
// 	compressedtext, err := compress(uncompressed)
// 	if err != nil {
// 		return
// 	}
// 	if err = source.Close(); err != nil {
// 		return
// 	}

// 	// Encrypt
// 	ciphertext, err := cryptopasta.Encrypt(compressedtext, key)
// 	if err != nil {
// 		return
// 	}

// 	// Write
// 	if _, err = destination.Write(ciphertext); err != nil {
// 		return
// 	}
// 	return destination.Close()
// }

// // Read decrypts and decompresses file, returning plaintext
// func Read(name string) (plaintext []byte, err error) {

// 	// Get file if exists
// 	file, err := os.Open(name)
// 	if err != nil {
// 		return
// 	}

// 	// Read file contents
// 	ciphertext, err := ioutil.ReadAll(file)
// 	if err != nil {
// 		return
// 	}

// 	// Close file
// 	if err = file.Close(); err != nil {
// 		return
// 	}

// 	// Decrypt
// 	compressed, err := cryptopasta.Decrypt(ciphertext, key)
// 	if err != nil {
// 		return
// 	}

// 	// Decompress
// 	plaintext, err = decompress(compressed)
// 	if err != nil {
// 		return
// 	}

// 	return
// }

// // Put adds a file to the cache without any modifications
// // This is best used with data received from cloud storage
// // TODO: update to use cache path
// func Put(name string, rc io.ReadCloser) (err error) {
// 	file, err := os.Create(name)
// 	if err != nil {
// 		return
// 	}

// 	_, err = io.Copy(file, rc)
// 	file.Close()
// 	return
// }

// // Get returns readCloser to a cached file; caller responsible for closing
// // This is best used for providing cloud storage the bytes to transmit
// // TODO: update to use cache path
// func Get(name string) (*os.File, error) {
// 	return os.Open(name)
// }

// // compress responsible for compressing contents of a file
// // REVIEW: update one day to support streaming
// func compress(uncompressed []byte) (compressed []byte, err error) {

// 	// Compress file contents
// 	var buf bytes.Buffer
// 	zw := gzip.NewWriter(&buf)
// 	if _, err = zw.Write(uncompressed); err != nil {
// 		return
// 	}

// 	// Close compression writer
// 	if err = zw.Close(); err != nil {
// 		return
// 	}

// 	compressed = buf.Bytes()
// 	return
// }

// // decompress responsible for decompressing contents
// // REVIEW: update one day to support streaming
// func decompress(compressed []byte) (decompressed []byte, err error) {

// 	// Create new decompression reader
// 	zr, err := gzip.NewReader(bytes.NewReader(compressed))
// 	if err != nil {
// 		return
// 	}

// 	// Read into bytes
// 	decompressed, err = ioutil.ReadAll(zr)
// 	if err != nil {
// 		return
// 	}

// 	// Close decompression reader
// 	err = zr.Close()
// 	return
// }
