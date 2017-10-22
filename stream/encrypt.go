package stream

import (
	"bytes"
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

// REVIEW: https://blog.cloudflare.com/recycling-memory-buffers-in-go/
// manage memory carefully
// TODO: https://stackoverflow.com/questions/16971741/how-do-you-clear-a-slice-in-go
// I love you, internet

// Encrypt encrypts a stream of data in chunks; CALLER MUST USE TooLargeToChunk
// to determine if it's safe to finish running this
// TODO: explore async options - probably use io.ReadSeeker
// TODO: Security Issue: TooLargeToChunk can help when the size of stream is known, but when it is now, we need to monitor for repeating Nonce and throw error if one is encountered!!
func Encrypt(ctx context.Context, key [KeySize]byte, r io.Reader, w io.Writer) (int64, error) {
	var (
		singleByte = make([]byte, 1)                                       // used to pull single byte from r
		buf        = bytes.NewBuffer(make([]byte, 0, EncryptionChunkSize)) // collect bytes for encryption
		written    int64
	)

	for nonce := GenerateNonce(); ; nonce = IncrementNonce(nonce) {
		select {
		case <-ctx.Done():
			return written, ctx.Err()

		default:
			// BUG: if slowly receiving bytes (from gzip, for example),
			// then Read won't get the full chunk size even if there are
			// more bytes on the way. Instead, it's necessary to buffer
			// the bytes read until we either hit an err of io.EOF, or
			// we fill a chunk.

			// Read single byte, return err if non-EOF is received
			_, readErr := r.Read(singleByte)
			if readErr != nil && readErr != io.EOF {
				return written, readErr
			}

			if readErr != io.EOF {
				// Add byte to buffer
				if err := buf.WriteByte(singleByte[0]); err != nil {
					return written, err
				}

				// See if we have enough data to encrypt
				if buf.Len() < EncryptionChunkSize {
					continue // get more data
				}
			}

			// During EOF wrapup, ensure we don't try to encrypt 0 bytes
			// BUG: buf.Len() will be > 0 if on second chunk or beyond because buf.Next does not remove bytes from buf!!
			// TODO: get away from using bufio... instead, just write a simple package for reading from reader into a byte slice until the slice is either full or we get an error (successful return on either full slice or on io.EOF, copying chunk to new slice matching appropriate size). This sounds a lot like .Read, but it isn't: .Read(p) will read up to p bytes that are available at the time of operation even if io.EOF hasn't been reached yet. Normally this wouldn't be necessary, but I have to encrypt bytes in the same lengths so I can (1) use my chunkSize to limit overhead and (2) be sure that I'm pulling bytes out of an encrypted file in the "same" sized slices that I put them in at (i.e. I want to fetch chunk 3 instead of parts of chunk 1 and 2 when it's time to fetch chunk 3)
			if buf.Len() > 0 {
				chunk := make([]byte, EncryptionChunkSize)

				chunkLen, err := buf.Read(chunk)
				if err != nil && err != io.EOF {
					return written, err
				}

				encryptedChunk, err := EncryptBytes(key, nonce, chunk[:chunkLen])
				if err != nil {
					fmt.Println("EncryptBytes:", err) // TODO: remove (testing)
					return written, err
				}

				num, err := w.Write(encryptedChunk)
				if err != nil {
					fmt.Println("Write:", err) // TODO: remove (testing)
					return written, err
				}
				written += int64(num)

				// Since this was EOF and we had a chance to encrypt the final chunk, it's time to return
				if readErr == io.EOF {
					return written, nil
				}
			}
		}
	}
}

// Decrypt decrypts a stream of data in chunks
// TODO: explore async options - probably use io.ReadSeeker
// TODO: update to generate initial nonce, then increment for subsequent chunks
func Decrypt(ctx context.Context, key [KeySize]byte, r io.Reader, w io.Writer) (int64, error) {
	var (
		singleByte = make([]byte, 1)                                       // used to pull single byte from r
		buf        = bytes.NewBuffer(make([]byte, 0, DecryptionChunkSize)) // collect bytes for decryption
		written    int64                                                   // record how many bytes are written to w
	)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("CONTEXT ERROR HIT") // TODO: remove (testing)
			return written, ctx.Err()

		default:

			// Read single byte, return err if non-EOF is received
			_, readErr := r.Read(singleByte)
			if readErr != nil && readErr != io.EOF {
				return written, readErr
			}

			if readErr != io.EOF {
				// Add byte to buffer
				if err := buf.WriteByte(singleByte[0]); err != nil {
					return written, err
				}

				// See if we have enough data to decrypt
				if buf.Len() < DecryptionChunkSize {
					continue // get more data
				}
			}

			// During EOF wrapup, ensure we don't try to decrypt 0 bytes
			if buf.Len() > 0 {
				decryptedChunk, err := DecryptBytes(key, buf.Next(DecryptionChunkSize))
				if err != nil {
					fmt.Println("DecryptBytes:", err) // TODO: remove (testing)
					return written, err
				}

				num, err := w.Write(decryptedChunk)
				if err != nil {
					fmt.Println("Write:", err) // TODO: remove (testing)
					return written, err
				}
				written += int64(num)
			}

			// Since this was EOF and we had a chance to write decrypt final chunk, it's time to return
			if readErr == io.EOF {
				return written, nil
			}
		}
	}
}

// TooLargeToChunk determines if a file is too large to safely chunk,
// considering our ChunkSize and NonceSize
func TooLargeToChunk(size int64) bool {
	numOfChunks := (float64)(size / EncryptionChunkSize)
	if numOfChunks > maxChunkCount {
		return true
	}
	return false
}
