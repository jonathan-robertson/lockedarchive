package secure_test

import (
	"bytes"
	"testing"

	"github.com/jonathan-robertson/lockedarchive/secure"
)

const (
	plaintext = "Text that is plain"
)

func TestPassphrase(t *testing.T) {
	passphrase := []byte("test passphrase!")

	pc, err := secure.ProtectPassphrase(passphrase)
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Destroy()

	salt, err := secure.GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}

	kc, err := pc.DeriveKeyContainer(salt)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("key generated: %x", kc.Key())
	kc.Destroy()
}

func TestSecure(t *testing.T) {
	kc := makeKeyContainer(t)
	defer kc.Destroy()

	plaintext := "string to test"

	// Make a copy of plaintext since we need to verify it matches decrypted bytes
	plaintextCopy := make([]byte, len(plaintext))
	copy(plaintextCopy, []byte(plaintext))

	encrypted := encryptAndWipe(t, kc.Key(), plaintextCopy)
	decrypted := decrypt(t, kc.Key(), encrypted)
	assertBytesEqual(t, []byte(plaintext), decrypted)

	t.Log("data encrypted and decrypted to get same result")
}

func assertBytesEqual(t *testing.T, x, y []byte) {
	if !bytes.Equal(x, y) {
		t.Fatal("plaintext does not equal decrypted text")
	}
}

func encryptAndWipe(t *testing.T, key secure.Key, plaintext []byte) []byte {
	return secure.EncryptAndWipe(key, makeNonce(t), plaintext)
}

func decrypt(t *testing.T, key secure.Key, ciphertext []byte) []byte {
	decrypted, err := secure.Decrypt(key, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	return decrypted
}

func makeNonce(t *testing.T) secure.Nonce {
	nonce, err := secure.GenerateNonce()
	if err != nil {
		t.Fatal(err)
	}
	return nonce
}

func makeKeyContainer(t *testing.T) *secure.KeyContainer {
	kc, err := secure.GenerateKeyContainer()
	if err != nil {
		t.Fatal(err)
	}
	return kc
}
