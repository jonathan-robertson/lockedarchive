// Package cache is responsible for managing the local cache
package cache

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

const (
	// KeySize represents the size of the key in bytes
	KeySize = 32 // 256-bit

	// NonceSize represents the size of the nonce in bytes
	NonceSize = 24 // 192-bit

	// ChunkSize represents the size of each encrypted chunk in bytes
	ChunkSize = 16384 // 16kb
)

var (
	// ErrEncrypt is an error that occurred during encryption
	ErrEncrypt = errors.New("secret: encryption failed")

	// ErrDecrypt is an error that occurred during decryption
	ErrDecrypt = errors.New("secret: decryption failed")
)

// REVIEW: https://leanpub.com/gocrypto/read#leanpub-auto-nacl

// GenerateKey creates a new random secret key and panics if the source
// of randomness fails
// TODO: harvest more entropy?
func GenerateKey() *[KeySize]byte {
	key := new([KeySize]byte)
	if _, err := io.ReadFull(rand.Reader, key[:]); err != nil {
		panic(err)
	}
	return key
}

// GenerateNonce creates a new random nonce and panics if the source of
// randomness fails
func GenerateNonce() *[NonceSize]byte {
	nonce := new([NonceSize]byte)
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		panic(err)
	}
	return nonce
}

// Encrypt generates a random nonce and encrypts the input using
// NaCl's secretbox package. The nonce is prepended to the ciphertext.
// A sealed message will the same size as the original message plus
// secretbox.Overhead bytes long.
// REVIEW: Keep in mind that random nonces are not always the right choice.
// We’ll talk more about this in a chapter on key exchanges, where we’ll talk
// about how we actually get and share the keys that we’re using.
func Encrypt(key *[KeySize]byte, message []byte) ([]byte, error) {
	nonce := GenerateNonce()

	out := make([]byte, len(nonce))
	copy(out, nonce[:])
	out = secretbox.Seal(out, message, nonce, key)
	return out, nil
}

// Decrypt extracts the nonce from the ciphertext, and attempts to
// decrypt with NaCl's secretbox.
func Decrypt(key *[KeySize]byte, message []byte) ([]byte, error) {
	if len(message) < (NonceSize + secretbox.Overhead) {
		return nil, ErrDecrypt
	}

	var nonce [NonceSize]byte
	copy(nonce[:], message[:NonceSize])
	out, ok := secretbox.Open(nil, message[NonceSize:], &nonce, key)
	if !ok {
		return nil, ErrDecrypt
	}

	return out, nil
}

// REVIEW: https://blog.cloudflare.com/recycling-memory-buffers-in-go/
// manage memory carefully
// TODO: https://stackoverflow.com/questions/16971741/how-do-you-clear-a-slice-in-go
// I love you, internet

// EncryptStream encrypts a stream of data in chunks
// TODO: explore async options - probably use io.ReadSeeker
// TODO: update to generate initial nonce, then increment for subsequent chunks
func EncryptStream(ctx context.Context, key *[KeySize]byte, r io.Reader, w io.Writer) error {
	chunk := make([]byte, ChunkSize)
	// nonce := GenerateNonce()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			len, err := r.Read(chunk)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			encryptedChunk, err := Encrypt(key, chunk[:len])
			if err != nil {
				return err
			}

			len, err = w.Write(encryptedChunk)
			if err != nil {
				return err
			}
		}
	}
}

// DecryptStream decrypts a stream of data in chunks
// TODO: explore async options - probably use io.ReadSeeker
// TODO: update to generate initial nonce, then increment for subsequent chunks
func DecryptStream(ctx context.Context, key *[KeySize]byte, r io.Reader, w io.Writer) error {
	chunk := make([]byte, ChunkSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			len, err := r.Read(chunk)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			decryptedChunk, err := Decrypt(key, chunk[:len])
			if err != nil {
				return err
			}

			len, err = w.Write(decryptedChunk)
			if err != nil {
				return err
			}
		}
	}
}

// CompressStream compresses a stream of data
// TODO: Update to return an io.reader or io.writer?
func CompressStream(r io.Reader, w io.Writer) (written int64, err error) {
	zw := gzip.NewWriter(w)

	written, err = io.Copy(zw, r)
	if err != nil {
		return
	}

	if err = zw.Flush(); err != nil {
		return
	}

	err = zw.Close()
	return
}

// DecompressStream decompresses a stream of data
// TODO: Update to return an io.reader or io.writer?
func DecompressStream(r io.Reader, w io.Writer) (written int64, err error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return
	}

	written, err = io.Copy(w, zr)
	if err != nil {
		return
	}

	err = zr.Close()
	return
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
