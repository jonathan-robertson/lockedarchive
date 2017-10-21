package stream

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"

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

	// ErrEncryptSize is an error that occurred during encryption
	ErrEncryptSize = fmt.Errorf("encrypt: file is too large to safely encrypt with a %d-byte chunk size", ChunkSize)

	maxChunkCount = math.Exp2(NonceSize)
)

// REVIEW: https://leanpub.com/gocrypto/read#leanpub-auto-nacl

// GenerateKey creates a new random secret key and panics if the source
// of randomness fails
// TODO: harvest more entropy?
func GenerateKey() [KeySize]byte {
	key := new([KeySize]byte)
	if _, err := io.ReadFull(rand.Reader, key[:]); err != nil {
		panic(err)
	}
	return *key
}

// GenerateNonce creates a new random nonce and panics if the source of
// randomness fails
func GenerateNonce() [NonceSize]byte {
	nonce := new([NonceSize]byte)
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		panic(err)
	}
	return *nonce
}

// IncrementNonce treats the received nonce as a big-endian value and increments it
func IncrementNonce(nonce [NonceSize]byte) [NonceSize]byte {
	for i := NonceSize - 1; i > 0; i-- {
		nonce[i]++
		if nonce[i] != 0 {
			break
		}
	}
	return nonce
}

// EncryptBytes generates a random nonce and encrypts the input using
// NaCl's secretbox package. The nonce is prepended to the ciphertext.
// A sealed message will the same size as the original message plus
// secretbox.Overhead bytes long.
// REVIEW: Keep in mind that random nonces are not always the right choice.
// We’ll talk more about this in a chapter on key exchanges, where we’ll talk
// about how we actually get and share the keys that we’re using.
func EncryptBytes(key [KeySize]byte, nonce [NonceSize]byte, message []byte) ([]byte, error) {
	out := make([]byte, len(nonce))
	copy(out, nonce[:])
	out = secretbox.Seal(out, message, &nonce, &key)
	return out, nil
}

// DecryptBytes extracts the nonce from the ciphertext, and attempts to
// decrypt with NaCl's secretbox.
func DecryptBytes(key [KeySize]byte, message []byte) ([]byte, error) {
	if len(message) < (NonceSize + secretbox.Overhead) {
		return nil, ErrDecrypt
	}

	var nonce [NonceSize]byte
	copy(nonce[:], message[:NonceSize])
	out, ok := secretbox.Open(nil, message[NonceSize:], &nonce, &key)
	if !ok {
		return nil, ErrDecrypt
	}

	return out, nil
}

// REVIEW: https://blog.cloudflare.com/recycling-memory-buffers-in-go/
// manage memory carefully
// TODO: https://stackoverflow.com/questions/16971741/how-do-you-clear-a-slice-in-go
// I love you, internet

// Encrypt encrypts a stream of data in chunks; CALLER MUST USE TooLargeToChunk
// to determine if it's safe to finish running this
// TODO: explore async options - probably use io.ReadSeeker
// TODO: update to generate initial nonce, then increment for subsequent chunks
func Encrypt(ctx context.Context, key [KeySize]byte, r io.Reader, w io.Writer) error {
	chunk := make([]byte, ChunkSize)

	for nonce := GenerateNonce(); ; nonce = IncrementNonce(nonce) {
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

			encryptedChunk, err := EncryptBytes(key, nonce, chunk[:len])
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

// Decrypt decrypts a stream of data in chunks
// TODO: explore async options - probably use io.ReadSeeker
// TODO: update to generate initial nonce, then increment for subsequent chunks
func Decrypt(ctx context.Context, key [KeySize]byte, r io.Reader, w io.Writer) error {
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

			decryptedChunk, err := DecryptBytes(key, chunk[:len])
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

// TooLargeToChunk determines if a file is too large to safely chunk,
// considering our ChunkSize and NonceSize
func TooLargeToChunk(size int64) bool {
	numOfChunks := (float64)(size / ChunkSize)
	if numOfChunks > maxChunkCount {
		return true
	}
	return false
}
