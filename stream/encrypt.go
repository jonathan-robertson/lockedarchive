package stream

import (
	"context"
	"fmt"
	"io"
	"math"

	"golang.org/x/crypto/nacl/secretbox"

	"github.com/jonathan-robertson/lockedarchive/secure"
)

const (

	// EncryptionChunkSize represents the number of bytes to encrypt in each chunk
	EncryptionChunkSize = 3927 // 16kb

	// DecryptionChunkSize represents the number of bytes we need order to decrypt each chunk
	DecryptionChunkSize = EncryptionChunkSize + secure.NonceSize + secretbox.Overhead
)

var (

	// ErrEncryptSize is an error that occurred during encryption
	ErrEncryptSize = fmt.Errorf("encrypt: file is too large to safely encrypt with a %d-byte chunk size", EncryptionChunkSize)

	maxChunkCount = math.Exp2(secure.NonceSize)
)

// Encrypt encrypts a stream of data in chunks.
// IF SIZE IS KNOWN, caller should first use TooLargeToChunk
func Encrypt(ctx context.Context, key secure.Key, r io.Reader, w io.Writer) (int64, error) {
	nonce, err := secure.GenerateNonce()
	if err != nil {
		return 0, err
	}

	// Record nonce starting point to protect against looping of nonce
	initialNonce := new([secure.NonceSize]byte)
	copy(initialNonce[:], nonce[:])

	var written int64
	chunk := make([]byte, EncryptionChunkSize)
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		default:
			secure.IncrementNonce(nonce)

			// Protect against the nonce repeating
			if initialNonce == nonce {
				return 0, ErrEncryptSize
			}

			length, err := GetChunk(ctx, r, chunk)
			if err != nil && err != io.EOF {
				return 0, err
			}

			if length > 0 {
				encryptedChunk, encErr := secure.Encrypt(key, nonce, chunk[:length])
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
func Decrypt(ctx context.Context, key secure.Key, r io.Reader, w io.Writer) (int64, error) {
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
				decryptedChunk, encErr := secure.Decrypt(key, chunk[:length])
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

// REVIEW: Not in use... will this ever be used?
func readAndWriteChunk(ctx context.Context, chunk []byte, r io.Reader, w io.Writer) (written int64, err error) {
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
