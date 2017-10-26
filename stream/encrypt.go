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

	// EncryptionChunkSize represents the number of bytes to encrypt in each chunk
	EncryptionChunkSize = 3927 // 16kb

	// DecryptionChunkSize represents the number of bytes we need order to decrypt each chunk
	DecryptionChunkSize = EncryptionChunkSize + NonceSize + secretbox.Overhead
)

var (
	// ErrEncrypt is an error that occurred during encryption
	ErrEncrypt = errors.New("secret: encryption failed")

	// ErrDecrypt is an error that occurred during decryption
	ErrDecrypt = errors.New("secret: decryption failed")

	// ErrEncryptSize is an error that occurred during encryption
	ErrEncryptSize = fmt.Errorf("encrypt: file is too large to safely encrypt with a %d-byte chunk size", EncryptionChunkSize)

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
	fmt.Printf("encrypting chunk of size %d\n", len(message)) // TODO: remove (testing)

	out := make([]byte, len(nonce))
	copy(out, nonce[:])
	out = secretbox.Seal(out, message, &nonce, &key)
	return out, nil
}

// DecryptBytes extracts the nonce from the ciphertext, and attempts to
// decrypt with NaCl's secretbox.
func DecryptBytes(key [KeySize]byte, message []byte) ([]byte, error) {
	fmt.Printf("decrypting chunk of size %d\n", len(message)) // TODO: remove (testing)

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

// Encrypt encrypts a stream of data in chunks.
// IF SIZE IS KNOWN, caller should first use TooLargeToChunk
func Encrypt(ctx context.Context, key [KeySize]byte, r io.Reader, w io.Writer) (int64, error) {
	var (
		chunk        = make([]byte, EncryptionChunkSize)
		written      int64
		nonce        = GenerateNonce()
		initialNonce [NonceSize]byte
	)
	copy(initialNonce[:], nonce[:])

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		default:
			nonce = IncrementNonce(nonce)

			// Protect against the nonce repeating
			if initialNonce == nonce {
				return 0, ErrEncryptSize
			}

			length, err := GetChunk(ctx, r, chunk)
			if err != nil && err != io.EOF {
				return 0, err
			}

			if length > 0 {
				encryptedChunk, encErr := EncryptBytes(key, nonce, chunk[:length])
				if encErr != nil {
					return 0, encErr
				}

				bytesWritten, writeErr := w.Write(encryptedChunk)
				if writeErr != nil {
					return 0, writeErr
				}
				written += int64(bytesWritten)
			}

			if err == io.EOF {
				return written, nil
			}
		}
	}
}

// Decrypt decrypts a stream of data in chunks
func Decrypt(ctx context.Context, key [KeySize]byte, r io.Reader, w io.Writer) (int64, error) {
	var (
		chunk   = make([]byte, DecryptionChunkSize)
		written int64
	)

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		default:
			length, err := GetChunk(ctx, r, chunk)
			if err != nil && err != io.EOF {
				return 0, err
			}

			if length > 0 {
				decryptedChunk, encErr := DecryptBytes(key, chunk[:length])
				if encErr != nil {
					return 0, encErr
				}

				bytesWritten, writeErr := w.Write(decryptedChunk)
				if writeErr != nil {
					return 0, writeErr
				}
				written += int64(bytesWritten)
			}

			if err == io.EOF {
				return written, nil // EOF marks success
			}
		}
	}
}

// IsTooLargeToChunk determines if a file is too large to safely chunk,
// considering our ChunkSize and NonceSize
func IsTooLargeToChunk(size int64) bool {
	numOfChunks := (float64)(size / EncryptionChunkSize)
	if numOfChunks > maxChunkCount {
		return true
	}
	return false
}

func readAndWriteChunk(ctx context.Context, chunk []byte, key [KeySize]byte, r io.Reader, w io.Writer) (written int64, err error) {
	length, err := GetChunk(ctx, r, chunk)
	if err != nil && err != io.EOF {
		return 0, err
	}

	if length > 0 {
		bytesWritten, writeErr := w.Write(chunk[:length])
		if writeErr != nil {
			return 0, writeErr
		}
		written += int64(bytesWritten)
	}

	return written, err // return err (may be io.EOF)
}
