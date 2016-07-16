package crypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

var (
	block cipher.Block
)

// InitializeBlock sets the blockchain to use the provided key for encryption and decryption
func InitializeBlock(key string) {
	var err error
	block, err = aes.NewCipher([]byte(key))
	if err != nil {
		panic(err)
	}
}

// Encrypt receives plaintext bytes and returns encrypted bytes
// Based heavily off of https://golang.org/pkg/crypto/cipher/#example_NewCBCEncrypter
func Encrypt(plaintext []byte) (ciphertext []byte, err error) {
	if block == nil {
		panic("blockchain not initialized with key")
	}

	// CBC mode works on blocks so plaintexts may need to be padded to the
	// next whole block. For an example of such padding, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. Here we'll
	// assume that the plaintext is already of the correct length.
	if len(plaintext)%aes.BlockSize != 0 {
		paddingLength := aes.BlockSize - (len(plaintext) % aes.BlockSize) // Calculate length needed

		padding := make([]byte, paddingLength)    // Make byte slice with zeros for padding
		plaintext = append(plaintext, padding...) // Add zero padding to the end of plaintext bytes
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext = make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	// It's important to remember that ciphertexts must be authenticated
	// (i.e. by using crypto/hmac) as well as being encrypted in order to
	// be secure.

	return
}

// EncryptStringToHexString receives an unencrypted string and returns an encrypted hex string
func EncryptStringToHexString(unencryptedString string) (encryptedHexString string, err error) {

	unencryptedBytes := []byte(unencryptedString)

	encryptedBytes, err := Encrypt(unencryptedBytes)
	if err != nil {
		return
	}

	encryptedHexString = hex.EncodeToString(encryptedBytes)

	return
}

// Decrypt receives encrypted bytes and returns decrypted bytes
// Based heavily off of https://golang.org/pkg/crypto/cipher/#example_NewCBCDecrypter
func Decrypt(ciphertext []byte) (plaintext []byte, err error) {
	if block == nil {
		panic("blockchain not initialized with key")
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		err = errors.New("ciphertext too short")
		return
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// CBC mode always works in whole blocks.
	if len(ciphertext)%aes.BlockSize != 0 {
		err = errors.New("ciphertext is not a multiple of the block size")
		return
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(ciphertext, ciphertext)

	// If the original plaintext lengths are not a multiple of the block
	// size, padding would have to be added when encrypting, which would be
	// removed at this point. For an example, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. However, it's
	// critical to note that ciphertexts must be authenticated (i.e. by
	// using crypto/hmac) before being decrypted in order to avoid creating
	// a padding oracle.

	ciphertext = bytes.TrimRight(ciphertext, "\x00") // REVIEW: is it safe to just trim padding characters off the end?

	return ciphertext, nil
}

// DecryptHexStringToString receives an encrypted hex string and returns a decrypted string
func DecryptHexStringToString(encryptedHexString string) (decryptedString string, err error) {

	encryptedBytes, err := hex.DecodeString(encryptedHexString)
	if err != nil {
		return
	}

	decryptedBytes, err := Decrypt(encryptedBytes)
	if err != nil {
		return
	}

	decryptedString = string(decryptedBytes)

	return
}
