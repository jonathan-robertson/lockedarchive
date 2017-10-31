package secure

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

const (

	// KeySize represents the size of the key in bytes
	KeySize = 32 // 256-bit

	// NonceSize represents the size of the nonce in bytes
	NonceSize = 24 // 192-bit
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

// IncrementNonce treats the received nonce as a big-endian value and increments it
func IncrementNonce(nonce *[NonceSize]byte) {
	for i := NonceSize - 1; i > 0; i-- {
		nonce[i]++
		if nonce[i] != 0 {
			break
		}
	}
}

// Encrypt generates a random nonce and encrypts the input using
// NaCl's secretbox package. The nonce is prepended to the ciphertext.
// A sealed message will the same size as the original message plus
// secretbox.Overhead bytes long.
// REVIEW: Keep in mind that random nonces are not always the right choice.
// We’ll talk more about this in a chapter on key exchanges, where we’ll talk
// about how we actually get and share the keys that we’re using.
func Encrypt(key *[KeySize]byte, nonce *[NonceSize]byte, message []byte) ([]byte, error) {
	fmt.Printf("encrypting chunk of size %d\n", len(message)) // TODO: remove (testing)

	out := make([]byte, len(nonce))
	copy(out, nonce[:])
	out = secretbox.Seal(out, message, nonce, key)
	return out, nil
}

// Decrypt extracts the nonce from the ciphertext, and attempts to
// decrypt with NaCl's secretbox.
func Decrypt(key *[KeySize]byte, message []byte) ([]byte, error) {
	fmt.Printf("decrypting chunk of size %d\n", len(message)) // TODO: remove (testing)

	if len(message) < (NonceSize + secretbox.Overhead) {
		return nil, ErrDecrypt
	}

	nonce := new([NonceSize]byte)
	copy(nonce[:], message[:NonceSize])
	out, ok := secretbox.Open(nil, message[NonceSize:], nonce, key)
	if !ok {
		return nil, ErrDecrypt
	}

	return out, nil
}
