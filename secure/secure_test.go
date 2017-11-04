package secure_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jonathan-robertson/lockedarchive/secure"
)

const (
	srcFilename = "src.txt"
)

func TestSecure(t *testing.T) {
	kc := makeKeyContainer(t)
	defer kc.Destroy()

	plaintext := read(t)
	encrypted := encrypt(t, plaintext, kc.Key())
	decrypted := decrypt(t, encrypted, kc.Key())

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("plaintext does not equal decrypted text")
	}
	t.Log("data encrypted and decrypted to get same result")
}

func read(t *testing.T) []byte {
	file, err := os.Open(srcFilename)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func encrypt(t *testing.T, plaintext []byte, key secure.Key) []byte {
	ciphertext, err := secure.Encrypt(key, makeNonce(t), plaintext)
	if err != nil {
		t.Fatal(err)
	}
	return ciphertext
}

func decrypt(t *testing.T, ciphertext []byte, key secure.Key) []byte {
	plaintext, err := secure.Decrypt(key, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	return plaintext
}

func makeNonce(t *testing.T) secure.Nonce {
	nonce, err := secure.GenerateNonce()
	if err != nil {
		t.Fatal(err)
	}
	return nonce
}

func makeKeyContainer(t *testing.T) *secure.Container {
	container, err := secure.GenerateKeyContainer()
	if err != nil {
		t.Fatal(err)
	}
	return container
}
