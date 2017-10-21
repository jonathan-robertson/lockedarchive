package stream_test

import (
	"context"
	"testing"

	"github.com/jonathan-robertson/lockedarchive/stream"
)

const (
	encSrcFilename = "src.txt"
	encWrkFilename = "encrypted.txt"
	encDstFilename = "decrypted.txt"
)

func TestEncryption(t *testing.T) {
	key := stream.GenerateKey()
	t.Run("Encrypt", func(t *testing.T) { runEncryption(t, key) })
	t.Run("Decrypt", func(t *testing.T) { runDecryption(t, key) })
	compareAndCleanup(t, encSrcFilename, encWrkFilename, encDstFilename)
}

func runEncryption(t *testing.T, key [stream.KeySize]byte) {
	src, dst := setup(t, encSrcFilename, encWrkFilename)
	defer src.Close()
	defer dst.Close()

	if err := stream.Encrypt(context.Background(), key, src, dst); err != nil {
		t.Fatal(err)
	}

	if err := dst.Sync(); err != nil {
		t.Fatal(err)
	}

	t.Logf("successfully encrypted data from %s to %s", encSrcFilename, encWrkFilename)
}

func runDecryption(t *testing.T, key [stream.KeySize]byte) {
	src, dst := setup(t, encWrkFilename, encDstFilename)
	defer src.Close()
	defer dst.Close()

	if err := stream.Decrypt(context.Background(), key, src, dst); err != nil {
		t.Fatal(err)
	}

	if err := dst.Sync(); err != nil {
		t.Fatal(err)
	}

	t.Logf("successfully decrypted data from %s to %s", encWrkFilename, encDstFilename)
}
