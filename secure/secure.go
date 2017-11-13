package secure

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"unsafe"

	"github.com/awnumar/memguard"

	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/scrypt"
)

// Container is responsible for securing encryption keys in memory
type Container struct {
	*memguard.LockedBuffer
}

// Nonce is used in encryption and should be random, but not secret
type Nonce = *[NonceSize]byte

// Key is used in encryption and should be kept secret
type Key = *[KeySize]byte

// Salt is used in derriving a key from a passphrase
type Salt = *[SaltSize]byte

const (

	// KeySize represents the size of the key in bytes
	KeySize = 32 // 256-bit

	// NonceSize represents the size of the nonce in bytes
	NonceSize = 24 // 192-bit

	// SaltSize represents the size of the salt in bytes
	SaltSize = 8 // 64-bit
)

var (

	// ErrEncrypt is an error that occurred during encryption
	ErrEncrypt = errors.New("secret: encryption failed")

	// ErrDecrypt is an error that occurred during decryption
	ErrDecrypt = errors.New("secret: decryption failed")
)

// Key returns an unsafe pointer to a byte array for use in encryption/decryption methods
func (container *Container) Key() Key {
	return (Key)(unsafe.Pointer(&container.Buffer()[0]))
}

// REVIEW: https://leanpub.com/gocrypto/read#leanpub-auto-nacl

// GenerateKeyContainer creates a new random secret key inside a safe container
func GenerateKeyContainer() (*Container, error) {
	buf, err := memguard.NewImmutableRandom(KeySize)
	return &Container{LockedBuffer: buf}, err
}

// DeriveKeyContainer generates a new KeyContainer from a passphrase and wipes passphrase's bytes once done, even on err
func DeriveKeyContainer(passphrase []byte, salt Salt) (*Container, error) {
	dk, err := scrypt.Key(passphrase, salt[:], 1<<15, 8, 1, KeySize)
	Wipe(passphrase) // zero bytes from passphrase asap
	if err != nil {
		return nil, err
	}

	buf, err := memguard.NewImmutableFromBytes(dk)
	return &Container{LockedBuffer: buf}, err
}

// GenerateSalt creates a new random Salt
func GenerateSalt() (Salt, error) {
	salt := new([SaltSize]byte)
	_, err := io.ReadFull(rand.Reader, salt[:])
	return salt, err
}

// GenerateNonce creates a new random Nonce
func GenerateNonce() (Nonce, error) {
	nonce := new([NonceSize]byte)
	_, err := io.ReadFull(rand.Reader, nonce[:])
	return nonce, err
}

// IncrementNonce treats the received Nonce as big-endian value and increments it
func IncrementNonce(nonce Nonce) {
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
func Encrypt(key Key, nonce Nonce, message []byte) ([]byte, error) {
	fmt.Printf("encrypting chunk of size %d\n", len(message)) // TODO: remove (testing)

	out := make([]byte, len(nonce))
	copy(out, nonce[:])
	out = secretbox.Seal(out, message, nonce, key)
	Wipe(message) // zero bytes of original message in memory asap
	return out, nil
}

// Decrypt extracts the nonce from the ciphertext, and attempts to
// decrypt with NaCl's secretbox.
func Decrypt(key Key, message []byte) ([]byte, error) {
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

// Wipe attempts to zero out bytes
func Wipe(data []byte) {
	for i := 0; i < len(data); i++ {
		data[i] = 0
	}
}
