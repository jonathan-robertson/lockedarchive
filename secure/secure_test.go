package secure_test

import (
	"bytes"
	"testing"

	"github.com/awnumar/memguard"

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

	encrypted := encrypt(t, kc, plaintext)
	decrypted := decrypt(t, kc, encrypted)
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

	keyString, err := secure.EncryptWithSaltToString(pc, kc.Key()[:])
	if err != nil {
		t.Fatal(err)
	}

	dkc, err := secure.DecryptWithSaltFromStringToKey(pc, keyString)
	if err != nil {
		t.Fatal(err)
	}

	if match, err := memguard.Equal(kc.LockedBuffer, dkc.LockedBuffer); err != nil {
		t.Fatal(err)
	} else if !match {
		t.Fatal("key before encryption does not match key after decryption")
	}

	t.Log("key successfully encrypted and decrypted")
}

func assertBytesEqual(t *testing.T, x, y []byte) {
	if !bytes.Equal(x, y) {
		t.Fatalf("byte slices do not equal\nx: %s\ny: %s", x, y)
	}
}

func encrypt(t *testing.T, kc *secure.KeyContainer, plaintext []byte) []byte {
	return secure.Encrypt(kc, makeNonce(t), plaintext)
}

func decrypt(t *testing.T, kc *secure.KeyContainer, ciphertext []byte) []byte {
	decrypted, err := secure.Decrypt(kc, ciphertext)
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
