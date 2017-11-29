package secure_test

import (
	"bytes"
	"testing"

	"github.com/jonathan-robertson/lockedarchive/secure"
)

const (
	plaintext = "Text that is plain"
)

func TestDerriveKeyContainer(t *testing.T) {
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

	t.Logf("key based on passphrase was successfully generated")
	kc.Destroy()
}

func TestEncrypt(t *testing.T) {
	kc := makeKeyContainer(t)
	defer kc.Destroy()

	plaintext := []byte("string to test")

	encrypted := encrypt(t, kc.Key(), plaintext)
	decrypted := decrypt(t, kc.Key(), encrypted)
	assertBytesEqual(t, plaintext, decrypted)

	t.Log("data encrypted and decrypted to get same result")
}

func TestEncryptKeyToString(t *testing.T) {
	passphrase := []byte("test passphrase!")
	pc, err := secure.ProtectPassphrase(passphrase)
	if err != nil {
		t.Fatal(err)
	}

	kc, err := secure.GenerateKeyContainer()
	if err != nil {
		t.Fatal(err)
	}

	keyString, err := secure.EncryptKeyToString(pc, kc)
	if err != nil {
		t.Fatal(err)
	}

	dkc, err := secure.DecryptKeyFromString(pc, keyString)
	if err != nil {
		t.Fatal(err)
	}

	assertBytesEqual(t, kc.Key()[:], dkc.Key()[:])
	t.Log("key successfully encrypted and decrypted")
}

func assertBytesEqual(t *testing.T, x, y []byte) {
	if !bytes.Equal(x, y) {
		t.Fatalf("byte slices do not equal\nx: %s\ny: %s", x, y)
	}
}

func encrypt(t *testing.T, key secure.Key, plaintext []byte) []byte {
	return secure.Encrypt(key, makeNonce(t), plaintext)
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
